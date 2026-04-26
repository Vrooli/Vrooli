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

.PHONY: help start stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check check

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

restart: ## Restart this scenario
	@echo "Restarting..."

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

check: fmt lint test ## Run all checks
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

.PHONY: help start stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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

restart: ## Restart this scenario
	@echo "Restarting..."

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

check: fmt lint test ## Run all checks
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

.PHONY: help start stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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

restart: ## Restart this scenario
	@echo "Restarting..."

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

check: fmt lint test ## Run all checks
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

.PHONY: help start stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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

restart: ## Restart this scenario
	@echo "Restarting..."

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

check: fmt lint test ## Run all checks
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

.PHONY: help start stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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

# Missing: stop, test, logs, status, clean, build, fmt, fmt-go, fmt-ui, lint, lint-go, lint-ui, check (13 targets)

# Development shortcuts
dev: start
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 14 violations for missing targets (17 canonical - 3 defined: help, start, dev)
	missingCount := 0
	for _, v := range violations {
		if strings.Contains(v.Message, "Required target") {
			missingCount++
		}
	}
	if missingCount != 14 {
		t.Errorf("expected 14 missing target violations, got %d", missingCount)
	}
}

// TestCheckMakefileStructure_MissingCheckTarget verifies that a Makefile with
// all targets EXCEPT "check" is correctly flagged. This was a false negative
// prior to adding "check" to the required targets list.
func TestCheckMakefileStructure_MissingCheckTarget(t *testing.T) {
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

.PHONY: help start stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

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

start: ## Start
	@echo "Starting..."
stop: ## Stop
	@echo "Stopping..."
restart: ## Restart
	@echo "Restarting..."
test: ## Test
	@echo "Testing..."
logs: ## Logs
	@echo "Logs..."
status: ## Status
	@echo "Status..."
clean: ## Clean
	@echo "Cleaning..."
build: ## Build
	@echo "Building..."
dev: ## Dev
	@echo "Dev..."
fmt: fmt-go fmt-ui ## Fmt
fmt-go: ## Fmt Go
	@echo "Fmt Go..."
fmt-ui: ## Fmt UI
	@echo "Fmt UI..."
lint: lint-go lint-ui ## Lint
lint-go: ## Lint Go
	@echo "Lint Go..."
lint-ui: ## Lint UI
	@echo "Lint UI..."
`
	violations, err := CheckMakefileStructure(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should flag missing "check" in .PHONY and missing "check" target definition.
	foundPhony := false
	foundTarget := false
	for _, v := range violations {
		if strings.Contains(v.Message, "Missing required targets in .PHONY") {
			foundPhony = true
		}
		if strings.Contains(v.Message, "Required target 'check'") {
			foundTarget = true
		}
	}
	if !foundPhony {
		t.Error("expected violation about missing 'check' in .PHONY")
	}
	if !foundTarget {
		t.Error("expected violation about missing 'check' target definition")
	}
}

// TestCheckMakefileStructure_RequiredTargetsMatchCanonicalSet verifies that
// the required targets list in the structure rule exactly matches the
// required targets defined in makefile_util.go (excluding known aliases like "run").
func TestCheckMakefileStructure_RequiredTargetsMatchCanonicalSet(t *testing.T) {
	required := make(map[string]struct{})
	for _, tgt := range requiredTargets() {
		required[tgt] = struct{}{}
	}
	canonical := required

	// Extract the required list by checking a Makefile that has only "help".
	// Every required target except "help" should produce a
	// "Required target" violation.
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

.PHONY: help

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
`
	violations, _ := CheckMakefileStructure(content, "Makefile")

	// Collect all targets mentioned in "Required target 'X'" violations.
	missingTargets := make(map[string]bool)
	for _, v := range violations {
		if strings.Contains(v.Message, "Required target '") {
			// Extract target name from: "Required target 'X' declared in ..."
			start := strings.Index(v.Message, "'") + 1
			end := strings.Index(v.Message[start:], "'") + start
			missingTargets[v.Message[start:end]] = true
		}
	}

	// Every canonical target except "help" (which is defined) should be missing.
	for target := range canonical {
		if target == "help" {
			continue
		}
		if !missingTargets[target] {
			t.Errorf("canonical target %q is not enforced by the structure rule", target)
		}
	}

	// Every missing target should be in the required set.
	for target := range missingTargets {
		if _, ok := canonical[target]; !ok {
			t.Errorf("structure rule requires %q but it's not in requiredTargets()", target)
		}
	}
}
