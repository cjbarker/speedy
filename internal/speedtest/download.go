package speedtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

const (
	minChunk = 1 << 20   // 1 MiB
	maxChunk = 256 << 20 // 256 MiB
)

func (c *Client) measureDownload(ctx context.Context, prog chan<- Progress) (float64, error) {
	var chunk atomic.Int64
	chunk.Store(minChunk)

	w := func(ctx context.Context, m *meter) error {
		n := chunk.Load()
		url := fmt.Sprintf("%s/__down?bytes=%d", c.cfg.BaseURL, n)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			io.Copy(io.Discard, resp.Body)
			return fmt.Errorf("download: unexpected status %s", resp.Status)
		}
		buf := make([]byte, 64<<10)
		cr := &countingReader{r: resp.Body, m: m}
		if _, err := io.CopyBuffer(io.Discard, cr, buf); err != nil {
			return err
		}
		if n < maxChunk {
			chunk.CompareAndSwap(n, n*2)
		}
		return nil
	}

	return c.runPhase(ctx, "download", prog, w)
}
