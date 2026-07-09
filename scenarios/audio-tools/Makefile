# Audio Tools Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/audio-tools-api). ALWAYS use these commands.
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

.PHONY: help setup start stop restart status logs test temporal-models open orient run dev endpoints fmt fmt-go fmt-ui lint lint-go lint-ui check clean build

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

help: ## Show the supported scenario entrypoints
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

start: ## Start the scenario
	@vrooli scenario start "$(SCENARIO_NAME)"

stop: ## Stop the scenario
	@vrooli scenario stop "$(SCENARIO_NAME)"

restart: ## Restart the scenario
	@vrooli scenario restart "$(SCENARIO_NAME)"

status: ## Show scenario runtime status
	@vrooli scenario status "$(SCENARIO_NAME)"

logs: ## Show scenario logs
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

temporal-models: ## Verify formal flow artifacts via the flow-verifier scenario CLI
	@flow-verifier verify check --root .

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

orient: ## Show generated-scenario orientation progress
	@template-manager orient "$(SCENARIO_NAME)"

endpoints: ## Regenerate .vrooli/endpoints.json from API modules (validated against cli/manifest.json)
	@cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true

build: ## Build the scenario
	@if [ -f api/go.mod ]; then \
		cd api && go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f cli/go.mod ]; then \
		cd cli && go build -o "$(SCENARIO_NAME)" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@if command -v gofumpt >/dev/null 2>&1; then \
		[ -d api ] && (cd api && gofumpt -w .); \
		[ -d cli ] && (cd cli && gofumpt -w .); \
	elif command -v gofmt >/dev/null 2>&1; then \
		[ -d api ] && (cd api && gofmt -w .); \
		[ -d cli ] && (cd cli && gofmt -w .); \
	fi

fmt-ui: ## Format UI code
	@true

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@if command -v golangci-lint >/dev/null 2>&1; then \
		[ -d api ] && (cd api && golangci-lint run ./...); \
		[ -d cli ] && (cd cli && golangci-lint run ./...); \
	else \
		[ -d api ] && (cd api && go vet ./...); \
		[ -d cli ] && (cd cli && go vet ./...); \
	fi

lint-ui: ## Lint UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run lint 2>/dev/null || true; \
	fi

check: fmt lint test ## Run full quality gates

# Development shortcuts
run: start
dev: start
