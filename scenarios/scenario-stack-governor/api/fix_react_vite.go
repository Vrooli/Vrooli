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

	// Verify the patched JSON is valid AND contains the expected step.
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
	if err := verifyUIInstallStep(verifyDoc); err != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
			Error:        fmt.Sprintf("patched service.json failed semantic validation (bug in fixer): %v", err),
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

// verifyUIInstallStep checks that the parsed JSON contains a setup step with
// "pnpm install --ignore-workspace". This catches cases where string-level
// patching produced valid JSON but mangled the step content.
func verifyUIInstallStep(doc map[string]any) error {
	lifecycle, _ := doc["lifecycle"].(map[string]any)
	if lifecycle == nil {
		return fmt.Errorf("lifecycle field missing or not an object")
	}
	setup, _ := lifecycle["setup"].(map[string]any)
	if setup == nil {
		return fmt.Errorf("lifecycle.setup field missing or not an object")
	}
	steps, _ := setup["steps"].([]any)
	if steps == nil {
		return fmt.Errorf("lifecycle.setup.steps field missing or not an array")
	}
	for _, s := range steps {
		step, _ := s.(map[string]any)
		run, _ := step["run"].(string)
		if strings.Contains(run, "pnpm install") && strings.Contains(run, "--ignore-workspace") {
			return nil
		}
	}
	return fmt.Errorf("no step found with 'pnpm install --ignore-workspace'")
}

// replaceRunInRaw performs a targeted replacement of a "run" field value within
// raw JSON bytes. Unlike naive string replacement, it finds the value
// specifically after a "run" key to avoid corrupting other occurrences of the
// same string elsewhere in the document.
func replaceRunInRaw(raw []byte, oldRun, newRun string) []byte {
	oldEncoded, _ := json.Marshal(oldRun)
	newEncoded, _ := json.Marshal(newRun)
	oldLiteral := fmt.Sprintf("%q", oldRun)
	newLiteral := fmt.Sprintf("%q", newRun)

	text := string(raw)

	// Search for "run": <value> patterns and replace only the value that
	// matches oldRun, avoiding false matches in other fields.
	runKeyPattern := `"run"`
	pos := 0
	for {
		keyIdx := jsonAwareIndex(text, runKeyPattern, pos)
		if keyIdx < 0 {
			break
		}
		// Skip past the key and any whitespace/colon to find the value.
		afterKey := keyIdx + len(runKeyPattern)
		valueStart := skipJSONWhitespaceAndColon(text, afterKey)
		if valueStart < 0 {
			pos = afterKey
			continue
		}

		// Try both encoded forms at this position.
		remainder := text[valueStart:]
		if strings.HasPrefix(remainder, string(oldEncoded)) {
			return []byte(text[:valueStart] + string(newEncoded) + text[valueStart+len(oldEncoded):])
		}
		if strings.HasPrefix(remainder, oldLiteral) {
			return []byte(text[:valueStart] + newLiteral + text[valueStart+len(oldLiteral):])
		}
		pos = afterKey
	}

	// No "run" key found with the expected value. Return the input unchanged
	// rather than doing a naive string replacement that could corrupt other
	// fields containing the same text. The caller's semantic validation will
	// catch the untouched value and report the failure.
	return raw
}

// skipJSONWhitespaceAndColon advances past optional whitespace and a colon
// after a JSON key. Returns the index of the value start, or -1 if no colon found.
func skipJSONWhitespaceAndColon(text string, pos int) int {
	for pos < len(text) && (text[pos] == ' ' || text[pos] == '\t' || text[pos] == '\n' || text[pos] == '\r') {
		pos++
	}
	if pos >= len(text) || text[pos] != ':' {
		return -1
	}
	pos++ // skip colon
	for pos < len(text) && (text[pos] == ' ' || text[pos] == '\t' || text[pos] == '\n' || text[pos] == '\r') {
		pos++
	}
	return pos
}

// insertStepInRaw inserts a new install-ui-deps step into the steps array
// within the raw JSON bytes. desiredRun is the run command string.
// If no steps array exists, it falls back to a full marshal.
func insertStepInRaw(raw []byte, desiredRun string, lifecycle, setup map[string]any, stepsAny []any) []byte {
	text := string(raw)

	stepsClose := findStepsArrayClose(text)
	indent := detectStepIndent(text)

	if stepsClose >= 0 && indent != "" {
		closingIndent := detectClosingBracketIndent(text, stepsClose)
		propIndent := indent + "  "
		stepJSON := "{\n" + propIndent + `"name": "install-ui-deps",` + "\n" + propIndent + `"run": ` + fmt.Sprintf("%q", desiredRun) + "\n" + indent + "}"

		before := text[:stepsClose]
		after := text[stepsClose:]

		if len(stepsAny) > 0 {
			trimmed := strings.TrimRight(before, " \t\n\r")
			return []byte(trimmed + ",\n" + indent + stepJSON + "\n" + closingIndent + after)
		}
		return []byte(before + "\n" + indent + stepJSON + "\n" + closingIndent + after)
	}

	// Fallback: re-parse and marshal (loses key order but guarantees correctness).
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)

	if lifecycle == nil {
		lifecycle = map[string]any{}
	}
	if setup == nil {
		setup = map[string]any{}
	}

	// Guard against duplicate insertion: check if a step with the desired
	// command already exists before appending.
	for _, s := range stepsAny {
		step, _ := s.(map[string]any)
		run, _ := step["run"].(string)
		if strings.Contains(run, "pnpm install") && strings.Contains(run, "--ignore-workspace") {
			return raw // already present, nothing to do
		}
	}

	newStep := map[string]any{
		"name": "install-ui-deps",
		"run":  desiredRun,
	}
	steps := append(stepsAny, newStep)
	setup["steps"] = steps
	lifecycle["setup"] = setup
	doc["lifecycle"] = lifecycle
	afterBytes, _ := json.MarshalIndent(doc, "", "  ")
	return append(afterBytes, '\n')
}

// detectStepIndent returns the whitespace prefix used for step objects inside the
// "steps" array. It scans for the first `{` after the `"steps"` key and returns
// everything between the preceding newline and that brace.
// Returns empty string if the text has no newlines (compact JSON).
func detectStepIndent(text string) string {
	idx := jsonAwareIndex(text, `"steps"`, 0)
	if idx < 0 {
		return "        " // 8-space default
	}
	// Find the first '{' after "steps" (skipping the '[').
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
// It uses JSON-aware scanning to correctly handle brackets inside string values.
func findStepsArrayClose(text string) int {
	idx := jsonAwareIndex(text, `"steps"`, 0)
	if idx < 0 {
		return -1
	}
	// Find the opening '[' after "steps".
	bracketStart := -1
	for i := idx + len(`"steps"`); i < len(text); i++ {
		if text[i] == '[' {
			bracketStart = i
			break
		}
		if text[i] != ' ' && text[i] != '\t' && text[i] != '\n' && text[i] != '\r' && text[i] != ':' {
			return -1 // unexpected token
		}
	}
	if bracketStart < 0 {
		return -1
	}
	return jsonAwareMatchingBracket(text, bracketStart)
}

// jsonAwareIndex finds the first occurrence of needle in text starting at pos,
// but only when the needle is NOT inside a JSON string value. This prevents
// matching keys or values that happen to contain the needle as a substring.
func jsonAwareIndex(text, needle string, pos int) int {
	inString := false
	for i := pos; i < len(text); i++ {
		if inString {
			if text[i] == '\\' {
				i++ // skip escaped character
				continue
			}
			if text[i] == '"' {
				inString = false
			}
			continue
		}
		if text[i] == '"' {
			// Check if the needle starts here.
			if strings.HasPrefix(text[i:], needle) {
				return i
			}
			inString = true
			continue
		}
	}
	return -1
}

// jsonAwareMatchingBracket finds the matching closing bracket for an opening
// '[' at the given position, correctly skipping brackets inside JSON strings.
func jsonAwareMatchingBracket(text string, openPos int) int {
	if openPos >= len(text) || text[openPos] != '[' {
		return -1
	}
	depth := 0
	inString := false
	for i := openPos; i < len(text); i++ {
		if inString {
			if text[i] == '\\' {
				i++ // skip escaped character
				continue
			}
			if text[i] == '"' {
				inString = false
			}
			continue
		}
		switch text[i] {
		case '"':
			inString = true
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
