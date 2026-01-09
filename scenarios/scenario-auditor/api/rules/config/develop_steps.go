package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	developDefaultRecommendation     = "Provide start-api, start-ui (when UI_PORT is defined), and show-urls steps in lifecycle.develop so scenarios restart predictably."
	startUIRunRecommendation         = "Serve the built ui/dist bundle (node server.(js|cjs) or static file server) per PRODUCTION_BUNDLES.md so lifecycle restarts rebuild stale assets."
	startUIBackgroundRecommendation  = "Set start-ui background: true so develop can continue to show-urls and agents regain their shell."
	startAPIBackgroundRecommendation = "Set start-api background: true so lifecycle can inject resource environment variables (DATABASE_URL, REDIS_HOST, etc.), monitor the process via health checks, and return shell control to agents for show-urls and subsequent steps. Foreground processes block the lifecycle and prevent orchestration."
	developDefaultUIBundlePath       = "ui/dist/index.html"
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
  <expected-message>start-api step must execute "cd api && ./demo-api" or "./api/demo-api"</expected-message>
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
          "run": "cd ui && NODE_ENV=production node server.js",
          "background": true,
          "description": "Serve production UI bundle",
          "condition": {"file_exists": "ui/dist/index.html"}
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

<test-case id="start-api-direct-binary" should-fail="false">
  <description>start-api can execute the binary directly via ./api/&lt;name&gt;</description>
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
          "run": "ENV=dev ./api/demo-api",
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
          "run": "cd ui && NODE_ENV=production node server.js",
          "background": true,
          "condition": {"file_exists": "ui/dist/index.html"}
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

<test-case id="start-ui-dev-server" should-fail="true">
  <description>start-ui cannot launch Vite/npm dev servers under production bundle requirements</description>
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
          "name": "start-ui",
          "run": "cd ui && npm run dev",
          "background": true,
          "condition": {"file_exists": "ui/package.json"}
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
  <expected-violations>2</expected-violations>
  <expected-message>built dist bundle</expected-message>
</test-case>

<test-case id="start-ui-not-backgrounded" should-fail="true">
  <description>start-ui must run in the background so develop can continue</description>
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
          "description": "Start Go API server",
          "background": true,
          "condition": {"file_exists": "api/demo-api"}
        },
        {
          "name": "start-ui",
          "run": "cd ui && node server.js",
          "description": "Serve production UI bundle",
          "condition": {"file_exists": "ui/dist/index.html"}
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
  <expected-violations>1</expected-violations>
  <expected-message>background</expected-message>
</test-case>

<test-case id="start-ui-no-condition" should-fail="true">
  <description>start-ui must include a file_exists condition that tracks ui/dist/index.html</description>
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
          "name": "start-ui",
          "run": "cd ui && node server.js",
          "description": "Serve production bundle",
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
  <expected-violations>1</expected-violations>
  <expected-message>ui/dist</expected-message>
</test-case>

<test-case id="start-ui-missing-bundle-check" should-fail="true">
  <description>start-ui must guard against stale bundles by checking ui/dist/index.html</description>
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
          "name": "start-ui",
          "run": "cd ui && NODE_ENV=production node server.js",
          "background": true,
          "condition": {"file_exists": "ui/package.json"}
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
  <expected-violations>1</expected-violations>
  <expected-message>ui/dist/index.html</expected-message>
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

	var payload map[string]any
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

	stepsSlice, ok := stepsRaw.([]any)
	if !ok || len(stepsSlice) == 0 {
		line := findJSONLineDevelop(source, "\"develop\"", "\"steps\"")
		return []Violation{newDevelopViolation(filePath, line, "lifecycle.develop.steps must include at least the start and show-urls commands")}
	}

	scenarioName := strings.TrimSpace(getScenarioName(payload))
	bundlePath := resolveUIBundlePath(payload)
	var violations []Violation
	developLine := findJSONLineDevelop(source, "\"develop\"")
	stepsLine := findJSONLineDevelop(source, "\"develop\"", "\"steps\"")

	if requireAPIStart(payload) {
		startAPIStep, _ := findStepByName(stepsSlice, "start-api")
		if startAPIStep == nil {
			line := contextualLine(findJSONLineDevelop(source, "\"start-api\""), stepsLine, developLine)
			msg := "Develop steps must include a \"start-api\" command so lifecycle restarts can launch the Go API server with injected resource environment variables (DATABASE_URL, REDIS_HOST, etc.) and monitor it via health checks. Without start-api, operators must manually start the API, orchestration tools can't detect the API endpoint, and resource connections may fail."
			violations = append(violations, newDevelopViolation(filePath, line, msg))
		} else {
			runVal := strings.TrimSpace(getString(startAPIStep, "run"))
			lineRun := findJSONLineDevelop(source, "\"start-api\"", "\"run\"")

			binaryName := extractBinaryName(runVal)
			expectedBinary := ""
			if scenarioName != "" {
				expectedBinary = fmt.Sprintf("%s-api", scenarioName)
			}

			if !isAllowedAPIRun(runVal, expectedBinary, binaryName) {
				violations = append(violations, newDevelopViolation(filePath, lineRun, buildAPIStartMessage(expectedBinary, scenarioName), buildAPIRunRecommendation(expectedBinary, binaryName)))
			}

			if desc := strings.TrimSpace(getString(startAPIStep, "description")); desc == "" {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"description\"")
				msg := "start-api description must be provided so operators and agents understand this step launches the Go API server that lifecycle monitors via health checks. Clear descriptions help agents troubleshoot startup failures and understand which services are running."
				violations = append(violations, newDevelopViolation(filePath, line, msg))
			}

			if bg, ok := startAPIStep["background"].(bool); !ok || !bg {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"background\"")
				msg := "start-api must run in the background so lifecycle can inject resource environment variables, monitor the process via health checks, and return shell control to agents for show-urls and subsequent steps. Foreground processes block the lifecycle, prevent orchestration, and cause agents to hang waiting for shell access."
				violations = append(violations, newDevelopViolation(filePath, line, msg, startAPIBackgroundRecommendation))
			}

			if cond, ok := startAPIStep["condition"].(map[string]any); ok {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"file_exists\"")
				fileExists := strings.TrimSpace(getString(cond, "file_exists"))
				if fileExists == "" {
					violations = append(violations, newDevelopViolation(filePath, line, "start-api condition.file_exists must reference the API binary", buildAPIConditionRecommendation(expectedBinary, binaryName)))
				} else if binaryName != "" && !strings.Contains(fileExists, binaryName) {
					expectedPath := fmt.Sprintf("api/%s", binaryName)
					violations = append(violations, newDevelopViolation(filePath, line, fmt.Sprintf("start-api condition.file_exists should reference %q", expectedPath), buildAPIConditionRecommendation(expectedBinary, binaryName)))
				}
			} else {
				line := findJSONLineDevelop(source, "\"start-api\"", "\"condition\"")
				violations = append(violations, newDevelopViolation(filePath, line, "start-api must include a file_exists condition for the API binary", buildAPIConditionRecommendation(expectedBinary, binaryName)))
			}
		}
	}

	if requireUIStart(payload) {
		startUIStep, _ := findStepByName(stepsSlice, "start-ui")
		if startUIStep == nil {
			line := contextualLine(findJSONLineDevelop(source, "\"start-ui\""), stepsLine, developLine)
			msg := "Develop steps must include a \"start-ui\" command when the UI is enabled so lifecycle restarts can serve the production bundle from ui/dist. Without start-ui, operators must manually start the UI, orchestration tools like app-monitor can't detect the UI endpoint, and users can't access the web interface. Verify that lifecycle.setup includes a build-ui step and that setup.condition includes a ui-bundle check pointing to the same bundle path."
			violations = append(violations, newDevelopViolation(filePath, line, msg))
		} else {
			if bg, ok := startUIStep["background"].(bool); !ok || !bg {
				line := findJSONLineDevelop(source, "\"start-ui\"", "\"background\"")
				msg := "start-ui must run in the background so develop can continue to show-urls and agents regain their shell. Foreground UI servers block the lifecycle from completing, prevent show-urls from displaying connection info, and cause automation to hang waiting for shell access."
				violations = append(violations, newDevelopViolation(filePath, line, msg, startUIBackgroundRecommendation))
			}

			runVal := strings.TrimSpace(getString(startUIStep, "run"))
			lineRun := findJSONLineDevelop(source, "\"start-ui\"", "\"run\"")
			if runVal == "" {
				msg := "start-ui run command must launch the production bundle server so lifecycle restarts serve the built ui/dist assets instead of running dev servers that bypass staleness detection. Production servers ensure cache-busting works, iframes load correctly, and restart automation rebuilds when ui/src changes per docs/scenarios/PRODUCTION_BUNDLES.md."
				violations = append(violations, newDevelopViolation(filePath, lineRun, msg, startUIRunRecommendation))
			} else {
				if usesDevServerCommand(runVal) {
					msg := "start-ui must serve the built dist bundle (node server.js/Express) instead of npm run dev per PRODUCTION_BUNDLES.md. Dev servers bypass lifecycle staleness detection, preventing auto-rebuild on restart. This causes stale UI code to persist after ui/src changes, breaks cache-busting in production-like environments, and makes iframe embedding unreliable. Replace with 'node server.js' that serves ui/dist and verify setup.condition includes a ui-bundle check."
					violations = append(violations, newDevelopViolation(filePath, lineRun, msg, startUIRunRecommendation))
				} else if !isProductionUIServerRun(runVal, bundlePath) {
					msg := "start-ui must launch a production bundle server (node server.(js|cjs) or equivalent static asset server) so restarts detect stale ui/dist builds and trigger setup to rebuild when ui/src changes. Without production serving, lifecycle can't correlate bundle staleness with source changes, auto-rebuild fails, and users see outdated UI code after git pull or agent fixes."
					violations = append(violations, newDevelopViolation(filePath, lineRun, msg, startUIRunRecommendation))
				}
			}

			condMap, hasCondition := startUIStep["condition"].(map[string]any)
			fileExistsLine := findJSONLineDevelop(source, "\"start-ui\"", "\"file_exists\"")
			if !hasCondition {
				message := fmt.Sprintf("start-ui must include a file_exists condition that tracks %s for bundle staleness", bundlePathOrDefault(bundlePath))
				violations = append(violations, newDevelopViolation(filePath, fileExistsLine, message, buildUIConditionRecommendation(bundlePath)))
			} else {
				bundleCheck := strings.TrimSpace(getString(condMap, "file_exists"))
				if bundleCheck == "" || !referencesBundleArtifact(bundleCheck, bundlePath) {
					message := fmt.Sprintf("start-ui condition.file_exists must point at %s so lifecycle restarts rebuild stale bundles", bundlePathOrDefault(bundlePath))
					violations = append(violations, newDevelopViolation(filePath, fileExistsLine, message, buildUIConditionRecommendation(bundlePath)))
				}
			}
		}
	}

	showStep, showIndex := findStepByName(stepsSlice, "show-urls")
	if showStep == nil {
		line := contextualLine(findJSONLineDevelop(source, "\"show-urls\""), stepsLine, developLine)
		msg := "Develop steps must include a \"show-urls\" command so operators and agents know which URLs to access after lifecycle completes. Without show-urls, users don't know the API_PORT or UI_PORT values, making it impossible to open the correct endpoints or verify the scenario is running correctly."
		violations = append(violations, newDevelopViolation(filePath, line, msg))
	} else if showIndex != len(stepsSlice)-1 {
		line := findJSONLineDevelop(source, "\"show-urls\"")
		msg := "show-urls step must be the final develop step so backgrounded services complete before displaying connection info. Showing URLs before services start causes agents and operators to attempt connections while health checks are still pending, leading to 'connection refused' errors and false failure reports."
		violations = append(violations, newDevelopViolation(filePath, line, msg))
	}

	return deduplicateDevelopViolations(violations)
}

func newDevelopViolation(filePath string, line int, message string, recommendation ...string) Violation {
	if line <= 0 {
		line = 1
	}
	rec := developDefaultRecommendation
	if len(recommendation) > 0 {
		if custom := strings.TrimSpace(recommendation[0]); custom != "" {
			rec = custom
		}
	}
	return Violation{
		Type:           "config_develop_steps",
		Severity:       "medium",
		Title:          "Develop lifecycle configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: rec,
		Standard:       "configuration-v1",
	}
}

func getMap(root map[string]any, key string) (map[string]any, bool) {
	if root == nil {
		return nil, false
	}
	raw, ok := root[key]
	if !ok {
		return nil, false
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	return m, true
}

func getString(root map[string]any, key string) string {
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

func getScenarioName(payload map[string]any) string {
	serviceMap, _ := getMap(payload, "service")
	if serviceMap == nil {
		return ""
	}
	return getString(serviceMap, "name")
}

func findStepByName(steps []any, name string) (map[string]any, int) {
	for idx, stepRaw := range steps {
		if stepMap, ok := stepRaw.(map[string]any); ok {
			if strings.TrimSpace(getString(stepMap, "name")) == name {
				return stepMap, idx
			}
		}
	}
	return nil, -1
}

func requireAPIStart(payload map[string]any) bool {
	enabled, present := componentEnabled(payload, "api")
	if present {
		return enabled
	}
	return hasPort(payload, "api")
}

func requireUIStart(payload map[string]any) bool {
	enabled, present := componentEnabled(payload, "ui")
	if present {
		return enabled
	}
	return hasPort(payload, "ui")
}

func resolveUIBundlePath(payload map[string]any) string {
	lifecycle, _ := getMap(payload, "lifecycle")
	setup, _ := getMap(lifecycle, "setup")
	condition, _ := getMap(setup, "condition")
	if condition != nil {
		if checksRaw, ok := condition["checks"]; ok {
			if checks, ok := checksRaw.([]any); ok {
				for _, entry := range checks {
					check, ok := entry.(map[string]any)
					if !ok {
						continue
					}
					if strings.EqualFold(strings.TrimSpace(getString(check, "type")), "ui-bundle") {
						if path := strings.TrimSpace(getString(check, "bundle_path")); path != "" {
							return path
						}
					}
				}
			}
		}
		if path := strings.TrimSpace(getString(condition, "bundle_path")); path != "" {
			return path
		}
	}
	return developDefaultUIBundlePath
}

func componentEnabled(payload map[string]any, name string) (bool, bool) {
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

func hasPort(payload map[string]any, name string) bool {
	ports, _ := getMap(payload, "ports")
	if ports == nil {
		return false
	}
	if entry, ok := ports[name]; ok {
		_, ok := entry.(map[string]any)
		return ok
	}
	return false
}

func extractBinaryName(run string) string {
	run = strings.TrimSpace(run)
	if run == "" {
		return ""
	}

	relevant := run
	if idx := strings.LastIndex(run, "cd api"); idx != -1 {
		relevant = run[idx:]
	}

	if last := strings.LastIndex(relevant, "./"); last != -1 {
		candidate := relevant[last+2:]
		candidate = trimCommandToken(candidate)
		if slash := strings.LastIndex(candidate, "/"); slash != -1 {
			candidate = candidate[slash+1:]
		}
		candidate = strings.Trim(candidate, `"'`)
		if isBinaryToken(candidate) {
			return candidate
		}
	}

	re := regexp.MustCompile(`\./([A-Za-z0-9._-]+)`)
	matches := re.FindAllStringSubmatch(relevant, -1)
	if len(matches) > 0 {
		candidate := matches[len(matches)-1][1]
		if candidate != "" {
			return candidate
		}
	}

	return ""
}

func trimCommandToken(value string) string {
	end := len(value)
	for i, r := range value {
		switch r {
		case ' ', '\t', '\n', '\r', '&', '|', ';':
			end = i
			return value[:end]
		}
	}
	return value
}

func isBinaryToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if r == '.' || r == '-' || r == '_' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		return false
	}
	return true
}

func isAllowedAPIRun(runValue, expectedBinary, binaryName string) bool {
	trimmed := strings.TrimSpace(runValue)
	if trimmed == "" {
		return false
	}

	hasCdApi := strings.Contains(trimmed, "cd api")

	if expectedBinary == "" {
		if binaryName == "" {
			return false
		}
		if hasCdApi {
			return true
		}
		return strings.Contains(trimmed, "./api/"+binaryName)
	}

	if hasCdApi && binaryName == expectedBinary {
		return true
	}

	directInvocation := fmt.Sprintf("./api/%s", expectedBinary)
	if strings.Contains(trimmed, directInvocation) {
		return true
	}

	return false
}

func usesDevServerCommand(run string) bool {
	lower := strings.ToLower(run)
	devTokens := []string{
		"npm run dev",
		"pnpm dev",
		"pnpm run dev",
		"yarn dev",
		"bun dev",
		"vite dev",
		"npx vite",
		"next dev",
		"nuxt dev",
		"react-scripts start",
	}
	for _, token := range devTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func isProductionUIServerRun(run string, bundlePath string) bool {
	lower := strings.ToLower(run)
	if strings.TrimSpace(lower) == "" {
		return false
	}
	allowed := []string{
		"node server.js",
		"node ./server.js",
		"node server.cjs",
		"node ./server.cjs",
		"node server.mjs",
		"node ./server.mjs",
		"serve -s",
		"npx serve",
		"npm run preview",
		"npm run start:prod",
		"npm run start:production",
		"npm run serve:prod",
	}
	for _, token := range allowed {
		if strings.Contains(lower, token) {
			return true
		}
	}
	staticServerTokens := []string{
		"python3 -m http.server",
		"python -m http.server",
		"npx http-server",
		"http-server",
		"bunx serve",
		"bun run serve",
		"go run",
		"static-web-server",
		"deno run",
	}
	if bundlePath == "" {
		bundlePath = developDefaultUIBundlePath
	}
	normalizedBundle := normalizeSlashes(strings.ToLower(bundlePath))
	bundleDir := normalizedBundle
	if strings.HasSuffix(bundleDir, "index.html") {
		bundleDir = strings.TrimSuffix(bundleDir, "index.html")
	}
	bundleDir = strings.TrimRight(bundleDir, "/")
	for _, token := range staticServerTokens {
		if strings.Contains(lower, token) {
			if normalizedBundle != "" && strings.Contains(lower, normalizedBundle) {
				return true
			}
			if bundleDir != "" && bundleDir != "." && strings.Contains(lower, bundleDir) {
				return true
			}
		}
	}
	if strings.Contains(lower, "node server") && strings.Contains(lower, "dist") {
		return true
	}
	return false
}

func referencesBundleArtifact(path string, bundlePath string) bool {
	candidate := normalizeSlashes(strings.ToLower(strings.TrimSpace(path)))
	if candidate == "" {
		return false
	}
	expected := normalizeSlashes(strings.ToLower(strings.TrimSpace(bundlePath)))
	if expected == "" {
		expected = normalizeSlashes(strings.ToLower(developDefaultUIBundlePath))
	}
	if strings.Contains(candidate, expected) {
		return true
	}
	if dir := strings.TrimRight(filepath.Dir(expected), "/"); dir != "." && dir != "" {
		if strings.Contains(candidate, dir) {
			return true
		}
	}
	if strings.Contains(candidate, "ui/dist") {
		return true
	}
	if strings.Contains(candidate, "/dist/") && (strings.HasSuffix(candidate, ".html") || strings.Contains(candidate, "index")) {
		return true
	}
	return false
}

func buildAPIStartMessage(expectedBinary, scenarioName string) string {
	binary := expectedBinary
	if binary == "" {
		binary = "<scenario>-api"
	}
	nameSuffix := "the scenario"
	if scenarioName != "" {
		nameSuffix = fmt.Sprintf("%s", scenarioName)
	}
	return fmt.Sprintf(
		"start-api step must execute \"cd api && ./%[1]s\" or \"./api/%[1]s\" so the lifecycle can restart %[2]s with the same binary, inject resource env vars, and detect stale builds. Launching the API through wrapper scripts or alternate names bypasses that lifecycle logic.",
		binary,
		nameSuffix,
	)
}

func buildAPIRunRecommendation(expectedBinary, binaryName string) string {
	binary := firstNonEmpty(expectedBinary, binaryName, "<scenario>-api")
	return fmt.Sprintf("Execute ./api/%[1]s (or cd api && ./%[1]s) directly so restart detection, resource injection, and health checks continue to work.", binary)
}

func buildAPIConditionRecommendation(expectedBinary, binaryName string) string {
	binary := firstNonEmpty(binaryName, expectedBinary, "<scenario>-api")
	return fmt.Sprintf("Set start-api condition.file_exists to 'api/%s' so lifecycle detects when the Go sources require a rebuild.", binary)
}

func buildUIConditionRecommendation(bundlePath string) string {
	expected := bundlePathOrDefault(bundlePath)
	return fmt.Sprintf("Point start-ui condition.file_exists to %s so restarts know when to rebuild the UI bundle.", expected)
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func bundlePathOrDefault(bundlePath string) string {
	if strings.TrimSpace(bundlePath) == "" {
		return developDefaultUIBundlePath
	}
	return bundlePath
}

func normalizeSlashes(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

func contextualLine(values ...int) int {
	for _, line := range values {
		if line > 1 {
			return line
		}
	}
	return 1
}
