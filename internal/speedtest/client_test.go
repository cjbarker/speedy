package speedtest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testBackend mimics the Cloudflare __down/__up/latency endpoints.
func testBackend(concurrent *atomic.Int32, maxSeen *atomic.Int32) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/__down", func(w http.ResponseWriter, r *http.Request) {
		if concurrent != nil {
			cur := concurrent.Add(1)
			defer concurrent.Add(-1)
			for {
				m := maxSeen.Load()
				if cur <= m || maxSeen.CompareAndSwap(m, cur) {
					break
				}
			}
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("bytes"))
		w.WriteHeader(http.StatusOK)
		if n <= 0 {
			return
		}
		buf := make([]byte, 32<<10)
		for n > 0 {
			c := len(buf)
			if c > n {
				c = n
			}
			if _, err := w.Write(buf[:c]); err != nil {
				return
			}
			n -= c
		}
	})
	mux.HandleFunc("/__up", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	})
	return httptest.NewServer(mux)
}

func TestMeasureDownload(t *testing.T) {
	var cur, max atomic.Int32
	srv := testBackend(&cur, &max)
	defer srv.Close()

	c := New(Config{
		BaseURL:  srv.URL,
		Streams:  4,
		Duration: 400 * time.Millisecond,
		Warmup:   100 * time.Millisecond,
	})
	mbps, err := c.measureDownload(context.Background(), nil)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if mbps <= 0 {
		t.Fatalf("expected positive throughput, got %v", mbps)
	}
	if max.Load() < 2 {
		t.Fatalf("expected parallel requests, max concurrency = %d", max.Load())
	}
}

func TestMeasureUpload(t *testing.T) {
	srv := testBackend(nil, nil)
	defer srv.Close()

	c := New(Config{
		BaseURL:  srv.URL,
		Streams:  2,
		Duration: 300 * time.Millisecond,
		Warmup:   50 * time.Millisecond,
	})
	mbps, err := c.measureUpload(context.Background(), nil)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if mbps <= 0 {
		t.Fatalf("expected positive throughput, got %v", mbps)
	}
}

func TestMeasureLatency(t *testing.T) {
	srv := testBackend(nil, nil)
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	lat, jit, err := c.measureLatency(context.Background())
	if err != nil {
		t.Fatalf("latency: %v", err)
	}
	if lat <= 0 {
		t.Fatalf("expected positive latency, got %v", lat)
	}
	if jit < 0 {
		t.Fatalf("jitter must be non-negative, got %v", jit)
	}
}

func TestRunContextCancel(t *testing.T) {
	srv := testBackend(nil, nil)
	defer srv.Close()

	c := New(Config{
		BaseURL:  srv.URL,
		Streams:  2,
		Duration: 10 * time.Second,
		Warmup:   0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.Run(ctx, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("Run did not honor context cancellation promptly")
	}
}

func TestProbeSuccess(t *testing.T) {
	srv := testBackend(nil, nil)
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	if err := c.Probe(context.Background()); err != nil {
		t.Fatalf("probe should succeed against test backend: %v", err)
	}
}

func TestProbeUnsupportedServer(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	err := c.Probe(context.Background())
	if err == nil {
		t.Fatal("probe should fail against a server without /__down")
	}
	if !strings.Contains(err.Error(), "does not support the speed test API") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRunResultFields(t *testing.T) {
	srv := testBackend(nil, nil)
	defer srv.Close()

	c := New(Config{
		BaseURL:    srv.URL,
		Streams:    2,
		Duration:   200 * time.Millisecond,
		Warmup:     0,
		DoDownload: true,
		DoUpload:   true,
		DoLatency:  true,
		DoJitter:   true,
	})
	res, err := c.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.HasPrefix(res.Server, "http") {
		t.Fatalf("server not set: %q", res.Server)
	}
	if res.DownloadMbps <= 0 || res.UploadMbps <= 0 {
		t.Fatalf("expected download and upload set, got %+v", res)
	}
}
