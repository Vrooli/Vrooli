.PHONY: help fmt lint type test check

.DEFAULT_GOAL := help

help: ## Show the supported package quality entrypoints
	@printf "api-core package entrypoints\n\n"
	@printf "  make fmt                        Format Go code\n"
	@printf "  make lint                       Lint Go code\n"
	@printf "  make type                       Compile-check Go packages without running tests\n"
	@printf "  make test                       Run Go tests\n"
	@printf "  make check                      Run lint, type, and test quality gates\n"

fmt: ## Format Go code
	@if command -v gofumpt >/dev/null; then \
		gofumpt -w .; \
	elif command -v gofmt >/dev/null; then \
		gofmt -w .; \
	else \
		echo "Neither gofumpt nor gofmt is available"; \
		exit 1; \
	fi

lint: ## Lint Go code
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; falling back to go vet"; \
		go vet ./...; \
	fi

type: ## Compile-check Go packages without running tests
	@go test -run '^$$' ./...

test: ## Run Go tests
	@go test ./...

check: lint type test ## Run lint, type, and test quality gates
