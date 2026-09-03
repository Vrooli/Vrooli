# System Monitor Scenario Makefile
#
# This Makefile is a thin wrapper around the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/system-monitor-api). ALWAYS use these targets
# or the `vrooli scenario ...` CLI, which owns all user-facing output.

.PHONY: help setup start stop restart status logs test open run dev clean build fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

SCENARIO_CUSTOM_TARGETS += fmt-ui lint-go lint-ui
# Shared toolchain floor (mk/toolchain.mk) and build targets (mk/scenario.mk); a custom body for one of them is declared in SCENARIO_CUSTOM_TARGETS.
-include ../../mk/scenario.mk

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

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

run: start

dev: start

# --- Scenario-specific quality targets (not part of the lifecycle wrapper) ---

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true

fmt: fmt-go fmt-ui ## Format all code

fmt-ui: ## Format UI code
	@if [ -f ui/package.json ]; then cd ui && pnpm run format 2>/dev/null || true; fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && go vet ./...; \
		fi; \
	fi

lint-ui: ## Lint UI code (eslint + tsc type-check)
	@if [ -f ui/package.json ]; then cd ui && pnpm run lint && pnpm run type-check; fi

check: fmt lint test ## Run full quality gates
