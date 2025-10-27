package config

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
  <input language="json"><![CDATA[
{
  "service": {
    "name": "demo"
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle</expected-message>
</test-case>

<test-case id="missing-test-object" should-fail="true">
  <description>service.json missing lifecycle.test</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "develop": {}
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.test</expected-message>
</test-case>

<test-case id="missing-steps-array" should-fail="true">
  <description>service.json missing lifecycle.test.steps</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "description": "custom"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>steps</expected-message>
</test-case>

<test-case id="empty-steps-array" should-fail="true">
  <description>lifecycle.test.steps defined but empty</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "steps": []
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>cannot be empty</expected-message>
</test-case>

<test-case id="steps-not-array" should-fail="true">
  <description>lifecycle.test.steps is not an array</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "steps": "run-tests"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>array</expected-message>
</test-case>

<test-case id="missing-required-step" should-fail="true">
  <description>steps defined but required run-tests step missing</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>test/run-tests.sh</expected-message>
</test-case>

<test-case id="valid-test-steps" should-fail="false">
  <description>Valid lifecycle.test.steps configuration including extra step</description>
  <input language="json"><![CDATA[
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
  ]]></input>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule ignores non service.json files</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "steps": []
    }
  }
}
  ]]></input>
</test-case>

<test-case id="valid-run-tests-variant" should-fail="false">
  <description>Accepts commands that prepare the environment before invoking the shared runner</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "steps": [
        {
          "name": "run-tests",
          "run": "cd test && export API_PORT=${API_PORT:-18000} && ./run-tests.sh --all",
          "description": "Delegate testing to shared runner with environment bootstrap"
        }
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="incorrect-runner-path" should-fail="true">
  <description>Reject runner invocations that do not target the shared orchestrator</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "steps": [
        {
          "name": "run-tests",
          "run": "scripts/run-tests.sh",
          "description": "Incorrect runner location"
        }
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>test/run-tests.sh</expected-message>
</test-case>

<test-case id="non-executing-reference" should-fail="true">
  <description>Reject commands that only mention the runner without executing it</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "test": {
      "steps": [
        {
          "name": "noop",
          "run": "echo 'test/run-tests.sh would run'",
          "description": "Mentions runner without executing it"
        }
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>test/run-tests.sh</expected-message>
</test-case>
*/

// CheckLifecycleTestSteps validates that lifecycle.test.steps includes a step invoking the shared test/run-tests.sh runner.
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
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test.steps cannot be empty; add a step that invokes test/run-tests.sh")}
	}

	hasSharedRunner := false
	for _, entry := range steps {
		stepMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		if usesSharedTestRunner(stepMap) {
			hasSharedRunner = true
			break
		}
	}

	if !hasSharedRunner {
		line := findJSONLine(string(content), "\"steps\"")
		return []Violation{newTestStepsViolation(filePath, line, "service.json lifecycle.test.steps must include a step that invokes test/run-tests.sh")}
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
		Recommendation: "Ensure lifecycle.test.steps invokes the shared test/run-tests.sh runner",
		Standard:       "configuration-v1",
	}
}

func usesSharedTestRunner(step map[string]interface{}) bool {
	runValue, ok := step["run"].(string)
	if !ok {
		return false
	}
	return referencesSharedTestRunner(runValue)
}

func referencesSharedTestRunner(command string) bool {
	segments := splitCommandSegments(command)
	for _, segment := range segments {
		if executesSharedTestRunner(segment) {
			return true
		}
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

func splitCommandSegments(command string) []string {
	normalized := strings.ReplaceAll(command, "\r", "\n")
	replacer := strings.NewReplacer("&&", "\n", "||", "\n", ";", "\n", "|", "\n")
	normalized = replacer.Replace(normalized)
	return strings.Split(normalized, "\n")
}

func executesSharedTestRunner(segment string) bool {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return false
	}

	tokens := strings.Fields(segment)
	if len(tokens) == 0 {
		return false
	}

	idx := 0
	for idx < len(tokens) && isEnvAssignment(tokens[idx]) {
		idx++
	}
	if idx >= len(tokens) {
		return false
	}

	token := tokens[idx]

	wrappers := map[string]struct{}{
		"bash": {},
		"sh":   {},
		"sudo": {},
	}

	if _, ok := wrappers[strings.ToLower(token)]; ok {
		idx++
		if idx >= len(tokens) {
			return false
		}
		token = tokens[idx]
	}

	if strings.EqualFold(token, "env") {
		idx++
		for idx < len(tokens) && isEnvAssignment(tokens[idx]) {
			idx++
		}
		if idx >= len(tokens) {
			return false
		}
		token = tokens[idx]
	}

	normalized := strings.ToLower(strings.ReplaceAll(token, "\\", "/"))
	if normalized == "test/run-tests.sh" || normalized == "./test/run-tests.sh" || normalized == "./run-tests.sh" || normalized == "run-tests.sh" {
		return true
	}

	if strings.HasSuffix(normalized, "/test/run-tests.sh") {
		return true
	}

	return false
}

func isEnvAssignment(token string) bool {
	if !strings.Contains(token, "=") {
		return false
	}
	parts := strings.SplitN(token, "=", 2)
	if len(parts) != 2 {
		return false
	}
	name := parts[0]
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
