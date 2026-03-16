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

func RunReactViteUIInstallsDependencies(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    "REACT_VITE_UI_INSTALLS_DEPENDENCIES",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = !hasActionableFindings(result.Findings)
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
		if !ent.IsDir() || !isScenarioDir(ent.Name()) {
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
		// If ui/ directory exists but package.json is missing, flag it.
		uiDir := filepath.Join(scenarioDir, "ui")
		if dirExists(uiDir) {
			findings = append(findings, Finding{
				Level:        "info",
				Message:      fmt.Sprintf("%s: ui/ directory exists but package.json is missing — is this intentional?", scenarioName),
				ScenarioName: scenarioName,
				Evidence: []Evidence{
					{Type: "path", Ref: uiDir},
				},
			})
		}
		return findings
	}

	serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if !fileExists(serviceJSONPath) {
		findings = append(findings, Finding{
			Level:        "warn",
			Message:      fmt.Sprintf("%s: UI present but .vrooli/service.json missing", scenarioName),
			ScenarioName: scenarioName,
			Evidence: []Evidence{
				{Type: "file", Ref: uiPackageJSON},
			},
		})
		return findings
	}

	installResult := checkUIInstall(serviceJSONPath)
	if installResult.parseErr != "" {
		findings = append(findings, Finding{
			Level:        "error",
			Message:      fmt.Sprintf("%s: malformed service.json structure: %s", scenarioName, installResult.parseErr),
			ScenarioName: scenarioName,
			Evidence: []Evidence{
				{Type: "file", Ref: serviceJSONPath},
				{Type: "note", Detail: "Expected lifecycle.setup.steps to be a JSON array of step objects"},
			},
		})
	} else if !installResult.ok {
		detail := "Expected setup step: cd ui && pnpm install --ignore-workspace"
		if installResult.evidence != "" && strings.Contains(installResult.evidence, "npm install") && !strings.Contains(installResult.evidence, "pnpm") {
			detail = "npm is not supported — this monorepo uses pnpm workspaces. Replace with: cd ui && pnpm install --ignore-workspace"
		}
		findings = append(findings, Finding{
			Level:        "error",
			Message:      fmt.Sprintf("%s: lifecycle setup must install UI deps with `pnpm install --ignore-workspace` (pnpm required, not npm/yarn)", scenarioName),
			ScenarioName: scenarioName,
			Evidence: []Evidence{
				{Type: "file", Ref: serviceJSONPath},
				{Type: "note", Detail: detail},
				{Type: "note", Detail: "Found: " + installResult.evidence},
			},
		})
	}

	uiNodeModules := filepath.Join(scenarioDir, "ui", "node_modules")
	if !dirExists(uiNodeModules) {
		findings = append(findings, Finding{
			Level:        "info",
			Message:      fmt.Sprintf("%s: ui/node_modules missing (run setup or install UI deps)", scenarioName),
			ScenarioName: scenarioName,
			Evidence: []Evidence{
				{Type: "path", Ref: uiNodeModules},
				{Type: "command", Ref: "cd ui && pnpm install --ignore-workspace"},
			},
		})
	}

	return findings
}

// uiInstallResult describes the outcome of checking a service.json for UI install steps.
type uiInstallResult struct {
	ok       bool   // true if a correct pnpm install --ignore-workspace step was found
	evidence string // best matching run command found, for diagnostics
	parseErr string // non-empty if the service.json structure was malformed
}

func hasUIInstallIgnoreWorkspace(serviceJSONPath string) (bool, string) {
	r := checkUIInstall(serviceJSONPath)
	return r.ok, r.evidence
}

func checkUIInstall(serviceJSONPath string) uiInstallResult {
	b, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return uiInstallResult{parseErr: fmt.Sprintf("cannot read service.json: %v", err)}
	}

	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return uiInstallResult{parseErr: fmt.Sprintf("invalid JSON in service.json: %v", err)}
	}

	lifecycleRaw, lifecycleExists := doc["lifecycle"]
	lifecycle, lifecycleOK := lifecycleRaw.(map[string]any)
	if !lifecycleOK {
		if lifecycleExists {
			return uiInstallResult{parseErr: "lifecycle field is not an object"}
		}
		return uiInstallResult{parseErr: "lifecycle field missing from service.json"}
	}

	setupRaw, setupExists := lifecycle["setup"]
	setup, setupOK := setupRaw.(map[string]any)
	if !setupOK {
		if setupExists {
			return uiInstallResult{parseErr: "lifecycle.setup field is not an object"}
		}
		return uiInstallResult{parseErr: "lifecycle.setup field missing from service.json"}
	}

	stepsRaw, stepsExists := setup["steps"]
	stepsAny, stepsOK := stepsRaw.([]any)
	if !stepsOK {
		if stepsExists {
			return uiInstallResult{parseErr: "lifecycle.setup.steps field is not an array"}
		}
		return uiInstallResult{parseErr: "lifecycle.setup.steps field missing from service.json"}
	}

	best := ""
	for _, stepAny := range stepsAny {
		step, _ := stepAny.(map[string]any)
		run, _ := step["run"].(string)
		if run == "" {
			continue
		}
		name, _ := step["name"].(string)
		isUIRelated := strings.Contains(run, "ui") || strings.Contains(name, "ui")
		// Track any package manager install step related to ui.
		if (strings.Contains(run, "pnpm install") || strings.Contains(run, "npm install")) && isUIRelated {
			best = run
			if strings.Contains(run, "pnpm install") && strings.Contains(run, "--ignore-workspace") {
				return uiInstallResult{ok: true, evidence: run}
			}
		}
	}

	return uiInstallResult{evidence: best}
}
