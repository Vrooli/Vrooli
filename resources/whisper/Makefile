.PHONY: help fmt fmt-go lint lint-go type type-go test test-go check check-go build-native-macos-arm64

.DEFAULT_GOAL := help

CLI_DIR := cli

# whisper.cpp does not publish a macOS whisper-server executable. This target
# is intentionally a native-host build recipe; it is not a cross-compilation
# shortcut or release evidence by itself.
WHISPER_CPP_ROOT ?=
NATIVE_BUILD_DIR ?= $(CURDIR)/.native-build/macos-arm64
NATIVE_DIST_DIR ?= $(CURDIR)/dist
HOST_OS := $(shell uname -s 2>/dev/null || printf unknown)
HOST_ARCH := $(shell uname -m 2>/dev/null || printf unknown)

help: ## Show the supported resource quality entrypoints
	@printf "Vrooli resource quality entrypoints\n\n"
	@printf "  make fmt                        Format Go code\n"
	@printf "  make lint                       Lint Go code\n"
	@printf "  make type                       Compile-check Go packages without running tests\n"
	@printf "  make test                       Run Go tests\n"
	@printf "  make check                      Run lint, type, and test quality gates\n"
	@printf "  make build-native-macos-arm64  Build a macOS arm64 candidate on Darwin/arm64\n"

fmt: fmt-go ## Format Go code

fmt-go: ## Format Go code
	@if [ -d "$(CLI_DIR)" ]; then \
		if command -v gofumpt >/dev/null; then \
			cd "$(CLI_DIR)" && gofumpt -w .; \
		elif command -v gofmt >/dev/null; then \
			cd "$(CLI_DIR)" && gofmt -w .; \
		else \
			echo "Neither gofumpt nor gofmt is available"; \
			exit 1; \
		fi; \
	fi

lint: lint-go ## Lint Go code

lint-go: ## Lint Go code
	@if [ -d "$(CLI_DIR)" ]; then \
		if command -v golangci-lint >/dev/null; then \
			cd "$(CLI_DIR)" && golangci-lint run ./...; \
		else \
			cd "$(CLI_DIR)" && go vet ./...; \
		fi; \
	fi

type: type-go ## Compile-check Go packages without running tests

type-go: ## Compile-check Go packages without running tests
	@if [ -d "$(CLI_DIR)" ]; then \
		cd "$(CLI_DIR)" && go test -run '^$$' ./...; \
	fi

test: test-go ## Run Go tests

test-go: ## Run Go tests
	@if [ -d "$(CLI_DIR)" ]; then \
		cd "$(CLI_DIR)" && go test ./...; \
	fi

check: check-go ## Run Go quality gates

check-go: lint-go type-go test-go ## Run Go quality gates

build-native-macos-arm64: ## Build the macOS arm64 whisper-server on a matching host
	@test "$(HOST_OS)" = "Darwin" -a "$(HOST_ARCH)" = "arm64" || (echo "native macOS arm64 build requires host Darwin/arm64, found $(HOST_OS)/$(HOST_ARCH)" >&2; exit 1)
	@test -n "$(WHISPER_CPP_ROOT)" || (echo "WHISPER_CPP_ROOT must point to the pinned whisper.cpp source tree" >&2; exit 1)
	@test -f "$(WHISPER_CPP_ROOT)/CMakeLists.txt" || (echo "WHISPER_CPP_ROOT is not a whisper.cpp source tree: $(WHISPER_CPP_ROOT)" >&2; exit 1)
	@command -v cmake >/dev/null || (echo "cmake is required for the native macOS build" >&2; exit 1)
	cmake -S "$(WHISPER_CPP_ROOT)" -B "$(NATIVE_BUILD_DIR)" -DCMAKE_BUILD_TYPE=Release -DWHISPER_BUILD_SERVER=ON -DWHISPER_BUILD_TESTS=OFF -DWHISPER_BUILD_EXAMPLES=OFF
	cmake --build "$(NATIVE_BUILD_DIR)" --target whisper-server --config Release
	@test -x "$(NATIVE_BUILD_DIR)/bin/whisper-server" || (echo "whisper-server was not produced at $(NATIVE_BUILD_DIR)/bin/whisper-server" >&2; exit 1)
	mkdir -p "$(NATIVE_DIST_DIR)"
	cp "$(NATIVE_BUILD_DIR)/bin/whisper-server" "$(NATIVE_DIST_DIR)/whisper-server_macos_arm64"
	chmod 0755 "$(NATIVE_DIST_DIR)/whisper-server_macos_arm64"
	@echo "Built candidate: $(NATIVE_DIST_DIR)/whisper-server_macos_arm64"
