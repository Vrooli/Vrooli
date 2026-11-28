.PHONY: help fmt lint type test check native-test release-stage validate-stage build-native-linux-amd64 build-native-linux-arm64 build-native-macos-arm64 build-native-windows-amd64 bundle-native-linux-amd64 bundle-native-linux-arm64 bundle-native-macos-arm64 bundle-native-windows-amd64

.DEFAULT_GOAL := help

SIGN ?= 1
PUBLICATION_URL ?=
UPSTREAM_PROVENANCE ?=

help:
	@printf "sherpa-onnx resource quality entrypoints\n\n"
	@printf "  make check                     Run CLI and native-server checks\n"
	@printf "  make native-test ROOT=...     Run cgo server tests against a pinned runtime\n"
	@printf "  make release-stage TARGET=... ROOT=... OUT=...  Build and validate a target-native stage (signed by default)\n"
	@printf "  make release-stage ... SIGN=0   Prepare an unsigned local stage for inspection\n"
	@printf "  make validate-stage OUT=...     Validate a local stage against resource.json offline\n"

fmt:
	@if command -v gofumpt >/dev/null; then gofumpt -w cli server; else gofmt -w cli server; fi

lint:
	cd cli && go vet ./...
	cd server && go vet ./...

type:
	cd cli && go test -run '^$$' ./...
	cd server && go test -run '^$$' ./...

test:
	python3 ../../scripts/test_validate_sherpa_native_stage.py
	python3 ../../scripts/test_package_sherpa_native_release.py
	cd cli && go test ./...
	cd server && go test ./...

check: lint type test

release-stage:
	@test -n "$(TARGET)" || (echo 'TARGET is required (linux-amd64, linux-arm64, macos-arm64, or windows-amd64)' >&2; exit 1)
	@test -n "$(ROOT)" || (echo 'ROOT is required' >&2; exit 1)
	@test -n "$(OUT)" || (echo 'OUT is required' >&2; exit 1)
	TARGET="$(TARGET)" ROOT="$(ROOT)" OUT="$(OUT)" VERSION="$(VERSION)" SIGN="$(SIGN)" PUBLICATION_URL="$(PUBLICATION_URL)" UPSTREAM_PROVENANCE="$(UPSTREAM_PROVENANCE)" ../../scripts/package-sherpa-native-release.sh

validate-stage:
	@test -n "$(OUT)" || (echo 'OUT is required' >&2; exit 1)
	python3 ../../scripts/validate-sherpa-native-stage.py "$(OUT)" resource.json

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
