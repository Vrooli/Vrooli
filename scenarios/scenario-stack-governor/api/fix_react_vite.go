package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FixReactViteUIInstallsDependencies patches service.json to add --ignore-workspace to pnpm install.
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
	if lifecycle == nil {
		lifecycle = map[string]any{}
		doc["lifecycle"] = lifecycle
	}
	setup, _ := lifecycle["setup"].(map[string]any)
	if setup == nil {
		setup = map[string]any{}
		lifecycle["setup"] = setup
	}
	stepsAny, _ := setup["steps"].([]any)

	// Find existing pnpm install step for ui.
	found := false
	for i, stepAny := range stepsAny {
		step, ok := stepAny.(map[string]any)
		if !ok {
			continue
		}
		run, _ := step["run"].(string)
		if run == "" {
			continue
		}
		if strings.Contains(run, "pnpm install") && strings.Contains(run, "ui") {
			if strings.Contains(run, "--ignore-workspace") {
				// Already correct.
				found = true
				break
			}
			// Patch: add --ignore-workspace.
			patched := strings.Replace(run, "pnpm install", "pnpm install --ignore-workspace", 1)
			step["run"] = patched
			stepsAny[i] = step
			changes = append(changes, FixChange{
				Type:   "patched_step",
				Detail: fmt.Sprintf("Added --ignore-workspace to existing step: %s", patched),
			})
			found = true
			break
		}
	}

	if !found {
		// Add new step.
		newStep := map[string]any{
			"name": "install-ui-deps",
			"run":  "cd ui && pnpm install --ignore-workspace",
		}
		stepsAny = append(stepsAny, newStep)
		changes = append(changes, FixChange{
			Type:   "added_step",
			Detail: "Added setup step: cd ui && pnpm install --ignore-workspace",
		})
	}

	if len(changes) == 0 {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
		}}
	}

	setup["steps"] = stepsAny
	lifecycle["setup"] = setup
	doc["lifecycle"] = lifecycle

	afterBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     serviceJSONPath,
			Error:        err.Error(),
		}}
	}
	afterBytes = append(afterBytes, '\n')

	var diff *FileDiff
	if dryRun {
		diff = &FileDiff{Before: string(raw), After: string(afterBytes)}
	} else {
		if err := os.WriteFile(serviceJSONPath, afterBytes, 0o644); err != nil {
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
