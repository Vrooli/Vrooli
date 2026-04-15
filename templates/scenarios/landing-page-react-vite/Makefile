# Landing Page React Vite Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/{{SCENARIO_ID}}-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📦 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME)-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test: ## Run scenario tests
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show scenario logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Show scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@for dir in api cli; do \
		if [ -d $$dir ] && find $$dir -name "*.go" | head -1 | grep -q .; then \
			echo "$(BLUE)🎨 Formatting Go code in $$dir...$(RESET)"; \
			if command -v gofumpt >/dev/null 2>&1; then \
				cd $$dir && gofumpt -w . && cd ..; \
			elif command -v gofmt >/dev/null 2>&1; then \
				cd $$dir && gofmt -w . && cd ..; \
			fi; \
			echo "$(GREEN)✓ Go code formatted in $$dir$(RESET)"; \
		fi; \
	done

fmt-ui: ## Format UI code
	@if [ -f ui/package.json ]; then \
		echo "$(BLUE)🎨 Formatting UI code...$(RESET)"; \
		cd ui && pnpm run lint:fix; \
		echo "$(GREEN)✓ UI code formatted$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to format$(RESET)"; \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@for dir in api cli; do \
		if [ -d $$dir ] && find $$dir -name "*.go" | head -1 | grep -q .; then \
			echo "$(BLUE)🔍 Linting Go code in $$dir...$(RESET)"; \
			if command -v golangci-lint >/dev/null 2>&1; then \
				cd $$dir && golangci-lint run ./... && cd ..; \
			elif command -v go >/dev/null 2>&1; then \
				cd $$dir && go vet ./... && cd ..; \
			fi; \
			echo "$(GREEN)✓ Go code linted in $$dir$(RESET)"; \
		fi; \
	done

lint-ui: ## Lint UI code
	@if [ -f ui/package.json ]; then \
		echo "$(BLUE)🔍 Linting UI code...$(RESET)"; \
		cd ui && pnpm run lint && pnpm run type-check; \
		echo "$(GREEN)✓ UI code linted$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to lint$(RESET)"; \
	fi

check: fmt lint test ## Run full quality gates

# Development shortcuts
dev: start
