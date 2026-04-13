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
Rule: UI Shared Lifecycle Launch
Description: UI package scripts must use the shared api-base lifecycle launcher instead of legacy shell guards
Reason: Shared lifecycle enforcement keeps UI startup policy centralized and prevents helper-script drift across scenarios
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

<test-case id="shared-launcher-passes" should-fail="false" path="testdata/ui-lifecycle-shared">
  <description>Shared api-base launcher is an approved UI lifecycle entrypoint</description>
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

		for _, scriptName := range []string{"start", "dev"} {
			run := strings.TrimSpace(manifest.Scripts[scriptName])
			if run == "" {
				continue
			}
			if strings.Contains(run, "scripts/lib/ui-guard.sh") {
				violations = append(violations, newUIViolation(rel, fmt.Sprintf("Legacy UI guard helper is not allowed in %s; use the shared api-base launcher instead", scriptName), "high"))
			}
		}
	}

	return violations, nil
}
