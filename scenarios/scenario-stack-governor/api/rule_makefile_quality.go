package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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

// RunMakefileQuality runs the quality check rule against scenario Makefiles.
func RunMakefileQuality(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    "MAKEFILE_QUALITY",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = !hasActionableFindings(result.Findings)
	}()

	paths, err := locateMakefiles(repoRoot, scenarioName)
	if err != nil {
		result.Findings = append(result.Findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("failed to locate Makefiles: %v", err),
		})
		return result
	}
	if len(paths) == 0 {
		result.Findings = append(result.Findings, Finding{
			Level:   "warn",
			Message: "no Makefiles found",
		})
		return result
	}

	for _, path := range paths {
		scenSlug := filepath.Base(filepath.Dir(path))
		content, err := os.ReadFile(path)
		if err != nil {
			result.Findings = append(result.Findings, Finding{
				Level:        "error",
				Message:      fmt.Sprintf("failed to read %s: %v", path, err),
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: path},
				},
			})
			continue
		}

		violations, _ := CheckMakefileQuality(string(content), path)
		for _, v := range violations {
			result.Findings = append(result.Findings, Finding{
				Level:        "warn",
				Message:      v.Message,
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: v.FilePath},
					{Type: "note", Detail: fmt.Sprintf("line %d: %s", v.Line, v.Message)},
				},
			})
		}
	}

	return result
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
	// Filter out comment-only recipe lines before matching, so patterns
	// only match actual executable code, not comments that happen to
	// contain the right keywords.
	var executableLines []string
	for _, line := range recipe {
		stripped := strings.TrimSpace(line)
		// Strip Make recipe prefixes (@, -, +) to find the actual command.
		for len(stripped) > 0 && (stripped[0] == '@' || stripped[0] == '-' || stripped[0] == '+') {
			stripped = strings.TrimSpace(stripped[1:])
		}
		if strings.HasPrefix(stripped, "#") {
			continue
		}
		executableLines = append(executableLines, line)
	}
	joined := strings.Join(executableLines, "\n")
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

	prereqs := strings.Fields(commandPart)

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

// goDir matches either "api" or "cli" as the Go source directory.
// Scenarios may have only an api/, only a cli/, or both — the quality rule
// accepts either directory as a valid guard target.
const goDirPattern = `(?:api|cli)`

func qualityLintGoRequirements() []qualityRequirement {
	return []qualityRequirement{
		newQualityRequirement("lint-go target must guard execution with a Go directory check", `-d\s+`+goDirPattern),
		newQualityRequirement("lint-go target must inspect Go sources before linting", `find\s+`+goDirPattern+`\s+-name`, goDirPattern+`/go\.mod`, `\[[^\]]*-d\s+`+goDirPattern+`[^\]]*\]`, `test\s+-d\s+`+goDirPattern, `ls\s+`+goDirPattern, `stat\s+`+goDirPattern),
		newQualityRequirement("lint-go target must lint from within the Go directory", `cd\s+`+goDirPattern+`[^\n]*golangci-lint`, `\(cd\s+`+goDirPattern),
		newQualityRequirement("lint-go target must invoke golangci-lint", `golangci-lint\s+run`),
		newQualityRequirement("lint-go target must handle missing golangci-lint gracefully", `go\s+vet\s+\.\/\.\.`, `go\s+test[^\n]*-vet`, `command\s+-v\s+golangci-lint`, `which\s+golangci-lint`, `hash\s+golangci-lint`, `type\s+golangci-lint`, `exit\s+1`, `false`, `return\s+1`),
	}
}

func qualityFmtGoRequirements() []qualityRequirement {
	return []qualityRequirement{
		newQualityRequirement("fmt-go target must guard execution with a Go directory check", `-d\s+`+goDirPattern),
		newQualityRequirement("fmt-go target must inspect Go sources before formatting", `find\s+`+goDirPattern+`\s+-name`, goDirPattern+`/go\.mod`, `\[[^\]]*-d\s+`+goDirPattern+`[^\]]*\]`, `test\s+-d\s+`+goDirPattern, `ls\s+`+goDirPattern, `stat\s+`+goDirPattern),
		newQualityRequirement("fmt-go target must run formatting from within the Go directory", `cd\s+`+goDirPattern+`[^\n]*gofumpt`, `cd\s+`+goDirPattern+`[^\n]*go\s+fmt`, `cd\s+`+goDirPattern+`[^\n]*gofmt`, `\(cd\s+`+goDirPattern),
		newQualityRequirement("fmt-go target must invoke gofumpt when available", `gofumpt`),
		newQualityRequirement("fmt-go target must handle missing gofumpt gracefully", `gofmt`, `go\s+fmt`, `command\s+-v\s+gofumpt`, `which\s+gofumpt`, `hash\s+gofumpt`, `type\s+gofumpt`, `exit\s+1`, `false`, `return\s+1`),
	}
}
