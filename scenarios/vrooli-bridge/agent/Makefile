# Vrooli Bridge node-agent Makefile
#
# The node-agent is a separate Go module from api/ and cli/. It is the thin,
# cross-compiled client installed on each trusted node (OT-P0-007). It MUST
# build CGO_ENABLED=0 for the full linux/darwin/windows × amd64/arm64 matrix so
# a node can run any supported OS from day one.
#
# Usage:
#   make build        - Build the agent for the host OS/arch into bin/
#   make matrix       - Cross-compile the full OS×arch matrix into dist/
#   make test         - Run unit tests
#   make fmt          - gofumpt (falls back to gofmt)
#   make vet          - go vet
#   make check        - fmt + vet + test + matrix (the green gate)
#   make clean        - Remove build artifacts

.PHONY: help build matrix test fmt vet check clean tidy

BINARY      := vrooli-bridge-agent
PKG         := vrooli-bridge/agent/internal/buildinfo
VERSION     ?= $(shell git -C ../../.. describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git -C ../../.. rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE        ?= $(shell date -u +%Y-%m-%d 2>/dev/null || echo unknown)
LDFLAGS     := -s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).Commit=$(COMMIT) \
	-X $(PKG).Date=$(DATE)

# The full cross-compile matrix. Keep this list authoritative — CI and the
# Phase-0 gate both build every entry CGO_ENABLED=0.
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

.DEFAULT_GOAL := help

help: ## Show targets
	@printf "vrooli-bridge node-agent targets:\n"
	@printf "  make build    Build for the host OS/arch into bin/\n"
	@printf "  make matrix   Cross-compile the full OS×arch matrix into dist/\n"
	@printf "  make test     Run unit tests\n"
	@printf "  make check    fmt + vet + test + matrix (green gate)\n"

tidy: ## Tidy the module
	GOWORK=off go mod tidy

build: ## Build for the host into bin/
	CGO_ENABLED=0 GOWORK=off go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

matrix: ## Cross-compile the full matrix (CGO_ENABLED=0) into dist/
	@set -e; for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="dist/$(BINARY)-$$os-$$arch$$ext"; \
		echo "  build $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch GOWORK=off \
			go build -trimpath -ldflags '$(LDFLAGS)' -o "$$out" . ; \
	done
	@echo "matrix complete"

test: ## Run unit tests
	CGO_ENABLED=0 GOWORK=off go test ./...

vet: ## go vet
	GOWORK=off go vet ./...

fmt: ## Format (gofumpt, falling back to gofmt)
	@if command -v gofumpt >/dev/null 2>&1; then gofumpt -w .; else gofmt -w .; fi

check: fmt vet test matrix ## Full green gate

clean: ## Remove build artifacts
	@rm -rf bin dist
