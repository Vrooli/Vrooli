# Workspace Sandbox Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/workspace-sandbox-api). ALWAYS use these commands.
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

.PHONY: help setup start stop restart status logs test open run dev endpoints clean build fmt fmt-go fmt-ui lint lint-go lint-ui lint-fix type-check cross-compile check

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

help: ## Show the supported scenario entrypoints
	@printf "Vrooli scenario entrypoints for %s\n\n" "$(SCENARIO_NAME)"
	@printf "  make setup                      Run scenario setup lifecycle\n"
	@printf "  make start                      Start the scenario\n"
	@printf "  make stop                       Stop the scenario\n"
	@printf "  make restart                    Restart the scenario\n"
	@printf "  make status                     Show scenario runtime status\n"
	@printf "  make logs                       Show scenario logs\n"
	@printf "  make test                       Run scenario tests\n"
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make endpoints                  Regenerate .vrooli/endpoints.json from API modules\n"
	@printf "  make run                        Alias for make start\n"
	@printf "  make dev                        Alias for make start\n"
	@printf "\n"
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\n"

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
	@vrooli scenario logs "$(SCENARIO_NAME)" --tail 50

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

clean: ## Clean build artifacts
	@rm -rf ui/dist || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) || true

build: ## Build the scenario
	@if [ -f api/go.mod ]; then \
		cd api && GOWORK=off go build -o "$(SCENARIO_NAME)-api" .; \
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
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd cli && GOWORK=off gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd cli && GOWORK=off gofmt -w .; \
		fi; \
	fi

fmt-ui: ## Format UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm exec eslint src --ext ts,tsx --fix; \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && GOWORK=off go vet ./... && GOWORK=off go fmt ./...; \
		fi; \
	fi
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd cli && golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd cli && GOWORK=off go vet ./... && GOWORK=off go fmt ./...; \
		fi; \
	fi

lint-ui: ## Lint UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run lint && pnpm run type-check; \
	fi

lint-fix: ## Auto-fix UI lint issues when supported
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm exec eslint src --ext ts,tsx --fix; \
	fi

type-check: ## Run UI type checking
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run type-check; \
	fi

cross-compile: ## Cross-compile the api module for the non-Linux OS-seam targets (CGO-free)
	@if [ -f api/go.mod ]; then \
		cd api && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 GOWORK=off go build ./... \
			&& GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOWORK=off go build ./...; \
	fi

check: fmt lint cross-compile test ## Run full quality gates

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

endpoints: ## Regenerate .vrooli/endpoints.json from API modules
	@cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json --seed cmd/gen-endpoints/cli_commands_seed.json

# Development shortcuts
run: start
dev: start
