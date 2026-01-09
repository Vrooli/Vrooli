package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RunReactViteUIInstallsDependencies(ctx context.Context, repoRoot, scenarioName string) RuleResult {
	start := time.Now()
	result := RuleResult{
		RuleID:    "REACT_VITE_UI_INSTALLS_DEPENDENCIES",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = len(result.Findings) == 0
	}()

	_ = ctx

	cleaned := strings.TrimSpace(scenarioName)
	if cleaned != "" {
		scenarioDir := filepath.Join(repoRoot, "scenarios", cleaned)
		result.Findings = append(result.Findings, checkScenarioUIInstallRule(scenarioDir, cleaned)...)
		return result
	}

	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		result.Findings = append(result.Findings, Finding{Level: "error", Message: err.Error()})
		return result
	}

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		scenarioDir := filepath.Join(scenariosRoot, ent.Name())
		result.Findings = append(result.Findings, checkScenarioUIInstallRule(scenarioDir, ent.Name())...)
	}

	return result
}

func checkScenarioUIInstallRule(scenarioDir, scenarioName string) []Finding {
	findings := []Finding{}

	uiPackageJSON := filepath.Join(scenarioDir, "ui", "package.json")
	if !fileExists(uiPackageJSON) {
		return findings
	}

	serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if !fileExists(serviceJSONPath) {
		findings = append(findings, Finding{
			Level:   "warn",
			Message: fmt.Sprintf("%s: UI present but .vrooli/service.json missing", scenarioName),
			Evidence: []Evidence{
				{Type: "file", Ref: uiPackageJSON},
			},
		})
		return findings
	}

	lifecycleInstallOK, installRun := hasUIInstallIgnoreWorkspace(serviceJSONPath)
	if !lifecycleInstallOK {
		findings = append(findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("%s: lifecycle setup must install UI deps with `pnpm install --ignore-workspace`", scenarioName),
			Evidence: []Evidence{
				{Type: "file", Ref: serviceJSONPath},
				{Type: "note", Detail: "Expected setup step like: cd ui && pnpm install --ignore-workspace"},
				{Type: "note", Detail: "Found: " + installRun},
			},
		})
	}

	uiNodeModules := filepath.Join(scenarioDir, "ui", "node_modules")
	if !dirExists(uiNodeModules) {
		findings = append(findings, Finding{
			Level:   "info",
			Message: fmt.Sprintf("%s: ui/node_modules missing (run setup or install UI deps)", scenarioName),
			Evidence: []Evidence{
				{Type: "path", Ref: uiNodeModules},
				{Type: "command", Ref: "cd ui && pnpm install --ignore-workspace"},
			},
		})
	}

	return findings
}

func hasUIInstallIgnoreWorkspace(serviceJSONPath string) (bool, string) {
	b, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return false, ""
	}

	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return false, ""
	}

	lifecycle, _ := doc["lifecycle"].(map[string]any)
	setup, _ := lifecycle["setup"].(map[string]any)
	stepsAny, _ := setup["steps"].([]any)
	best := ""

	for _, stepAny := range stepsAny {
		step, _ := stepAny.(map[string]any)
		run, _ := step["run"].(string)
		if run == "" {
			continue
		}
		if strings.Contains(run, "pnpm install") && strings.Contains(run, "ui") {
			best = run
			// Require ignore-workspace because workspace-mode installs can skip local node_modules.
			if strings.Contains(run, "--ignore-workspace") {
				return true, run
			}
		}
	}

	return false, best
}
