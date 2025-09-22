package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

/*
Rule: Develop Lifecycle Steps
Description: Ensure lifecycle.develop steps start the scenario services consistently
Reason: Consistent develop workflow keeps API/UI orchestration predictable across scenarios
Category: config
Severity: medium
Standard: configuration-v1
Targets: service_json

<test-case id="missing-develop-steps" should-fail="true">
  <description>service.json without lifecycle develop steps</description>
  <input language="json">
{
  "service": {"name": "demo-svc"},
  "lifecycle": {}
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.develop.steps</expected-message>
</test-case>

<test-case id="missing-start-api" should-fail="true">
  <description>Develop steps missing the required start-api command</description>
  <input language="json">
{
  "service": {"name": "demo"},
  "lifecycle": {
    "develop": {
      "steps": [
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>start-api</expected-message>
</test-case>

<test-case id="bad-start-api" should-fail="true">
  <description>start-api exists but uses the wrong binary name</description>
  <input language="json">
{
  "service": {"name": "demo"},
  "lifecycle": {
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./wrong",
          "background": true,
          "description": "Start Go API server in background",
          "condition": {"file_exists": "api/demo-api"}
        },
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>cd api && ./demo-api</expected-message>
</test-case>

<test-case id="missing-start-ui" should-fail="true">
  <description>UI port present but no start-ui step</description>
  <input language="json">
{
  "service": {"name": "demo"},
  "ports": {
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./demo-api",
          "description": "Start Go API server in background",
          "background": true,
          "condition": {"file_exists": "api/demo-api"}
        },
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>start-ui</expected-message>
</test-case>

<test-case id="show-urls-last" should-fail="true">
  <description>show-urls is not the final develop step</description>
  <input language="json">
{
  "service": {"name": "demo"},
  "lifecycle": {
    "develop": {
      "steps": [
        {"name": "show-urls", "run": "echo first"},
        {
          "name": "start-api",
          "run": "cd api && ./demo-api",
          "description": "Start Go API server in background",
          "background": true,
          "condition": {"file_exists": "api/demo-api"}
        }
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>show-urls</expected-message>
</test-case>

<test-case id="valid-develop" should-fail="false">
  <description>Valid develop configuration with API and UI steps</description>
  <input language="json">
{
  "service": {"name": "demo"},
  "ports": {
    "api": {"env_var": "API_PORT"},
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./demo-api",
          "description": "Start Go API server in background",
          "background": true,
          "condition": {"file_exists": "api/demo-api"}
        },
        {
          "name": "start-ui",
          "run": "cd ui && npm run dev",
          "background": true,
          "description": "Start UI dev server"
        },
        {
          "name": "show-urls",
          "run": "echo done"
        }
      ]
    }
  }
}
  </input>
</test-case>
*/

// CheckDevelopLifecycleSteps validates lifecycle.develop configuration in service.json.
func CheckDevelopLifecycleSteps(content []byte, filePath string) []Violation {
	if !shouldCheckDevelopServiceJSON(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		return []Violation{newDevelopViolation(filePath, 1, "service.json is empty; expected lifecycle configuration")}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate develop steps: %v", err)
		return []Violation{newDevelopViolation(filePath, 1, msg)}
	}

	lifecycleMap, _ := getMap(payload, "lifecycle")
	developMap, _ := getMap(lifecycleMap, "develop")

	if developMap == nil {
		line := findJSONLineDevelop(source, "\"develop\"")
		return []Violation{newDevelopViolation(filePath, line, "lifecycle.develop.steps must be defined with required steps")}
	}

	stepsRaw, ok := developMap["steps"]
	if !ok {
		line := findJSONLineDevelop(source, "\"develop\"", "\"steps\"")
		return []Violation{newDevelopViolation(filePath, line, "lifecycle.develop.steps must be defined as an array")}
	}

	stepsSlice, ok := stepsRaw.([]interface{})
	if !ok || len(stepsSlice) == 0 {
		line := findJSONLineDevelop(source, "\"develop\"", "\"steps\"")
		return []Violation{newDevelopViolation(filePath, line, "lifecycle.develop.steps must include at least the start and show-urls commands")}
	}

	scenarioName := strings.TrimSpace(getScenarioName(payload))
	var violations []Violation

	startAPIStep, _ := findStepByName(stepsSlice, "start-api")
	if startAPIStep == nil {
		line := findJSONLineDevelop(source, "\"start-api\"")
		violations = append(violations, newDevelopViolation(filePath, line, "Develop steps must include a \"start-api\" command"))
	} else {
		expectedRun := fmt.Sprintf("cd api && ./%s-api", scenarioName)
		if scenarioName == "" {
			expectedRun = "cd api && ./<scenario>-api"
		}

		if runVal := strings.TrimSpace(getString(startAPIStep, "run")); runVal != expectedRun {
			line := findJSONLineDevelop(source, "\"start-api\"", "\"run\"")
			violations = append(violations, newDevelopViolation(filePath, line, fmt.Sprintf("start-api run command must be %q", expectedRun)))
		}

		if desc := strings.TrimSpace(getString(startAPIStep, "description")); desc != "Start Go API server in background" {
			line := findJSONLineDevelop(source, "\"start-api\"", "\"description\"")
			violations = append(violations, newDevelopViolation(filePath, line, "start-api description must be \"Start Go API server in background\""))
		}

		if bg, ok := startAPIStep["background"].(bool); !ok || !bg {
			line := findJSONLineDevelop(source, "\"start-api\"", "\"background\"")
			violations = append(violations, newDevelopViolation(filePath, line, "start-api must run in the background"))
		}

		if cond, ok := startAPIStep["condition"].(map[string]interface{}); ok {
			expectedPath := fmt.Sprintf("api/%s-api", scenarioName)
			if scenarioName == "" {
				expectedPath = "api/<scenario>-api"
			}
			if fileExists := strings.TrimSpace(getString(cond, "file_exists")); fileExists != expectedPath {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"file_exists\"")
				violations = append(violations, newDevelopViolation(filePath, line, fmt.Sprintf("start-api condition.file_exists must be %q", expectedPath)))
			}
		} else {
			line := findJSONLineDevelop(source, "\"start-api\"", "\"condition\"")
			violations = append(violations, newDevelopViolation(filePath, line, "start-api must include a file_exists condition for the API binary"))
		}
	}

	hasUI := strings.Contains(source, "UI_PORT")
	if hasUI {
		startUIStep, _ := findStepByName(stepsSlice, "start-ui")
		if startUIStep == nil {
			line := findJSONLineDevelop(source, "\"start-ui\"")
			violations = append(violations, newDevelopViolation(filePath, line, "Develop steps must include a \"start-ui\" command when UI_PORT is present"))
		} else if bg, ok := startUIStep["background"].(bool); !ok || !bg {
			line := findJSONLineDevelop(source, "\"start-ui\"", "\"background\"")
			violations = append(violations, newDevelopViolation(filePath, line, "start-ui must run in the background when provided"))
		}
	}

	lastStepName := strings.TrimSpace(getString(stepsSlice[len(stepsSlice)-1].(map[string]interface{}), "name"))
	if lastStepName != "show-urls" {
		line := findJSONLineDevelop(source, "\"show-urls\"")
		violations = append(violations, newDevelopViolation(filePath, line, "show-urls step must be the final develop step"))
	}

	showStep, showIndex := findStepByName(stepsSlice, "show-urls")
	if showStep == nil {
		line := findJSONLineDevelop(source, "\"show-urls\"")
		violations = append(violations, newDevelopViolation(filePath, line, "Develop steps must include a \"show-urls\" command"))
	} else if showIndex != len(stepsSlice)-1 {
		line := findJSONLineDevelop(source, "\"show-urls\"")
		violations = append(violations, newDevelopViolation(filePath, line, "show-urls step must be the final develop step"))
	}

	return deduplicateDevelopViolations(violations)
}

func newDevelopViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_develop_steps",
		Severity:       "medium",
		Title:          "Develop lifecycle configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Provide start-api, start-ui (when UI_PORT is defined), and show-urls steps in lifecycle.develop",
		Standard:       "configuration-v1",
	}
}

func getMap(root map[string]interface{}, key string) (map[string]interface{}, bool) {
	if root == nil {
		return nil, false
	}
	raw, ok := root[key]
	if !ok {
		return nil, false
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, false
	}
	return m, true
}

func getString(root map[string]interface{}, key string) string {
	if root == nil {
		return ""
	}
	if val, ok := root[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getScenarioName(payload map[string]interface{}) string {
	serviceMap, _ := getMap(payload, "service")
	if serviceMap == nil {
		return ""
	}
	return getString(serviceMap, "name")
}

func findStepByName(steps []interface{}, name string) (map[string]interface{}, int) {
	for idx, stepRaw := range steps {
		if stepMap, ok := stepRaw.(map[string]interface{}); ok {
			if strings.TrimSpace(getString(stepMap, "name")) == name {
				return stepMap, idx
			}
		}
	}
	return nil, -1
}

func shouldCheckDevelopServiceJSON(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if base == "service.json" {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".json") {
		return true
	}
	return false
}

func findJSONLineDevelop(content string, tokens ...string) int {
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

func deduplicateDevelopViolations(list []Violation) []Violation {
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
