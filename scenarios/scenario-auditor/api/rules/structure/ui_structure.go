package structure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	rules "scenario-auditor/rules"
)

var (
	ruleDir = discoverRuleDir()
)

type uiStructurePayload struct {
	Scenario string   `json:"scenario"`
	Files    []string `json:"files"`
}

/*
Rule: Scenario UI Structure
Description: Validates that scenarios supplying a UI ship a usable shell with required entry points
Reason: Monitoring and orchestration flows depend on a consistent UI entry point when present
Category: structure
Severity: high
Targets: structure

<test-case id="missing-ui-directory" should-fail="false">
  <description>Scenario missing ui directory is allowed</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/main.go"
  ]
}
  </input>
</test-case>

<test-case id="missing-index" should-fail="true" path="testdata/missing-index">
  <description>ui directory exists but index.html missing</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/app.js",
    "ui/src/main.tsx"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI entry file</expected-message>
</test-case>

<test-case id="react-success" should-fail="false" path="testdata/react-success">
  <description>React/Vite style UI with entrypoints wired up</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html",
    "ui/package.json",
    "ui/src/App.tsx",
    "ui/src/main.tsx",
    "ui/vite.config.ts",
    "ui/tsconfig.json"
  ]
}
  </input>
</test-case>

<test-case id="static-success" should-fail="false" path="testdata/static-success">
  <description>Static HTML UI with application assets</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html",
    "ui/app.js",
    "ui/styles.css"
  ]
}
  </input>
</test-case>

<test-case id="app-monitor-example" should-fail="false" scenario="app-monitor" path="testdata/app-monitor-example">
  <description>App monitor scenario illustrates optional UI handling</description>
  <input language="json">
{
  "scenario": "app-monitor",
  "files": [
    "ui/index.html",
    "ui/src/main.tsx"
  ]
}
  </input>
</test-case>
*/

// CheckUIStructure validates the UI structure entrypoint expected by the scenario lifecycle.
func CheckUIStructure(content []byte, scenarioPath string, scenario string) ([]rules.Violation, error) {
	return CheckUICore(content, scenarioPath, scenario)
}

// CheckUICore validates the UI shell for required entry assets.
func CheckUICore(content []byte, scenarioPath string, scenario string) ([]rules.Violation, error) {
	var payload uiStructurePayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return []rules.Violation{newUIViolation("ui", fmt.Sprintf("UI structure payload is invalid JSON: %v", err), "high")}, nil
	}

	scenarioName := strings.TrimSpace(payload.Scenario)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(scenario)
	}

	scenarioPath = resolveScenarioRoot(scenarioPath, payload.Scenario)
	if scenarioPath == "" {
		scenarioPath = resolveScenarioRoot(payload.Scenario, "")
	}
	if scenarioName == "" && strings.TrimSpace(scenarioPath) != "" {
		scenarioName = filepath.Base(filepath.Clean(scenarioPath))
	}

	filesSet := make(map[string]struct{}, len(payload.Files))
	for _, f := range payload.Files {
		filesSet[filepath.ToSlash(strings.TrimSpace(f))] = struct{}{}
	}

	var violations []rules.Violation

	if !uiDirectoryExists(scenarioPath, "ui", filesSet) {
		// UI is optional. If a scenario has no UI assets we consider it compliant.
		return nil, nil
	}

	if !uiFileExists(scenarioPath, "ui/index.html", filesSet) {
		violations = append(violations, newUIViolation("ui/index.html", "Missing required UI entry file: ui/index.html", "high"))
	}

	if !hasUIEntrypoint(scenarioPath, filesSet) {
		violations = append(violations, newUIViolation("ui", "Missing required UI entry script", "high"))
	}

	return violations, nil
}

func hasUIEntrypoint(root string, files map[string]struct{}) bool {
	entryCandidates := []string{
		"ui/src/main.tsx",
		"ui/src/main.ts",
		"ui/src/main.jsx",
		"ui/src/main.js",
		"ui/src/index.tsx",
		"ui/src/index.ts",
		"ui/src/index.jsx",
		"ui/src/index.js",
		"ui/main.tsx",
		"ui/main.ts",
		"ui/main.jsx",
		"ui/main.js",
		"ui/app.js",
		"ui/app.ts",
		"ui/app.tsx",
		"ui/script.js",
		"ui/script.ts",
		"ui/js/app.js",
		"ui/js/main.js",
		"ui/server.js",
		"ui/server.ts",
	}

	for _, candidate := range entryCandidates {
		if uiFileExists(root, candidate, files) {
			return true
		}
	}
	return false
}

func newUIViolation(path, message string, severity string) rules.Violation {
	severity = strings.TrimSpace(severity)
	if severity == "" {
		severity = "high"
	}
	recommendation := fmt.Sprintf("Add the required resource at %s", path)
	return rules.Violation{
		Severity:       severity,
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: recommendation,
	}
}

func uiFileExists(root, rel string, known map[string]struct{}) bool {
	rel = filepath.ToSlash(rel)
	if _, ok := known[rel]; ok {
		return true
	}
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

func uiDirectoryExists(root, rel string, known map[string]struct{}) bool {
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

func resolveScenarioRoot(input string, fallback string) string {
	candidates := []string{strings.TrimSpace(input), strings.TrimSpace(fallback)}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if resolved := resolveCandidate(candidate); resolved != "" {
			return resolved
		}
	}
	return ""
}

func resolveCandidate(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return filepath.Clean(path)
		}
		return ""
	}

	tryPaths := []string{
		path,
		filepath.Join(ruleDir, path),
		filepath.Join(filepath.Dir(ruleDir), path),
		filepath.Join(filepath.Dir(filepath.Dir(ruleDir)), path),
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			tryPaths = append(tryPaths, filepath.Join(base, path))
		}
	}

	if wd, err := os.Getwd(); err == nil {
		tryPaths = append(tryPaths,
			filepath.Join(wd, path),
			filepath.Join(wd, "rules", "structure", path),
			filepath.Join(wd, "api", "rules", "structure", path),
			filepath.Join(wd, "scenarios", "scenario-auditor", "api", "rules", "structure", path),
		)
	}

	for _, candidate := range tryPaths {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}

	return ""
}

func discoverRuleDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		if strings.HasSuffix(file, "ui_structure.go") {
			return filepath.Dir(file)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		if dir := searchRuleDirFrom(wd); dir != "" {
			return dir
		}
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			if dir := searchRuleDirFrom(base); dir != "" {
				return dir
			}
		}
	}

	return "."
}

func searchRuleDirFrom(start string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	current := filepath.Clean(start)
	visited := map[string]struct{}{}
	for {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		candidates := []string{
			current,
			filepath.Join(current, "rules", "structure"),
			filepath.Join(current, "api", "rules", "structure"),
			filepath.Join(current, "scenarios", "scenario-auditor", "api", "rules", "structure"),
		}

		for _, candidate := range candidates {
			file := filepath.Join(candidate, "ui_structure.go")
			if info, err := os.Stat(file); err == nil && !info.IsDir() {
				return filepath.Dir(file)
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}
