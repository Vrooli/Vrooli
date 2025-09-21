package rules

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

/*
Rule: Test Lifecycle Steps
Description: Ensure lifecycle.test.steps runs the standard phased testing script
Reason: Guarantees scenarios execute the unified test suite for consistent coverage
Category: config
Severity: high
Standard: configuration-v1
Targets: service_json

<test-case id="missing-lifecycle" should-fail="true">
  <description>service.json missing lifecycle object</description>
  <input language="json">
{
  "service": {
    "name": "demo"
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle</expected-message>
</test-case>

<test-case id="missing-test-object" should-fail="true">
  <description>service.json missing lifecycle.test</description>
  <input language="json">
{
  "lifecycle": {
    "develop": {}
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.test</expected-message>
</test-case>

<test-case id="missing-steps-array" should-fail="true">
  <description>service.json missing lifecycle.test.steps</description>
  <input language="json">
{
  "lifecycle": {
    "test": {
      "description": "custom"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>steps</expected-message>
</test-case>

<test-case id="steps-not-array" should-fail="true">
  <description>lifecycle.test.steps is not an array</description>
  <input language="json">
{
  "lifecycle": {
    "test": {
      "steps": "run-tests"
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>array</expected-message>
</test-case>

<test-case id="missing-required-step" should-fail="true">
  <description>steps defined but required run-tests step missing</description>
  <input language="json">
{
  "lifecycle": {
    "test": {
      "steps": [
        {
          "name": "custom",
          "run": "scripts/test.sh"
        }
      ]
    }
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>run-tests</expected-message>
</test-case>

<test-case id="valid-test-steps" should-fail="false">
  <description>Valid lifecycle.test.steps configuration including extra step</description>
  <input language="json">
{
  "lifecycle": {
    "test": {
      "description": "custom description ok",
      "steps": [
        {
          "name": "run-tests",
          "run": "test/run-tests.sh",
          "description": "Execute comprehensive phased testing (structure, dependencies, unit, integration, business, performance)"
        },
        {
          "name": "notify",
          "run": "scripts/notify.sh"
        }
      ]
    }
  }
}
  </input>
</test-case>
*/

// CheckLifecycleTestSteps validates that lifecycle.test.steps includes the standard run-tests entry.
func CheckLifecycleTestSteps(content []byte, filePath string) []Violation {
	if !shouldCheckLifecycleServiceJSON(filePath) {
		return nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate lifecycle test steps: %v", err)
		return []Violation{newTestStepsViolation(filePath, 1, msg)}
	}

	lifecycleRaw, ok := payload["lifecycle"]
	if !ok {
		line := findJSONLine(string(content), "\"lifecycle\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json must define a \"lifecycle\" object with test steps")}
	}

	lifecycle, ok := lifecycleRaw.(map[string]interface{})
	if !ok {
		line := findJSONLine(string(content), "\"lifecycle\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle must be an object")}
	}

	testRaw, ok := lifecycle["test"]
	if !ok {
		line := findJSONLine(string(content), "\"test\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json must define \"lifecycle.test\" with steps")}
	}

	testSection, ok := testRaw.(map[string]interface{})
	if !ok {
		line := findJSONLine(string(content), "\"test\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test must be an object")}
	}

	stepsRaw, ok := testSection["steps"]
	if !ok {
		line := findJSONLine(string(content), "\"steps\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test must define a \"steps\" array")}
	}

	steps, ok := stepsRaw.([]interface{})
	if !ok {
		line := findJSONLine(string(content), "\"steps\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test.steps must be an array")}
	}

	if len(steps) == 0 {
		line := findJSONLine(string(content), "\"steps\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test.steps must include the run-tests entry")}
	}

	requiredStep := map[string]string{
		"name":        "run-tests",
		"run":         "test/run-tests.sh",
		"description": "Execute comprehensive phased testing (structure, dependencies, unit, integration, business, performance)",
	}

	hasRequired := false
	for _, entry := range steps {
		stepMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		match := true
		for key, expected := range requiredStep {
			value, ok := stepMap[key].(string)
			if !ok || value != expected {
				match = false
				break
			}
		}
		if match {
			hasRequired = true
			break
		}
	}

	if !hasRequired {
		line := findJSONLine(string(content), "\"steps\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test.steps must include the standard run-tests step")}
	}

	return nil
}

func shouldCheckLifecycleServiceJSON(path string) bool {
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

func newTestStepsViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_test_steps",
		Severity:       "high",
		Title:          "Test lifecycle configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Ensure lifecycle.test.steps includes the shared run-tests step (test/run-tests.sh)",
		Standard:       "configuration-v1",
	}
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
