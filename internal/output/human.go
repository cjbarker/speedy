package output

import (
	"fmt"
	"io"
	"os"

	"github.com/cjbarker/speedy/internal/speedtest"
)

const (
	clrReset = "\x1b[0m"
	clrBold  = "\x1b[1m"
	clrCyan  = "\x1b[36m"
	clrGreen = "\x1b[32m"
)

// IsTTY reports whether f is an interactive terminal.
func IsTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func color(tty bool, c, s string) string {
	if !tty {
		return s
	}
	return c + s + clrReset
}

// RenderProgress redraws a single in-place progress line. It is a no-op when
// w is not a terminal so piped/redirected output stays clean.
func RenderProgress(w io.Writer, tty bool, p speedtest.Progress) {
	if !tty {
		return
	}
	width := 24
	filled := int(p.FracDone * float64(width))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "#"
		} else {
			bar += "-"
		}
	}
	fmt.Fprintf(w, "\r%s%-9s%s [%s] %s%7.2f Mbps%s ",
		clrCyan, p.Phase, clrReset, bar, clrBold, p.Mbps, clrReset)
}

// RenderResult prints the final summary table.
func RenderResult(w io.Writer, tty bool, r *speedtest.Result) {
	if tty {
		fmt.Fprint(w, "\r\x1b[K")
	}
	fmt.Fprintf(w, "%s\n", color(tty, clrBold, "speedy — internet speed test"))
	fmt.Fprintf(w, "  Server:   %s\n", r.Server)
	fmt.Fprintf(w, "  Time:     %s\n", r.Timestamp.Format("2006-01-02 15:04:05 MST"))
	if r.LatencyMs > 0 {
		fmt.Fprintf(w, "  Latency:  %.2f ms\n", r.LatencyMs)
	}
	if r.JitterMs > 0 {
		fmt.Fprintf(w, "  Jitter:   %.2f ms\n", r.JitterMs)
	}
	if r.DownloadMbps > 0 {
		fmt.Fprintf(w, "  Download: %s\n", color(tty, clrGreen, fmt.Sprintf("%.2f Mbps", r.DownloadMbps)))
	}
	if r.UploadMbps > 0 {
		fmt.Fprintf(w, "  Upload:   %s\n", color(tty, clrGreen, fmt.Sprintf("%.2f Mbps", r.UploadMbps)))
	}
}
