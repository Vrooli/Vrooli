# Scenario Makefile
# 
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/binary). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make run   - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help run stop test logs status clean build dev

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name from current directory
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
	@echo "    Always use 'make run' or 'vrooli scenario run $(SCENARIO_NAME)'"

run: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@if [ -f test.sh ]; then \
		./test.sh; \
	elif [ -f api/go.mod ]; then \
		cd api && go test ./... -v; \
	elif [ -f ui/package.json ] && [ -f ui/package-lock.json ]; then \
		cd ui && npm test; \
	else \
		vrooli test scenario $(SCENARIO_NAME); \
	fi

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log api/$(SCENARIO_NAME) api/$(SCENARIO_NAME)-api
	@if [ -d api ]; then cd api && go clean 2>/dev/null || true; fi
	@if [ -d ui/dist ]; then rm -rf ui/dist; fi
	@if [ -d ui/build ]; then rm -rf ui/build; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario (if applicable)
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go build -o $(SCENARIO_NAME)-api ./cmd/server/main.go 2>/dev/null || go build -o $(SCENARIO_NAME)-api main.go 2>/dev/null || echo "Could not build Go API"; \
	elif [ -f ui/package.json ]; then \
		cd ui && npm run build 2>/dev/null || echo "Could not build UI"; \
	else \
		echo "$(YELLOW)No build needed for this scenario$(RESET)"; \
	fi

# Shortcuts
r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
