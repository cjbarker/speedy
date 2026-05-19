VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build run test vet fmt clean install

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
