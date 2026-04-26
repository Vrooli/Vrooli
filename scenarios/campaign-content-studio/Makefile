# Campaign Content Studio Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/campaign-content-studio-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts
#
# This Makefile is a thin wrapper. All user-facing output (status, banners,
# colors) is owned by the `vrooli scenario` CLI itself.

.PHONY: help start run stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check setup open

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

help: ## Show this help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

start: ## Start this scenario
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@vrooli scenario stop $(SCENARIO_NAME)

restart: ## Restart this scenario
	@vrooli scenario restart $(SCENARIO_NAME)

test: ## Run scenario tests
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show scenario logs
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Show scenario status
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true

build: ## Build the scenario
	@if [ -f api/go.mod ]; then \
		cd api && go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
	fi

fmt-ui: ## Format UI code
	@true

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && go vet ./... && go fmt ./...; \
		fi; \
	fi

lint-ui: ## Lint UI code
	@true

check: fmt lint test ## Run full quality gates

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

# Development shortcuts
run: start
dev: start
