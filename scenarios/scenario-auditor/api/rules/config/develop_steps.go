package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
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
  <input language="json"><![CDATA[
{
  "service": {"name": "demo-svc"},
  "lifecycle": {}
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.develop.steps</expected-message>
</test-case>

<test-case id="missing-start-api" should-fail="true">
  <description>Develop steps missing the required start-api command</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "demo"},
  "ports": {
    "api": {"env_var": "API_PORT"}
  },
  "lifecycle": {
    "develop": {
      "steps": [
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>start-api</expected-message>
</test-case>

<test-case id="bad-start-api" should-fail="true">
  <description>start-api exists but uses the wrong binary name</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "demo"},
  "ports": {
    "api": {"env_var": "API_PORT"}
  },
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
  ]]></input>
  <expected-violations>2</expected-violations>
  <expected-message>must execute "./demo-api"</expected-message>
</test-case>

<test-case id="missing-start-ui" should-fail="true">
  <description>UI port present but no start-ui step</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>start-ui</expected-message>
</test-case>

<test-case id="show-urls-last" should-fail="true">
  <description>show-urls is not the final develop step</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>show-urls</expected-message>
</test-case>

<test-case id="valid-develop" should-fail="false">
  <description>Valid develop configuration with API and UI steps</description>
  <input language="json"><![CDATA[
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
  ]]></input>
</test-case>

<test-case id="start-api-env-preamble" should-fail="false">
  <description>start-api allows environment preparation before launching binary</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "demo"},
  "ports": {
    "api": {"env_var": "API_PORT"}
  },
  "lifecycle": {
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "export DATABASE_URL=$(./lib/get-db-url.sh) && cd api && ENVIRONMENT=dev ./demo-api",
          "description": "Start Go API server with prepared environment",
          "background": true,
          "condition": {"file_exists": "api/demo-api"}
        },
        {
          "name": "show-urls",
          "run": "echo done"
        }
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="ui-disabled" should-fail="false">
  <description>UI component disabled so start-ui step is not required</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "demo"},
  "ports": {
    "api": {"env_var": "API_PORT"},
    "ui": {"env_var": "UI_PORT"}
  },
  "components": {
    "ui": {
      "enabled": false
    }
  },
  "lifecycle": {
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./demo-api",
          "description": "Start Go API server",
          "background": true,
          "condition": {"file_exists": "api/demo-api"}
        },
        {
          "name": "show-urls",
          "run": "echo done"
        }
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="ui-only" should-fail="false">
  <description>No API port so start-api step is not required</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "demo"},
  "ports": {
    "ui": {"env_var": "UI_PORT"}
  },
  "components": {
    "api": {
      "enabled": false
    }
  },
  "lifecycle": {
    "develop": {
      "steps": [
        {
          "name": "start-ui",
          "run": "cd ui && npm run dev",
          "background": true
        },
        {
          "name": "show-urls",
          "run": "echo done"
        }
      ]
    }
  }
}
  ]]></input>
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

	if requireAPIStart(payload) {
		startAPIStep, _ := findStepByName(stepsSlice, "start-api")
		if startAPIStep == nil {
			line := findJSONLineDevelop(source, "\"start-api\"")
			violations = append(violations, newDevelopViolation(filePath, line, "Develop steps must include a \"start-api\" command"))
		} else {
			runVal := strings.TrimSpace(getString(startAPIStep, "run"))
			lineRun := findJSONLineDevelop(source, "\"start-api\"", "\"run\"")
			if !strings.Contains(runVal, "cd api") {
				violations = append(violations, newDevelopViolation(filePath, lineRun, "start-api run command must change into the api directory (e.g. `cd api && ...`)"))
			}

			binaryName := extractBinaryName(runVal)
			expectedBinary := ""
			if scenarioName != "" {
				expectedBinary = fmt.Sprintf("%s-api", scenarioName)
			}

			if expectedBinary != "" && binaryName != expectedBinary {
				violations = append(violations, newDevelopViolation(filePath, lineRun, fmt.Sprintf("start-api must execute \"./%s\"", expectedBinary)))
			} else if expectedBinary == "" && binaryName == "" {
				violations = append(violations, newDevelopViolation(filePath, lineRun, "start-api run command must execute the scenario API binary (e.g. ./<scenario>-api)"))
			}

			if desc := strings.TrimSpace(getString(startAPIStep, "description")); desc == "" {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"description\"")
				violations = append(violations, newDevelopViolation(filePath, line, "start-api description must be provided"))
			}

			if bg, ok := startAPIStep["background"].(bool); !ok || !bg {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"background\"")
				violations = append(violations, newDevelopViolation(filePath, line, "start-api must run in the background"))
			}

			if cond, ok := startAPIStep["condition"].(map[string]interface{}); ok {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"file_exists\"")
				fileExists := strings.TrimSpace(getString(cond, "file_exists"))
				if fileExists == "" {
					violations = append(violations, newDevelopViolation(filePath, line, "start-api condition.file_exists must reference the API binary"))
				} else if binaryName != "" && !strings.Contains(fileExists, binaryName) {
					expectedPath := fmt.Sprintf("api/%s", binaryName)
					violations = append(violations, newDevelopViolation(filePath, line, fmt.Sprintf("start-api condition.file_exists should reference %q", expectedPath)))
				}
			} else {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"condition\"")
				violations = append(violations, newDevelopViolation(filePath, line, "start-api must include a file_exists condition for the API binary"))
			}
		}
	}

	if requireUIStart(payload) {
		startUIStep, _ := findStepByName(stepsSlice, "start-ui")
		if startUIStep == nil {
			line := findJSONLineDevelop(source, "\"start-ui\"")
			violations = append(violations, newDevelopViolation(filePath, line, "Develop steps must include a \"start-ui\" command when the UI is enabled"))
		} else if bg, ok := startUIStep["background"].(bool); !ok || !bg {
			line := findJSONLineDevelop(source, "\"start-ui\"", "\"background\"")
			violations = append(violations, newDevelopViolation(filePath, line, "start-ui must run in the background when provided"))
		}
	}

	lastStepMap, ok := stepsSlice[len(stepsSlice)-1].(map[string]interface{})
	if ok {
		lastStepName := strings.TrimSpace(getString(lastStepMap, "name"))
		if lastStepName != "show-urls" {
			line := findJSONLineDevelop(source, "\"show-urls\"")
			violations = append(violations, newDevelopViolation(filePath, line, "show-urls step must be the final develop step"))
		}
	} else {
		line := findJSONLineDevelop(source, "\"develop\"", "\"steps\"")
		violations = append(violations, newDevelopViolation(filePath, line, "Each develop step must be an object with a name"))
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

func requireAPIStart(payload map[string]interface{}) bool {
	enabled, present := componentEnabled(payload, "api")
	if present {
		return enabled
	}
	return hasPort(payload, "api")
}

func requireUIStart(payload map[string]interface{}) bool {
	enabled, present := componentEnabled(payload, "ui")
	if present {
		return enabled
	}
	return hasPort(payload, "ui")
}

func componentEnabled(payload map[string]interface{}, name string) (bool, bool) {
	components, _ := getMap(payload, "components")
	component, present := getMap(components, name)
	if !present || component == nil {
		return false, false
	}
	if enabledVal, ok := component["enabled"]; ok {
		if enabled, ok := enabledVal.(bool); ok {
			return enabled, true
		}
	}
	return true, true
}

func hasPort(payload map[string]interface{}, name string) bool {
	ports, _ := getMap(payload, "ports")
	if ports == nil {
		return false
	}
	if entry, ok := ports[name]; ok {
		_, ok := entry.(map[string]interface{})
		return ok
	}
	return false
}

func extractBinaryName(run string) string {
	idx := strings.LastIndex(run, "cd api")
	search := run
	if idx != -1 {
		search = run[idx:]
	}
	re := regexp.MustCompile(`\./([A-Za-z0-9._-]+)`)
	match := re.FindStringSubmatch(search)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
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
