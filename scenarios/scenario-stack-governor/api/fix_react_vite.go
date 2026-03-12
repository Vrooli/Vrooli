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

	// Only fix scenarios that actually have a UI (ui/package.json).
	// Without this guard, the fix would blindly add a UI install step to
	// scenarios that have no UI directory, which would fail at runtime.
	uiPackageJSON := filepath.Join(scenarioDir, "ui", "package.json")
	if !fileExists(uiPackageJSON) {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
		}}
	}

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
	originalRaw := string(raw)

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
			raw = insertStepInRaw(raw, desiredRun, lifecycle, setup, stepsAny)
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

	// Verify the patched JSON is still valid before writing. String-level
	// patching can produce invalid JSON if the replacement doesn't match the
	// exact encoding used in the file.
	var verifyDoc map[string]any
	if err := json.Unmarshal(raw, &verifyDoc); err != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
			Error:        fmt.Sprintf("patched service.json is invalid JSON (bug in fixer): %v", err),
		}}
	}

	var diff *FileDiff
	if dryRun {
		diff = &FileDiff{Before: originalRaw, After: string(raw)}
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

// insertStepInRaw inserts a new install-ui-deps step into the steps array
// within the raw JSON bytes. desiredRun is the run command string.
// If no steps array exists, it falls back to a full marshal.
func insertStepInRaw(raw []byte, desiredRun string, lifecycle, setup map[string]any, stepsAny []any) []byte {
	text := string(raw)

	if stepsAny != nil && len(stepsAny) > 0 {
		stepsClose := findStepsArrayClose(text)
		indent := detectStepIndent(text)
		// Only attempt string-level insertion when we can detect indentation
		// (i.e. the JSON is multi-line). Compact JSON falls through to marshal.
		if stepsClose >= 0 && indent != "" {
			closingIndent := detectClosingBracketIndent(text, stepsClose)
			propIndent := indent + "  "
			stepJSON := "{\n" + propIndent + `"name": "install-ui-deps",` + "\n" + propIndent + `"run": ` + fmt.Sprintf("%q", desiredRun) + "\n" + indent + "}"

			before := text[:stepsClose]
			after := text[stepsClose:]
			trimmed := strings.TrimRight(before, " \t\n\r")
			result := trimmed + ",\n" + indent + stepJSON + "\n" + closingIndent + after
			return []byte(result)
		}
	} else if stepsAny != nil {
		stepsClose := findStepsArrayClose(text)
		indent := detectStepIndent(text)
		if stepsClose >= 0 && indent != "" {
			closingIndent := detectClosingBracketIndent(text, stepsClose)
			propIndent := indent + "  "
			stepJSON := "{\n" + propIndent + `"name": "install-ui-deps",` + "\n" + propIndent + `"run": ` + fmt.Sprintf("%q", desiredRun) + "\n" + indent + "}"

			before := text[:stepsClose]
			after := text[stepsClose:]
			result := before + "\n" + indent + stepJSON + "\n" + closingIndent + after
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
		"run":  desiredRun,
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

// detectStepIndent returns the whitespace prefix used for step objects inside the
// "steps" array. It scans for the first `{` after the `"steps"` key and returns
// everything between the preceding newline and that brace.
// Returns empty string if the text has no newlines (compact JSON).
func detectStepIndent(text string) string {
	idx := strings.Index(text, `"steps"`)
	if idx < 0 {
		return "        " // 8-space default
	}
	// Find the first '{' after "steps".
	rest := text[idx:]
	braceIdx := strings.Index(rest, "{")
	if braceIdx < 0 {
		return "        "
	}
	abs := idx + braceIdx
	// Walk backwards from the '{' to the preceding newline.
	start := abs
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	indent := text[start:abs]
	// If we hit the start of the string without finding a newline, the JSON
	// is compact (no newlines). Return empty to signal the caller should use
	// the marshal fallback.
	if start == 0 && !strings.ContainsRune(indent, '\n') && strings.ContainsAny(indent, `"{}[],:`) {
		return ""
	}
	return indent
}

// detectClosingBracketIndent returns the whitespace that should precede the
// closing ']' of the steps array. It looks at the whitespace before the ']'
// at the given position.
func detectClosingBracketIndent(text string, closingPos int) string {
	start := closingPos
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	return text[start:closingPos]
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
