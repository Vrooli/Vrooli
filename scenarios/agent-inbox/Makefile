# Agent Inbox Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/binary). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start run dev stop status logs test clean setup build fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📱 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-10s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME) directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start the scenario services
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

run: start ## Start this scenario (alias for 'start')

dev: start ## Start development environment (alias for 'start')

stop: ## Stop the scenario services
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

status: ## Show service status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

logs: ## Show service logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

test: ## Run all test phases
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

setup: build ## Initialize scenario (database, dependencies)
	@echo "$(BLUE)🔧 Setting up $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario setup $(SCENARIO_NAME)
	@echo "$(GREEN)🔧 Setup complete. Use 'make start' to run the scenario.$(RESET)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log api/$(SCENARIO_NAME) api/$(SCENARIO_NAME)-api
	@if [ -d api ]; then cd api && go clean 2>/dev/null || true; fi
	@if [ -d ui/dist ]; then rm -rf ui/dist; fi
	@if [ -d ui/build ]; then rm -rf ui/build; fi
	@if [ -d ui/node_modules ]; then rm -rf ui/node_modules; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario (if applicable)
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		echo "$(BLUE)  Building Go API...$(RESET)"; \
		cd api && go build -o $(SCENARIO_NAME)-api .; \
	fi
	@if [ -f ui/package.json ]; then \
		echo "$(BLUE)  Installing UI dependencies...$(RESET)"; \
		cd ui && pnpm install; \
	fi
	@echo "$(GREEN)✓ Build complete$(RESET)"

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "$(BLUE)📝 Formatting Go code...$(RESET)"
	@if [ -d api ]; then \
		if command -v gofumpt &>/dev/null; then \
			cd api && gofumpt -w . 2>/dev/null || true; \
		else \
			echo "$(YELLOW)⚠️  gofumpt not found, skipping Go formatting$(RESET)"; \
		fi \
	fi

fmt-ui: ## Format UI code
	@echo "$(BLUE)📝 Formatting UI code...$(RESET)"
	@if [ -d ui ] && [ -f ui/package.json ]; then \
		if command -v prettier &>/dev/null; then \
			cd ui && prettier --write "src/**/*.{ts,tsx,js,jsx,json,css}" 2>/dev/null || true; \
		else \
			echo "$(YELLOW)⚠️  prettier not found, skipping UI formatting$(RESET)"; \
		fi \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@if [ -d api ]; then \
		if command -v golangci-lint &>/dev/null; then \
			cd api && golangci-lint run 2>/dev/null || true; \
		else \
			echo "$(YELLOW)⚠️  golangci-lint not found, running go vet$(RESET)"; \
			cd api && go vet ./...; \
		fi \
	fi

lint-ui: ## Lint UI code
	@echo "$(BLUE)🔍 Linting UI code...$(RESET)"
	@if [ -d ui ] && [ -f ui/package.json ]; then \
		if grep -q '"lint"' ui/package.json 2>/dev/null; then \
			cd ui && pnpm run lint 2>/dev/null || true; \
		else \
			echo "$(YELLOW)⚠️  No lint script in package.json$(RESET)"; \
		fi \
	fi

check: fmt lint test ## Format, lint, and test code (pre-commit workflow)
