package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

/*
Rule: Health Lifecycle Event
Description: Ensure lifecycle.health defines required endpoints and checks for API/UI services
Reason: Missing health checks break lifecycle orchestration and monitoring
Category: config
Severity: high
Standard: configuration-v1
Targets: service_json

<test-case id="missing-lifecycle" should-fail="true" path=".vrooli/service.json">
  <description>service.json missing lifecycle section</description>
  <input language="json">
{
  "service": {"name": "demo"}
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.health</expected-message>
</test-case>

<test-case id="missing-endpoints" should-fail="true" path=".vrooli/service.json">
  <description>health block without endpoints</description>
  <input language="json">
{
  "lifecycle": {
    "health": {
      "checks": []
    }
  }
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>endpoints</expected-message>
</test-case>

<test-case id="missing-api-endpoint" should-fail="true" path=".vrooli/service.json">
  <description>health endpoints missing api route</description>
  <input language="json">
{
  "lifecycle": {
    "health": {
      "endpoints": {},
      "checks": []
    }
  }
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>api endpoint</expected-message>
</test-case>

<test-case id="missing-api-check" should-fail="true" path=".vrooli/service.json">
  <description>health checks missing api_endpoint entry</description>
  <input language="json">
{
  "lifecycle": {
    "health": {
      "endpoints": {"api": "/health"},
      "checks": []
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api_endpoint</expected-message>
</test-case>

<test-case id="ui-requires-endpoint" should-fail="true" path=".vrooli/service.json">
  <description>UI port defined but no UI endpoint</description>
  <input language="json">
{
  "ports": {
    "api": {"env_var": "API_PORT"},
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "health": {
      "endpoints": {"api": "/health"},
      "checks": [
        {
          "name": "api_endpoint",
          "type": "http",
          "target": "http://localhost:${API_PORT}/health",
          "critical": true,
          "timeout": 5000,
          "interval": 30000
        }
      ]
    }
  }
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>ui endpoint</expected-message>
</test-case>

<test-case id="valid-health-config" should-fail="false" path=".vrooli/service.json">
  <description>Valid lifecycle health configuration for API and UI</description>
  <input language="json">
{
  "ports": {
    "api": {"env_var": "API_PORT"},
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "health": {
      "description": "Demo health checks",
      "endpoints": {
        "api": "/health",
        "ui": "/health"
      },
      "checks": [
        {
          "name": "api_endpoint",
          "type": "http",
          "target": "http://localhost:${API_PORT}/health",
          "critical": true,
          "timeout": 5000,
          "interval": 30000
        },
        {
          "name": "ui_endpoint",
          "type": "http",
          "target": "http://localhost:${UI_PORT}/health",
          "critical": false,
          "timeout": 3000,
          "interval": 45000
        }
      ]
    }
  }
}
  </input>
</test-case>
*/

// CheckServiceHealthLifecycle validates lifecycle health configuration in service.json.
func CheckServiceHealthLifecycle(content []byte, filePath string) []Violation {
	if !shouldCheckHealthServiceJSON(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		line := findHealthJSONLine(source)
		return []Violation{newHealthViolation(filePath, line, "service.json is empty; expected lifecycle.health configuration")}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate lifecycle health: %v", err)
		return []Violation{newHealthViolation(filePath, 1, msg)}
	}

	lifecycleRaw, ok := payload["lifecycle"]
	if !ok {
		line := findHealthJSONLine(source, "\"lifecycle\"")
		return []Violation{newHealthViolation(filePath, line, "service.json must define lifecycle.health configuration")}
	}

	lifecycleMap, ok := lifecycleRaw.(map[string]interface{})
	if !ok {
		line := findHealthJSONLine(source, "\"lifecycle\"")
		return []Violation{newHealthViolation(filePath, line, "service.json lifecycle must be an object")}
	}

	healthRaw, ok := lifecycleMap["health"]
	if !ok {
		line := findHealthJSONLine(source, "\"health\"", "\"lifecycle\"")
		return []Violation{newHealthViolation(filePath, line, "service.json must define lifecycle.health configuration")}
	}

	healthMap, ok := healthRaw.(map[string]interface{})
	if !ok {
		line := findHealthJSONLine(source, "\"health\"")
		return []Violation{newHealthViolation(filePath, line, "lifecycle.health must be an object")}
	}

	var violations []Violation

	hasUI := scenarioHasUIPort(payload)

	// Validate endpoints
	endpointLine := findHealthJSONLine(source, "\"endpoints\"")
	endpointsRaw, endpointsOk := healthMap["endpoints"].(map[string]interface{})
	if !endpointsOk {
		violations = append(violations, newHealthViolation(filePath, endpointLine, "lifecycle.health must define endpoints"))
	} else {
		apiEndpoint, apiOk := endpointsRaw["api"].(string)
		if !apiOk || strings.TrimSpace(apiEndpoint) == "" {
			violations = append(violations, newHealthViolation(filePath, findHealthJSONLine(source, "\"api\"", "\"endpoints\""), "lifecycle.health must define an api endpoint at '/health'"))
		} else if apiEndpoint != "/health" {
			violations = append(violations, newHealthViolation(filePath, findHealthJSONLine(source, "\"api\"", "\"endpoints\""), "lifecycle.health api endpoint must be '/health'"))
		}

		if hasUI {
			uiEndpoint, uiOk := endpointsRaw["ui"].(string)
			if !uiOk || strings.TrimSpace(uiEndpoint) == "" {
				violations = append(violations, newHealthViolation(filePath, findHealthJSONLine(source, "\"ui\"", "\"endpoints\""), "lifecycle.health must define a ui endpoint at '/health' when UI ports are configured"))
			} else if uiEndpoint != "/health" {
				violations = append(violations, newHealthViolation(filePath, findHealthJSONLine(source, "\"ui\"", "\"endpoints\""), "lifecycle.health ui endpoint must be '/health' when UI ports are configured"))
			}
		}
	}

	// Validate checks
	checksLine := findHealthJSONLine(source, "\"checks\"")
	checksRaw, checksOk := healthMap["checks"].([]interface{})
	if !checksOk {
		violations = append(violations, newHealthViolation(filePath, checksLine, "lifecycle.health must define checks"))
		return dedupeHealthViolations(violations)
	}

	apiCheck := findHealthCheck(checksRaw, "api_endpoint")
	if apiCheck == nil {
		violations = append(violations, newHealthViolation(filePath, checksLine, "lifecycle.health.checks must include an api_endpoint entry"))
	} else {
		violations = append(violations, validateCheck(apiCheck, filePath, source, "api_endpoint", "${API_PORT}")...)
	}

	if hasUI {
		uiCheck := findHealthCheck(checksRaw, "ui_endpoint")
		if uiCheck == nil {
			violations = append(violations, newHealthViolation(filePath, checksLine, "lifecycle.health.checks must include a ui_endpoint entry when UI ports are configured"))
		} else {
			violations = append(violations, validateCheck(uiCheck, filePath, source, "ui_endpoint", "${UI_PORT}")...)
		}
	}

	return dedupeHealthViolations(violations)
}

func findHealthCheck(checks []interface{}, name string) map[string]interface{} {
	for _, entry := range checks {
		checkMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if checkName, ok := checkMap["name"].(string); ok && checkName == name {
			return checkMap
		}
	}
	return nil
}

func validateCheck(check map[string]interface{}, filePath, source, name, portVar string) []Violation {
	var violations []Violation

	checkLine := findHealthJSONLine(source, fmt.Sprintf("\"name\": \"%s\"", name))

	// type must be http
	if checkType, ok := check["type"].(string); !ok || strings.TrimSpace(checkType) == "" {
		violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check must define type 'http'", name)))
	} else if strings.ToLower(checkType) != "http" {
		violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check type must be 'http'", name)))
	}

	// target must include port variable and end with /health
	if target, ok := check["target"].(string); !ok || strings.TrimSpace(target) == "" {
		violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check must define a target", name)))
	} else {
		if !strings.Contains(target, portVar) {
			violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check target must reference %s", name, portVar)))
		}
		if !strings.HasSuffix(target, "/health") {
			violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check target must end with /health", name)))
		}
	}

	// critical must be boolean
	if _, ok := check["critical"].(bool); !ok {
		violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check must define critical as boolean", name)))
	}

	// timeout and interval numeric
	if !hasPositiveNumber(check, "timeout") {
		violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check must define a positive timeout", name)))
	}
	if !hasPositiveNumber(check, "interval") {
		violations = append(violations, newHealthViolation(filePath, checkLine, fmt.Sprintf("%s check must define a positive interval", name)))
	}

	return violations
}

func hasPositiveNumber(obj map[string]interface{}, key string) bool {
	value, ok := obj[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case float64:
		return v > 0
	case int:
		return v > 0
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return false
		}
		return f > 0
	default:
		return false
	}
}

func scenarioHasUIPort(payload map[string]interface{}) bool {
	portsRaw, ok := payload["ports"].(map[string]interface{})
	if !ok {
		return false
	}
	uiRaw, ok := portsRaw["ui"]
	if !ok {
		return false
	}
	uiMap, ok := uiRaw.(map[string]interface{})
	if !ok {
		return true
	}
	if env, ok := uiMap["env_var"].(string); ok && strings.TrimSpace(env) != "" {
		return true
	}
	if _, ok := uiMap["range"]; ok {
		return true
	}
	if _, ok := uiMap["fixed"]; ok {
		return true
	}
	return true
}

func newHealthViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_service_health",
		Severity:       "high",
		Title:          "Lifecycle health configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Define lifecycle.health endpoints and checks for API_PORT and UI_PORT services",
		Standard:       "configuration-v1",
	}
}

func dedupeHealthViolations(list []Violation) []Violation {
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

func shouldCheckHealthServiceJSON(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if base == "service.json" {
		return true
	}
	return false
}

func findHealthJSONLine(content string, tokens ...string) int {
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
