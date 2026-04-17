package structure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: Scenario Required Structure
Description: Ensures every scenario contains the canonical lifecycle wrapper Makefile, manifest, and documentation assets
Reason: Missing or drifted core files breaks the shared scenario operator contract
Category: structure
Severity: critical
Targets: structure

<test-case id="missing-makefile" should-fail="true">
  <description>Scenario missing Makefile and PRD</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
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
    ".vrooli/service.json"
  ]
}
  </input>
</test-case>
*/

// Violation captures missing structure elements.
type Violation = rules.Violation

var scenarioMakefileTargetRegexp = regexp.MustCompile(`(?m)^([A-Za-z0-9_.-]+)\s*:`)

// Check validates the presence of required scenario files and the standard lifecycle wrapper Makefile.
func Check(content string, scenarioPath string, scenario string) ([]Violation, error) {
	var payload struct {
		Scenario string   `json:"scenario"`
		Files    []string `json:"files"`
	}

	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return []Violation{newStructureViolation("scenario", fmt.Sprintf("Structure payload is invalid JSON: %v", err))}, nil
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
		"Makefile",
		"PRD.md",
		"README.md",
	}

	for _, rel := range requiredFiles {
		if !fileExists(scenarioPath, rel, filesSet) {
			violations = append(violations, newStructureViolation(rel, fmt.Sprintf("Missing required file: %s", rel)))
		}
	}

	if fileExists(scenarioPath, "Makefile", filesSet) {
		data, err := os.ReadFile(filepath.Join(scenarioPath, "Makefile"))
		if err != nil {
			violations = append(violations, newStructureViolation("Makefile", fmt.Sprintf("Unable to read Makefile: %v", err)))
		} else if !matchesScenarioLifecycleWrapper(string(data), scenarioPath) {
			violations = append(violations, newStructureViolation("Makefile", "Makefile must provide the standard scenario lifecycle wrapper targets"))
		}
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

func newStructureViolation(path, message string) Violation {
	recommendation := fmt.Sprintf("Add the required resource at %s", path)
	return Violation{
		Severity:       "critical",
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: recommendation,
	}
}

func normalizeFileContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.TrimSpace(content)
}

func canonicalScenarioMakefile() string {
	if cwd, err := os.Getwd(); err == nil {
		return canonicalScenarioMakefileForScenarioPath(filepath.Join(filepath.Dir(cwd), "demo"))
	}
	return canonicalScenarioMakefileFallback()
}

func canonicalScenarioMakefileForScenarioPath(scenarioPath string) string {
	repoRoot := filepath.Dir(filepath.Dir(filepath.Clean(scenarioPath)))
	templatePath := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "Makefile")
	if data, err := os.ReadFile(templatePath); err == nil {
		return normalizeFileContent(string(data))
	}

	return canonicalScenarioMakefileFallback()
}

func canonicalScenarioMakefileFallback() string {
	return strings.TrimSpace(`
.PHONY: help setup start stop restart status logs test open run dev

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

help: ## Show the supported scenario entrypoints
	@printf "Vrooli scenario entrypoints for %s\n\n" "$(SCENARIO_NAME)"
	@printf "  make setup                      Run scenario setup lifecycle\n"
	@printf "  make start                      Start the scenario\n"
	@printf "  make stop                       Stop the scenario\n"
	@printf "  make restart                    Restart the scenario\n"
	@printf "  make status                     Show scenario runtime status\n"
	@printf "  make logs                       Show scenario logs\n"
	@printf "  make test                       Run scenario tests\n"
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make run                        Alias for make start\n"
	@printf "  make dev                        Alias for make start\n"
	@printf "\n"
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\n"

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

start: ## Start the scenario
	@vrooli scenario start "$(SCENARIO_NAME)"

stop: ## Stop the scenario
	@vrooli scenario stop "$(SCENARIO_NAME)"

restart: ## Restart the scenario
	@vrooli scenario restart "$(SCENARIO_NAME)"

status: ## Show scenario runtime status
	@vrooli scenario status "$(SCENARIO_NAME)"

logs: ## Show scenario logs
	@vrooli scenario logs "$(SCENARIO_NAME)"

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

run: start

dev: start
`)
}

func matchesScenarioLifecycleWrapper(content string, scenarioPath string) bool {
	normalized := normalizeFileContent(content)
	if normalized == canonicalScenarioMakefileForScenarioPath(scenarioPath) {
		return true
	}
	return looksLikeScenarioLifecycleWrapper(normalized)
}

func looksLikeScenarioLifecycleWrapper(content string) bool {
	requiredSnippets := []string{
		".DEFAULT_GOAL := help",
		"SCENARIO_NAME := $(notdir $(CURDIR))",
		"help: ## Show the supported scenario entrypoints",
		"vrooli scenario start",
		"vrooli scenario stop",
		"vrooli scenario restart",
		"vrooli scenario status",
		"vrooli scenario logs",
		"vrooli scenario test",
		"vrooli scenario open",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			return false
		}
	}

	requiredTargets := []string{"help", "setup", "start", "stop", "restart", "status", "logs", "test", "open", "run", "dev"}
	foundTargets := make(map[string]struct{}, len(requiredTargets))
	for _, match := range scenarioMakefileTargetRegexp.FindAllStringSubmatch(content, -1) {
		if len(match) == 2 {
			foundTargets[match[1]] = struct{}{}
		}
	}
	for _, target := range requiredTargets {
		if _, ok := foundTargets[target]; !ok {
			return false
		}
	}

	return strings.Contains(content, "run: start") && strings.Contains(content, "dev: start")
}
