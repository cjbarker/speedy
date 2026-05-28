package speedtest

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is Cloudflare's public, no-API-key speed test backend.
const DefaultBaseURL = "https://speed.cloudflare.com"

// Config controls a speed test run.
type Config struct {
	BaseURL    string
	Streams    int
	Duration   time.Duration
	Warmup     time.Duration
	Timeout    time.Duration
	DoDownload bool
	DoUpload   bool
	DoLatency  bool
	DoJitter   bool
}

// Client runs speed tests against a backend.
type Client struct {
	http *http.Client
	cfg  Config
}

// New builds a Client with a transport tuned for bandwidth measurement:
// HTTP/2 is disabled so each stream is an independent TCP connection, and
// compression is disabled so the zero-byte test payloads aren't deflated
// into fictitious throughput.
func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Streams < 1 {
		cfg.Streams = 4
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}
	if cfg.Warmup < 0 {
		cfg.Warmup = 0
	}

	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        cfg.Streams * 2,
		MaxIdleConnsPerHost: cfg.Streams * 2,
		MaxConnsPerHost:     cfg.Streams * 2,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,
		TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
	}

	return &Client{
		http: &http.Client{Transport: tr},
		cfg:  cfg,
	}
}

// CheckConnectivity performs a fast DNS lookup on the target server's hostname
// to verify basic internet connectivity before starting the speed test.
func (c *Client) CheckConnectivity(ctx context.Context) error {
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	host := u.Hostname()

	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if _, err := net.DefaultResolver.LookupHost(checkCtx, host); err != nil {
		return fmt.Errorf("no internet connection: unable to resolve %q (check your network)", host)
	}
	return nil
}

// Probe checks that the server supports the speed test API by issuing a
// small download request. It returns a descriptive error if the server is
// unreachable or does not expose the expected endpoints.
func (c *Client) Probe(ctx context.Context) error {
	url := c.cfg.BaseURL + "/__down?bytes=0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("server %q is not reachable: %w", c.cfg.BaseURL, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("server %q does not support the speed test API (expected /__down and /__up endpoints)", c.cfg.BaseURL)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server %q returned %s on probe", c.cfg.BaseURL, resp.Status)
	}
	return nil
}

// Run executes the configured phases and returns a Result. Progress updates
// for download/upload phases are sent on prog if it is non-nil; Run closes
// prog before returning.
func (c *Client) Run(ctx context.Context, prog chan<- Progress) (*Result, error) {
	if prog != nil {
		defer close(prog)
	}

	res := &Result{
		Server:    c.cfg.BaseURL,
		Timestamp: time.Now().UTC(),
		Duration:  c.cfg.Duration.String(),
	}

	if c.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
		defer cancel()
	}

	if c.cfg.DoLatency || c.cfg.DoJitter {
		lat, jit, err := c.measureLatency(ctx)
		if err != nil {
			return nil, err
		}
		if c.cfg.DoLatency {
			res.LatencyMs = lat
		}
		if c.cfg.DoJitter {
			res.JitterMs = jit
		}
	}

	if c.cfg.DoDownload {
		v, err := c.measureDownload(ctx, prog)
		if err != nil {
			return nil, err
		}
		res.DownloadMbps = v
	}

	if c.cfg.DoUpload {
		v, err := c.measureUpload(ctx, prog)
		if err != nil {
			return nil, err
		}
		res.UploadMbps = v
	}

	return res, nil
}
