package requiredlayout

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	auditrules "structure-health/internal/packs/auditrules"
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
type Violation = auditrules.Violation

var scenarioMakefileTargetRegexp = regexp.MustCompile(`(?m)^([A-Za-z0-9_.-]+)\s*:`)
var templateReadmeScaffoldRegexp = regexp.MustCompile(`(?i)generated from the .*react-vite.*template`)

type scenarioMakefileTarget struct {
	Header  string
	Recipes []string
}

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

// CheckScenarioDocumentation enforces the durable scenario identity surface.
// A README must describe the scenario's capability, not the template that
// happened to create it.
func CheckScenarioDocumentation(_ string, scenarioPath string, _ string) []Violation {
	if strings.TrimSpace(scenarioPath) == "" {
		return nil
	}
	readme, err := os.ReadFile(filepath.Join(scenarioPath, "README.md"))
	if err != nil || !templateReadmeScaffoldRegexp.Match(readme) {
		return nil
	}
	return []Violation{{
		Severity:       "critical",
		Message:        "scenario README contains template scaffold language",
		FilePath:       "README.md",
		Recommendation: "Rewrite the README to describe the scenario's permanent capability and remove template provenance boilerplate.",
	}}
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
	if matchesScenarioLifecycleTemplate(normalized, canonicalScenarioMakefileForScenarioPath(scenarioPath)) {
		return true
	}
	return matchesScenarioLifecycleTemplate(normalized, canonicalScenarioMakefileFallback())
}

func matchesScenarioLifecycleTemplate(content string, template string) bool {
	if normalizeFileContent(content) == normalizeFileContent(template) {
		return true
	}

	return matchesScenarioLifecycleTargets(content, template)
}

func matchesScenarioLifecycleTargets(content string, template string) bool {
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

	candidateTargets, candidatePhony := parseScenarioMakefileTargets(content)
	templateTargets, templatePhony := parseScenarioMakefileTargets(template)
	if len(templateTargets) == 0 {
		return false
	}

	candidatePhonySet := make(map[string]struct{}, len(candidatePhony))
	for _, target := range candidatePhony {
		candidatePhonySet[target] = struct{}{}
	}
	for _, target := range templatePhony {
		if _, ok := candidatePhonySet[target]; !ok {
			return false
		}
	}

	for name, expected := range templateTargets {
		actual, ok := candidateTargets[name]
		if !ok {
			return false
		}
		if normalizeFileContent(actual.Header) != normalizeFileContent(expected.Header) {
			return false
		}
		if len(actual.Recipes) != len(expected.Recipes) {
			return false
		}
		for i := range expected.Recipes {
			if !matchesScenarioLifecycleRecipe(name, actual.Recipes[i], expected.Recipes[i]) {
				return false
			}
		}
	}

	return true
}

func matchesScenarioLifecycleRecipe(target string, actual string, expected string) bool {
	normalizedActual := normalizeFileContent(actual)
	normalizedExpected := normalizeFileContent(expected)
	if normalizedActual == normalizedExpected {
		return true
	}

	if target != "logs" {
		return false
	}

	return normalizedActual == `@vrooli scenario logs "$(SCENARIO_NAME)" --tail 50` &&
		normalizedExpected == `@vrooli scenario logs "$(SCENARIO_NAME)"`
}

func parseScenarioMakefileTargets(content string) (map[string]scenarioMakefileTarget, []string) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	targets := make(map[string]scenarioMakefileTarget)
	phony := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ".PHONY:") {
			phony = append(phony, strings.Fields(strings.TrimSpace(strings.TrimPrefix(trimmed, ".PHONY:")))...)
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ".") || strings.Contains(trimmed, "=") {
			continue
		}
		match := scenarioMakefileTargetRegexp.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}

		name := strings.TrimSpace(match[1])
		var recipes []string
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			if strings.HasPrefix(next, "\t") {
				recipes = append(recipes, strings.TrimRight(next, " \t"))
				continue
			}
			if strings.TrimSpace(next) == "" {
				continue
			}
			break
		}

		targets[name] = scenarioMakefileTarget{
			Header:  strings.TrimSpace(line),
			Recipes: recipes,
		}
	}

	return targets, phony
}
