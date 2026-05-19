# speedy

A simple, fast, single-binary internet speed test CLI. Estimates your ISP
bandwidth (download and upload) against Cloudflare's public speed test
backend, with optional latency and jitter. Zero external dependencies — pure
Go standard library.

## Install

```sh
go install github.com/cjbarker/speedy@latest
```

Or build locally:

```sh
make build      # produces ./speedy
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

## Development

```sh
make test   # go test ./... -race -cover
make vet
make fmt
```
