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

// FixMakefileAll orchestrates the Makefile fix for a single scenario and returns
// one FixResult per MAKEFILE_* rule ID.
func FixMakefileAll(ctx context.Context, repoRoot, scenarioName string, dryRun bool) []FixResult {
	ruleIDs := []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"}

	scenarioDir := filepath.Join(repoRoot, "scenarios", scenarioName)
	makefilePath := filepath.Join(scenarioDir, "Makefile")

	// Read existing content (may not exist).
	existingContent, _ := os.ReadFile(makefilePath)

	// Extract custom targets from existing Makefile.
	var customs []customTarget
	if len(existingContent) > 0 {
		customs = extractCustomTargets(string(existingContent))
	}

	// Generate canonical Makefile.
	canonical := generateCanonicalMakefile(scenarioName)

	// Merge custom targets into canonical output.
	output := canonical
	if len(customs) > 0 {
		output = mergeCustomTargets(canonical, customs)
	}

	// Determine changes.
	var changes []FixChange
	if len(existingContent) == 0 {
		changes = append(changes, FixChange{Type: "generated", Detail: "Created canonical Makefile from template"})
	} else if string(existingContent) != output {
		changes = append(changes, FixChange{Type: "replaced", Detail: "Replaced Makefile with canonical version"})
	}
	for _, c := range customs {
		changes = append(changes, FixChange{Type: "preserved_custom", Detail: fmt.Sprintf("Preserved custom target '%s'", c.name)})
	}

	fixed := len(changes) > 0 && (len(existingContent) == 0 || string(existingContent) != output)

	var diff *FileDiff
	if dryRun && fixed {
		diff = &FileDiff{Before: string(existingContent), After: output}
	}

	// Write if not dry-run and there are changes.
	if fixed && !dryRun {
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
		results = append(results, FixResult{
			ScenarioName: scenarioName,
			RuleID:       id,
			Fixed:        fixed,
			FilePath:     makefilePath,
			Changes:      changes,
			Diff:         diff,
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

// extractCustomTargets parses an existing Makefile and returns targets not in the canonical set.
func extractCustomTargets(content string) []customTarget {
	canonical := canonicalTargetSet()
	// Also exclude common shortcut targets that appear in the shortcuts section.
	shortcuts := map[string]struct{}{"dev": {}, "restart": {}, "rebuild": {}}

	lines := strings.Split(content, "\n")
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
	for _, c := range customs {
		if !strings.Contains(output, c.name) {
			continue
		}
		// Find the .PHONY line and append custom target name.
		phonyIdx := strings.Index(output, ".PHONY:")
		if phonyIdx == -1 {
			continue
		}
		eol := strings.Index(output[phonyIdx:], "\n")
		if eol == -1 {
			output += " " + c.name
		} else {
			pos := phonyIdx + eol
			output = output[:pos] + " " + c.name + output[pos:]
		}
	}

	return output
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
