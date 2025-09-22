package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

/*
Rule: Setup Steps Configuration
Description: Ensure lifecycle.setup.steps include consistent initialization tasks
Reason: Reliable setup steps prevent missing binaries and inconsistent developer environments
Category: config
Severity: medium
Standard: configuration-v1
Targets: service_json

<test-case id="missing-setup-steps" should-fail="true">
  <description>service.json without lifecycle.setup steps</description>
  <input language="json">
{
  "service": {
    "name": "file-tools"
  },
  "lifecycle": {}
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.setup.steps</expected-message>
</test-case>

<test-case id="missing-install-cli" should-fail="true">
  <description>Setup steps missing the install-cli task</description>
  <input language="json">
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>install-cli</expected-message>
</test-case>

<test-case id="mismatched-build-api" should-fail="true">
  <description>Build step does not output scenario-name-api binary</description>
  <input language="json">
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go build", "description": "Build Go API server"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>build-api</expected-message>
</test-case>

<test-case id="show-urls-last" should-fail="true">
  <description>show-urls step is not last in the list</description>
  <input language="json">
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "show-urls", "run": "echo urls"},
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>show-urls</expected-message>
</test-case>

<test-case id="install-cli-run-mismatch" should-fail="true">
  <description>install-cli step uses an unexpected run command</description>
  <input language="json">
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>install-cli run command</expected-message>
</test-case>

<test-case id="missing-service-name" should-fail="true">
  <description>install-cli exists but service.name is missing</description>
  <input language="json">
{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>service.name</expected-message>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule is skipped for files other than service.json</description>
  <input language="json">
{
  "service": {"name": "file-tools"}
}
  </input>
</test-case>

<test-case id="valid-setup" should-fail="false">
  <description>Setup steps include install-cli, scenario-specific build, and show-urls last</description>
  <input language="json">
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "install-cli",
          "run": "cd cli && ./install.sh",
          "description": "Install CLI command globally",
          "condition": {"file_exists": "cli/install.sh"}
        },
        {
          "name": "build-api",
          "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go",
          "description": "Build Go API server",
          "condition": {"file_exists": "api/go.mod"}
        },
        {
          "name": "show-urls",
          "run": "echo 'UI: http://localhost:${UI_PORT}'"
        }
      ]
    }
  }
}
  </input>
</test-case>
*/

// CheckSetupStepsConfiguration validates lifecycle.setup.steps for required tasks.
func CheckSetupStepsConfiguration(content []byte, filePath string) []Violation {
	if !shouldCheckSetupStepsJSON(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		return []Violation{newSetupStepsViolation(filePath, 1, "service.json is empty; expected lifecycle.setup.steps")}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate setup steps: %v", err)
		return []Violation{newSetupStepsViolation(filePath, 1, msg)}
	}

	serviceName := extractSetupStepServiceName(payload)
	if serviceName == "" {
		line := findSetupJSONLine(source, "\"service\"", "\"name\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.name must be set to validate setup steps")}
	}

	lifecycleRaw, ok := payload["lifecycle"]
	if !ok {
		line := findSetupJSONLine(source, "\"lifecycle\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.json must define lifecycle.setup.steps")}
	}

	lifecycleMap, ok := lifecycleRaw.(map[string]interface{})
	if !ok {
		line := findSetupJSONLine(source, "\"lifecycle\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.json lifecycle must be an object")}
	}

	setupRaw, ok := lifecycleMap["setup"]
	if !ok {
		line := findSetupJSONLine(source, "\"setup\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.json must define lifecycle.setup.steps")}
	}

	setupMap, ok := setupRaw.(map[string]interface{})
	if !ok {
		line := findSetupJSONLine(source, "\"setup\"")
		return []Violation{newSetupStepsViolation(filePath, line, "lifecycle.setup must be an object")}
	}

	stepsRaw, ok := setupMap["steps"]
	if !ok {
		line := findSetupJSONLine(source, "\"steps\"")
		return []Violation{newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must be defined")}
	}

	stepsSlice, ok := stepsRaw.([]interface{})
	if !ok || len(stepsSlice) == 0 {
		line := findSetupJSONLine(source, "\"steps\"")
		return []Violation{newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must be a non-empty array")}
	}

	var (
		installStep map[string]interface{}
		buildStep   map[string]interface{}
		lastStep    map[string]interface{}
	)

	lastStepRaw := stepsSlice[len(stepsSlice)-1]
	lastStep, _ = lastStepRaw.(map[string]interface{})

	for _, step := range stepsSlice {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			continue
		}
		name := strings.TrimSpace(toStringOrDefault(stepMap["name"]))
		switch name {
		case "install-cli":
			installStep = stepMap
		case "build-api":
			buildStep = stepMap
		}
	}

	var violations []Violation

	if installStep == nil {
		line := findSetupJSONLine(source, "\"install-cli\"")
		violations = append(violations, newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must include an install-cli step"))
	} else {
		violations = append(violations, validateInstallCLI(filePath, source, installStep)...)
	}

	if buildStep == nil {
		line := findSetupJSONLine(source, "\"build-api\"")
		violations = append(violations, newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must include a build-api step"))
	} else {
		violations = append(violations, validateBuildAPI(filePath, source, buildStep, serviceName)...)
	}

	if lastStep == nil {
		line := findSetupJSONLine(source, "\"steps\"")
		violations = append(violations, newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must end with a show-urls step"))
	} else {
		name := strings.TrimSpace(toStringOrDefault(lastStep["name"]))
		if name != "show-urls" {
			line := findSetupJSONLine(source, "\"name\": \"show-urls\"")
			violations = append(violations, newSetupStepsViolation(filePath, line, "show-urls must be the final setup step"))
		}
	}

	return dedupeSetupStepsViolations(violations)
}

func validateInstallCLI(filePath, source string, step map[string]interface{}) []Violation {
	var violations []Violation

	line := findSetupJSONLine(source, "\"install-cli\"")

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	if run != "cd cli && ./install.sh" {
		violations = append(violations, newSetupStepsViolation(filePath, line, "install-cli run command must be 'cd cli && ./install.sh'"))
	}

	desc := strings.TrimSpace(toStringOrDefault(step["description"]))
	if desc != "Install CLI command globally" {
		violations = append(violations, newSetupStepsViolation(filePath, line, "install-cli description must be 'Install CLI command globally'"))
	}

	if cond, ok := step["condition"].(map[string]interface{}); ok {
		fileExists := strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
		if fileExists != "cli/install.sh" {
			violations = append(violations, newSetupStepsViolation(filePath, line, "install-cli condition.file_exists must be 'cli/install.sh'"))
		}
	}

	return violations
}

func validateBuildAPI(filePath, source string, step map[string]interface{}, serviceName string) []Violation {
	var violations []Violation

	line := findSetupJSONLine(source, "\"build-api\"")

	expectedRun := fmt.Sprintf("cd api && go mod download && go build -o %s-api ./cmd/server/main.go", serviceName)
	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	if run != expectedRun {
		violations = append(violations, newSetupStepsViolation(filePath, line, fmt.Sprintf("build-api run must be '%s'", expectedRun)))
	}

	desc := strings.TrimSpace(toStringOrDefault(step["description"]))
	if desc != "Build Go API server" {
		violations = append(violations, newSetupStepsViolation(filePath, line, "build-api description must be 'Build Go API server'"))
	}

	if cond, ok := step["condition"].(map[string]interface{}); ok {
		fileExists := strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
		if fileExists != "api/go.mod" {
			violations = append(violations, newSetupStepsViolation(filePath, line, "build-api condition.file_exists must be 'api/go.mod'"))
		}
	}

	return violations
}

func newSetupStepsViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_setup_steps",
		Severity:       "medium",
		Title:          "Setup steps configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Ensure lifecycle.setup.steps include install-cli, scenario-specific build-api, and show-urls as the final step",
		Standard:       "configuration-v1",
	}
}

func extractSetupStepServiceName(payload map[string]interface{}) string {
	serviceRaw, ok := payload["service"].(map[string]interface{})
	if !ok {
		return ""
	}
	name := strings.TrimSpace(toStringOrDefault(serviceRaw["name"]))
	return name
}

func toStringOrDefault(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func shouldCheckSetupStepsJSON(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(trimmed))
	if base == "service.json" {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".json") {
		return true
	}
	return false
}

func findSetupJSONLine(content string, tokens ...string) int {
	if len(tokens) == 0 {
		return 1
	}
	lines := strings.Split(content, "\n")
	for idx, line := range lines {
		for _, token := range tokens {
			if strings.Contains(line, token) {
				return idx + 1
			}
		}
	}
	return 1
}

func dedupeSetupStepsViolations(list []Violation) []Violation {
	if len(list) == 0 {
		return list
	}
	seen := make(map[string]bool)
	var deduped []Violation
	for _, v := range list {
		key := fmt.Sprintf("%s|%s|%d", v.Description, v.FilePath, v.LineNumber)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, v)
	}
	return deduped
}
