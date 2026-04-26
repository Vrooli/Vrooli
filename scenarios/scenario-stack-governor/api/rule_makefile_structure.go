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
	targets              map[string][]string
	targetRecipeLines    map[string][]int
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

var (
	structureTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)
	shortcutAliasPattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]+(?:\s+[A-Za-z0-9_.-]+)*$`)
)

// RunMakefileStructure runs the structure check rule against scenario Makefiles.
func RunMakefileStructure(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    "MAKEFILE_STRUCTURE",
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

		violations, _ := CheckMakefileStructure(string(content), path)
		for _, v := range violations {
			result.Findings = append(result.Findings, Finding{
				Level:        "error",
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

// CheckMakefileStructure ensures the Makefile conforms to the required structure scaffolding.
//
// User-facing UX (colors, banners, help-text formatting) is owned by the Vrooli
// CLI, not by per-scenario Makefiles. The structure rule enforces only what's
// needed for interoperability: the lifecycle-warning header, the canonical
// .PHONY target list, the standard variable defaults, a help target that exists,
// and a stable top-of-file ordering.
func CheckMakefileStructure(content string, filepath string) ([]MakefileStructureViolation, error) {
	data := parseStructureMakefile(content)
	var violations []MakefileStructureViolation

	violations = append(violations, structureValidateHeader(data, filepath)...)
	violations = append(violations, structureValidatePhony(data, filepath)...)
	violations = append(violations, structureValidateDefaults(data, filepath)...)
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
	// STRICT: All canonical targets required for consistency and interoperability.
	// Scenarios without UI/Go code should provide no-op implementations.
	required := requiredTargets()
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

func structureValidateHelp(data structureMakefileData, path string) []MakefileStructureViolation {
	// We only require that the help target exists with at least one recipe line.
	// The recipe contents are not enforced — scenarios can render help however
	// they like (the canonical template uses the standard `## comment` + grep/awk
	// pattern, which Make conventions already document for free).
	lines := structureNormalizeRecipes(data.targets["help"])
	hasRecipe := false
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			hasRecipe = true
			break
		}
	}
	if hasRecipe {
		return nil
	}
	return []MakefileStructureViolation{{
		Severity: "high",
		Message:  "help target is missing or empty",
		FilePath: path,
		Line:     structureFindLine(data.lines, "help:"),
	}}
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

	// After the shortcuts comment, only shortcut aliases and comments/blanks
	// are expected. Non-alias targets (with recipes or complex bodies) should
	// be placed before the shortcuts section, not after it.
	for i := data.shortcutsCommentLine + 1; i < len(data.lines); i++ {
		trimmed := strings.TrimSpace(data.lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if _, ok := shortcutLines[i+1]; ok {
			continue
		}
		// Allow target definition lines (they'll be validated as shortcuts
		// if they were parsed into shortcutTargets). Only flag non-target,
		// non-blank, non-comment lines that aren't part of a shortcut.
		if structureTargetRegexp.MatchString(trimmed) {
			continue
		}
		// Recipe lines (tab-prefixed) for non-shortcut targets after the
		// shortcuts comment indicate misplaced content.
		if strings.HasPrefix(data.lines[i], "\t") {
			violations = append(violations, MakefileStructureViolation{
				Severity: "high",
				Message:  "Recipe content after '# Development shortcuts' must be shortcut aliases only — move complex targets before the shortcuts section",
				FilePath: path,
				Line:     i + 1,
			})
			break
		}
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

		if strings.Contains(trimmedLeft, ":=") && !strings.HasPrefix(raw, "\t") {
			// Skip arbitrary variable assignments — the rule no longer cares
			// about non-canonical variables in the structural validation.
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
	return 0
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

func structureIsCommentOrBlank(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(trimmed, "#")
}
