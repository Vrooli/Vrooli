package config

import (
	"fmt"
	"regexp"
	"strings"
)

/*
Rule: Makefile Core Structure
Description: Enforces canonical Makefile structure with STRICT consistency for interoperability. All scenarios must follow identical structure including fmt-go/lint-go/fmt-ui/lint-ui targets (use no-op stubs if needed).
Reason: STRICT consistency ensures agents and humans can rely on standard targets across all scenarios. Any deviation breaks tooling and creates confusion.
Category: makefile
Severity: high
Targets: makefile

<test-case id="structure-valid" should-fail="false">
  <description>Canonical Makefile follows the template</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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
]]></input>
</test-case>

<test-case id="structure-missing-header" should-fail="true">
  <description>Header lines missing required guidance</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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
]]></input>
<expected-violations>1</expected-violations>
<expected-message>Header must explain lifecycle requirement</expected-message>
</test-case>

<test-case id="structure-order-phony" should-fail="true">
  <description>.PHONY must be the first directive after the header</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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
]]></input>
<expected-violations>2</expected-violations>
<expected-message>.PHONY target declaration</expected-message>
</test-case>

<test-case id="structure-help-first" should-fail="true">
  <description>help target must be defined before other targets</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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
]]></input>
<expected-violations>2</expected-violations>
<expected-message>help target must be defined before any other targets</expected-message>
</test-case>

<test-case id="structure-shortcuts-terminal" should-fail="true">
  <description>Shortcut targets must be the final content in the Makefile</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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

# Development shortcuts
dev: start
restart: stop start

.PHONY: make-executable
make-executable:
	@echo "fix permissions"
]]></input>
<expected-violations>3</expected-violations>
<expected-message>Shortcut targets must be the final content in the Makefile</expected-message>
</test-case>

<test-case id="structure-shortcuts-valid" should-fail="false">
  <description>Shortcut aliases are present and terminate the file</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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

# Development shortcuts
dev: start
restart: stop start
rebuild: clean build
]]></input>
</test-case>

<test-case id="structure-missing-target-definitions" should-fail="true">
  <description>Targets declared in .PHONY but not defined</description>
  <input language="make"><![CDATA[
# Demo Scenario Makefile
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
]]></input>
<expected-violations>12</expected-violations>
<expected-message>Required target</expected-message>
</test-case>

*/

type MakefileStructureViolation struct {
	Severity string
	Message  string
	FilePath string
	Line     int
}

type structureMakefileData struct {
	lines                []string
	header               []string
	phony                []string
	defaultGoal          string
	scenarioName         string
	colors               map[string]string
	targets              map[string][]string
	targetRecipeLines    map[string][]int
	colorLines           map[string]int
	phonyLine            int
	defaultLine          int
	scenarioLine         int
	helpLine             int
	targetOrder          []string
	targetLines          map[string]int
	shortcutsCommentLine int
	shortcutTargets      []string
	shortcutTargetLines  []int
	firstNonHeaderLine   int
	headerEndLine        int
	lastNonEmptyLine     int
}

var structureTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)
var shortcutAliasPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+(?:\s+[A-Za-z0-9_.-]+)*$`)

// CheckMakefileStructure ensures the Makefile conforms to the required structure scaffolding.
func CheckMakefileStructure(content string, filepath string) ([]MakefileStructureViolation, error) {
	data := parseStructureMakefile(content)
	var violations []MakefileStructureViolation

	violations = append(violations, structureValidateHeader(data, filepath)...)
	violations = append(violations, structureValidatePhony(data, filepath)...)
	violations = append(violations, structureValidateDefaults(data, filepath)...)
	violations = append(violations, structureValidateColors(data, filepath)...)
	violations = append(violations, structureValidateHelp(data, filepath)...)
	violations = append(violations, structureValidateOrdering(data, filepath)...)
	violations = append(violations, structureValidateShortcuts(data, filepath)...)

	return violations, nil
}

func structureValidateHeader(data structureMakefileData, path string) []MakefileStructureViolation {
	requiredLines := []struct {
		index   int
		check   func(string) bool
		message string
	}{
		{0, func(line string) bool {
			return strings.HasPrefix(strings.TrimSpace(line), "# ") && strings.HasSuffix(strings.TrimSpace(line), "Scenario Makefile")
		}, "Header must be a comment ending with 'Scenario Makefile' (e.g., '# Demo Scenario Makefile')"},
		{1, func(line string) bool { return strings.TrimSpace(line) == "#" }, "Second header line must be '# '"},
		{2, func(line string) bool {
			return strings.TrimSpace(line) == "# This Makefile ensures scenarios are always run through the Vrooli lifecycle system."
		}, "Header must explain lifecycle requirement"},
		{3, func(line string) bool {
			trimmed := strings.TrimSpace(line)
			return strings.HasPrefix(trimmed, "# NEVER run scenarios directly (") && strings.Contains(trimmed, "). ALWAYS use these commands.")
		}, "Header must warn against direct execution"},
		{4, func(line string) bool { return strings.TrimSpace(line) == "#" }, "Header spacing line missing"},
		{5, func(line string) bool { return strings.TrimSpace(line) == "# Usage:" }, "Header must introduce usage section"},
		{6, func(line string) bool { return strings.TrimSpace(line) == "#   make       - Show help" }, "Usage entry for 'make' missing"},
		{7, func(line string) bool { return strings.TrimSpace(line) == "#   make start - Start this scenario" }, "Usage entry for 'make start' missing"},
		{8, func(line string) bool { return strings.TrimSpace(line) == "#   make stop  - Stop this scenario" }, "Usage entry for 'make stop' missing"},
		{9, func(line string) bool { return strings.TrimSpace(line) == "#   make test  - Run scenario tests" }, "Usage entry for 'make test' missing"},
		{10, func(line string) bool { return strings.TrimSpace(line) == "#   make logs  - Show scenario logs" }, "Usage entry for 'make logs' missing"},
		{11, func(line string) bool { return strings.TrimSpace(line) == "#   make clean - Clean build artifacts" }, "Usage entry for 'make clean' missing"},
	}

	var violations []MakefileStructureViolation
	if len(data.header) < len(requiredLines) {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "Makefile header is incomplete",
			FilePath: path,
			Line:     1,
		})
		return violations
	}

	for _, requirement := range requiredLines {
		line := data.header[requirement.index]
		if !requirement.check(line) {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  requirement.message,
				FilePath: path,
				Line:     requirement.index + 1,
			})
		}
	}

	return violations
}

func structureValidatePhony(data structureMakefileData, path string) []MakefileStructureViolation {
	// STRICT: All targets required for consistency and interoperability across scenarios
	// Scenarios without UI/Go code should provide no-op implementations
	required := []string{"help", "start", "stop", "test", "logs", "status", "clean", "build", "dev", "fmt", "fmt-go", "fmt-ui", "lint", "lint-go", "lint-ui"}
	var violations []MakefileStructureViolation

	if len(data.phony) == 0 {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  ".PHONY declaration missing required targets (STRICT: required for consistency and interoperability)",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".PHONY"),
		})
		return violations
	}

	if data.phony[0] != "help" {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "help must be the first entry in .PHONY (STRICT: maintains standard structure)",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".PHONY"),
		})
	}

	if !structureContainsAll(required, data.phony) {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "Missing required targets in .PHONY (STRICT: required for consistency. Use no-op implementations if needed)",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".PHONY"),
		})
	}

	// Validate that all required targets actually exist as definitions
	for _, target := range required {
		if _, exists := data.targets[target]; !exists {
			// Find the line where this target should be defined, or default to .PHONY line
			lineNum := data.phonyLine
			if lineNum == 0 {
				lineNum = structureFindLine(data.lines, ".PHONY")
			}
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  fmt.Sprintf("Required target '%s' declared in .PHONY but not defined (STRICT: all .PHONY targets must have definitions)", target),
				FilePath: path,
				Line:     lineNum,
			})
		}
	}

	return violations
}

func structureValidateDefaults(data structureMakefileData, path string) []MakefileStructureViolation {
	var violations []MakefileStructureViolation

	if data.defaultGoal != "help" {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  ".DEFAULT_GOAL must be set to 'help'",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".DEFAULT_GOAL"),
		})
	}

	if data.scenarioName != "$(notdir $(CURDIR))" {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "SCENARIO_NAME must be defined as '$(notdir $(CURDIR))'",
			FilePath: path,
			Line:     structureFindLine(data.lines, "SCENARIO_NAME"),
		})
	}

	return violations
}

func structureValidateColors(data structureMakefileData, path string) []MakefileStructureViolation {
	expected := map[string]string{
		"GREEN":  "\\033[1;32m",
		"YELLOW": "\\033[1;33m",
		"BLUE":   "\\033[1;34m",
		"RED":    "\\033[1;31m",
		"RESET":  "\\033[0m",
	}

	var violations []MakefileStructureViolation

	// Check that all expected colors are defined correctly
	for name, value := range expected {
		if data.colors[name] != value {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  fmt.Sprintf("Color %s must be defined exactly as '%s'", name, value),
				FilePath: path,
				Line:     structureFindLine(data.lines, name+" :="),
			})
		}
	}

	// Allow additional color definitions without failing; enforce only the canonical palette above.

	return violations
}

func structureValidateHelp(data structureMakefileData, path string) []MakefileStructureViolation {
	lines := structureNormalizeRecipes(data.targets["help"])
	// STRICT: Required for consistent help output across all scenarios
	requiredSnippets := []string{
		"@echo \"$(BLUE)",
		"Scenario Commands$(RESET)\"",
		"@echo \"$(YELLOW)Usage:$(RESET)\"",
		"@echo \"  make <command>\"",
		"@echo \"$(YELLOW)Commands:$(RESET)\"",
		"@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST)",
		"awk",  // Relaxed: allow any awk formatting
		"printf", // Must use printf for formatting
		"$(GREEN)", // Must use color variables
		"Never run ./api/",
	}

	var violations []MakefileStructureViolation
	if len(lines) == 0 {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "help target is missing",
			FilePath: path,
			Line:     structureFindLine(data.lines, "help:"),
		})
		return violations
	}

	for _, snippet := range requiredSnippets {
		if !structureContainsSnippet(lines, snippet) {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  fmt.Sprintf("help target must include text containing '%s'", snippet),
				FilePath: path,
				Line:     structureFindLine(data.lines, "help:"),
			})
		}
	}

	// Check for either 'make start' or 'make run' variant
	hasStartOrRun := structureContainsSnippet(lines, "Always use 'make start' or 'vrooli scenario start") ||
		structureContainsSnippet(lines, "Always use 'make run' or 'vrooli scenario start")
	if !hasStartOrRun {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "help target must include text containing \"Always use 'make start' or 'vrooli scenario start\" (or 'make run' variant)",
			FilePath: path,
			Line:     structureFindLine(data.lines, "help:"),
		})
	}

	return violations
}

func structureValidateOrdering(data structureMakefileData, path string) []MakefileStructureViolation {
	var violations []MakefileStructureViolation

	if data.firstNonHeaderLine > 0 {
		firstLine := strings.TrimSpace(data.lines[data.firstNonHeaderLine-1])
		if !strings.HasPrefix(firstLine, ".PHONY") {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  "First directive after header must be .PHONY target declaration",
				FilePath: path,
				Line:     data.firstNonHeaderLine,
			})
		}
	}

	if data.phonyLine > 0 && data.defaultLine > 0 && data.defaultLine < data.phonyLine {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  ".DEFAULT_GOAL must be defined after the .PHONY block",
			FilePath: path,
			Line:     data.defaultLine,
		})
	}

	if data.defaultLine > 0 && data.scenarioLine > 0 && data.scenarioLine < data.defaultLine {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "SCENARIO_NAME definition must follow .DEFAULT_GOAL",
			FilePath: path,
			Line:     data.scenarioLine,
		})
	}

	colorOrder := []string{"GREEN", "YELLOW", "BLUE", "RED", "RESET"}
	prevLine := 0
	prevName := ""
	firstColorLine := 0
	lastColorLine := 0
	for _, name := range colorOrder {
		line := data.colorLines[name]
		if line == 0 {
			continue
		}
		if prevLine > 0 && line < prevLine {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  fmt.Sprintf("Color %s must be declared after %s to preserve palette order", name, prevName),
				FilePath: path,
				Line:     line,
			})
		}
		if firstColorLine == 0 || line < firstColorLine {
			firstColorLine = line
		}
		if line > lastColorLine {
			lastColorLine = line
		}
		prevLine = line
		prevName = name
	}

	if data.scenarioLine > 0 && firstColorLine > 0 {
		for i := data.scenarioLine; i < firstColorLine-1; i++ {
			if !structureIsCommentOrBlank(data.lines[i]) {
				violations = append(violations, MakefileStructureViolation{
					Severity: "high",
					Message:  "Color palette must immediately follow SCENARIO_NAME declaration",
					FilePath: path,
					Line:     i + 1,
				})
				break
			}
		}
	}

	if lastColorLine > 0 && data.helpLine > 0 {
		if data.helpLine < lastColorLine {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  "help target must appear after color palette definitions",
				FilePath: path,
				Line:     data.helpLine,
			})
		} else {
			for i := lastColorLine; i < data.helpLine-1; i++ {
				if !structureIsCommentOrBlank(data.lines[i]) {
					violations = append(violations, MakefileStructureViolation{
						Severity: "high",
						Message:  "No additional directives allowed between color palette and help target",
						FilePath: path,
						Line:     i + 1,
					})
					break
				}
			}
		}
	}

	if len(data.targetOrder) > 0 && data.targetOrder[0] != "help" {
		first := data.targetOrder[0]
		line := data.targetLines[first]
		if line == 0 {
			line = data.firstNonHeaderLine
		}
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "help target must be defined before any other targets",
			FilePath: path,
			Line:     line,
		})
	}

	return violations
}

func structureValidateShortcuts(data structureMakefileData, path string) []MakefileStructureViolation {
	if data.shortcutsCommentLine == 0 {
		return nil
	}

	var violations []MakefileStructureViolation
	if len(data.shortcutTargets) == 0 {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "Shortcut section must define at least one shortcut target",
			FilePath: path,
			Line:     data.shortcutsCommentLine,
		})
		return violations
	}

	shortcutLines := make(map[int]struct{})
	for idx, target := range data.shortcutTargets {
		line := data.shortcutTargetLines[idx]
		shortcutLines[line] = struct{}{}
		if len(data.targetRecipeLines[target]) > 0 {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  fmt.Sprintf("Shortcut target '%s' must be a single-line alias without its own recipe", target),
				FilePath: path,
				Line:     line,
			})
		}
		for _, recipeLine := range data.targetRecipeLines[target] {
			shortcutLines[recipeLine] = struct{}{}
		}
		for _, snippet := range data.targets[target] {
			trimmed := strings.TrimSpace(strings.TrimPrefix(snippet, "\t"))
			if trimmed == "" {
				continue
			}
			if !shortcutAliasPattern.MatchString(trimmed) {
				violations = append(violations, MakefileStructureViolation{
					Severity: "high",
					Message:  fmt.Sprintf("Shortcut target '%s' must only reference other targets", target),
					FilePath: path,
					Line:     line,
				})
				break
			}
		}
	}

	for i := data.shortcutsCommentLine + 1; i < len(data.lines); i++ {
		trimmed := strings.TrimSpace(data.lines[i])
		if trimmed == "" {
			continue
		}
		if _, ok := shortcutLines[i+1]; ok {
			continue
		}
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "Shortcut targets must be the final content in the Makefile",
			FilePath: path,
			Line:     i + 1,
		})
		break
	}

	return violations
}

func parseStructureMakefile(content string) structureMakefileData {
	lines := strings.Split(content, "\n")
	header := structureExtractHeader(lines)
	data := structureMakefileData{
		lines:             lines,
		header:            header,
		headerEndLine:     len(header),
		colors:            map[string]string{},
		colorLines:        make(map[string]int),
		targets:           make(map[string][]string),
		targetLines:       make(map[string]int),
		targetRecipeLines: make(map[string][]int),
	}

	var currentTarget string
	for i, raw := range lines {
		trimmedLeft := strings.TrimLeft(raw, "\t ")
		trimmed := strings.TrimSpace(raw)

		if trimmed != "" {
			data.lastNonEmptyLine = i + 1
		}

		if data.firstNonHeaderLine == 0 && i >= data.headerEndLine {
			if !structureIsCommentOrBlank(raw) {
				data.firstNonHeaderLine = i + 1
			}
		}

		if strings.HasPrefix(strings.TrimLeft(raw, " \t"), "#") {
			// STRICT: Only accept exact "# Development shortcuts" format
			if trimmed == "# Development shortcuts" && data.shortcutsCommentLine == 0 {
				data.shortcutsCommentLine = i + 1
			}
		}

		if strings.HasPrefix(trimmedLeft, ".PHONY:") {
			if data.phonyLine == 0 {
				data.phony = structureParseList(trimmedLeft)
				data.phonyLine = i + 1
			}
			continue
		}

		if strings.HasPrefix(trimmedLeft, ".DEFAULT_GOAL") {
			parts := strings.Split(trimmedLeft, ":=")
			if len(parts) == 2 {
				data.defaultGoal = strings.TrimSpace(parts[1])
			}
			if data.defaultLine == 0 {
				data.defaultLine = i + 1
			}
			continue
		}

		if strings.HasPrefix(trimmedLeft, "SCENARIO_NAME") {
			parts := strings.Split(trimmedLeft, ":=")
			if len(parts) == 2 {
				data.scenarioName = strings.TrimSpace(parts[1])
			}
			if data.scenarioLine == 0 {
				data.scenarioLine = i + 1
			}
			continue
		}

		if strings.Contains(trimmedLeft, ":=") {
			assignParts := strings.SplitN(trimmedLeft, ":=", 2)
			if len(assignParts) == 2 {
				name := strings.TrimSpace(assignParts[0])
				value := strings.TrimSpace(assignParts[1])
				switch name {
				case "GREEN", "YELLOW", "BLUE", "RED", "RESET", "CYAN":
					data.colors[name] = value
					if _, recorded := data.colorLines[name]; !recorded {
						data.colorLines[name] = i + 1
					}
				}
			}
			continue
		}

		if strings.HasPrefix(raw, "\t") && currentTarget != "" {
			data.targets[currentTarget] = append(data.targets[currentTarget], raw)
			data.targetRecipeLines[currentTarget] = append(data.targetRecipeLines[currentTarget], i+1)
			continue
		}

		matches := structureTargetRegexp.FindStringSubmatch(trimmedLeft)
		if len(matches) == 3 {
			currentTarget = matches[1]
			if _, ok := data.targets[currentTarget]; !ok {
				data.targets[currentTarget] = []string{}
			}
			remainder := strings.TrimSpace(matches[2])
			if remainder != "" {
				data.targets[currentTarget] = append(data.targets[currentTarget], "\t"+remainder)
			}
			if _, seen := data.targetLines[currentTarget]; !seen {
				data.targetLines[currentTarget] = i + 1
				data.targetOrder = append(data.targetOrder, currentTarget)
				if currentTarget == "help" {
					data.helpLine = i + 1
				}
				if data.shortcutsCommentLine > 0 && i+1 > data.shortcutsCommentLine {
					data.shortcutTargets = append(data.shortcutTargets, currentTarget)
					data.shortcutTargetLines = append(data.shortcutTargetLines, i+1)
				}
			}
			continue
		}

		currentTarget = ""
	}

	return data
}

func structureExtractHeader(lines []string) []string {
	header := []string{}
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			header = append(header, raw)
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(raw, " \t"), "#") {
			header = append(header, raw)
			continue
		}
		break
	}
	return header
}

func structureParseList(line string) []string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	tokens := strings.Fields(strings.ReplaceAll(parts[1], ",", " "))
	return tokens
}

func structureNormalizeRecipes(lines []string) []string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized = append(normalized, strings.TrimSpace(line))
	}
	return normalized
}

func structureFindLine(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 0 // Return 0 to indicate "not found" instead of defaulting to line 1
}

func structureContainsAll(required []string, actual []string) bool {
	set := make(map[string]struct{}, len(required))
	for _, item := range required {
		set[item] = struct{}{}
	}
	for _, item := range actual {
		delete(set, item)
	}
	return len(set) == 0
}

func structureContainsSnippet(lines []string, snippet string) bool {
	for _, line := range lines {
		if strings.Contains(line, snippet) {
			return true
		}
	}
	return false
}

func structureIsCommentOrBlank(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(trimmed, "#")
}
