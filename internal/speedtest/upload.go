package speedtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

func (c *Client) measureUpload(ctx context.Context, prog chan<- Progress) (float64, error) {
	var chunk atomic.Int64
	chunk.Store(minChunk)

	w := func(ctx context.Context, m *meter) error {
		n := chunk.Load()
		url := c.cfg.BaseURL + "/__up"
		body := &zeroReader{remaining: n, ctx: ctx, m: m}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
		if err != nil {
			return err
		}
		req.ContentLength = n
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		if resp.StatusCode == http.StatusForbidden {
			if n > minChunk {
				chunk.CompareAndSwap(n, n/2)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
			return nil
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("upload: unexpected status %s", resp.Status)
		}
		if n < maxChunk {
			chunk.CompareAndSwap(n, n*2)
		}
		return nil
	}

	return c.runPhase(ctx, "upload", prog, w)
}
