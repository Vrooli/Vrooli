package main

import (
	"strings"
	"testing"
)

func TestCheckMakefileStructure_Valid(t *testing.T) {
	content := `# Demo Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/demo-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📅 $(SCENARIO_NAME) Scenario Commands$(RESET)"
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
	@echo "Starting..."

stop: ## Stop this scenario
	@echo "Stopping..."

test: ## Run scenario tests
	@echo "Testing..."

logs: ## Show scenario logs
	@echo "Logs..."

status: ## Show scenario status
	@echo "Status..."

clean: ## Clean build artifacts
	@echo "Cleaning..."

build: ## Build the scenario
	@echo "Building..."

dev: ## Start development mode
	@echo "Dev mode..."

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "Formatting Go..."

fmt-ui: ## Format UI code
	@echo "Formatting UI..."

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "Linting Go..."

lint-ui: ## Lint UI code
	@echo "Linting UI..."
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileStructure_MissingHeader(t *testing.T) {
	content := `# Demo Scenario Makefile
#
# Missing lifecycle guidance on purpose
# NEVER run scenarios directly (./api/demo-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📅 $(SCENARIO_NAME) Scenario Commands$(RESET)"
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
	@echo "Starting..."

stop: ## Stop this scenario
	@echo "Stopping..."

test: ## Run scenario tests
	@echo "Testing..."

logs: ## Show scenario logs
	@echo "Logs..."

status: ## Show scenario status
	@echo "Status..."

clean: ## Clean build artifacts
	@echo "Cleaning..."

build: ## Build the scenario
	@echo "Building..."

dev: ## Start development mode
	@echo "Dev mode..."

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "Formatting Go..."

fmt-ui: ## Format UI code
	@echo "Formatting UI..."

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "Linting Go..."

lint-ui: ## Lint UI code
	@echo "Linting UI..."
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected violations for missing header guidance")
	}
	found := false
	for _, v := range violations {
		if v.Message == "Header must explain lifecycle requirement" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected violation about lifecycle requirement header")
	}
}

func TestCheckMakefileStructure_PhonyOrder(t *testing.T) {
	content := `# Demo Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/demo-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.DEFAULT_GOAL := help

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📅 $(SCENARIO_NAME) Scenario Commands$(RESET)"
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
	@echo "Starting..."

stop: ## Stop this scenario
	@echo "Stopping..."

test: ## Run scenario tests
	@echo "Testing..."

logs: ## Show scenario logs
	@echo "Logs..."

status: ## Show scenario status
	@echo "Status..."

clean: ## Clean build artifacts
	@echo "Cleaning..."

build: ## Build the scenario
	@echo "Building..."

dev: ## Start development mode
	@echo "Dev mode..."

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "Formatting Go..."

fmt-ui: ## Format UI code
	@echo "Formatting UI..."

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "Linting Go..."

lint-ui: ## Lint UI code
	@echo "Linting UI..."
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.Message == "First directive after header must be .PHONY target declaration" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected violation about .PHONY ordering")
	}
}

func TestCheckMakefileStructure_HelpFirst(t *testing.T) {
	content := `# Demo Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/demo-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"

help: ## Show this help message
	@echo "$(BLUE)📅 $(SCENARIO_NAME) Scenario Commands$(RESET)"
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
	@echo "Starting..."

stop: ## Stop this scenario
	@echo "Stopping..."

test: ## Run scenario tests
	@echo "Testing..."

status: ## Show scenario status
	@echo "Status..."

clean: ## Clean build artifacts
	@echo "Cleaning..."

build: ## Build the scenario
	@echo "Building..."

dev: ## Start development mode
	@echo "Dev mode..."

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "Formatting Go..."

fmt-ui: ## Format UI code
	@echo "Formatting UI..."

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "Linting Go..."

lint-ui: ## Lint UI code
	@echo "Linting UI..."
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.Message == "help target must be defined before any other targets" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected violation about help target ordering")
	}
}

func TestCheckMakefileStructure_MissingTargetDefinitions(t *testing.T) {
	content := `# Demo Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/demo-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📅 $(SCENARIO_NAME) Scenario Commands$(RESET)"
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
	@echo "Starting..."

# Missing: stop, test, logs, status, clean, build, fmt, fmt-go, fmt-ui, lint, lint-go, lint-ui (12 targets)

# Development shortcuts
dev: start
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 12 violations for missing targets
	missingCount := 0
	for _, v := range violations {
		if strings.Contains(v.Message, "Required target") {
			missingCount++
		}
	}
	if missingCount != 12 {
		t.Errorf("expected 12 missing target violations, got %d", missingCount)
	}
}
