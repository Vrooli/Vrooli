package rules

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

/*
Rule: Ports Configuration
Description: Ensure service.json defines reusable API/UI ports with approved ranges
Reason: Consistent port allocation prevents collisions between scenarios and enables lifecycle orchestration
Category: config
Severity: high
Standard: configuration-v1
Targets: service_json

<test-case id="missing-ports-block" should-fail="true">
  <description>service.json without a ports section</description>
  <input language="json">
{
  "service": {
    "name": "sample"
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ports</expected-message>
</test-case>

<test-case id="incorrect-api-env-var" should-fail="true">
  <description>API port defined with wrong env var</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "PORT",
      "range": "15000-19999"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>API_PORT</expected-message>
</test-case>

<test-case id="incorrect-api-range" should-fail="true">
  <description>API port range outside the allowed window</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "10000-12000"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>15000-19999</expected-message>
</test-case>

<test-case id="ui-env-var-check" should-fail="true">
  <description>UI port defined without the expected env var</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "PORT",
      "range": "35000-39999"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>UI_PORT</expected-message>
</test-case>

<test-case id="valid-config" should-fail="false">
  <description>Valid ports configuration with api and ui ranges</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "description": "Primary API",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999",
      "description": "Optional UI"
    }
  }
}
  </input>
</test-case>
*/

// CheckServicePortConfiguration validates that service.json declares expected port entries.
func CheckServicePortConfiguration(content []byte, filePath string) []Violation {
	if !shouldCheckServiceJSON(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		return []Violation{newPortsViolation(filePath, 1, "service.json is empty; expected a ports configuration block")}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate ports: %v", err)
		return []Violation{newPortsViolation(filePath, 1, msg)}
	}

	portsRaw, ok := payload["ports"]
	if !ok {
		line := findJSONLine(source, "\"ports\"")
		return []Violation{newPortsViolation(filePath, line, "service.json must define a top-level \"ports\" object")}
	}

	portsMap, ok := portsRaw.(map[string]interface{})
	if !ok {
		line := findJSONLine(source, "\"ports\"")
		return []Violation{newPortsViolation(filePath, line, "service.json ports must be an object of named port configurations")}
	}

	var violations []Violation

	apiRaw, hasAPI := portsMap["api"]
	if !hasAPI {
		line := findJSONLine(source, "\"api\"", "\"ports\"")
		violations = append(violations, newPortsViolation(filePath, line, "ports configuration must define an \"api\" entry"))
	} else if apiMap, ok := apiRaw.(map[string]interface{}); ok {
		validateAPIPort(apiMap, source, filePath, &violations)
	} else {
		line := findJSONLine(source, "\"api\"")
		violations = append(violations, newPortsViolation(filePath, line, "ports.api must be an object with env_var and range fields"))
	}

	if uiRaw, hasUI := portsMap["ui"]; hasUI {
		if uiMap, ok := uiRaw.(map[string]interface{}); ok {
			validateUIPort(uiMap, source, filePath, &violations)
		} else {
			line := findJSONLine(source, "\"ui\"")
			violations = append(violations, newPortsViolation(filePath, line, "ports.ui must be an object when provided"))
		}
	}

	return deduplicateViolations(violations)
}

func validateAPIPort(entry map[string]interface{}, source, filePath string, violations *[]Violation) {
	envVar, ok := entry["env_var"].(string)
	lineEnv := findJSONLine(source, "\"api\"", "\"env_var\"")
	if !ok || strings.TrimSpace(envVar) == "" {
		*violations = append(*violations, newPortsViolation(filePath, lineEnv, "ports.api.env_var must be set to \"API_PORT\""))
	} else if envVar != "API_PORT" {
		*violations = append(*violations, newPortsViolation(filePath, lineEnv, "ports.api.env_var must be \"API_PORT\""))
	}

	rangeLine := findJSONLine(source, "\"api\"", "\"range\"")
	rangeVal, hasRange := entry["range"]
	if !hasRange {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.api.range must be \"15000-19999\""))
		return
	}

	rangeStr, ok := rangeVal.(string)
	if !ok || strings.TrimSpace(rangeStr) == "" {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.api.range must be \"15000-19999\""))
		return
	}

	if rangeStr != "15000-19999" {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.api.range must be \"15000-19999\""))
	}
}

func validateUIPort(entry map[string]interface{}, source, filePath string, violations *[]Violation) {
	envVarLine := findJSONLine(source, "\"ui\"", "\"env_var\"")
	envVar, ok := entry["env_var"].(string)
	if !ok || strings.TrimSpace(envVar) == "" || envVar != "UI_PORT" {
		*violations = append(*violations, newPortsViolation(filePath, envVarLine, "ports.ui.env_var must be \"UI_PORT\" when the ui port is defined"))
	}

	if rangeVal, hasRange := entry["range"]; hasRange {
		rangeLine := findJSONLine(source, "\"ui\"", "\"range\"")
		rangeStr, ok := rangeVal.(string)
		if !ok || strings.TrimSpace(rangeStr) == "" {
			*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.ui.range must be \"35000-39999\" when a range is specified"))
			return
		}
		if rangeStr != "35000-39999" {
			*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.ui.range must be \"35000-39999\" when a range is specified"))
		}
	}
}

func newPortsViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_service_ports",
		Severity:       "high",
		Title:          "Ports configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Define API_PORT (range 15000-19999) and UI_PORT (range 35000-39999 when used) in .vrooli/service.json",
		Standard:       "configuration-v1",
	}
}

func deduplicateViolations(list []Violation) []Violation {
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

func shouldCheckServiceJSON(path string) bool {
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

func findJSONLine(content string, tokens ...string) int {
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
