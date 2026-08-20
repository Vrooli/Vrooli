.PHONY: help fmt lint type test check native-test release-stage build-native-linux-amd64 build-native-linux-arm64 build-native-macos-arm64 build-native-windows-amd64 bundle-native-linux-amd64 bundle-native-linux-arm64 bundle-native-macos-arm64 bundle-native-windows-amd64

.DEFAULT_GOAL := help

VERSION ?= 1.13.2-vrooli.1
SIGN ?= 1
PUBLICATION_URL ?=

help:
	@printf "sherpa-onnx resource quality entrypoints\n\n"
	@printf "  make check                     Run CLI and native-server checks\n"
	@printf "  make native-test ROOT=...     Run cgo server tests against a pinned runtime\n"
	@printf "  make release-stage TARGET=... ROOT=... OUT=...  Build, checksum, and sign one target-native bundle\n"

fmt:
	@if command -v gofumpt >/dev/null; then gofumpt -w cli server; else gofmt -w cli server; fi

lint:
	cd cli && go vet ./...
	cd server && go vet ./...

type:
	cd cli && go test -run '^$$' ./...
	cd server && go test -run '^$$' ./...

test:
	cd cli && go test ./...
	cd server && go test ./...

check: lint type test

release-stage:
	@test -n "$(TARGET)" || (echo 'TARGET is required (linux-amd64, linux-arm64, macos-arm64, or windows-amd64)' >&2; exit 1)
	@test -n "$(ROOT)" || (echo 'ROOT is required' >&2; exit 1)
	@test -n "$(OUT)" || (echo 'OUT is required' >&2; exit 1)
	TARGET="$(TARGET)" ROOT="$(ROOT)" OUT="$(OUT)" VERSION="$(VERSION)" SIGN="$(SIGN)" PUBLICATION_URL="$(PUBLICATION_URL)" ./scripts/package-native-release.sh

native-test:
	@test -n "$(ROOT)" || (echo 'ROOT is required' >&2; exit 1)
	$(MAKE) -C server test-native ROOT="$(ROOT)"

build-native-linux-amd64:
	$(MAKE) -C server build-native-linux-amd64 ROOT="$(ROOT)"

build-native-linux-arm64:
	$(MAKE) -C server build-native-linux-arm64 ROOT="$(ROOT)"

build-native-macos-arm64:
	$(MAKE) -C server build-native-macos-arm64 ROOT="$(ROOT)"

build-native-windows-amd64:
	$(MAKE) -C server build-native-windows-amd64 ROOT="$(ROOT)"

bundle-native-linux-amd64:
	$(MAKE) -C server bundle-native-linux-amd64 ROOT="$(ROOT)" OUT="$(OUT)"

bundle-native-linux-arm64:
	$(MAKE) -C server bundle-native-linux-arm64 ROOT="$(ROOT)" OUT="$(OUT)"

bundle-native-macos-arm64:
	$(MAKE) -C server bundle-native-macos-arm64 ROOT="$(ROOT)" OUT="$(OUT)"

bundle-native-windows-amd64:
	$(MAKE) -C server bundle-native-windows-amd64 ROOT="$(ROOT)" OUT="$(OUT)"
