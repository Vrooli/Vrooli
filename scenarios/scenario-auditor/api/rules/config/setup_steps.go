package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
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
  <input language="json"><![CDATA[
{
  "service": {
    "name": "file-tools"
  },
  "lifecycle": {}
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.setup.steps</expected-message>
</test-case>

<test-case id="missing-install-cli" should-fail="true">
  <description>Setup steps missing the install-cli task</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>install-cli</expected-message>
</test-case>

<test-case id="mismatched-build-api" should-fail="true">
  <description>Build step does not output scenario-name-api binary</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>build-api</expected-message>
</test-case>

<test-case id="show-urls-last" should-fail="true">
  <description>show-urls step is not last in the list</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>show-urls</expected-message>
</test-case>

<test-case id="install-cli-run-mismatch" should-fail="true">
  <description>install-cli step uses an unexpected run command</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>change into the cli directory</expected-message>
</test-case>

<test-case id="install-cli-bash-script" should-fail="false">
  <description>install-cli step invokes cli/install.sh without changing directories</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "bash cli/install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="missing-service-name" should-fail="true">
  <description>install-cli exists but service.name is missing</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>service.name</expected-message>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule is skipped for files other than service.json</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"}
}
  ]]></input>
</test-case>

<test-case id="valid-setup" should-fail="false">
  <description>Setup steps include install-cli, scenario-specific build, and show-urls last</description>
  <input language="json"><![CDATA[
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
  ]]></input>
</test-case>

<test-case id="valid-build-api-dot" should-fail="false">
  <description>build-api allows building the scenario binary from current directory</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go mod tidy && go build -o file-tools-api .", "description": "Build Go API binary"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="valid-install-cli-symlink" should-fail="false">
  <description>install-cli can chmod and symlink the binary into PATH</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "install-cli",
          "run": "cd cli && chmod +x file-tools && ln -sf $(pwd)/file-tools ~/.local/bin/file-tools",
          "description": "Install CLI binary to PATH"
        },
        {
          "name": "build-api",
          "run": "cd api && go build -o file-tools-api ./cmd/server/main.go",
          "description": "Build Go API binary"
        },
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="missing-go-build" should-fail="true">
  <description>build-api step must run go build</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && npm run build", "description": "Build API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>go build</expected-message>
</test-case>

<test-case id="wrong-output-name" should-fail="true">
  <description>build-api must emit the scenario-specific binary name</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go build -o tools-api .", "description": "Build Go API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>output must be file-tools-api</expected-message>
</test-case>

<test-case id="install-cli-no-install" should-fail="true">
  <description>install-cli must perform an installation action</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && echo 'Skipping install'", "description": "CLI setup"},
        {"name": "build-api", "run": "cd api && go build -o file-tools-api .", "description": "Build Go API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>install the CLI binary</expected-message>
</test-case>

<test-case id="missing-show-urls-step" should-fail="true">
  <description>Setup must include a final show-urls step</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go build -o file-tools-api .", "description": "Build Go API"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>final show-urls step</expected-message>
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
		installStep   map[string]interface{}
		buildStep     map[string]interface{}
		lastStep      map[string]interface{}
		showStepIndex = -1
		showStepLine  = findSetupJSONLine(source, "\"show-urls\"")
	)

	lastStepRaw := stepsSlice[len(stepsSlice)-1]
	lastStep, _ = lastStepRaw.(map[string]interface{})

	for idx, step := range stepsSlice {
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
		case "show-urls":
			showStepIndex = idx
			showStepLine = findSetupJSONLine(source, "\"name\": \"show-urls\"")
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

	if showStepIndex == -1 {
		line := findSetupJSONLine(source, "\"show-urls\"")
		violations = append(violations, newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must include a final show-urls step"))
	} else {
		if lastStep == nil {
			line := findSetupJSONLine(source, "\"steps\"")
			violations = append(violations, newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must include a final show-urls step"))
		} else {
			name := strings.TrimSpace(toStringOrDefault(lastStep["name"]))
			if name != "show-urls" {
				violations = append(violations, newSetupStepsViolation(filePath, showStepLine, "show-urls must be the final setup step"))
			}
		}
	}

	return dedupeSetupStepsViolations(violations)
}

func validateInstallCLI(filePath, source string, step map[string]interface{}) []Violation {
	var violations []Violation

	line := findSetupJSONLine(source, "\"install-cli\"")

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	runLower := strings.ToLower(run)
	hasCdCli := strings.Contains(runLower, "cd cli")
	invokesInstallScript := strings.Contains(runLower, "cli/install.sh")
	if !hasCdCli && !invokesInstallScript {
		violations = append(violations, newSetupStepsViolation(filePath, line, "install-cli step must change into the cli directory"))
	}

	installIndicators := []string{"cli/install.sh", "install.sh", "ln -sf", "cp ", "install-cli"}
	indicatorFound := false
	for _, indicator := range installIndicators {
		if strings.Contains(runLower, indicator) {
			indicatorFound = true
			break
		}
	}
	if !indicatorFound {
		violations = append(violations, newSetupStepsViolation(filePath, line, "install-cli step must install the CLI binary (expected install.sh, symlink, or copy command)"))
		return violations
	}

	desc := strings.TrimSpace(toStringOrDefault(step["description"]))
	descLower := strings.ToLower(desc)
	if !(strings.Contains(descLower, "install") && strings.Contains(descLower, "cli")) {
		violations = append(violations, newSetupStepsViolation(filePath, line, "install-cli description must mention installing the CLI"))
	}

	return violations
}

func validateBuildAPI(filePath, source string, step map[string]interface{}, serviceName string) []Violation {
	var violations []Violation

	line := findSetupJSONLine(source, "\"build-api\"")

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	runLower := strings.ToLower(run)
	if !strings.Contains(runLower, "cd api") {
		violations = append(violations, newSetupStepsViolation(filePath, line, "build-api step must change into the api directory"))
	}
	if !strings.Contains(runLower, "go build") {
		violations = append(violations, newSetupStepsViolation(filePath, line, "build-api step must invoke 'go build'"))
		return violations
	}

	outputMatch := buildOutputArgument(run)
	if outputMatch == "" {
		violations = append(violations, newSetupStepsViolation(filePath, line, fmt.Sprintf("build-api run must specify '-o %s-api'", serviceName)))
	} else {
		expectedBinary := fmt.Sprintf("%s-api", serviceName)
		trimmed := strings.Trim(outputMatch, "\"'")
		if filepath.Base(trimmed) != expectedBinary {
			violations = append(violations, newSetupStepsViolation(filePath, line, fmt.Sprintf("build-api output must be %s", expectedBinary)))
		}
	}

	desc := strings.TrimSpace(toStringOrDefault(step["description"]))
	descLower := strings.ToLower(desc)
	if !(strings.Contains(descLower, "build") && strings.Contains(descLower, "api")) {
		violations = append(violations, newSetupStepsViolation(filePath, line, "build-api description must describe building the Go API"))
	}

	if cond, ok := step["condition"].(map[string]interface{}); ok {
		fileExists := strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
		if fileExists != "" && fileExists != "api/go.mod" {
			violations = append(violations, newSetupStepsViolation(filePath, line, "build-api condition.file_exists should reference 'api/go.mod'"))
		}
	}

	return violations
}

var buildOutputRegex = regexp.MustCompile(`-o\s+([^\s]+)`)

func buildOutputArgument(run string) string {
	match := buildOutputRegex.FindStringSubmatch(run)
	if len(match) < 2 {
		return ""
	}
	return match[1]
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
