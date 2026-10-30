.PHONY: help fmt fmt-go lint lint-go type type-go test test-go check check-go

.DEFAULT_GOAL := help

CLI_DIR := cli

help: ## Show the supported resource quality entrypoints
	@printf "Vrooli resource quality entrypoints\n\n"
	@printf "  make fmt                        Format Go code\n"
	@printf "  make lint                       Lint Go code\n"
	@printf "  make type                       Compile-check Go packages without running tests\n"
	@printf "  make test                       Run Go tests\n"
	@printf "  make check                      Run lint, type, and test quality gates\n"

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
