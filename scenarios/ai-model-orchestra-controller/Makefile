# Scenario Makefile for AI Model Orchestra Controller

.PHONY: help run stop test logs status clean build dev fmt lint check install-cli models health

.DEFAULT_GOAL := help

SCENARIO_NAME := ai-model-orchestra-controller
GREEN := \033[1;32m
YELLOW := \033[1;33m  
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help:
	@echo "$(BLUE)🎛️  AI Model Orchestra Controller Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Core Commands:$(RESET)"
	@echo "  $(GREEN)run$(RESET)        Start the scenario"
	@echo "  $(GREEN)stop$(RESET)       Stop the scenario"
	@echo "  $(GREEN)test$(RESET)       Run tests"
	@echo "  $(GREEN)build$(RESET)      Build Go API"
	@echo "  $(GREEN)clean$(RESET)      Clean artifacts"
	@echo ""
	@echo "$(CYAN)AI Orchestra:$(RESET)"
	@echo "  $(GREEN)models$(RESET)     List AI models"
	@echo "  $(GREEN)health$(RESET)     Check system health"
	@echo "  $(GREEN)install-cli$(RESET) Install CLI"

run:
	@echo "$(BLUE)🚀 Starting AI Model Orchestra Controller...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping AI Model Orchestra Controller...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing AI Model Orchestra Controller...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

build:
	@echo "$(BLUE)🏗️  Building AI Model Orchestra Controller...$(RESET)"
	@cd api && go mod tidy && go build -o ai-model-orchestra-controller-api .

clean:
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -f api/ai-model-orchestra-controller-api api/test-build
	@cd api && go clean 2>/dev/null || true

install-cli:
	@echo "$(BLUE)⚙️  Installing ai-orchestra CLI...$(RESET)"
	@cd cli && ./install.sh

models:
	@echo "$(CYAN)🤖 Checking AI models...$(RESET)"
	@ai-orchestra models --include-metrics || echo "Install CLI first: make install-cli"

health:
	@echo "$(CYAN)🏥 Checking system health...$(RESET)"
	@ai-orchestra health || echo "Install CLI first: make install-cli"

# Shortcuts
r: run
s: stop
t: test
b: build
c: clean
h: health
m: models
