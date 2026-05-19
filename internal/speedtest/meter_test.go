package speedtest

import (
	"context"
	"io"
	"sync"
	"testing"
)

func TestMeterConcurrentAdd(t *testing.T) {
	m := &meter{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				m.Add(7)
			}
		}()
	}
	wg.Wait()
	if got := m.Load(); got != 100*1000*7 {
		t.Fatalf("Load() = %d, want %d", got, 100*1000*7)
	}
}

func TestZeroReader(t *testing.T) {
	m := &meter{}
	zr := &zeroReader{remaining: 5000, ctx: context.Background(), m: m}
	n, err := io.Copy(io.Discard, zr)
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if n != 5000 {
		t.Fatalf("copied %d bytes, want 5000", n)
	}
	if m.Load() != 5000 {
		t.Fatalf("meter = %d, want 5000", m.Load())
	}
}

func TestZeroReaderContextCancel(t *testing.T) {
	m := &meter{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	zr := &zeroReader{remaining: 1 << 20, ctx: ctx, m: m}
	buf := make([]byte, 4096)
	if _, err := zr.Read(buf); err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestCountingReader(t *testing.T) {
	m := &meter{}
	src := &zeroReader{remaining: 1234, ctx: context.Background(), m: &meter{}}
	cr := &countingReader{r: src, m: m}
	if _, err := io.Copy(io.Discard, cr); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if m.Load() != 1234 {
		t.Fatalf("meter = %d, want 1234", m.Load())
	}
}
