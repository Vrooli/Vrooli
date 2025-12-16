# Scenario Makefile
#
# This Makefile ensures the scenario is always run through the Vrooli lifecycle system.
# NEVER run services directly (e.g. `./api/binary`). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs

.PHONY: help start run dev stop status logs test

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

# Colors for output
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
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-10s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME)-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start the scenario services
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

run: start ## Alias for `start`

dev: start ## Alias for `start` (dev lifecycle handled by `.vrooli/service.json`)

stop: ## Stop the scenario services
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

status: ## Show scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

logs: ## Show recent logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

test: ## Run scenario test lifecycle
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

