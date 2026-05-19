VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build run test vet fmt clean install version

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o speedy .

run:
	go run . $(ARGS)

test:
	go test ./... -race -cover

vet:
	go vet ./...

fmt:
	gofmt -l -w .

clean:
	rm -f speedy
	rm -rf dist

install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

version:
	@echo $(VERSION)
