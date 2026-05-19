# speedy

A simple, fast, single-binary internet speed test CLI. Estimates your ISP
bandwidth (download and upload) against Cloudflare's public speed test
backend, with optional latency and jitter. Zero external dependencies — pure
Go standard library.

## Quick start

```sh
go install github.com/cjbarker/speedy@latest   # install
speedy                                          # run a download + upload test
```

## Build

Requires Go 1.24+.

```sh
git clone https://github.com/cjbarker/speedy.git
cd speedy
make build        # produces ./speedy (trimmed, version-stamped)
./speedy -version
```

Cross-compile for another platform with the standard Go toolchain:

```sh
GOOS=linux   GOARCH=amd64 make build
GOOS=darwin  GOARCH=arm64 make build
GOOS=windows GOARCH=amd64 make build
```

## Test

```sh
make test   # go test ./... -race -cover  (unit + httptest integration)
make vet    # go vet ./...
make fmt    # gofmt -l -w .
```

The suite uses in-process `httptest` backends, so it needs no network
access and is safe to run in CI and sandboxed environments.

## Deploy

`speedy` is a single static binary with zero runtime dependencies — deploy
by copying it onto the target host:

```sh
make build
install -m 0755 speedy /usr/local/bin/speedy   # local/host install

# or install directly from source onto a server
go install github.com/cjbarker/speedy@latest    # -> $GOBIN/speedy
```

For scheduled monitoring, run it from cron/systemd with `-json` and append
the output to a log or pipe it to your metrics pipeline:

```sh
*/15 * * * * /usr/local/bin/speedy -json >> /var/log/speedy.jsonl 2>&1
```

## Usage

```sh
speedy                          # download + upload, pretty output
speedy -ping -jitter            # also measure latency and jitter
speedy -json                    # machine-readable JSON
speedy -server https://my.host  # custom backend (must expose /__down and /__up)
speedy -duration 5s -streams 8  # tune measurement window / parallelism
speedy -no-upload               # download only
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-server` | Cloudflare | Custom backend base URL |
| `-json` | false | Machine-readable JSON output |
| `-ping` | false | Also measure latency |
| `-jitter` | false | Also measure jitter (implies `-ping`) |
| `-duration` | 10s | Measurement time per direction |
| `-streams` | 4 | Parallel connections |
| `-warmup` | 2s | Warmup excluded from results (TCP slow-start) |
| `-no-download` | false | Skip the download test |
| `-no-upload` | false | Skip the upload test |
| `-timeout` | 60s | Global wall-clock cap |
| `-version` | | Print version and exit |

## How it works

- Saturates the link with N independent HTTP/1.1 connections (HTTP/2 is
  disabled so streams don't multiplex over one TCP connection).
- Compression is disabled so the zero-byte test payloads aren't deflated into
  fictitious throughput.
- A warmup window is excluded from the result to discard TCP slow-start.
- Latency is measured as time-to-first-byte on tiny requests; jitter is the
  standard deviation of those samples.
