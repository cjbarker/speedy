package speedtest

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptrace"
	"time"
)

const latencySamples = 20

// measureLatency issues small requests and times the first response byte
// (TTFB) to isolate round-trip latency from transfer time. It returns mean
// latency and jitter (population standard deviation), both in milliseconds.
func (c *Client) measureLatency(ctx context.Context) (latMs, jitMs float64, err error) {
	var samples []float64
	for i := 0; i < latencySamples; i++ {
		if err := ctx.Err(); err != nil {
			break
		}
		d, err := c.ttfb(ctx)
		if err != nil {
			return 0, 0, err
		}
		// Discard the first two samples (connection setup / cold cache).
		if i >= 2 {
			samples = append(samples, float64(d.Microseconds())/1000.0)
		}
	}
	if len(samples) == 0 {
		return 0, 0, fmt.Errorf("latency: no samples collected")
	}

	var sum float64
	for _, s := range samples {
		sum += s
	}
	mean := sum / float64(len(samples))

	var sq float64
	for _, s := range samples {
		sq += (s - mean) * (s - mean)
	}
	return mean, math.Sqrt(sq / float64(len(samples))), nil
}

func (c *Client) ttfb(ctx context.Context) (time.Duration, error) {
	url := fmt.Sprintf("%s/__down?bytes=0", c.cfg.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	var start time.Time
	var first time.Duration
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() { first = time.Since(start) },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start = time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if first == 0 {
		first = time.Since(start)
	}
	return first, nil
}
