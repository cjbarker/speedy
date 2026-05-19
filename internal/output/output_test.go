package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cjbarker/speedy/internal/speedtest"
)

func sampleResult() *speedtest.Result {
	return &speedtest.Result{
		DownloadMbps: 123.45,
		UploadMbps:   67.89,
		LatencyMs:    12.3,
		JitterMs:     1.5,
		Server:       "https://speed.cloudflare.com",
		Timestamp:    time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
		Duration:     "10s",
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, sampleResult()); err != nil {
		t.Fatalf("render: %v", err)
	}
	var got speedtest.Result
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.DownloadMbps != 123.45 || got.UploadMbps != 67.89 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestRenderResultPlain(t *testing.T) {
	var buf bytes.Buffer
	RenderResult(&buf, false, sampleResult())
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("non-tty output must not contain ANSI escapes: %q", out)
	}
	for _, want := range []string{"123.45 Mbps", "67.89 Mbps", "12.30 ms", "1.50 ms"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderProgressNonTTYNoOp(t *testing.T) {
	var buf bytes.Buffer
	RenderProgress(&buf, false, speedtest.Progress{Phase: "download", Mbps: 50})
	if buf.Len() != 0 {
		t.Fatalf("expected no output for non-tty, got %q", buf.String())
	}
}
