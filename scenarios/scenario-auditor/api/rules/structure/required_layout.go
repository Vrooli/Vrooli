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
Rule: Scenario Required Structure
Description: Ensures every scenario contains required lifecycle, API, CLI, and documentation assets
Reason: Missing core files prevents the lifecycle system and CLI tooling from functioning
Category: structure
Severity: critical
Targets: structure

Note: Testing is handled by test-genie via .vrooli/service.json lifecycle.test.
Test artifacts live in bas/ (playbooks, fixtures) and coverage/ (logs, reports).

<test-case id="missing-makefile" should-fail="true">
  <description>Scenario missing Makefile and PRD</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/main.go",
    "cli/install.sh",
    "cli/demo",
    ".vrooli/service.json",
    "README.md"
  ]
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Missing required file</expected-message>
</test-case>

<test-case id="complete-structure" should-fail="false">
  <description>Scenario with complete set of required files</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "Makefile",
    "PRD.md",
    "README.md",
    ".vrooli/service.json",
    "api/main.go",
    "cli/install.sh",
    "cli/demo"
  ]
}
  </input>
</test-case>
*/

// Violation captures missing structure elements.
type Violation = rules.Violation

// Check validates the presence of required scenario files and directories.
func Check(content string, scenarioPath string, scenario string) ([]Violation, error) {
	var payload struct {
		Scenario string   `json:"scenario"`
		Files    []string `json:"files"`
	}

	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return []Violation{newStructureViolation("scenario", fmt.Sprintf("Structure payload is invalid JSON: %v", err))}, nil
	}

	scenarioName := strings.TrimSpace(payload.Scenario)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(scenario)
	}
	if scenarioName == "" && scenarioPath != "" {
		scenarioName = filepath.Base(scenarioPath)
	}

	scenarioPath = strings.TrimSpace(scenarioPath)
	if scenarioPath == "" {
		// Without a real path we cannot validate existence; report once.
		return []Violation{newStructureViolation("scenario", "Unable to determine scenario root for structure validation")}, nil
	}

	filesSet := make(map[string]struct{}, len(payload.Files))
	for _, f := range payload.Files {
		filesSet[filepath.ToSlash(strings.TrimSpace(f))] = struct{}{}
	}

	var violations []Violation

	// Core required files.
	requiredFiles := []string{
		".vrooli/service.json",
		"api/main.go",
		"cli/install.sh",
		"Makefile",
		"PRD.md",
		"README.md",
	}

	for _, rel := range requiredFiles {
		if !fileExists(scenarioPath, rel, filesSet) {
			violations = append(violations, newStructureViolation(rel, fmt.Sprintf("Missing required file: %s", rel)))
		}
	}

	// CLI binary must be named after the scenario.
	if scenarioName != "" {
		cliBinary := filepath.ToSlash(filepath.Join("cli", scenarioName))
		if !fileExists(scenarioPath, cliBinary, filesSet) {
			violations = append(violations, newStructureViolation(cliBinary, fmt.Sprintf("Missing CLI entrypoint executable: %s", cliBinary)))
		}
	} else {
		violations = append(violations, newStructureViolation("cli/<scenario>", "Unable to determine scenario name for CLI binary validation"))
	}

	return violations, nil
}

func fileExists(root, rel string, known map[string]struct{}) bool {
	rel = filepath.ToSlash(rel)
	if _, ok := known[rel]; ok {
		return true
	}
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

func directoryExists(root, rel string, known map[string]struct{}) bool {
	rel = filepath.ToSlash(rel)
	if _, ok := known[rel]; ok {
		return true
	}
	prefix := rel
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for path := range known {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	info, err := os.Stat(filepath.Join(root, rel))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func newStructureViolation(path, message string) Violation {
	recommendation := fmt.Sprintf("Add the required resource at %s", path)
	return Violation{
		Severity:       "critical",
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: recommendation,
	}
}
