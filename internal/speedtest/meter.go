package speedtest

import (
	"context"
	"io"
	"sync/atomic"
)

// meter accumulates a running total of bytes transferred across goroutines.
type meter struct {
	bytes atomic.Int64
}

func (m *meter) Add(n int)   { m.bytes.Add(int64(n)) }
func (m *meter) Load() int64 { return m.bytes.Load() }

// countingReader wraps an io.Reader and reports every byte read to a meter.
type countingReader struct {
	r io.Reader
	m *meter
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.m.Add(n)
	}
	return n, err
}

// zeroReader streams up to `remaining` zero bytes, used as an upload body.
// It stops early when the context is cancelled and reports progress to a
// meter so uploads are accounted for the same way as downloads.
type zeroReader struct {
	remaining int64
	ctx       context.Context
	m         *meter
}

func (z *zeroReader) Read(p []byte) (int, error) {
	if z.remaining <= 0 {
		return 0, io.EOF
	}
	if err := z.ctx.Err(); err != nil {
		return 0, err
	}
	n := int64(len(p))
	if n > z.remaining {
		n = z.remaining
	}
	for i := int64(0); i < n; i++ {
		p[i] = 0
	}
	z.remaining -= n
	z.m.Add(int(n))
	return int(n), nil
}
