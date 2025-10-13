package config

import (
	"fmt"
	"regexp"
	"strings"
)

/*
Rule: Makefile Quality Targets
Description: Validates fmt/lint/check targets invoke the canonical sub-commands and enforce strict Go formatting/linting logic
Reason: Keeps code quality workflows discoverable and consistent across scenarios
Category: makefile
Severity: medium
Targets: makefile

<test-case id="quality-valid" should-fail="false">
  <description>Quality targets follow template structure</description>
  <input language="make"><![CDATA[
fmt:
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

lint:
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

check:
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
  ]]></input>
</test-case>

<test-case id="quality-missing-lint-go" should-fail="true">
  <description>Lint target does not call lint-go</description>
  <input language="make"><![CDATA[
fmt:
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

fmt-ui:
	@echo "Formatting UI assets..."

lint:
	@$(MAKE) lint-ui

lint-ui:
	@echo "Linting UI code..."

lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

check:
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lint target must include '@$(MAKE) lint-go'</expected-message>
</test-case>

*/

type MakefileQualityViolation struct {
	Severity string
	Message  string
	FilePath string
	Line     int
}

type qualityMakefileData struct {
	lines   []string
	targets map[string][]string
}

var qualityTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)

// CheckMakefileQuality ensures quality-related targets meet expectations.
func CheckMakefileQuality(content string, filepath string) ([]MakefileQualityViolation, error) {
	data := parseQualityMakefile(content)
	var violations []MakefileQualityViolation

	violations = append(violations, qualityEnsureContains(data, filepath, "check", []string{"@$(MAKE) fmt", "@$(MAKE) lint", "@$(MAKE) test"})...)
	violations = append(violations, qualityEnsureContains(data, filepath, "lint", []string{"@$(MAKE) lint-go"})...)
	violations = append(violations, qualityEnsureContains(data, filepath, "fmt", []string{"@$(MAKE) fmt-go"})...)
	violations = append(violations, qualityEnsureMatches(data, filepath, "lint-go", qualityLintGoTemplate())...)
	violations = append(violations, qualityEnsureMatches(data, filepath, "fmt-go", qualityFmtGoTemplate())...)

	return violations, nil
}

func qualityEnsureContains(data qualityMakefileData, path, target string, snippets []string) []MakefileQualityViolation {
	recipe := qualityNormalize(data.targets[target])
	var violations []MakefileQualityViolation
	if len(recipe) == 0 {
		violations = append(violations, MakefileQualityViolation{
			Severity: "medium",
			Message:  fmt.Sprintf("%s target missing", target),
			FilePath: path,
			Line:     qualityFindLine(data.lines, target+":"),
		})
		return violations
	}

	for _, snippet := range snippets {
		if !qualityContains(recipe, snippet) {
			violations = append(violations, MakefileQualityViolation{
				Severity: "medium",
				Message:  fmt.Sprintf("%s target must include '%s'", target, snippet),
				FilePath: path,
				Line:     qualityFindLine(data.lines, target+":"),
			})
		}
	}
	return violations
}

func qualityEnsureMatches(data qualityMakefileData, path, target string, expected []string) []MakefileQualityViolation {
	recipe := qualityNormalize(data.targets[target])
	if len(recipe) == 0 {
		return []MakefileQualityViolation{
			{
				Severity: "medium",
				Message:  fmt.Sprintf("%s target missing", target),
				FilePath: path,
				Line:     qualityFindLine(data.lines, target+":"),
			},
		}
	}

	var requiredFragments []string
	switch target {
	case "lint-go":
		requiredFragments = []string{
			"find api -name \"*.go\"",
			"golangci-lint run",
			"go vet ./...",
			"Go code linted",
		}
	case "fmt-go":
		requiredFragments = []string{
			"find api -name \"*.go\"",
			"Formatting Go code",
			"gofumpt -w .",
			"gofmt -w .",
			"Go code formatted",
		}
	default:
		requiredFragments = expected
	}

	var violations []MakefileQualityViolation
	for _, fragment := range requiredFragments {
		if !qualityContains(recipe, fragment) {
			violations = append(violations, MakefileQualityViolation{
				Severity: "medium",
				Message:  fmt.Sprintf("%s target must include '%s'", target, fragment),
				FilePath: path,
				Line:     qualityFindLine(data.lines, target+":"),
			})
		}
	}

	return violations
}

func parseQualityMakefile(content string) qualityMakefileData {
	lines := strings.Split(content, "\n")
	data := qualityMakefileData{
		lines:   lines,
		targets: make(map[string][]string),
	}

	var currentTarget string
	for _, raw := range lines {
		trimmed := strings.TrimLeft(raw, "\t ")

		if strings.HasPrefix(raw, "\t") && currentTarget != "" {
			data.targets[currentTarget] = append(data.targets[currentTarget], raw)
			continue
		}

		matches := qualityTargetRegexp.FindStringSubmatch(trimmed)
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
	}

	return data
}

func qualityNormalize(lines []string) []string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized = append(normalized, strings.TrimSpace(line))
	}
	return normalized
}

func qualityContains(lines []string, fragment string) bool {
	for _, line := range lines {
		if strings.Contains(line, fragment) {
			return true
		}
	}
	return false
}

func qualityFindLine(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 1
}

func qualityLintGoTemplate() []string {
	return []string{
		"@if [ -d api ] && find api -name \"*.go\" | head -1 | grep -q .; then \\",
		"\techo \"Linting Go code...\"; \\",
		"\tif command -v golangci-lint >/dev/null 2>&1; then \\",
		"\t\tcd api && golangci-lint run; \\",
		"\telse \\",
		"\t\tcd api && go vet ./...; \\",
		"\tfi; \\",
		"\techo \"$(GREEN)✓ Go code linted$(RESET)\"; \\",
		"fi",
	}
}

func qualityFmtGoTemplate() []string {
	return []string{
		"@if [ -d api ] && find api -name \"*.go\" | head -1 | grep -q .; then \\",
		"\techo \"Formatting Go code...\"; \\",
		"\tif command -v gofumpt >/dev/null 2>&1; then \\",
		"\t\tcd api && gofumpt -w .; \\",
		"\telif command -v gofmt >/dev/null 2>&1; then \\",
		"\t\tcd api && gofmt -w .; \\",
		"\tfi; \\",
		"\techo \"$(GREEN)✓ Go code formatted$(RESET)\"; \\",
		"fi",
	}
}
