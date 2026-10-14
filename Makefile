BIN ?= paginegialle
MODULE ?= github.com/h4shell/paginegialle-cli
VERSION ?= v0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X '$(MODULE)/internal/version.Version=$(VERSION)' -X '$(MODULE)/internal/version.Commit=$(COMMIT)' -X '$(MODULE)/internal/version.Date=$(DATE)'
PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build test vet cross install clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) .

test:
	go test ./...

vet:
	go vet ./...

cross:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=287199287199{p%/*}; arch=287199287199{p#*/}; \
		out=dist/$(BIN)-$$\{os\}-$$\{arch\}; \
		if [ "$$\{os\}" = "windows" ]; then out=$$\{out\}.exe; fi; \
		GOOS=$$\{os\} GOARCH=$$\{arch\} CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $$\{out\} .; \
		echo "built $$\{out\}"; \
	done

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -rf bin dist
