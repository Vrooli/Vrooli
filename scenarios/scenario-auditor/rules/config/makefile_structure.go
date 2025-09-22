package config

import (
	"fmt"
	"regexp"
	"strings"
)

/*
Rule: Makefile Core Structure
Description: Enforces the canonical Makefile header, phony declarations, goal, scenario variable, colour palette, and help target structure
Reason: Consistent scaffolding keeps lifecycle messaging accurate for agents and humans
Category: makefile
Severity: high
Targets: makefile

<test-case id="structure-valid" should-fail="false">
  <description>Canonical Makefile follows the template</description>
  <input language="make">
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

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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
  </input>
</test-case>

<test-case id="structure-missing-header" should-fail="true">
  <description>Header lines missing required guidance</description>
  <input language="make">
# Demo Scenario Makefile

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help
SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help:
	@echo "missing guidance"
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Header must explain lifecycle requirement</expected-message>
</test-case>

*/

type MakefileStructureViolation struct {
	Severity string
	Message  string
	FilePath string
	Line     int
}

type structureMakefileData struct {
	lines        []string
	header       []string
	phony        []string
	defaultGoal  string
	scenarioName string
	colors       map[string]string
	targets      map[string][]string
}

var structureTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)

// CheckMakefileStructure ensures the Makefile conforms to the required structure scaffolding.
func CheckMakefileStructure(content string, filepath string) ([]MakefileStructureViolation, error) {
	data := parseStructureMakefile(content)
	var violations []MakefileStructureViolation

	violations = append(violations, structureValidateHeader(data, filepath)...)
	violations = append(violations, structureValidatePhony(data, filepath)...)
	violations = append(violations, structureValidateDefaults(data, filepath)...)
	violations = append(violations, structureValidateColors(data, filepath)...)
	violations = append(violations, structureValidateHelp(data, filepath)...)

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
		}, "Header must start with '# <Scenario> Scenario Makefile'"},
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
	required := []string{"help", "start", "stop", "test", "logs", "status", "clean", "build", "dev", "fmt", "fmt-go", "fmt-ui", "lint", "lint-go", "lint-ui", "check"}
	var violations []MakefileStructureViolation

	if len(data.phony) == 0 {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  ".PHONY declaration missing required targets",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".PHONY"),
		})
		return violations
	}

	if data.phony[0] != "help" {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "help must be the first entry in .PHONY",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".PHONY"),
		})
	}

	if !structureContainsAll(required, data.phony) {
		violations = append(violations, MakefileStructureViolation{
			Severity: "high",
			Message:  "Missing required targets in .PHONY",
			FilePath: path,
			Line:     structureFindLine(data.lines, ".PHONY"),
		})
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

	return violations
}

func structureValidateHelp(data structureMakefileData, path string) []MakefileStructureViolation {
	lines := structureNormalizeRecipes(data.targets["help"])
	requiredSnippets := []string{
		"@echo \"$(BLUE)",
		"Scenario Commands$(RESET)\"",
		"@echo \"$(YELLOW)Usage:$(RESET)\"",
		"@echo \"  make <command>\"",
		"@echo \"$(YELLOW)Commands:$(RESET)\"",
		"@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST)",
		"awk 'BEGIN {FS = \":.*?## \"}; {printf \"  $(GREEN)%-",
		"Never run ./api/",
		"Always use 'make start' or 'vrooli scenario start",
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

	return violations
}

func parseStructureMakefile(content string) structureMakefileData {
	lines := strings.Split(content, "\n")
	data := structureMakefileData{
		lines:   lines,
		header:  []string{},
		colors:  map[string]string{},
		targets: make(map[string][]string),
	}

	var currentTarget string
	for i, raw := range lines {
		trimmedLeft := strings.TrimLeft(raw, "\t ")

		if len(data.header) == 0 {
			data.header = structureExtractHeader(lines)
		}

		if strings.HasPrefix(trimmedLeft, ".PHONY:") {
			data.phony = structureParseList(trimmedLeft)
			continue
		}

		if strings.HasPrefix(trimmedLeft, ".DEFAULT_GOAL") {
			parts := strings.Split(trimmedLeft, ":=")
			if len(parts) == 2 {
				data.defaultGoal = strings.TrimSpace(parts[1])
			}
			continue
		}

		if strings.HasPrefix(trimmedLeft, "SCENARIO_NAME") {
			parts := strings.Split(trimmedLeft, ":=")
			if len(parts) == 2 {
				data.scenarioName = strings.TrimSpace(parts[1])
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
				}
			}
		}

		if strings.HasPrefix(raw, "\t") && currentTarget != "" {
			data.targets[currentTarget] = append(data.targets[currentTarget], raw)
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
			continue
		}

		currentTarget = ""
		_ = i
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
	return 1
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
