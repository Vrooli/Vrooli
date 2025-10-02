package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
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

<test-case id="missing-ports-block" should-fail="true" path=".vrooli/service.json">
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

<test-case id="incorrect-api-env-var" should-fail="true" path=".vrooli/service.json">
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

<test-case id="incorrect-api-range" should-fail="true" path=".vrooli/service.json">
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

<test-case id="ui-env-var-check" should-fail="true" path=".vrooli/service.json">
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

<test-case id="valid-config" should-fail="false" path=".vrooli/service.json">
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

<test-case id="invalid-json" should-fail="true" path=".vrooli/service.json">
  <description>service.json contains malformed JSON</description>
  <input language="json">{</input>
  <expected-violations>1</expected-violations>
  <expected-message>valid JSON</expected-message>
</test-case>

<test-case id="missing-api-entry" should-fail="true" path=".vrooli/service.json">
  <description>ports block defined without api entry</description>
  <input language="json">
{
  "ports": {
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ports configuration must define</expected-message>
</test-case>

<test-case id="api-range-missing" should-fail="true" path=".vrooli/service.json">
  <description>API port missing range definition</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ports.api.range</expected-message>
</test-case>

<test-case id="ui-range-mismatch" should-fail="true" path=".vrooli/service.json">
  <description>UI range provided but outside approved window</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "32000-32500"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ports.ui.range</expected-message>
</test-case>

<test-case id="ui-port-fixed-valid" should-fail="false" path=".vrooli/service.json">
  <description>UI port uses fixed assignment with port field in valid range</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "port": 38000
    }
  }
}
  </input>
</test-case>

<test-case id="ui-port-fixed-reserved" should-fail="true" path=".vrooli/service.json">
  <description>UI port uses fixed assignment in reserved range</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "port": 3500
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>reserved range</expected-message>
</test-case>

<test-case id="api-range-invalid-format" should-fail="true" path=".vrooli/service.json">
  <description>API range has invalid format</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-notanumber"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>invalid format</expected-message>
</test-case>

<test-case id="api-range-inverted" should-fail="true" path=".vrooli/service.json">
  <description>API range has start greater than end</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "19999-15000"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>start must be less than end</expected-message>
</test-case>

<test-case id="api-range-overlaps-reserved" should-fail="true" path=".vrooli/service.json">
  <description>API range overlaps with reserved ranges</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "3000-4000"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>reserved range</expected-message>
</test-case>

<test-case id="ports-api-not-object" should-fail="true" path=".vrooli/service.json">
  <description>api entry is not an object</description>
  <input language="json">
{
  "ports": {
    "api": 8080
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ports.api must be</expected-message>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule should ignore files that are not service.json</description>
  <input language="json">
{
  "ports": {
    "api": {
      "env_var": "PORT",
      "range": "10000-12000"
    }
  }
}
  </input>
</test-case>
*/

// CheckServicePortConfiguration validates that service.json declares expected port entries.
func CheckServicePortConfiguration(content []byte, filePath string) []Violation {
	if !shouldCheckPortsServiceJSON(filePath) {
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
		line := findPortsJSONLine(source, "\"ports\"")
		return []Violation{newPortsViolation(filePath, line, "service.json must define a top-level \"ports\" object")}
	}

	portsMap, ok := portsRaw.(map[string]interface{})
	if !ok {
		line := findPortsJSONLine(source, "\"ports\"")
		return []Violation{newPortsViolation(filePath, line, "service.json ports must be an object of named port configurations")}
	}

	var violations []Violation

	apiRaw, hasAPI := portsMap["api"]
	if !hasAPI {
		line := findPortsJSONLine(source, "\"api\"", "\"ports\"")
		violations = append(violations, newPortsViolation(filePath, line, "ports configuration must define an \"api\" entry"))
	} else if apiMap, ok := apiRaw.(map[string]interface{}); ok {
		validateAPIPort(apiMap, source, filePath, &violations)
	} else {
		line := findPortsJSONLine(source, "\"api\"")
		violations = append(violations, newPortsViolation(filePath, line, "ports.api must be an object with env_var and range fields"))
	}

	if uiRaw, hasUI := portsMap["ui"]; hasUI {
		if uiMap, ok := uiRaw.(map[string]interface{}); ok {
			validateUIPort(uiMap, source, filePath, &violations)
		} else {
			line := findPortsJSONLine(source, "\"ui\"")
			violations = append(violations, newPortsViolation(filePath, line, "ports.ui must be an object when provided"))
		}
	}

	return dedupePortViolations(violations)
}

func validateAPIPort(entry map[string]interface{}, source, filePath string, violations *[]Violation) {
	envVar, ok := entry["env_var"].(string)
	lineEnv := findPortsJSONLine(source, "\"api\"", "\"env_var\"")
	if !ok || strings.TrimSpace(envVar) == "" {
		*violations = append(*violations, newPortsViolation(filePath, lineEnv, "ports.api.env_var must be set to \"API_PORT\""))
	} else if envVar != "API_PORT" {
		*violations = append(*violations, newPortsViolation(filePath, lineEnv, "ports.api.env_var must be \"API_PORT\""))
	}

	rangeLine := findPortsJSONLine(source, "\"api\"", "\"range\"")
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

	// Parse and validate range format
	start, end, err := parsePortRange(rangeStr)
	if err != nil {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.api.range has %s", err.Error())))
		return
	}

	// Check for overlap with reserved ranges FIRST (more specific error)
	if overlaps, reservedName := checkReservedRangeOverlap(start, end); overlaps {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.api.range overlaps with reserved range: %s", reservedName)))
		return
	}

	// Check if range is the approved one
	if rangeStr != "15000-19999" {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.api.range must be \"15000-19999\""))
	}
}

func validateUIPort(entry map[string]interface{}, source, filePath string, violations *[]Violation) {
	envVarLine := findPortsJSONLine(source, "\"ui\"", "\"env_var\"")
	envVar, ok := entry["env_var"].(string)
	if !ok || strings.TrimSpace(envVar) == "" || envVar != "UI_PORT" {
		*violations = append(*violations, newPortsViolation(filePath, envVarLine, "ports.ui.env_var must be \"UI_PORT\" when the ui port is defined"))
	}

	// Check for range field
	if rangeVal, hasRange := entry["range"]; hasRange {
		rangeLine := findPortsJSONLine(source, "\"ui\"", "\"range\"")
		rangeStr, ok := rangeVal.(string)
		if !ok || strings.TrimSpace(rangeStr) == "" {
			*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.ui.range must be \"35000-39999\" when a range is specified"))
			return
		}

		// Parse and validate range format
		start, end, err := parsePortRange(rangeStr)
		if err != nil {
			*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.ui.range has %s", err.Error())))
			return
		}

		// Check if range is the approved one
		if rangeStr != "35000-39999" {
			*violations = append(*violations, newPortsViolation(filePath, rangeLine, "ports.ui.range must be \"35000-39999\" when a range is specified"))
			return
		}

		// Check for overlap with reserved ranges
		if overlaps, reservedName := checkReservedRangeOverlap(start, end); overlaps {
			*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.ui.range overlaps with reserved range: %s", reservedName)))
		}
	}

	// Check for fixed port field
	if portVal, hasPort := entry["port"]; hasPort {
		portLine := findPortsJSONLine(source, "\"ui\"", "\"port\"")

		// Port can be either a number or a string (with env var substitution like "${PORT:-3500}")
		var port int
		switch v := portVal.(type) {
		case float64:
			port = int(v)
		case int:
			port = v
		case string:
			// If it's a string with env var substitution like "${PORT:-3500}", extract the default
			if strings.Contains(v, "${") {
				// Skip validation for env var strings - they're validated at runtime
				return
			}
			// Try to parse as number
			parsed, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				*violations = append(*violations, newPortsViolation(filePath, portLine, "ports.ui.port must be a valid port number"))
				return
			}
			port = parsed
		default:
			*violations = append(*violations, newPortsViolation(filePath, portLine, "ports.ui.port must be a number"))
			return
		}

		// Validate port is in valid range
		if port < 1 || port > 65535 {
			*violations = append(*violations, newPortsViolation(filePath, portLine, "ports.ui.port must be between 1 and 65535"))
			return
		}

		// Check if fixed port is in a reserved range FIRST (more specific error)
		if inReserved, reservedName := checkFixedPortInReserved(port); inReserved {
			*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.ui.port %d is in reserved range: %s", port, reservedName)))
			return
		}

		// Verify fixed port is in the UI range (35000-39999)
		if port < 35000 || port > 39999 {
			*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.ui.port %d should be in range 35000-39999", port)))
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

func dedupePortViolations(list []Violation) []Violation {
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

func shouldCheckPortsServiceJSON(path string) bool {
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

func findPortsJSONLine(content string, tokens ...string) int {
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

// Reserved port ranges that scenarios must NOT use (from port_registry.sh)
var reservedRanges = map[string][2]int{
	"vrooli_core": {3000, 4100},
	"databases":   {5432, 5499},
	"cache":       {6379, 6399},
	"debug":       {9200, 9299},
	"system":      {1, 1023},
}

// parsePortRange parses a range string like "15000-19999" and returns start, end, error
func parsePortRange(rangeStr string) (int, int, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format: expected 'start-end'")
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid format: start port must be a number")
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid format: end port must be a number")
	}

	if start < 1 || start > 65535 || end < 1 || end > 65535 {
		return 0, 0, fmt.Errorf("ports must be between 1 and 65535")
	}

	if start >= end {
		return 0, 0, fmt.Errorf("start must be less than end")
	}

	return start, end, nil
}

// checkReservedRangeOverlap checks if a port range overlaps with any reserved ranges
func checkReservedRangeOverlap(start, end int) (bool, string) {
	for name, reserved := range reservedRanges {
		// Check for overlap: range overlaps if start <= reserved_end && end >= reserved_start
		if start <= reserved[1] && end >= reserved[0] {
			return true, fmt.Sprintf("%s (%d-%d)", name, reserved[0], reserved[1])
		}
	}
	return false, ""
}

// checkFixedPortInReserved checks if a fixed port is in a reserved range
func checkFixedPortInReserved(port int) (bool, string) {
	for name, reserved := range reservedRanges {
		if port >= reserved[0] && port <= reserved[1] {
			return true, fmt.Sprintf("%s (%d-%d)", name, reserved[0], reserved[1])
		}
	}
	return false, ""
}
