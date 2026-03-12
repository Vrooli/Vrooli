package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed templates/canonical.mk
var canonicalMakefileTemplate string

// customTarget represents a non-canonical target extracted from an existing Makefile.
type customTarget struct {
	name          string
	definition    string // full target line (e.g. "export-variants: ## Export variants")
	recipe        []string
	prerequisites []string
}

// customVariable represents a non-canonical variable definition extracted from an existing Makefile.
type customVariable struct {
	name       string // e.g. "API_URL"
	definition string // full line: "API_URL ?= http://localhost:3000"
}

// FixMakefileAll orchestrates the Makefile fix for a single scenario and returns
// one FixResult per MAKEFILE_* rule ID.
func FixMakefileAll(ctx context.Context, repoRoot, scenarioName string, dryRun bool) []FixResult {
	ruleIDs := []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"}

	scenarioDir := filepath.Join(repoRoot, "scenarios", scenarioName)
	makefilePath := filepath.Join(scenarioDir, "Makefile")

	// Read existing content (may not exist).
	existingContent, _ := os.ReadFile(makefilePath)

	// Extract custom targets and variables from existing Makefile.
	var customs []customTarget
	var customVars []customVariable
	if len(existingContent) > 0 {
		customs = extractCustomTargets(string(existingContent))
		customVars = extractCustomVariables(string(existingContent))
	}

	// Generate canonical Makefile.
	canonical := generateCanonicalMakefile(scenarioName)

	// Merge custom variables into canonical output.
	output := canonical
	if len(customVars) > 0 {
		output = mergeCustomVariables(output, customVars)
	}

	// Merge custom targets into canonical output.
	if len(customs) > 0 {
		output = mergeCustomTargets(output, customs)
	}

	// Check which rules have violations against the existing content.
	var structureViolations []MakefileStructureViolation
	var lifecycleViolations []MakefileLifecycleViolation
	var qualityViolations []MakefileQualityViolation
	if len(existingContent) > 0 {
		structureViolations, _ = CheckMakefileStructure(string(existingContent), makefilePath)
		lifecycleViolations, _ = CheckMakefileLifecycle(string(existingContent), makefilePath)
		qualityViolations, _ = CheckMakefileQuality(string(existingContent), makefilePath)
	}

	// Build per-rule results.
	isNew := len(existingContent) == 0
	contentChanged := isNew || string(existingContent) != output

	// Determine per-rule fix status.
	ruleNeedsFix := map[string]bool{
		"MAKEFILE_STRUCTURE": isNew || len(structureViolations) > 0,
		"MAKEFILE_LIFECYCLE": isNew || len(lifecycleViolations) > 0,
		"MAKEFILE_QUALITY":   isNew || len(qualityViolations) > 0,
	}

	// Build per-rule changes.
	type ruleChanges struct {
		fixed   bool
		changes []FixChange
	}
	perRule := make(map[string]ruleChanges)
	for _, id := range ruleIDs {
		needsFix := ruleNeedsFix[id] && contentChanged
		var ruleSpecificChanges []FixChange
		if needsFix {
			if isNew {
				ruleSpecificChanges = append(ruleSpecificChanges, FixChange{Type: "generated", Detail: "Created canonical Makefile from template"})
			} else {
				ruleSpecificChanges = append(ruleSpecificChanges, FixChange{Type: "replaced", Detail: "Replaced Makefile with canonical version"})
			}
			for _, v := range customVars {
				ruleSpecificChanges = append(ruleSpecificChanges, FixChange{Type: "preserved_variable", Detail: fmt.Sprintf("Preserved custom variable '%s'", v.name)})
			}
			for _, c := range customs {
				ruleSpecificChanges = append(ruleSpecificChanges, FixChange{Type: "preserved_custom", Detail: fmt.Sprintf("Preserved custom target '%s'", c.name)})
			}
		}
		perRule[id] = ruleChanges{fixed: needsFix, changes: ruleSpecificChanges}
	}

	// Write if any rule needs fixing and not dry-run.
	anyFixed := false
	for _, rc := range perRule {
		if rc.fixed {
			anyFixed = true
			break
		}
	}

	// Validate the generated Makefile passes the rules it's supposed to fix.
	// This catches merge/template bugs before writing a broken file.
	if anyFixed {
		if vErr := validateGeneratedMakefile(output, makefilePath); vErr != "" {
			return errorResults(ruleIDs, scenarioName, makefilePath, fmt.Errorf("generated Makefile failed self-validation: %s", vErr))
		}
	}

	if anyFixed && !dryRun {
		if err := os.MkdirAll(filepath.Dir(makefilePath), 0o755); err != nil {
			return errorResults(ruleIDs, scenarioName, makefilePath, err)
		}
		if err := os.WriteFile(makefilePath, []byte(output), 0o644); err != nil {
			return errorResults(ruleIDs, scenarioName, makefilePath, err)
		}
	}

	// Return one result per rule.
	results := make([]FixResult, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		rc := perRule[id]
		var ruleDiff *FileDiff
		if dryRun && rc.fixed {
			ruleDiff = &FileDiff{Before: string(existingContent), After: output}
		}
		results = append(results, FixResult{
			ScenarioName: scenarioName,
			RuleID:       id,
			Fixed:        rc.fixed,
			FilePath:     makefilePath,
			Changes:      rc.changes,
			Diff:         ruleDiff,
		})
	}
	return results
}

// generateCanonicalMakefile produces the canonical Makefile content for a scenario.
func generateCanonicalMakefile(scenarioName string) string {
	// Convert scenario slug to title case for the header.
	title := scenarioSlugToTitle(scenarioName)
	output := strings.Replace(canonicalMakefileTemplate, "{{TITLE}}", title, 1)
	output = strings.ReplaceAll(output, "{{SCENARIO_ID}}", scenarioName)
	return output
}

// scenarioSlugToTitle converts "my-cool-scenario" to "My Cool Scenario".
func scenarioSlugToTitle(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

var targetLineRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)

// joinContinuationLines merges lines ending with backslash (\) into single logical lines.
// When a non-recipe line ends with \, subsequent lines are treated as continuations
// regardless of their indentation, until a line without trailing \ is found.
// Recipe lines (starting with tab) that are NOT part of a continuation are passed through as-is.
func joinContinuationLines(lines []string) []string {
	var result []string
	var builder strings.Builder
	inContinuation := false

	for _, raw := range lines {
		// If we're in a continuation, this line is part of it regardless of prefix.
		if inContinuation {
			builder.WriteByte(' ')
			trimmed := strings.TrimRight(strings.TrimSpace(raw), " \t")
			if strings.HasSuffix(trimmed, "\\") {
				builder.WriteString(strings.TrimSuffix(trimmed, "\\"))
				// Still in continuation.
			} else {
				builder.WriteString(trimmed)
				result = append(result, builder.String())
				builder.Reset()
				inContinuation = false
			}
			continue
		}

		// Not in continuation. Check if this line starts one.
		trimmedRight := strings.TrimRight(raw, " \t")
		if strings.HasSuffix(trimmedRight, "\\") {
			// Recipe lines starting with tab that end with \ are part of recipe
			// continuation — pass them through individually (recipes handle their
			// own continuations via the shell).
			if strings.HasPrefix(raw, "\t") {
				result = append(result, raw)
				continue
			}
			// Non-recipe line starts a continuation.
			inContinuation = true
			builder.WriteString(strings.TrimSuffix(trimmedRight, "\\"))
			continue
		}

		result = append(result, raw)
	}

	if builder.Len() > 0 {
		result = append(result, builder.String())
	}

	return result
}

// extractCustomTargets parses an existing Makefile and returns targets not in the canonical set.
func extractCustomTargets(content string) []customTarget {
	canonical := canonicalTargetSet()
	// Also exclude common shortcut targets that appear in the shortcuts section.
	shortcuts := map[string]struct{}{"dev": {}, "restart": {}, "rebuild": {}}

	// Join continuation lines so multi-line target definitions and prerequisites
	// are properly parsed as single logical lines.
	lines := joinContinuationLines(strings.Split(content, "\n"))
	var customs []customTarget
	var currentTarget string
	var currentDef string
	var currentRecipe []string
	var currentPrereqs []string

	flush := func() {
		if currentTarget == "" {
			return
		}
		if _, isCanonical := canonical[currentTarget]; isCanonical {
			currentTarget = ""
			currentDef = ""
			currentRecipe = nil
			currentPrereqs = nil
			return
		}
		if _, isShortcut := shortcuts[currentTarget]; isShortcut {
			currentTarget = ""
			currentDef = ""
			currentRecipe = nil
			currentPrereqs = nil
			return
		}
		customs = append(customs, customTarget{
			name:          currentTarget,
			definition:    currentDef,
			recipe:        currentRecipe,
			prerequisites: currentPrereqs,
		})
		currentTarget = ""
		currentDef = ""
		currentRecipe = nil
		currentPrereqs = nil
	}

	for _, raw := range lines {
		if strings.HasPrefix(raw, "\t") && currentTarget != "" {
			currentRecipe = append(currentRecipe, raw)
			continue
		}

		trimmed := strings.TrimLeft(raw, " ")

		// Skip Make directives and variable assignments.
		if strings.HasPrefix(trimmed, ".PHONY") ||
			strings.HasPrefix(trimmed, ".DEFAULT_GOAL") ||
			strings.HasPrefix(trimmed, "SCENARIO_NAME") ||
			strings.Contains(trimmed, ":=") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.TrimSpace(trimmed) == "" {
			flush()
			continue
		}

		matches := targetLineRegexp.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			flush()
			name := matches[1]
			currentTarget = name
			currentDef = raw
			remainder := strings.TrimSpace(matches[2])
			if remainder != "" {
				// Split off ## comment, extract prerequisites.
				if commentIdx := strings.Index(remainder, "##"); commentIdx != -1 {
					remainder = strings.TrimSpace(remainder[:commentIdx])
				}
				if remainder != "" {
					currentPrereqs = strings.Fields(remainder)
				}
			}
			continue
		}

		flush()
	}
	flush()

	return customs
}

// canonicalVariableSet returns variable names that are part of the canonical template.
var canonicalVariableSet = map[string]struct{}{
	"SCENARIO_NAME": {},
	"GREEN":         {},
	"YELLOW":        {},
	"BLUE":          {},
	"RED":           {},
	"RESET":         {},
}

// varAssignRegexp matches Make variable assignments: VAR_NAME := value, VAR_NAME ?= value, VAR_NAME = value
var varAssignRegexp = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*(?:\?=|:=|=)\s*(.*)$`)

// extractCustomVariables parses an existing Makefile and returns variable definitions
// that are NOT in the canonical set. Handles multi-line assignments (backslash continuations).
func extractCustomVariables(content string) []customVariable {
	// Join continuation lines so multi-line variable assignments are captured as one definition.
	lines := joinContinuationLines(strings.Split(content, "\n"))
	var vars []customVariable

	for _, raw := range lines {
		// Skip recipe lines (start with tab).
		if strings.HasPrefix(raw, "\t") {
			continue
		}

		trimmed := strings.TrimSpace(raw)

		// Skip empty lines, comments, .PHONY, .DEFAULT_GOAL.
		if trimmed == "" ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, ".PHONY") ||
			strings.HasPrefix(trimmed, ".DEFAULT_GOAL") {
			continue
		}

		matches := varAssignRegexp.FindStringSubmatch(trimmed)
		if len(matches) < 2 {
			continue
		}
		name := matches[1]

		// Skip canonical variables.
		if _, isCanonical := canonicalVariableSet[name]; isCanonical {
			continue
		}

		vars = append(vars, customVariable{
			name:       name,
			definition: raw,
		})
	}

	return vars
}

// mergeCustomVariables inserts custom variable definitions into the canonical Makefile.
// They are inserted after SCENARIO_NAME and before the color palette (before GREEN := ...).
func mergeCustomVariables(canonical string, vars []customVariable) string {
	lines := strings.Split(canonical, "\n")

	// Find the GREEN line (first color variable) to insert before it.
	insertIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "GREEN :=") {
			insertIdx = i
			break
		}
	}

	if insertIdx < 0 {
		// Fallback: insert after SCENARIO_NAME line.
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "SCENARIO_NAME :=") {
				insertIdx = i + 1
				break
			}
		}
	}

	if insertIdx < 0 {
		insertIdx = len(lines)
	}

	// Build the custom variable lines.
	var customLines []string
	for _, v := range vars {
		customLines = append(customLines, v.definition)
	}
	customLines = append(customLines, "") // blank separator after custom vars

	result := make([]string, 0, len(lines)+len(customLines))
	result = append(result, lines[:insertIdx]...)
	result = append(result, customLines...)
	result = append(result, lines[insertIdx:]...)

	return strings.Join(result, "\n")
}

// mergeCustomTargets inserts custom targets into the canonical Makefile output.
// Custom targets are placed after the `check:` target but before `# Development shortcuts`.
func mergeCustomTargets(canonical string, customs []customTarget) string {
	lines := strings.Split(canonical, "\n")

	// Find the "# Development shortcuts" line.
	insertIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "# Development shortcuts" {
			insertIdx = i
			break
		}
	}

	// Build the custom section.
	var customLines []string
	for _, c := range customs {
		customLines = append(customLines, c.definition)
		customLines = append(customLines, c.recipe...)
		customLines = append(customLines, "")
	}

	// Insert before shortcuts section.
	if insertIdx >= 0 {
		result := make([]string, 0, len(lines)+len(customLines))
		result = append(result, lines[:insertIdx]...)
		result = append(result, customLines...)
		result = append(result, lines[insertIdx:]...)
		lines = result
	} else {
		// No shortcuts section found; append at end.
		lines = append(lines, customLines...)
	}

	// Update .PHONY to include custom target names.
	output := strings.Join(lines, "\n")
	phonyIdx := strings.Index(output, ".PHONY:")
	if phonyIdx != -1 {
		eol := strings.Index(output[phonyIdx:], "\n")
		var phonyLine string
		if eol == -1 {
			phonyLine = output[phonyIdx:]
		} else {
			phonyLine = output[phonyIdx : phonyIdx+eol]
		}
		phonyTargets := strings.Fields(phonyLine)

		var toAdd []string
		for _, c := range customs {
			// Check if the target name is already in the .PHONY line.
			// Use word-level matching to avoid substring false positives
			// (e.g. "foo" matching "foobar").
			alreadyPresent := false
			for _, t := range phonyTargets {
				if t == c.name {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				toAdd = append(toAdd, c.name)
			}
		}

		if len(toAdd) > 0 {
			suffix := " " + strings.Join(toAdd, " ")
			if eol == -1 {
				output += suffix
			} else {
				pos := phonyIdx + eol
				output = output[:pos] + suffix + output[pos:]
			}
		}
	}

	return output
}

// validateGeneratedMakefile runs the three Makefile check rules against the
// generated content and returns a description of the first violation found,
// or empty string if all checks pass. This guards against template or merge
// bugs producing an invalid Makefile.
func validateGeneratedMakefile(content, path string) string {
	if sv, _ := CheckMakefileStructure(content, path); len(sv) > 0 {
		return fmt.Sprintf("STRUCTURE: %s (line %d)", sv[0].Message, sv[0].Line)
	}
	if lv, _ := CheckMakefileLifecycle(content, path); len(lv) > 0 {
		return fmt.Sprintf("LIFECYCLE: %s (line %d)", lv[0].Message, lv[0].Line)
	}
	if qv, _ := CheckMakefileQuality(content, path); len(qv) > 0 {
		return fmt.Sprintf("QUALITY: %s (line %d)", qv[0].Message, qv[0].Line)
	}
	return ""
}

func errorResults(ruleIDs []string, scenarioName, filePath string, err error) []FixResult {
	results := make([]FixResult, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		results = append(results, FixResult{
			ScenarioName: scenarioName,
			RuleID:       id,
			Fixed:        false,
			FilePath:     filePath,
			Error:        err.Error(),
		})
	}
	return results
}
