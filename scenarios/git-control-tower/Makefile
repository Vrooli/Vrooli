# Scenario Makefile
#
# This Makefile ensures the scenario is always run through the Vrooli lifecycle system.
# NEVER run services directly (e.g. `./api/binary`); ALWAYS use `make run`/`make start`.
#
# Usage:
#   make            - Show help
#   make start      - Start this scenario
#   make stop       - Stop this scenario
#   make test       - Run scenario tests
#   make logs       - Show scenario logs
#   make build      - Build API/UI (best-effort; no installs)
#   make clean      - Remove build artifacts
#   make fmt        - Format code (best-effort)
#   make lint       - Lint code (best-effort)
#   make check      - fmt + lint + test

.PHONY: help start run dev stop status logs test build clean fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📦 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Essential Commands:$(RESET)"
	@grep -E '^(start|stop|test|logs|help):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Additional Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -v -E '^(start|stop|test|logs|help):' | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME)-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

run: start ## Alias for `start`

dev: start ## Alias for `start` (dev lifecycle handled by `.vrooli/service.json`)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

status: ## Show scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

logs: ## Show recent logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

test: ## Run scenario tests (uses lifecycle test runner)
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

build: ## Build API/UI (best-effort; does not install deps)
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		echo "$(CYAN)Building Go API...$(RESET)"; \
		( cd api && go build -o git-control-tower-api . ); \
	fi
	@if [ -f ui/package.json ] && [ -d ui/node_modules ]; then \
		echo "$(CYAN)Building UI...$(RESET)"; \
		( cd ui && pnpm run build ); \
	else \
		echo "$(YELLOW)Skipping UI build (deps not installed)$(RESET)"; \
	fi

clean: ## Remove build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/git-control-tower-api 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(RESET)"

fmt: ## Format all code (best-effort)
	@echo "$(BLUE)🎨 Formatting all code...$(RESET)"
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "$(CYAN)Formatting Go...$(RESET)"; \
		gofmt -w api; \
	fi
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		echo "$(CYAN)Formatting Go CLI...$(RESET)"; \
		gofmt -w cli; \
	fi

fmt-ui: ## Format UI code (if prettier available)
	@if command -v prettier >/dev/null 2>&1 && [ -d ui ]; then \
		echo "$(CYAN)Formatting UI...$(RESET)"; \
		( cd ui && prettier --write "**/*.{ts,tsx,js,jsx,json,css,md}" ) || true; \
	else \
		echo "$(YELLOW)Skipping UI formatting (prettier not found)$(RESET)"; \
	fi

lint: ## Lint all code (best-effort)
	@echo "$(BLUE)🔍 Linting all code...$(RESET)"
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Lint Go code
	@if [ -d api ] && [ -f api/go.mod ]; then \
		echo "$(CYAN)Go vet (api)...$(RESET)"; \
		( cd api && go vet ./... ); \
	fi
	@if [ -d cli ] && [ -f cli/go.mod ]; then \
		echo "$(CYAN)Go vet (cli)...$(RESET)"; \
		( cd cli && go vet ./... ); \
	fi

lint-ui: ## Lint UI code (if eslint available)
	@if [ -f ui/package.json ] && command -v pnpm >/dev/null 2>&1 && grep -q '"lint"' ui/package.json; then \
		echo "$(CYAN)UI lint...$(RESET)"; \
		( cd ui && pnpm run lint ) || true; \
	else \
		echo "$(YELLOW)Skipping UI lint (no script/deps)$(RESET)"; \
	fi

check: ## fmt + lint + test
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
