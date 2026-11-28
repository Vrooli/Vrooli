.PHONY: help fmt fmt-go lint lint-go type type-go test test-go test-server test-server-onnx check check-go build-server build-server-onnx

.DEFAULT_GOAL := help

CLI_DIR := cli
SERVER_DIR := server
GO ?= go

help: ## Show the supported resource quality entrypoints
	@printf "Vrooli resource quality entrypoints\n\n"
	@printf "  make fmt           Format Go code (cli + server)\n"
	@printf "  make lint          Lint Go code\n"
	@printf "  make type          Compile-check Go packages without running tests\n"
	@printf "  make test          Run Go tests (cli + server)\n"
	@printf "  make check         Run lint, type, and test quality gates\n"
	@printf "  make build-server  Build the serving binary (regex-only stub engine)\n"
	@printf "  make build-server-onnx  Build with the pii_onnx NER tag (CGO-free; dlopen libonnxruntime.so at runtime)\n"
	@printf "  make test-server-onnx MODEL_DIR=<dir> ONNXRUNTIME_LIB=<.so>  Run the live NER + eval tests\n"

fmt: fmt-go ## Format Go code

fmt-go: ## Format Go code
	@for dir in "$(CLI_DIR)" "$(SERVER_DIR)"; do \
		if [ -d "$$dir" ]; then \
			if command -v gofumpt >/dev/null; then \
				cd "$$dir" && gofumpt -w . && cd - >/dev/null; \
			elif command -v gofmt >/dev/null; then \
				cd "$$dir" && gofmt -w . && cd - >/dev/null; \
			else \
				echo "Neither gofumpt nor gofmt is available"; exit 1; \
			fi; \
		fi; \
	done

lint: lint-go ## Lint Go code

lint-go: ## Lint Go code
	@if [ -d "$(CLI_DIR)" ]; then \
		if command -v golangci-lint >/dev/null; then \
			cd "$(CLI_DIR)" && golangci-lint run ./...; \
		else \
			cd "$(CLI_DIR)" && $(GO) vet ./...; \
		fi; \
	fi
	@if [ -d "$(SERVER_DIR)" ]; then cd "$(SERVER_DIR)" && GOWORK=off $(GO) vet ./...; fi

type: type-go ## Compile-check Go packages without running tests

type-go: ## Compile-check Go packages without running tests
	@if [ -d "$(CLI_DIR)" ]; then cd "$(CLI_DIR)" && $(GO) test -run '^$$' ./...; fi
	@if [ -d "$(SERVER_DIR)" ]; then cd "$(SERVER_DIR)" && GOWORK=off $(GO) test -run '^$$' ./...; fi

test: test-go ## Run Go tests

test-go: test-server ## Run Go tests
	@if [ -d "$(CLI_DIR)" ]; then cd "$(CLI_DIR)" && $(GO) test ./...; fi

test-server: ## Run the server module tests (separate go.mod; GOWORK=off)
	@if [ -d "$(SERVER_DIR)" ]; then cd "$(SERVER_DIR)" && GOWORK=off $(GO) test ./...; fi

check: check-go ## Run Go quality gates

check-go: lint-go type-go test-go ## Run Go quality gates

build-server: ## Build the serving binary with the regex-only stub engine (no ONNX)
	@test -d "$(SERVER_DIR)" || (echo 'server/ missing' >&2; exit 1)
	mkdir -p "$(SERVER_DIR)/dist"
	cd "$(SERVER_DIR)" && GOWORK=off $(GO) build -trimpath -buildvcs=false -o dist/resource-pii-detector-server .

# build-server-onnx builds the real NER path (engine_onnx.go, `pii_onnx` tag).
# The onnxruntime_go binding (approved: go/github.com/yalue/onnxruntime_go) is
# CGO-FREE: it dlopen()s libonnxruntime.so at runtime, so this build needs NO
# CGO toolchain, NO headers, and NO -lonnxruntime link flag. The native lib is a
# RUNTIME dependency, bundled beside the binary (sherpa-style: <bundle>/lib/
# libonnxruntime.so) or pointed at via PII_DETECTOR_ONNXRUNTIME_LIB. Stage the
# checksum-pinned small-tier model (model-policy.json models.*.artifact) into the
# model cache; the binary alone does not fetch it.
build-server-onnx: ## Build with the pii_onnx NER tag (CGO-free; libonnxruntime.so loaded at runtime)
	@test -d "$(SERVER_DIR)" || (echo 'server/ missing' >&2; exit 1)
	mkdir -p "$(SERVER_DIR)/dist"
	cd "$(SERVER_DIR)" && GOWORK=off $(GO) build -trimpath -buildvcs=false -tags pii_onnx -o dist/resource-pii-detector-server .

# test-server-onnx runs the server suite under the pii_onnx tag. Without
# MODEL_DIR + ONNXRUNTIME_LIB the live NER tests skip-with-reason; with them they
# load the staged model and assert real spans + the labeled-fixture eval recall.
test-server-onnx: ## Run the pii_onnx server tests (live NER tests run when MODEL_DIR + ONNXRUNTIME_LIB are set)
	@test -d "$(SERVER_DIR)" || (echo 'server/ missing' >&2; exit 1)
	cd "$(SERVER_DIR)" && PII_DETECTOR_MODEL_DIR="$(MODEL_DIR)" PII_DETECTOR_ONNXRUNTIME_LIB="$(ONNXRUNTIME_LIB)" GOWORK=off $(GO) test -tags pii_onnx ./...
