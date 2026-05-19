package speedtest

import (
	"math"
	"testing"
	"time"
)

func TestMbps(t *testing.T) {
	cases := []struct {
		name  string
		bytes int64
		d     time.Duration
		want  float64
	}{
		{"1MB in 1s", 1_000_000, time.Second, 8.0},
		{"125MB in 1s = 1Gbps", 125_000_000, time.Second, 1000.0},
		{"zero duration", 1_000_000, 0, 0},
		{"negative duration", 1_000_000, -time.Second, 0},
		{"zero bytes", 0, time.Second, 0},
		{"half second", 1_000_000, 500 * time.Millisecond, 16.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mbps(c.bytes, c.d)
			if math.Abs(got-c.want) > 1e-9 {
				t.Fatalf("mbps(%d, %v) = %v, want %v", c.bytes, c.d, got, c.want)
			}
		})
	}
}
