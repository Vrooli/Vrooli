# Scenario To Desktop Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/scenario-to-desktop-api). ALWAYS use these commands.
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
	@printf "  make run                        Alias for make start\n"
	@printf "  make dev                        Alias for make start\n"
	@printf "\n"
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\n"

start: ## Start the scenario
	@vrooli scenario start "$(SCENARIO_NAME)"

stop: ## Stop the scenario
	@vrooli scenario stop "$(SCENARIO_NAME)"

restart: ## Restart the scenario
	@vrooli scenario restart "$(SCENARIO_NAME)"

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

logs: ## Show scenario logs
	@vrooli scenario logs "$(SCENARIO_NAME)" --tail 50

status: ## Show scenario runtime status
	@vrooli scenario status "$(SCENARIO_NAME)"

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -rf api/bin 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true
	@rm -f api/coverage.out runtime/coverage.out 2>/dev/null || true

build: ## Build the scenario
	@if [ -f api/go.mod ]; then \
		cd api && go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@for surface in api cli runtime; do \
		if [ -d "$$surface" ] && find "$$surface" -name "*.go" -print -quit | grep -q .; then \
			if command -v gofumpt >/dev/null 2>&1; then \
				find "$$surface" -name "*.go" -type f -print0 | xargs -0 gofumpt -w; \
			else \
				find "$$surface" -name "*.go" -type f -print0 | xargs -0 gofmt -w; \
			fi; \
		fi; \
	done

fmt-ui: ## Format UI code
	@cd ui && pnpm run lint:fix && pnpm run format

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@for surface in api cli runtime; do \
		if [ -d "$$surface" ] && find "$$surface" -name "*.go" -print -quit | grep -q .; then \
			if command -v golangci-lint >/dev/null 2>&1; then \
				cd "$$surface" && golangci-lint run ./... && cd - >/dev/null; \
			else \
				cd "$$surface" && go vet ./... && cd - >/dev/null; \
			fi; \
		fi; \
	done

lint-ui: ## Lint UI code
	@cd ui && pnpm run lint && pnpm run type-check && pnpm run format:check

check: fmt lint test ## Run full quality gates

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

# Development shortcuts
run: start
dev: start
