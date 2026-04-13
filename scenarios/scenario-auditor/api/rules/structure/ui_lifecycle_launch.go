package structure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: UI Lifecycle Launch
Description: UI package scripts must use the hidden native lifecycle protector instead of raw commands or legacy launchers
Reason: Shared lifecycle enforcement belongs in the native vrooli CLI so package-manager differences cannot bypass or break it
Category: structure
Severity: high
Targets: structure

<test-case id="legacy-ui-guard-fails" should-fail="true" path="testdata/ui-lifecycle-legacy">
  <description>Legacy shell guard usage is no longer allowed</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/package.json"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Legacy UI guard helper is not allowed</expected-message>
</test-case>

<test-case id="raw-ui-command-fails" should-fail="true" path="testdata/ui-lifecycle-raw">
  <description>Raw lifecycle-sensitive UI commands must go through the shared launcher</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/package.json"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>must use vrooli lifecycle protect</expected-message>
</test-case>

<test-case id="old-shared-launcher-fails" should-fail="true" path="testdata/ui-lifecycle-missing-dependency">
  <description>The old package-bin launcher is no longer an approved UI entrypoint</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/package.json"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>must use vrooli lifecycle protect</expected-message>
</test-case>

<test-case id="shared-launcher-passes" should-fail="false" path="testdata/ui-lifecycle-shared">
  <description>Hidden native lifecycle protector is an approved UI lifecycle entrypoint</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/package.json"
  ]
}
  </input>
</test-case>
*/

type uiLifecyclePayload struct {
	Scenario string   `json:"scenario"`
	Files    []string `json:"files"`
}

func usesApprovedUILauncher(run string) bool {
	return strings.Contains(run, "vrooli lifecycle protect --")
}

func requiresLifecycleProtection(run string) bool {
	lifecycleSensitiveMarkers := []string{
		"scripts/lib/ui-guard.sh",
		"vrooli-ui-run",
		"vite",
		"vite preview",
		"npx vite",
		"react-scripts start",
		"node server.js",
		"node server.cjs",
		"node server.mjs",
		"node api/server.js",
		"node --watch server.js",
		"nodemon server.js",
		"electron .",
		"node build.js --watch",
		"ts-node template-generator.ts",
	}

	for _, marker := range lifecycleSensitiveMarkers {
		if strings.Contains(run, marker) {
			return true
		}
	}

	return false
}

func CheckUISharedLifecycleLaunch(content []byte, scenarioPath string, scenario string) ([]rules.Violation, error) {
	var payload uiLifecyclePayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return []rules.Violation{newUIViolation("ui", fmt.Sprintf("UI lifecycle payload is invalid JSON: %v", err), "high")}, nil
	}

	scenarioPath = resolveScenarioRoot(scenarioPath, payload.Scenario)
	if scenarioPath == "" {
		scenarioPath = resolveScenarioRoot(payload.Scenario, scenario)
	}
	if scenarioPath == "" {
		return nil, nil
	}

	packageFiles := []string{
		"ui/package.json",
		"ui/electron/package.json",
	}

	var violations []rules.Violation
	for _, rel := range packageFiles {
		abs := filepath.Join(scenarioPath, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}

		var manifest struct {
			Scripts map[string]string `json:"scripts"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			violations = append(violations, newUIViolation(rel, fmt.Sprintf("Invalid package.json for UI lifecycle validation: %v", err), "high"))
			continue
		}

		for _, scriptName := range []string{"start", "dev", "debug", "start:daemon"} {
			run := strings.TrimSpace(manifest.Scripts[scriptName])
			if run == "" {
				continue
			}
			if strings.Contains(run, "scripts/lib/ui-guard.sh") {
				violations = append(violations, newUIViolation(rel, fmt.Sprintf("Legacy UI guard helper is not allowed in %s; use vrooli lifecycle protect instead", scriptName), "high"))
				continue
			}
			if !requiresLifecycleProtection(run) {
				continue
			}
			if !usesApprovedUILauncher(run) {
				violations = append(violations, newUIViolation(rel, fmt.Sprintf("%s must use vrooli lifecycle protect for lifecycle-sensitive UI commands", scriptName), "high"))
			}
		}
	}

	return violations, nil
}
