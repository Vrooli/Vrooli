package config

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
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {}
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.setup.condition</expected-message>
</test-case>

<test-case id="missing-binary-check" should-fail="true">
  <description>First check is not binaries with <scenario>-api target</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "cli",
            "command": "scenario-auditor",
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>binaries</expected-message>
</test-case>

<test-case id="missing-cli-check" should-fail="true">
  <description>Second check is not CLI with scenario target</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>CLI check</expected-message>
</test-case>

<test-case id="incorrect-targets" should-fail="true">
  <description>Binaries check must reference api/<scenario>-api and CLI command must be present</description>
  <input language="json"><![CDATA[
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
  ]]></input>
  <expected-violations>2</expected-violations>
  <expected-message>CLI check</expected-message>
</test-case>

<test-case id="missing-service-name" should-fail="true">
  <description>condition defined but service.name missing</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {"type": "binaries", "targets": ["api/scenario-auditor-api"]},
          {"type": "cli", "command": "scenario-auditor", "targets": ["scenario-auditor"]}
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>service.name</expected-message>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule skips files outside of service.json</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"}
}
  ]]></input>
</test-case>

<test-case id="valid-condition" should-fail="false">
  <description>Valid setup condition with required binaries and CLI checks</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api", "api/extra-api"]
          },
          {
            "type": "ui-bundle",
            "bundle_path": "ui/dist/index.html",
            "source_dir": "ui/src"
          },
          {
            "type": "cli",
            "command": "scenario-auditor",
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
  ]]></input>
</test-case>

<test-case id="valid-with-ui-bundle" should-fail="false">
  <description>Valid setup condition that includes ui-bundle between binaries and CLI</description>
  <input language="json"><![CDATA[
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
            "type": "ui-bundle",
            "bundle_path": "ui/dist/index.html",
            "source_dir": "ui/src"
          },
          {
            "type": "cli",
            "command": "scenario-auditor",
            "targets": ["scenario-auditor"]
          }
        ]
      }
    }
  }
}
  ]]></input>
</test-case>

<test-case id="valid-cli-command" should-fail="false">
  <description>CLI readiness can be expressed via command name</description>
  <input language="json"><![CDATA[
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
            "type": "cli",
            "command": "scenario-auditor"
          }
        ]
      }
    }
  }
}
  ]]></input>
</test-case>

<test-case id="invalid-cli-command" should-fail="true">
  <description>CLI check without a command should fail because runtime cannot validate it</description>
  <input language="json"><![CDATA[
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
            "type": "ui-bundle",
            "bundle_path": "ui/dist/index.html",
            "source_dir": "ui/src"
          },
          {
            "type": "cli",
            "targets": ["scenario-auditor"]
          }
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>CLI check</expected-message>
</test-case>

<test-case id="cli-command-mismatch" should-fail="true">
  <description>CLI command that does not match service.name should be rejected</description>
  <input language="json"><![CDATA[
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
            "type": "cli",
            "command": "auditor"
          }
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>CLI check</expected-message>
</test-case>

<test-case id="ui-bundle-required" should-fail="true">
  <description>Scenarios with UI port must declare a ui-bundle check</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "ports": {
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api"]
          },
          {
            "type": "cli",
            "command": "scenario-auditor"
          }
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ui-bundle</expected-message>
</test-case>

<test-case id="web-port-detected" should-fail="true">
  <description>Ports labelled web should also require a ui-bundle check</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "ports": {
    "web": {"env_var": "WEB_PORT"}
  },
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api"]
          },
          {
            "type": "cli",
            "command": "scenario-auditor"
          }
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ui-bundle</expected-message>
</test-case>

<test-case id="ui-bundle-missing-paths" should-fail="true">
  <description>ui-bundle check must include bundle_path and source_dir fields</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "ports": {
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api"]
          },
          {
            "type": "ui-bundle"
          },
          {
            "type": "cli",
            "command": "scenario-auditor"
          }
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>bundle_path</expected-message>
</test-case>

<test-case id="ui-bundle-canonical-paths" should-fail="true">
  <description>ui-bundle must point at ui/dist/index.html and ui/src for rebuild detection</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "scenario-auditor"},
  "ports": {
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/scenario-auditor-api"]
          },
          {
            "type": "ui-bundle",
            "bundle_path": "ui/build/index.html",
            "source_dir": "ui/app"
          },
          {
            "type": "cli",
            "command": "scenario-auditor"
          }
        ]
      }
    }
  }
}
  ]]></input>
  <expected-violations>2</expected-violations>
  <expected-message>ui-bundle</expected-message>
</test-case>
*/

// CheckServiceSetupConditions validates lifecycle.setup.condition structure for service.json.
func CheckServiceSetupConditions(content []byte, filePath string) []Violation {
	if !shouldValidateServiceJSON(filePath) {
		return nil
	}

	var payload map[string]any
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

	checks, ok := checksVal.([]any)
	if !ok || len(checks) == 0 {
		return []Violation{newSetupViolation(filePath, findJSONLineSetup(string(content), "\"checks\""), "lifecycle.setup.condition.checks must be an array")}
	}

	var violations []Violation

	// Validate first check: binaries with scenario-api target
	firstCheck, firstOK := checks[0].(map[string]any)
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
			violations = append(violations, newSetupViolation(
				filePath,
				lineTargets,
				fmt.Sprintf("Binaries check must reference the canonical path '%s'. Update every other binary reference (build steps, develop steps, health checks) to use the same api/<service>-api naming.", expectedBinary),
			))
		}
	}

	// Validate CLI readiness checks: ensure a CLI check exists with a command (runtime uses the command field)
	cliChecks := extractChecksByType(checks, "cli")
	if len(cliChecks) == 0 {
		line := findJSONLineSetup(string(content), "\"checks\"", "\"cli\"")
		message := fmt.Sprintf("lifecycle.setup.condition must include a CLI check whose command is exactly '%s'. The executable lives under cli/%s, but readiness resolves commands via PATH so the check must use the bare service name.", serviceName, serviceName)
		violations = append(violations, newSetupViolation(filePath, line, message))
	} else {
		missingCommand := true
		matchingCommand := false
		for _, cliCheck := range cliChecks {
			command := strings.TrimSpace(extractCommand(cliCheck))
			if command == "" {
				continue
			}
			missingCommand = false
			if commandMatches(cliCheck, serviceName) {
				matchingCommand = true
				break
			}
		}
		if missingCommand || !matchingCommand {
			line := findJSONLineSetup(string(content), "\"checks\"", "\"cli\"")
			msg := fmt.Sprintf("CLI check must set command '%s'. Keep the executable under cli/%s, but this readiness probe runs 'command -v %s' so the entry must be the bare service name, not a filesystem path.", serviceName, serviceName, serviceName)
			if !missingCommand {
				msg = fmt.Sprintf("CLI check command must exactly match '%s' so command -v can find it on PATH (place the binary under cli/%s but expose it via that name).", serviceName, serviceName)
			}
			violations = append(violations, newSetupViolation(filePath, line, msg))
		}
	}

	// Enforce ui-bundle checks for scenarios with a UI surface
	if scenarioHasUI(payload) {
		bundleChecks := extractChecksByType(checks, "ui-bundle")
		if len(bundleChecks) == 0 {
			line := findJSONLineSetup(string(content), "\"checks\"", "\"ui-bundle\"")
			violations = append(violations, newSetupViolation(filePath, line, "UI scenarios must include a ui-bundle check with bundle_path and source_dir"))
		} else {
			for _, bundleCheck := range bundleChecks {
				bundlePath := strings.TrimSpace(stringField(bundleCheck, "bundle_path"))
				sourceDir := strings.TrimSpace(stringField(bundleCheck, "source_dir"))
				if bundlePath == "" || sourceDir == "" {
					line := findJSONLineSetup(string(content), "\"ui-bundle\"", "\"bundle_path\"", "\"source_dir\"")
					violations = append(violations, newSetupViolation(filePath, line, "ui-bundle check must include bundle_path and source_dir"))
					continue
				}

				if !isCanonicalBundlePath(bundlePath) {
					line := findJSONLineSetup(string(content), "\"ui-bundle\"", "\"bundle_path\"")
					violations = append(violations, newSetupViolation(filePath, line, "ui-bundle bundle_path must point to the production artifact (ui/dist/index.html) per PRODUCTION_BUNDLES.md"))
				}
				if !isCanonicalSourceDir(sourceDir) {
					line := findJSONLineSetup(string(content), "\"ui-bundle\"", "\"source_dir\"")
					violations = append(violations, newSetupViolation(filePath, line, "ui-bundle source_dir must reference the UI source directory (ui/src) so staleness detection works"))
				}
			}
		}
	}

	return dedupeSetupConditionViolations(violations)
}

func extractChecksByType(checks []any, checkType string) []map[string]any {
	var result []map[string]any
	for _, raw := range checks {
		check, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if valueEquals(check, "type", checkType) {
			result = append(result, check)
		}
	}
	return result
}

func extractServiceName(payload map[string]any) (string, error) {
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

func getObject(parent map[string]any, key string) (map[string]any, bool) {
	raw, ok := parent[key]
	if !ok {
		return nil, false
	}
	obj, ok := raw.(map[string]any)
	return obj, ok
}

func valueEquals(obj map[string]any, key, expected string) bool {
	value, ok := obj[key].(string)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), expected)
}

func containsTarget(check map[string]any, target string) bool {
	raw, ok := check["targets"]
	if !ok {
		return false
	}
	list, ok := raw.([]any)
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

func commandMatches(check map[string]any, serviceName string) bool {
	cmd, ok := check["command"].(string)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cmd), strings.TrimSpace(serviceName))
}

func extractCommand(check map[string]any) string {
	value, _ := check["command"].(string)
	return value
}

func stringField(check map[string]any, key string) string {
	if value, ok := check[key].(string); ok {
		return value
	}
	return ""
}

func scenarioHasUI(payload map[string]any) bool {
	if portsRaw, ok := payload["ports"]; ok {
		if ports, ok := portsRaw.(map[string]any); ok {
			for key := range ports {
				if isUIPortKey(key) {
					return true
				}
			}
		}
	}

	lifecycle, ok := getObject(payload, "lifecycle")
	if !ok {
		return false
	}
	develop, ok := getObject(lifecycle, "develop")
	if !ok {
		return false
	}
	stepsRaw, exists := develop["steps"]
	if !exists {
		return false
	}
	steps, ok := stepsRaw.([]any)
	if !ok {
		return false
	}
	for _, stepRaw := range steps {
		step, ok := stepRaw.(map[string]any)
		if !ok {
			continue
		}
		if value, ok := step["name"].(string); ok && uiIndicator(value) {
			return true
		}
		if value, ok := step["run"].(string); ok && uiIndicator(value) {
			return true
		}
	}
	return false
}

func uiIndicator(value string) bool {
	if value == "" {
		return false
	}
	v := strings.ToLower(value)
	tokens := []string{
		"start-ui",
		"build-ui",
		"-ui",
		"ui-",
		" ui",
		"ui ",
		"ui/",
		"cd ui",
		"ui_port",
		"ui-port",
		"ui server",
		"start-web",
		"web-server",
		"start-dashboard",
		"frontend",
		"start-frontend",
		"web ui",
		"webapp",
		"web-app",
		"spa",
	}
	for _, token := range tokens {
		if strings.Contains(v, token) {
			return true
		}
	}
	return false
}

func isUIPortKey(key string) bool {
	if key == "" {
		return false
	}
	key = strings.ToLower(strings.TrimSpace(key))
	tokens := []string{"ui", "web", "dashboard", "frontend", "spa", "client"}
	for _, token := range tokens {
		if strings.Contains(key, token) {
			return true
		}
	}
	return false
}

func isCanonicalBundlePath(path string) bool {
	if path == "" {
		return false
	}
	canonical := "ui/dist/index.html"
	if strings.EqualFold(path, canonical) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(path), "ui/dist/") && strings.HasSuffix(strings.ToLower(path), ".html")
}

func isCanonicalSourceDir(dir string) bool {
	if dir == "" {
		return false
	}
	canonical := "ui/src"
	if strings.EqualFold(dir, canonical) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(dir), canonical)
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
		Recommendation: "Configure lifecycle.setup.condition with binaries, CLI commands, and ui-bundle checks that match the scenario",
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

func dedupeSetupConditionViolations(list []Violation) []Violation {
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
