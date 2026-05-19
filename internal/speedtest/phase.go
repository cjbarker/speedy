package speedtest

import (
	"context"
	"sync"
	"time"
)

// worker performs one unit of transfer (a single HTTP request), feeding the
// shared meter as bytes move. It returns when it completes or ctx is cancelled.
type worker func(ctx context.Context, m *meter) error

// runPhase drives cfg.Streams workers in parallel for Warmup+Duration, then
// computes throughput from only the post-warmup bytes/time so TCP slow-start
// is excluded. A sampler goroutine emits live Progress on prog (if non-nil).
func (c *Client) runPhase(parent context.Context, phase string, prog chan<- Progress, w worker) (float64, error) {
	m := &meter{}
	total := c.cfg.Warmup + c.cfg.Duration

	ctx, cancel := context.WithTimeout(parent, total)
	defer cancel()

	var (
		mu        sync.Mutex
		markBytes int64
		markTime  time.Time
		marked    bool
	)
	start := time.Now()

	sampleDone := make(chan struct{})
	go func() {
		defer close(sampleDone)
		t := time.NewTicker(150 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				elapsed := now.Sub(start)
				mu.Lock()
				if !marked && elapsed >= c.cfg.Warmup {
					markBytes = m.Load()
					markTime = now
					marked = true
				}
				mb, mt, mk := markBytes, markTime, marked
				mu.Unlock()

				if prog == nil || !mk {
					continue
				}
				dur := now.Sub(mt)
				if dur <= 0 {
					continue
				}
				frac := float64(elapsed-c.cfg.Warmup) / float64(c.cfg.Duration)
				if frac < 0 {
					frac = 0
				}
				if frac > 1 {
					frac = 1
				}
				select {
				case prog <- Progress{Phase: phase, Mbps: mbps(m.Load()-mb, dur), FracDone: frac}:
				default:
				}
			}
		}
	}()

	errCh := make(chan error, c.cfg.Streams)
	var wg sync.WaitGroup
	for i := 0; i < c.cfg.Streams; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				if err := w(ctx, m); err != nil {
					if ctx.Err() != nil {
						return // expected: phase time-boxed out
					}
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}

	wg.Wait()
	cancel()
	<-sampleDone

	select {
	case err := <-errCh:
		return 0, err
	default:
	}

	mu.Lock()
	mb, mt, mk := markBytes, markTime, marked
	mu.Unlock()
	if !mk {
		// Phase ended before warmup elapsed: fall back to whole-run figure.
		return mbps(m.Load(), time.Since(start)), nil
	}
	return mbps(m.Load()-mb, time.Since(mt)), nil
}
