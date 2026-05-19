VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

DIST := dist
BINARY := speedy

.PHONY: build build-all run test vet fmt clean install version

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

build-all: clean
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(VERSION)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(VERSION)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(VERSION)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(VERSION)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(VERSION)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(VERSION)-windows-arm64.exe .

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
