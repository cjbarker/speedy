package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cjbarker/speedy/internal/output"
	"github.com/cjbarker/speedy/internal/speedtest"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var (
		server     = flag.String("server", "", "custom backend base URL (default: Cloudflare)")
		asJSON     = flag.Bool("json", false, "output machine-readable JSON")
		ping       = flag.Bool("ping", false, "also measure latency")
		jitter     = flag.Bool("jitter", false, "also measure jitter (implies -ping)")
		duration   = flag.Duration("duration", 10*time.Second, "measurement time per direction")
		streams    = flag.Int("streams", 4, "parallel connections")
		warmup     = flag.Duration("warmup", 2*time.Second, "warmup excluded from results (TCP slow-start)")
		noDownload = flag.Bool("no-download", false, "skip the download test")
		noUpload   = flag.Bool("no-upload", false, "skip the upload test")
		timeout    = flag.Duration("timeout", 60*time.Second, "global wall-clock cap")
		showVer    = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "speedy — simple internet speed test\n\nUsage: speedy [flags]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("speedy %s\ncommit: %s\nbuilt:  %s\n", version, commit, date)
		return
	}

	cfg := speedtest.Config{
		BaseURL:    *server,
		Streams:    *streams,
		Duration:   *duration,
		Warmup:     *warmup,
		Timeout:    *timeout,
		DoDownload: !*noDownload,
		DoUpload:   !*noUpload,
		DoLatency:  *ping || *jitter,
		DoJitter:   *jitter,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := speedtest.New(cfg)

	if err := client.Probe(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "speedy: %v\n", err)
		os.Exit(1)
	}

	tty := !*asJSON && output.IsTTY(os.Stdout)

	prog := make(chan speedtest.Progress, 16)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for p := range prog {
			output.RenderProgress(os.Stdout, tty, p)
		}
	}()

	res, err := client.Run(ctx, prog)
	wg.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nspeedy: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		if err := output.RenderJSON(os.Stdout, res); err != nil {
			fmt.Fprintf(os.Stderr, "speedy: %v\n", err)
			os.Exit(1)
		}
		return
	}
	output.RenderResult(os.Stdout, tty, res)
}
