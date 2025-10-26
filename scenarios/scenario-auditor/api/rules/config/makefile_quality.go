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

<test-case id="quality-canonical" should-fail="false">
  <description>Canonical quality workflow using prerequisites and Go API tooling</description>
  <input language="make"><![CDATA[
fmt: fmt-go fmt-ui ## Format all code

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

lint: lint-go lint-ui ## Lint all code

lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui:
	@echo "Linting UI code..."

check: fmt lint test ## Run full quality gates
  ]]></input>
</test-case>

<test-case id="quality-make-invocation-flags" should-fail="false">
  <description>Aggregator targets invoke canonical subcommands via $(MAKE) with flags</description>
  <input language="make"><![CDATA[
fmt: ## Format all code
	@$(MAKE) --no-print-directory fmt-go
	@$(MAKE) --no-print-directory fmt-ui

fmt-go:
	@if [ -d api ] && [ -f api/go.mod ]; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		else \
			cd api && gofmt -w .; \
		fi; \
	fi

fmt-ui:
	@echo "Formatting UI assets..."

lint: ## Lint all code
	@$(MAKE) --no-print-directory lint-go
	@$(MAKE) --no-print-directory lint-ui

lint-go:
	@if [ -d api ] && [ -f api/go.mod ]; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
	fi

lint-ui:
	@echo "Linting UI code..."

test:
	@echo "Running tests..."

check: ## Run full quality gates
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
  ]]></input>
</test-case>

<test-case id="quality-missing-fmt-go" should-fail="true">
  <description>fmt target omits fmt-go dependency</description>
  <input language="make"><![CDATA[
fmt: fmt-ui ## Format all code

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

lint: lint-go lint-ui

lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui:
	@echo "Linting UI code..."

check: fmt lint test
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>fmt target must depend on or invoke 'fmt-go'</expected-message>
</test-case>

<test-case id="quality-lint-cli" should-fail="true">
  <description>lint-go operates on cli directory instead of api</description>
  <input language="make"><![CDATA[
fmt: fmt-go fmt-ui

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

lint: lint-go lint-ui

lint-go:
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd cli && golangci-lint run ./...; \
		else \
			cd cli && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui:
	@echo "Linting UI code..."

check: fmt lint test
  ]]></input>
  <expected-violations>3</expected-violations>
  <expected-message>lint-go target must guard execution with an api directory check</expected-message>
</test-case>

<test-case id="quality-guard-without-fallback" should-fail="false">
  <description>Guarded gofmt/golangci checks without explicit fallbacks still pass</description>
  <input language="make"><![CDATA[
fmt: fmt-go

fmt-go:
	@if [ -d api ]; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		fi; \
	fi

lint: lint-go

lint-go:
	@if [ -d api ]; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		fi; \
	fi

check: fmt lint test

test:
	@echo "Running tests"
  ]]></input>
</test-case>

<test-case id="quality-missing-guards" should-fail="true">
  <description>fmt-go without directory guard or fallback should fail</description>
  <input language="make"><![CDATA[
fmt: fmt-go

fmt-go:
	cd api && gofumpt -w .

lint: lint-go

lint-go:
	cd api && golangci-lint run ./...

check: fmt lint test
  ]]></input>
  <expected-violations>6</expected-violations>
  <expected-message>fmt-go target must guard execution with an api directory check</expected-message>
</test-case>

<test-case id="quality-missing-fallbacks" should-fail="true">
  <description>fmt-go and lint-go guard api but offer no fallback or graceful handling</description>
  <input language="make"><![CDATA[
fmt: fmt-go

fmt-go:
	@if [ -d api ]; then \
		cd api && gofumpt -w .; \
	fi

lint: lint-go

lint-go:
	@if [ -d api ]; then \
		cd api && golangci-lint run ./...; \
	fi

check: fmt lint test
  ]]></input>
  <expected-violations>2</expected-violations>
  <expected-message>fmt-go target must handle missing gofumpt gracefully</expected-message>
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
	targets map[string]qualityTarget
}

type qualityTarget struct {
	prerequisites []string
	recipe        []string
	line          int
}

var qualityTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)

type qualityRequirement struct {
	patterns []*regexp.Regexp
	message  string
}

func newQualityRequirement(message string, patterns ...string) qualityRequirement {
	reqs := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		reqs = append(reqs, regexp.MustCompile(pattern))
	}
	return qualityRequirement{patterns: reqs, message: message}
}

// CheckMakefileQuality ensures quality-related targets meet expectations.
func CheckMakefileQuality(content string, filepath string) ([]MakefileQualityViolation, error) {
	data := parseQualityMakefile(content)
	var violations []MakefileQualityViolation

	violations = append(violations, qualityEnsureContains(data, filepath, "check", []string{"fmt", "lint", "test"})...)
	violations = append(violations, qualityEnsureContains(data, filepath, "lint", []string{"lint-go"})...)
	violations = append(violations, qualityEnsureContains(data, filepath, "fmt", []string{"fmt-go"})...)
	violations = append(violations, qualityEnsureMatches(data, filepath, "lint-go", qualityLintGoRequirements())...)
	violations = append(violations, qualityEnsureMatches(data, filepath, "fmt-go", qualityFmtGoRequirements())...)

	return violations, nil
}

func qualityEnsureContains(data qualityMakefileData, path, target string, required []string) []MakefileQualityViolation {
	info, ok := data.targets[target]
	if !ok {
		return []MakefileQualityViolation{{
			Severity: "medium",
			Message:  fmt.Sprintf("%s target missing", target),
			FilePath: path,
			Line:     qualityFindLine(data.lines, target+":"),
		}}
	}

	recipe := qualityNormalize(info.recipe)
	line := info.line
	if line == 0 {
		line = qualityFindLine(data.lines, target+":")
	}

	var violations []MakefileQualityViolation
	for _, req := range required {
		if !qualityHasPrerequisite(info.prerequisites, req) && !qualityInvokesMakeTarget(recipe, req) {
			violations = append(violations, MakefileQualityViolation{
				Severity: "medium",
				Message:  fmt.Sprintf("%s target must depend on or invoke '%s'", target, req),
				FilePath: path,
				Line:     line,
			})
		}
	}
	return violations
}

func qualityEnsureMatches(data qualityMakefileData, path, target string, requirements []qualityRequirement) []MakefileQualityViolation {
	info, ok := data.targets[target]
	if !ok {
		return []MakefileQualityViolation{{
			Severity: "medium",
			Message:  fmt.Sprintf("%s target missing", target),
			FilePath: path,
			Line:     qualityFindLine(data.lines, target+":"),
		}}
	}

	recipe := qualityNormalize(info.recipe)
	if len(recipe) == 0 {
		return []MakefileQualityViolation{{
			Severity: "medium",
			Message:  fmt.Sprintf("%s target missing", target),
			FilePath: path,
			Line:     info.line,
		}}
	}

	line := info.line
	if line == 0 {
		line = qualityFindLine(data.lines, target+":")
	}

	var violations []MakefileQualityViolation
	for _, requirement := range requirements {
		if !requirement.satisfied(recipe) {
			violations = append(violations, MakefileQualityViolation{
				Severity: "medium",
				Message:  requirement.message,
				FilePath: path,
				Line:     line,
			})
		}
	}

	return violations
}

func parseQualityMakefile(content string) qualityMakefileData {
	lines := strings.Split(content, "\n")
	data := qualityMakefileData{
		lines:   lines,
		targets: make(map[string]qualityTarget),
	}

	var currentTarget string
	for idx, raw := range lines {
		trimmed := strings.TrimLeft(raw, "\t ")

		if strings.HasPrefix(raw, "\t") && currentTarget != "" {
			target := data.targets[currentTarget]
			target.recipe = append(target.recipe, raw)
			data.targets[currentTarget] = target
			continue
		}

		matches := qualityTargetRegexp.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			currentTarget = matches[1]
			target := data.targets[currentTarget]
			target.line = idx + 1

			remainder := strings.TrimSpace(matches[2])
			if remainder != "" {
				prereqs, trailing := qualitySplitPrerequisites(remainder)
				target.prerequisites = append(target.prerequisites, prereqs...)
				if trailing != "" {
					target.recipe = append(target.recipe, "\t"+trailing)
				}
			}

			data.targets[currentTarget] = target
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

func qualityInvokesMakeTarget(recipe []string, target string) bool {
	if len(recipe) == 0 {
		return false
	}
	targetPattern := regexp.MustCompile(fmt.Sprintf(`(?i)(?:^|\s)(?:@)?\$\(\s*MAKE\s*\)(?:[^\n\r;]*)?\b%s\b`, regexp.QuoteMeta(target)))
	legacyPattern := regexp.MustCompile(fmt.Sprintf(`(?i)(?:^|\s)(?:@)?make(?:[^\n\r;]*)?\b%s\b`, regexp.QuoteMeta(target)))
	for _, line := range recipe {
		trimmed := strings.TrimSpace(line)
		if targetPattern.MatchString(trimmed) || legacyPattern.MatchString(trimmed) {
			return true
		}
	}
	return false
}

func (req qualityRequirement) satisfied(recipe []string) bool {
	if len(req.patterns) == 0 {
		return true
	}
	joined := strings.Join(recipe, "\n")
	for _, pattern := range req.patterns {
		if pattern.MatchString(joined) {
			return true
		}
	}
	return false
}

func qualityHasPrerequisite(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func qualitySplitPrerequisites(remainder string) ([]string, string) {
	commandPart := remainder
	if semicolon := strings.Index(remainder, ";"); semicolon != -1 {
		commandPart = strings.TrimSpace(remainder[:semicolon])
		remainder = strings.TrimSpace(remainder[semicolon+1:])
	} else {
		remainder = ""
	}

	if commentIdx := strings.Index(commandPart, "##"); commentIdx != -1 {
		commandPart = strings.TrimSpace(commandPart[:commentIdx])
	}

	var prereqs []string
	for _, field := range strings.Fields(commandPart) {
		prereqs = append(prereqs, field)
	}

	return prereqs, remainder
}

func qualityFindLine(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 1
}

func qualityLintGoRequirements() []qualityRequirement {
	return []qualityRequirement{
		newQualityRequirement("lint-go target must guard execution with an api directory check", `-d\s+api`),
		newQualityRequirement("lint-go target must inspect Go sources before linting", `find\s+api\s+-name`, `api/go\.mod`, `\[[^\]]*-d\s+api[^\]]*\]`, `test\s+-d\s+api`, `ls\s+api`, `stat\s+api`),
		newQualityRequirement("lint-go target must lint from within the api directory", `cd\s+api[^\n]*golangci-lint`, `\(cd\s+api`),
		newQualityRequirement("lint-go target must invoke golangci-lint", `golangci-lint\s+run`),
		newQualityRequirement("lint-go target must handle missing golangci-lint gracefully", `go\s+vet\s+\.\/\.\.`, `go\s+test[^\n]*-vet`, `command\s+-v\s+golangci-lint`, `which\s+golangci-lint`, `hash\s+golangci-lint`, `type\s+golangci-lint`, `exit\s+1`, `false`, `return\s+1`),
	}
}

func qualityFmtGoRequirements() []qualityRequirement {
	return []qualityRequirement{
		newQualityRequirement("fmt-go target must guard execution with an api directory check", `-d\s+api`),
		newQualityRequirement("fmt-go target must inspect Go sources before formatting", `find\s+api\s+-name`, `api/go\.mod`, `\[[^\]]*-d\s+api[^\]]*\]`, `test\s+-d\s+api`, `ls\s+api`, `stat\s+api`),
		newQualityRequirement("fmt-go target must run formatting from within the api directory", `cd\s+api[^\n]*gofumpt`, `cd\s+api[^\n]*go\s+fmt`, `cd\s+api[^\n]*gofmt`, `\(cd\s+api`),
		newQualityRequirement("fmt-go target must invoke gofumpt when available", `gofumpt`),
		newQualityRequirement("fmt-go target must handle missing gofumpt gracefully", `gofmt`, `go\s+fmt`, `command\s+-v\s+gofumpt`, `which\s+gofumpt`, `hash\s+gofumpt`, `type\s+gofumpt`, `exit\s+1`, `false`, `return\s+1`),
	}
}
