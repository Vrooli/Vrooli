package rules

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

/*
Rule: Setup Conditions
Description: Ensure lifecycle.setup.condition is configured with binary and CLI readiness checks
Reason: Proper setup guards prevent scenarios from running without compiled binaries or installed CLI entrypoints
Category: config
Severity: high
Standard: configuration-v1
Targets: service_json

<test-case id="missing-condition" should-fail="true">
  <description>Lifecycle setup condition block is missing</description>
  <input language="json">
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {}
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.setup.condition</expected-message>
</test-case>

<test-case id="missing-binary-check" should-fail="true">
  <description>First check is not binaries with <scenario>-api target</description>
  <input language="json">
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "cli",
            "targets": ["scenario-auditor"]
          },
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api"]
          }
        ]
      }
    }
  }
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>binaries</expected-message>
</test-case>

<test-case id="missing-cli-check" should-fail="true">
  <description>Second check is not CLI with scenario target</description>
  <input language="json">
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api"]
          },
          {
            "type": "binaries",
            "targets": ["scenario-auditor-api"]
          }
        ]
      }
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>cli</expected-message>
</test-case>

<test-case id="incorrect-targets" should-fail="true">
  <description>Binaries and CLI checks exist but targets do not match scenario name</description>
  <input language="json">
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/another-api"]
          },
          {
            "type": "cli",
            "targets": ["another"]
          }
        ]
      }
    }
  }
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>api/scenario-auditor-api</expected-message>
</test-case>

<test-case id="valid-condition" should-fail="false">
  <description>Valid setup condition with required binaries and CLI checks</description>
  <input language="json">
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api", "api/extra-api"]
          },
          {
            "type": "cli",
            "targets": ["scenario-auditor", "another-cli"]
          },
          {
            "type": "custom",
            "targets": ["anything"]
          }
        ]
      }
    }
  }
}
  </input>
</test-case>
*/

// CheckServiceSetupConditions validates lifecycle.setup.condition structure for service.json.
func CheckServiceSetupConditions(content []byte, filePath string) []Violation {
	if !shouldValidateServiceJSON(filePath) {
		return nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		return []Violation{newSetupViolation(filePath, 1, fmt.Sprintf("service.json must be valid JSON: %v", err))}
	}

	serviceName, err := extractServiceName(payload)
	if err != nil {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"service\""), err.Error())}
	}

	lifecycle, ok := getObject(payload, "lifecycle")
	if !ok {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"lifecycle\""), "lifecycle.setup.condition must be defined")}
	}

	setup, ok := getObject(lifecycle, "setup")
	if !ok {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"setup\""), "lifecycle.setup.condition must be defined")}
	}

	condition, ok := getObject(setup, "condition")
	if !ok {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"condition\""), "lifecycle.setup.condition must be defined")}
	}

	checksVal, exists := condition["checks"]
	if !exists {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"checks\""), "lifecycle.setup.condition.checks must be an array")}
	}

	checks, ok := checksVal.([]interface{})
	if !ok || len(checks) == 0 {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"checks\""), "lifecycle.setup.condition.checks must be an array")}
	}

	var violations []Violation

	// Validate first check: binaries with scenario-api target
	firstCheck, firstOK := checks[0].(map[string]interface{})
	firstValid := false
	if !firstOK {
		line := findJSONLineSetup(string(content), "\"checks\"")
		violations = append(violations, newSetupViolation(filePath, line, "First lifecycle.setup.condition check must be object with type 'binaries'"))
	} else {
		line := findJSONLineSetup(string(content), "\"checks\"", "\"type\"")
		if valueEquals(firstCheck, "type", "binaries") {
			firstValid = true
		} else {
			violations = append(violations, newSetupViolation(filePath, line, "First lifecycle.setup.condition check must be type 'binaries'"))
		}
		expectedBinary := fmt.Sprintf("api/%s-api", serviceName)
		if firstValid && !containsTarget(firstCheck, expectedBinary) {
			lineTargets := findJSONLineSetup(string(content), "\"checks\"", "\"targets\"")
			violations = append(violations, newSetupViolation(filePath, lineTargets, fmt.Sprintf("Binaries check must include target '%s'", expectedBinary)))
		}
	}

	// Validate second check: cli with scenario target
	if len(checks) < 2 {
		line := findJSONLineSetup(string(content), "\"checks\"")
		violations = append(violations, newSetupViolation(filePath, line, "Second lifecycle.setup.condition check must be type 'cli'"))
	} else if secondCheck, ok := checks[1].(map[string]interface{}); !ok {
		line := findJSONLineSetup(string(content), "\"checks\"")
		violations = append(violations, newSetupViolation(filePath, line, "Second lifecycle.setup.condition check must be object with type 'cli'"))
	} else {
		line := findJSONLineSetup(string(content), "\"checks\"", "\"type\"")
		secondValid := false
		if valueEquals(secondCheck, "type", "cli") {
			secondValid = true
		} else {
			violations = append(violations, newSetupViolation(filePath, line, "Second lifecycle.setup.condition check must be type 'cli'"))
		}
		if secondValid && !containsTarget(secondCheck, serviceName) {
			lineTargets := findJSONLineSetup(string(content), "\"checks\"", "\"targets\"")
			violations = append(violations, newSetupViolation(filePath, lineTargets, fmt.Sprintf("CLI check must include target '%s'", serviceName)))
		}
	}

	return deduplicateSetupViolations(violations)
}

func extractServiceName(payload map[string]interface{}) (string, error) {
	service, ok := getObject(payload, "service")
	if !ok {
		return "", fmt.Errorf("service.name must be defined to validate setup conditions")
	}
	nameVal, ok := service["name"].(string)
	if !ok || strings.TrimSpace(nameVal) == "" {
		return "", fmt.Errorf("service.name must be defined to validate setup conditions")
	}
	return nameVal, nil
}

func getObject(parent map[string]interface{}, key string) (map[string]interface{}, bool) {
	raw, ok := parent[key]
	if !ok {
		return nil, false
	}
	obj, ok := raw.(map[string]interface{})
	return obj, ok
}

func valueEquals(obj map[string]interface{}, key, expected string) bool {
	value, ok := obj[key].(string)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), expected)
}

func containsTarget(check map[string]interface{}, target string) bool {
	raw, ok := check["targets"]
	if !ok {
		return false
	}
	list, ok := raw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range list {
		if s, ok := item.(string); ok && strings.EqualFold(strings.TrimSpace(s), target) {
			return true
		}
	}
	return false
}

func newSetupViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_setup_conditions",
		Severity:       "high",
		Title:          "Lifecycle setup condition issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Configure lifecycle.setup.condition with binaries and CLI checks matching the scenario name",
		Standard:       "configuration-v1",
	}
}

func shouldValidateServiceJSON(path string) bool {
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

func findJSONLineSetup(content string, tokens ...string) int {
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

func deduplicateSetupViolations(list []Violation) []Violation {
	if len(list) == 0 {
		return list
	}
	seen := make(map[string]bool)
	var result []Violation
	for _, v := range list {
		key := fmt.Sprintf("%s|%s|%d", v.Description, v.FilePath, v.LineNumber)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}
	return result
}
