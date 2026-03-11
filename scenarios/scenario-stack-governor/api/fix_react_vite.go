package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FixReactViteUIInstallsDependencies patches service.json to ensure UI deps
// are installed with `pnpm install --ignore-workspace`. It handles both pnpm
// and npm install steps, replacing them in-place via string-level patching to
// preserve the original JSON key ordering.
func FixReactViteUIInstallsDependencies(ctx context.Context, repoRoot, scenarioName string, dryRun bool) []FixResult {
	ruleID := "REACT_VITE_UI_INSTALLS_DEPENDENCIES"

	scenarioDir := filepath.Join(repoRoot, "scenarios", scenarioName)
	serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")

	if !fileExists(serviceJSONPath) {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
			Error:        "service.json not found; cannot auto-fix",
		}}
	}

	raw, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
			Error:        err.Error(),
		}}
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
			Error:        fmt.Sprintf("invalid JSON: %v", err),
		}}
	}

	var changes []FixChange

	lifecycle, _ := doc["lifecycle"].(map[string]any)
	setup, _ := lifecycle["setup"].(map[string]any)
	stepsAny, _ := setup["steps"].([]any)

	const desiredRun = "cd ui && pnpm install --ignore-workspace"

	// Phase 1: Look for existing pnpm install step with "ui".
	pnpmFound := false
	for _, stepAny := range stepsAny {
		step, ok := stepAny.(map[string]any)
		if !ok {
			continue
		}
		run, _ := step["run"].(string)
		if run == "" {
			continue
		}
		if strings.Contains(run, "pnpm install") && isUIRelatedStep(step) {
			pnpmFound = true
			if strings.Contains(run, "--ignore-workspace") {
				// Already correct — nothing to do.
				break
			}
			// Patch: replace the run value in the raw JSON string.
			patched := strings.Replace(run, "pnpm install", "pnpm install --ignore-workspace", 1)
			changes = append(changes, FixChange{
				Type:   "patched_step",
				Detail: fmt.Sprintf("Added --ignore-workspace to existing step: %s", patched),
			})
			raw = replaceRunInRaw(raw, run, patched)
			break
		}
	}

	// Phase 2: If no pnpm step found, look for npm install step with "ui".
	if !pnpmFound {
		npmFound := false
		for _, stepAny := range stepsAny {
			step, ok := stepAny.(map[string]any)
			if !ok {
				continue
			}
			run, _ := step["run"].(string)
			if run == "" {
				continue
			}
			if strings.Contains(run, "npm install") && isUIRelatedStep(step) {
				npmFound = true
				// Replace the entire run value with the desired pnpm command.
				changes = append(changes, FixChange{
					Type:   "replaced_step",
					Detail: fmt.Sprintf("Replaced npm install with pnpm: %s -> %s", run, desiredRun),
				})
				raw = replaceRunInRaw(raw, run, desiredRun)
				break
			}
		}

		// Phase 3: Neither pnpm nor npm found — add new step.
		if !npmFound {
			newStepJSON := fmt.Sprintf("{\n      \"name\": \"install-ui-deps\",\n      \"run\": %q\n    }", desiredRun)
			raw = insertStepInRaw(raw, newStepJSON, lifecycle, setup, stepsAny)
			changes = append(changes, FixChange{
				Type:   "added_step",
				Detail: "Added setup step: " + desiredRun,
			})
		}
	}

	if len(changes) == 0 {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
		}}
	}

	var diff *FileDiff
	if dryRun {
		diff = &FileDiff{Before: string(raw[:0:0]), After: string(raw)}
		// Re-read original for Before.
		origBytes, _ := os.ReadFile(serviceJSONPath)
		diff.Before = string(origBytes)
		diff.After = string(raw)
	} else {
		if err := os.WriteFile(serviceJSONPath, raw, 0o644); err != nil {
			return []FixResult{{
				ScenarioName: scenarioName,
				RuleID:       ruleID,
				Fixed:        false,
				FilePath:     serviceJSONPath,
				Error:        err.Error(),
			}}
		}
	}

	return []FixResult{{
		ScenarioName: scenarioName,
		RuleID:       ruleID,
		Fixed:        true,
		FilePath:     serviceJSONPath,
		Changes:      changes,
		Diff:         diff,
	}}
}

// isUIRelatedStep returns true if the step's run command or name references "ui".
func isUIRelatedStep(step map[string]any) bool {
	run, _ := step["run"].(string)
	name, _ := step["name"].(string)
	return strings.Contains(run, "ui") || strings.Contains(name, "ui")
}

// replaceRunInRaw performs a targeted string replacement of a run value within
// the raw JSON bytes, preserving all other formatting and key ordering.
func replaceRunInRaw(raw []byte, oldRun, newRun string) []byte {
	// Use json.Marshal to produce the exact encoding the JSON encoder would use
	// (e.g., & is escaped as \u0026 by the standard encoder). We try both the
	// escaped and unescaped forms to handle hand-written vs machine-generated JSON.
	oldEscaped, _ := json.Marshal(oldRun)
	newEscaped, _ := json.Marshal(newRun)
	// Also try the literal (unescaped) form for hand-written JSON.
	oldLiteral := fmt.Sprintf("%q", oldRun)
	newLiteral := fmt.Sprintf("%q", newRun)

	text := string(raw)
	// Try escaped form first (machine-generated JSON).
	if strings.Contains(text, string(oldEscaped)) {
		return []byte(strings.Replace(text, string(oldEscaped), string(newEscaped), 1))
	}
	// Try literal form (hand-written JSON).
	return []byte(strings.Replace(text, oldLiteral, newLiteral, 1))
}

// insertStepInRaw inserts a new step JSON into the steps array within the raw
// JSON bytes. If no steps array exists, it falls back to a full marshal.
func insertStepInRaw(raw []byte, newStepJSON string, lifecycle, setup map[string]any, stepsAny []any) []byte {
	text := string(raw)

	if stepsAny != nil && len(stepsAny) > 0 {
		// Find the closing bracket of the steps array.
		// We look for the pattern where the steps array ends.
		stepsClose := findStepsArrayClose(text)
		if stepsClose >= 0 {
			// Insert before the closing bracket, adding a comma after the last element.
			before := text[:stepsClose]
			after := text[stepsClose:]
			// Trim trailing whitespace before the ] to find insertion point.
			trimmed := strings.TrimRight(before, " \t\n\r")
			whitespace := before[len(trimmed):]
			result := trimmed + ",\n    " + newStepJSON + whitespace + after
			return []byte(result)
		}
	} else if stepsAny != nil {
		// Empty steps array — find "steps": [] and insert inside.
		stepsClose := findStepsArrayClose(text)
		if stepsClose >= 0 {
			before := text[:stepsClose]
			after := text[stepsClose:]
			result := before + "\n    " + newStepJSON + "\n  " + after
			return []byte(result)
		}
	}

	// Fallback: marshal the whole thing (loses key order but at least works).
	if lifecycle == nil {
		lifecycle = map[string]any{}
	}
	if setup == nil {
		setup = map[string]any{}
	}
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	newStep := map[string]any{
		"name": "install-ui-deps",
		"run":  "cd ui && pnpm install --ignore-workspace",
	}
	stepsAny = append(stepsAny, newStep)
	if setup["steps"] == nil {
		setup["steps"] = []any{}
	}
	setup["steps"] = stepsAny
	if lifecycle["setup"] == nil {
		lifecycle["setup"] = setup
	}
	doc["lifecycle"] = lifecycle
	afterBytes, _ := json.MarshalIndent(doc, "", "  ")
	return append(afterBytes, '\n')
}

// findStepsArrayClose returns the index of the ']' that closes the "steps" array.
func findStepsArrayClose(text string) int {
	// Find "steps" key.
	idx := strings.Index(text, `"steps"`)
	if idx < 0 {
		return -1
	}
	// Find the opening '[' after "steps".
	rest := text[idx:]
	bracketStart := strings.Index(rest, "[")
	if bracketStart < 0 {
		return -1
	}
	absStart := idx + bracketStart
	// Walk forward counting brackets to find the matching ']'.
	depth := 0
	for i := absStart; i < len(text); i++ {
		switch text[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
