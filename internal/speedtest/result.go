package speedtest

import "time"

// Result is the final outcome of a speed test run.
type Result struct {
	DownloadMbps float64   `json:"download_mbps,omitempty"`
	UploadMbps   float64   `json:"upload_mbps,omitempty"`
	LatencyMs    float64   `json:"latency_ms,omitempty"`
	JitterMs     float64   `json:"jitter_ms,omitempty"`
	Server       string    `json:"server"`
	Timestamp    time.Time `json:"timestamp"`
	Duration     string    `json:"duration"`
}

// Progress is a live update emitted during a measurement phase.
type Progress struct {
	Phase    string  // "download" or "upload"
	Mbps     float64 // cumulative throughput so far
	FracDone float64 // 0..1 estimated completion of the phase
	Done     bool    // true on the final update for a phase
}

// mbps converts a byte count over a duration into megabits per second.
// A non-positive duration yields 0 rather than NaN/Inf.
func mbps(bytes int64, d time.Duration) float64 {
	if d <= 0 || bytes <= 0 {
		return 0
	}
	return (float64(bytes) * 8 / 1e6) / d.Seconds()
}
