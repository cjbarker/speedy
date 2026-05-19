package speedtest

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is Cloudflare's public, no-API-key speed test backend.
const DefaultBaseURL = "https://speed.cloudflare.com"

// Config controls a speed test run.
type Config struct {
	BaseURL    string
	Streams    int
	Duration   time.Duration
	Warmup     time.Duration
	Timeout    time.Duration
	DoDownload bool
	DoUpload   bool
	DoLatency  bool
	DoJitter   bool
}

// Client runs speed tests against a backend.
type Client struct {
	http *http.Client
	cfg  Config
}

// New builds a Client with a transport tuned for bandwidth measurement:
// HTTP/2 is disabled so each stream is an independent TCP connection, and
// compression is disabled so the zero-byte test payloads aren't deflated
// into fictitious throughput.
func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Streams < 1 {
		cfg.Streams = 4
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}
	if cfg.Warmup < 0 {
		cfg.Warmup = 0
	}

	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        cfg.Streams * 2,
		MaxIdleConnsPerHost: cfg.Streams * 2,
		MaxConnsPerHost:     cfg.Streams * 2,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,
		TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
	}

	return &Client{
		http: &http.Client{Transport: tr},
		cfg:  cfg,
	}
}

// Run executes the configured phases and returns a Result. Progress updates
// for download/upload phases are sent on prog if it is non-nil; Run closes
// prog before returning.
func (c *Client) Run(ctx context.Context, prog chan<- Progress) (*Result, error) {
	if prog != nil {
		defer close(prog)
	}

	res := &Result{
		Server:    c.cfg.BaseURL,
		Timestamp: time.Now().UTC(),
		Duration:  c.cfg.Duration.String(),
	}

	if c.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
		defer cancel()
	}

	if c.cfg.DoLatency || c.cfg.DoJitter {
		lat, jit, err := c.measureLatency(ctx)
		if err != nil {
			return nil, err
		}
		if c.cfg.DoLatency {
			res.LatencyMs = lat
		}
		if c.cfg.DoJitter {
			res.JitterMs = jit
		}
	}

	if c.cfg.DoDownload {
		v, err := c.measureDownload(ctx, prog)
		if err != nil {
			return nil, err
		}
		res.DownloadMbps = v
	}

	if c.cfg.DoUpload {
		v, err := c.measureUpload(ctx, prog)
		if err != nil {
			return nil, err
		}
		res.UploadMbps = v
	}

	return res, nil
}
