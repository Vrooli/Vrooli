package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	installCLIRecommendation    = "Add an install-cli step (e.g., \"cd cli && ./install.sh\") so lifecycle.setup.condition's CLI check and CLI users can find the scenario binary before develop runs."
	installUIDepsRecommendation = "Add install-ui-deps (npm|pnpm|yarn|bun install inside ui/) before build-ui so PRODUCTION_BUNDLES staleness detection can recreate ui/dist when ui/src changes."
	buildUIRecommendation       = "Run build-ui (npm|pnpm|yarn|bun build inside ui/) before show-urls so develop serves the ui/dist production bundle per docs/scenarios/PRODUCTION_BUNDLES.md."
)

type stepMatch struct {
	step          map[string]any
	index         int
	name          string
	matchedByName bool
}

func newStepMatch(step map[string]any, index int, name string, matched bool) *stepMatch {
	return &stepMatch{step: step, index: index, name: name, matchedByName: matched}
}

func buildAPIRecommendation(serviceName string) string {
	return fmt.Sprintf("Emit %s-api directly inside the api directory so lifecycle.setup.condition binaries checks and start-api reuse the same binary.", serviceName)
}

/*
Rule: Setup Steps Configuration
Description: Ensure lifecycle.setup.steps include consistent initialization tasks
Reason: Reliable setup steps prevent missing binaries and inconsistent developer environments
Category: config
Severity: medium
Standard: configuration-v1
Targets: service_json

<test-case id="missing-setup-steps" should-fail="true">
  <description>service.json without lifecycle.setup steps</description>
  <input language="json"><![CDATA[
{
  "service": {
    "name": "file-tools"
  },
  "lifecycle": {}
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.setup.steps</expected-message>
</test-case>

<test-case id="missing-install-cli" should-fail="true">
  <description>Setup steps missing the install-cli task</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>install-cli</expected-message>
</test-case>

<test-case id="mismatched-build-api" should-fail="true">
  <description>Build step does not output scenario-name-api binary</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go build", "description": "Build Go API server"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>build-api</expected-message>
</test-case>

<test-case id="show-urls-last" should-fail="true">
  <description>show-urls step is not last in the list</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "show-urls", "run": "echo urls"},
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>show-urls</expected-message>
</test-case>

<test-case id="install-cli-run-mismatch" should-fail="true">
  <description>install-cli step uses an unexpected run command</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>install-cli must run from cli/</expected-message>
</test-case>

<test-case id="install-cli-bash-script" should-fail="false">
  <description>install-cli step invokes cli/install.sh without changing directories</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "bash cli/install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="missing-service-name" should-fail="true">
  <description>install-cli exists but service.name is missing</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally", "condition": {"file_exists": "cli/install.sh"}},
        {"name": "build-api", "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go", "description": "Build Go API server", "condition": {"file_exists": "api/go.mod"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>service.name</expected-message>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule is skipped for files other than service.json</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"}
}
  ]]></input>
</test-case>

<test-case id="valid-setup" should-fail="false">
  <description>Setup steps include install-cli, scenario-specific build, and show-urls last</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "install-cli",
          "run": "cd cli && ./install.sh",
          "description": "Install CLI command globally",
          "condition": {"file_exists": "cli/install.sh"}
        },
        {
          "name": "build-api",
          "run": "cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go",
          "description": "Build Go API server",
          "condition": {"file_exists": "api/go.mod"}
        },
        {
          "name": "show-urls",
          "run": "echo 'UI: http://localhost:${UI_PORT}'"
        }
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="valid-build-api-dot" should-fail="false">
  <description>build-api allows building the scenario binary from current directory</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go mod tidy && go build -o file-tools-api .", "description": "Build Go API binary"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="valid-install-cli-symlink" should-fail="false">
  <description>install-cli can chmod and symlink the binary into PATH</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "install-cli",
          "run": "cd cli && chmod +x file-tools && ln -sf $(pwd)/file-tools ~/.local/bin/file-tools",
          "description": "Install CLI binary to PATH"
        },
        {
          "name": "build-api",
          "run": "cd api && go build -o file-tools-api ./cmd/server/main.go",
          "description": "Build Go API binary"
        },
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="install-cli-symlink-without-cd" should-fail="false">
  <description>install-cli may operate directly on cli/<service> paths without changing directories</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "core-debugger"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "install-cli",
          "run": "chmod +x cli/core-debugger && ln -sf $(pwd)/cli/core-debugger ~/.local/bin/core-debugger",
          "description": "Install CLI command globally"
        },
        {
          "name": "build-api",
          "run": "cd api && go mod download && go build -o core-debugger-api .",
          "description": "Build Go API binary"
        },
        {
          "name": "install-ui-deps",
          "run": "cd ui && npm install",
          "description": "Install UI dependencies",
          "condition": {"file_exists": "ui/package.json"}
        },
        {
          "name": "build-ui",
          "run": "cd ui && npm run build",
          "description": "Build UI bundle",
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

<test-case id="missing-go-build" should-fail="true">
  <description>build-api step must run go build</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && npm run build", "description": "Build API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>go build</expected-message>
</test-case>

<test-case id="wrong-output-name" should-fail="true">
  <description>build-api must emit the scenario-specific binary name</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go build -o tools-api .", "description": "Build Go API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>output must be file-tools-api</expected-message>
</test-case>

<test-case id="build-api-nested-output" should-fail="true">
  <description>build-api must not prefix the output with api/ when already running inside the api directory</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go build -o api/file-tools-api .", "description": "Build Go API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>api directory</expected-message>
</test-case>

<test-case id="install-cli-no-install" should-fail="true">
  <description>install-cli must perform an installation action</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && echo 'Skipping install'", "description": "CLI setup"},
        {"name": "build-api", "run": "cd api && go build -o file-tools-api .", "description": "Build Go API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>install the CLI binary</expected-message>
</test-case>

<test-case id="install-cli-out-of-order" should-fail="true">
  <description>install-cli must complete before build-api and show-urls announce readiness</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "order-check"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go build -o order-check-api .", "description": "Build Go API"},
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>install-cli must run before build-api</expected-message>
</test-case>

<test-case id="missing-show-urls-step" should-fail="true">
  <description>Setup must include a final show-urls step</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI command globally"},
        {"name": "build-api", "run": "cd api && go build -o file-tools-api .", "description": "Build Go API"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>final show-urls step</expected-message>
</test-case>

<test-case id="ui-missing-build-step" should-fail="true">
  <description>UI scenarios must build production bundles before develop shows URLs</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "app-issue-tracker"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "build-api", "run": "cd api && go build -o app-issue-tracker-api .", "description": "Build API"},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>2</expected-violations>
  <expected-message>install-ui-deps</expected-message>
</test-case>

<test-case id="ui-valid-setup" should-fail="false">
  <description>install-ui-deps and build-ui steps prepare the production bundle ahead of show-urls</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "system-monitor"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "build-api", "run": "cd api && go build -o system-monitor-api .", "description": "Build API"},
        {"name": "install-ui-deps", "run": "cd ui && npm install", "description": "Install UI dependencies", "condition": {"file_exists": "ui/package.json"}},
        {"name": "build-ui", "run": "cd ui && npm run build", "description": "Build production UI", "condition": {"file_exists": "ui/dist/index.html"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="ui-install-rename" should-fail="true">
  <description>UI dependency installation exists but is misnamed, so agents need guidance to rename it</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "rename-install"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "build-api", "run": "cd api && go build -o rename-install-api .", "description": "Build API"},
        {"name": "prepare-ui-deps", "run": "cd ui && npm install", "description": "Install React dependencies", "condition": {"file_exists": "ui/package.json"}},
        {"name": "build-ui", "run": "cd ui && npm run build", "description": "Build production UI", "condition": {"file_exists": "ui/dist/index.html"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>named install-ui-deps</expected-message>
</test-case>

<test-case id="ui-build-rename" should-fail="true">
  <description>build step runs npm run build but uses a custom name instead of build-ui</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "rename-build"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "build-api", "run": "cd api && go build -o rename-build-api .", "description": "Build API"},
        {"name": "install-ui-deps", "run": "cd ui && npm install", "description": "Install UI dependencies", "condition": {"file_exists": "ui/package.json"}},
        {"name": "compile-ui", "run": "cd ui && npm run build", "description": "Build production bundle", "condition": {"file_exists": "ui/dist/index.html"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>named build-ui</expected-message>
</test-case>

<test-case id="ui-install-condition-missing" should-fail="true">
  <description>install-ui-deps must gate on ui/package.json so restarts rerun npm install when needed</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "ui-conditions"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "build-api", "run": "cd api && go build -o ui-conditions-api .", "description": "Build API"},
        {"name": "install-ui-deps", "run": "cd ui && npm install", "description": "Install UI deps"},
        {"name": "build-ui", "run": "cd ui && npm run build", "description": "Build production UI", "condition": {"file_exists": "ui/dist/index.html"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>condition.file_exists: "ui/package.json"</expected-message>
</test-case>

<test-case id="ui-build-condition-mismatch" should-fail="true">
  <description>build-ui must watch ui/dist/index.html so lifecycle checks know when to rebuild</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "ui-bundle-condition"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh", "description": "Install CLI"},
        {"name": "build-api", "run": "cd api && go build -o ui-bundle-condition-api .", "description": "Build API"},
        {"name": "install-ui-deps", "run": "cd ui && npm install", "description": "Install UI deps", "condition": {"file_exists": "ui/package.json"}},
        {"name": "build-ui", "run": "cd ui && npm run build", "description": "Build production UI", "condition": {"file_exists": "ui/package.json"}},
        {"name": "show-urls", "run": "echo done"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ui/dist/index.html</expected-message>
</test-case>
*/

// CheckSetupStepsConfiguration validates lifecycle.setup.steps for required tasks.
func CheckSetupStepsConfiguration(content []byte, filePath string) []Violation {
	if !shouldCheckSetupStepsJSON(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		return []Violation{newSetupStepsViolation(filePath, 1, "service.json is empty; expected lifecycle.setup.steps")}
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate setup steps: %v", err)
		return []Violation{newSetupStepsViolation(filePath, 1, msg)}
	}

	serviceName := extractSetupStepServiceName(payload)
	if serviceName == "" {
		line := findSetupJSONLine(source, "\"service\"", "\"name\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.name must be set to validate setup steps")}
	}

	lifecycleRaw, ok := payload["lifecycle"]
	if !ok {
		line := findSetupJSONLine(source, "\"lifecycle\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.json must define lifecycle.setup.steps")}
	}

	lifecycleMap, ok := lifecycleRaw.(map[string]any)
	if !ok {
		line := findSetupJSONLine(source, "\"lifecycle\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.json lifecycle must be an object")}
	}

	setupRaw, ok := lifecycleMap["setup"]
	if !ok {
		line := findSetupJSONLine(source, "\"setup\"")
		return []Violation{newSetupStepsViolation(filePath, line, "service.json must define lifecycle.setup.steps")}
	}

	setupMap, ok := setupRaw.(map[string]any)
	if !ok {
		line := findSetupJSONLine(source, "\"setup\"")
		return []Violation{newSetupStepsViolation(filePath, line, "lifecycle.setup must be an object")}
	}

	stepsRaw, ok := setupMap["steps"]
	if !ok {
		line := findSetupJSONLine(source, "\"steps\"")
		return []Violation{newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must be defined")}
	}

	stepsSlice, ok := stepsRaw.([]any)
	if !ok || len(stepsSlice) == 0 {
		line := findSetupJSONLine(source, "\"steps\"")
		return []Violation{newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must be a non-empty array")}
	}

	var (
		installMatch   *stepMatch
		buildMatch     *stepMatch
		uiInstallMatch *stepMatch
		uiBuildMatch   *stepMatch
		lastStep       map[string]any
		showStepIndex  = -1
		showStepLine   = findSetupJSONLine(source, "\"show-urls\"")
	)

	lastStepRaw := stepsSlice[len(stepsSlice)-1]
	lastStep, _ = lastStepRaw.(map[string]any)

	for idx, step := range stepsSlice {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(toStringOrDefault(stepMap["name"]))
		switch name {
		case "install-cli":
			installMatch = newStepMatch(stepMap, idx, name, true)
		case "build-api":
			buildMatch = newStepMatch(stepMap, idx, name, true)
		case "show-urls":
			showStepIndex = idx
			showStepLine = findSetupJSONLine(source, "\"name\": \"show-urls\"")
		case "install-ui-deps":
			uiInstallMatch = newStepMatch(stepMap, idx, name, true)
		case "build-ui":
			uiBuildMatch = newStepMatch(stepMap, idx, name, true)
		}

		if (uiInstallMatch == nil || !uiInstallMatch.matchedByName) && name != "install-ui-deps" {
			if looksLikeInstallUIDepsStep(stepMap) {
				uiInstallMatch = newStepMatch(stepMap, idx, name, false)
			}
		}
		if (uiBuildMatch == nil || !uiBuildMatch.matchedByName) && name != "build-ui" {
			if looksLikeBuildUIStep(stepMap) {
				uiBuildMatch = newStepMatch(stepMap, idx, name, false)
			}
		}
	}

	var violations []Violation

	if installMatch == nil {
		line := findMissingStepLine(source, "install-cli")
		msg := "lifecycle.setup.steps must include an install-cli step so lifecycle.setup.condition's CLI check and operators can find the binary before develop runs. Without it, CLI tooling fails even if the API built successfully."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installCLIRecommendation))
	} else {
		violations = append(violations, validateInstallCLI(filePath, source, installMatch.step)...)
	}

	if buildMatch == nil {
		line := findMissingStepLine(source, "build-api")
		msg := fmt.Sprintf("lifecycle.setup.steps must include a build-api step so lifecycle.setup.condition's binaries check can detect stale Go code and start-api reuses the %s-api binary instead of rebuilding ad hoc.", serviceName)
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildAPIRecommendation(serviceName)))
	} else {
		violations = append(violations, validateBuildAPI(filePath, source, buildMatch.step, serviceName)...)
	}

	if installMatch != nil && buildMatch != nil && installMatch.index > buildMatch.index {
		line := findStepNameLine(source, installMatch.name)
		msg := "install-cli must run before build-api so CLI binaries and symlinks are refreshed before we rebuild the Go server; otherwise lifecycle.setup.condition's CLI probe can pass with yesterday's binary while the API just changed."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installCLIRecommendation))
	}

	if showStepIndex == -1 {
		line := findMissingStepLine(source, "show-urls")
		msg := "lifecycle.setup.steps must include a final show-urls step so humans and agents know setup completed and which endpoints to hit."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg))
	} else {
		if lastStep == nil {
			line := findSetupJSONLine(source, "\"steps\"")
			violations = append(violations, newSetupStepsViolation(filePath, line, "lifecycle.setup.steps must include a final show-urls step so restart logs end with the URLs operators should open."))
		} else {
			name := strings.TrimSpace(toStringOrDefault(lastStep["name"]))
			if name != "show-urls" {
				violations = append(violations, newSetupStepsViolation(filePath, showStepLine, "show-urls must be the final setup step so no additional work runs silently after we announce completion."))
			}
		}
	}

	if setupScenarioHasUI(payload, stepsSlice) {
		if uiInstallMatch == nil {
			line := findMissingStepLine(source, "install-ui-deps")
			msg := "UI scenarios must include an install-ui-deps step so restart staleness detection reinstalls node_modules before build-ui runs; otherwise PRODUCTION_BUNDLES can't rebuild when ui/src changes. Add install-ui-deps ahead of build-ui and confirm develop/start-ui serves ui/dist."
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
		} else {
			if !uiInstallMatch.matchedByName {
				line := findStepNameLine(source, uiInstallMatch.name)
				msg := fmt.Sprintf("Step \"%s\" already installs frontend dependencies, but it must be named install-ui-deps so lifecycle.setup.condition's ui-bundle check and restart automation can correlate it with build-ui.", uiInstallMatch.name)
				violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
			}
			violations = append(violations, validateUIInstallStep(filePath, source, uiInstallMatch.step)...)
		}

		if uiBuildMatch == nil {
			line := findMissingStepLine(source, "build-ui")
			msg := "UI scenarios must include a build-ui step that emits ui/dist production bundles so develop serves the right assets; without it agents keep running dev servers that bypass PRODUCTION_BUNDLES."
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
		} else {
			if !uiBuildMatch.matchedByName {
				line := findStepNameLine(source, uiBuildMatch.name)
				msg := fmt.Sprintf("Step \"%s\" already builds the production UI, but it must be named build-ui so restart logs, auto-fixers, and docs/scenarios/PRODUCTION_BUNDLES.md know when ui/dist was generated. Rename it and verify develop/start-ui serves the built assets instead of \"npm run dev\".", uiBuildMatch.name)
				violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
			}

			installIdx := -1
			if uiInstallMatch != nil {
				installIdx = uiInstallMatch.index
			}
			violations = append(violations, validateUIBuildStep(filePath, source, uiBuildMatch.step, installIdx, uiBuildMatch.index, showStepIndex)...)
		}
	}

	return dedupeSetupStepsViolations(violations)
}

func validateInstallCLI(filePath, source string, step map[string]any) []Violation {
	var violations []Violation

	line := findSetupJSONLine(source, "\"install-cli\"")

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	runLower := strings.ToLower(run)
	hasCdCli := strings.Contains(runLower, "cd cli")
	touchesCliBinary := strings.Contains(runLower, "cli/")
	invokesInstallScript := strings.Contains(runLower, "cli/install.sh")
	if !hasCdCli && !touchesCliBinary && !invokesInstallScript {
		msg := "install-cli must run from cli/ or operate on cli/<service> binaries so the installed artifact matches what lifecycle.setup.condition checks."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installCLIRecommendation))
	}

	installIndicators := []string{"cli/install.sh", "install.sh", "ln -sf", "cp ", "install-cli"}
	indicatorFound := false
	for _, indicator := range installIndicators {
		if strings.Contains(runLower, indicator) {
			indicatorFound = true
			break
		}
	}
	if !indicatorFound {
		msg := "install-cli step must actually install the CLI binary (install.sh, ln -sf, or cp) so users and lifecycle checks can invoke it after setup."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installCLIRecommendation))
		return violations
	}

	desc := strings.TrimSpace(toStringOrDefault(step["description"]))
	descLower := strings.ToLower(desc)
	if !(strings.Contains(descLower, "install") && strings.Contains(descLower, "cli")) {
		msg := "install-cli description must mention installing the CLI so agents (and humans) know this step produces the binary required by lifecycle.setup.condition."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installCLIRecommendation))
	}

	return violations
}

func validateUIInstallStep(filePath, source string, step map[string]any) []Violation {
	var violations []Violation
	line := findMissingStepLine(source, "install-ui-deps")
	if step == nil {
		msg := "UI scenarios must include an install-ui-deps step that installs front-end dependencies before builds (see docs/scenarios/PRODUCTION_BUNDLES.md)"
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
		return violations
	}

	name := strings.TrimSpace(toStringOrDefault(step["name"]))
	if customLine := findStepNameLine(source, name); customLine > 0 {
		line = customLine
	}

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	runLower := strings.ToLower(run)
	if !stepRunsInUIDirectory(runLower) {
		msg := "install-ui-deps must run inside ui/ so node_modules timestamps update and lifecycle.setup.condition's ui-bundle check knows the workspace is fresh."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
	}

	if !stepInstallsUIDependencies(runLower) {
		msg := "install-ui-deps must run npm|pnpm|yarn|bun install so PRODUCTION_BUNDLES can rebuild ui/dist when ui/src changes."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
	}

	cond, ok := step["condition"].(map[string]any)
	fileExists := ""
	if ok {
		fileExists = strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
	}
	if !ok || fileExists == "" {
		msg := "install-ui-deps must declare condition.file_exists: \"ui/package.json\" so lifecycle.setup.condition can rerun npm install whenever the UI workspace changes."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
	} else if fileExists != "ui/package.json" {
		msg := "install-ui-deps condition.file_exists must point to ui/package.json so restart automation knows when dependencies went stale."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
	}

	return violations
}

func validateUIBuildStep(filePath, source string, step map[string]any, installIndex, buildIndex, showIndex int) []Violation {
	var violations []Violation
	line := findMissingStepLine(source, "build-ui")
	if step == nil {
		msg := "UI scenarios must include a build-ui step that produces ui/dist production bundles (see docs/scenarios/PRODUCTION_BUNDLES.md)"
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
		return violations
	}

	name := strings.TrimSpace(toStringOrDefault(step["name"]))
	if customLine := findStepNameLine(source, name); customLine > 0 {
		line = customLine
	}

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	runLower := strings.ToLower(run)
	if !stepRunsInUIDirectory(runLower) {
		msg := "build-ui must execute from ui/ so the bundle lands under ui/dist and the ui-bundle condition can detect stale assets."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}

	if !stepBuildsUIBundle(runLower) {
		msg := "build-ui must run npm|pnpm|yarn|bun build to create the production bundle that develop/start-ui serves."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}

	cond, ok := step["condition"].(map[string]any)
	fileExists := ""
	if ok {
		fileExists = strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
	}
	if !ok || fileExists == "" {
		msg := "build-ui must declare condition.file_exists: \"ui/dist/index.html\" so lifecycle.setup.condition's ui-bundle check can detect when production assets need to be rebuilt."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	} else if fileExists != "ui/dist/index.html" {
		msg := "build-ui condition.file_exists must point to ui/dist/index.html so restart automation confirms the production bundle that develop/start-ui will serve."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}

	if installIndex != -1 && buildIndex != -1 && installIndex > buildIndex {
		msg := "install-ui-deps must run before build-ui so dependencies are in place when the production bundle is generated."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}
	if showIndex != -1 && buildIndex != -1 && buildIndex > showIndex {
		msg := "build-ui must finish before show-urls so the restart message reflects the freshly built bundle."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}

	return violations
}

func validateBuildAPI(filePath, source string, step map[string]any, serviceName string) []Violation {
	var violations []Violation

	line := findSetupJSONLine(source, "\"build-api\"")
	rec := buildAPIRecommendation(serviceName)

	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	runLower := strings.ToLower(run)
	if !strings.Contains(runLower, "cd api") {
		msg := "build-api step must change into the api directory so the compiled binary lands next to go.mod for lifecycle.setup.condition's binaries check."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
	}
	if !strings.Contains(runLower, "go build") {
		msg := "build-api step must invoke 'go build' so restarts rebuild the Go server instead of leaving stale binaries in place."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
		return violations
	}

	outputMatch := buildOutputArgument(run)
	if outputMatch == "" {
		msg := fmt.Sprintf("build-api run must specify '-o %s-api' so start-api and lifecycle.setup.condition agree on the binary name.", serviceName)
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
	} else {
		expectedBinary := fmt.Sprintf("%s-api", serviceName)
		trimmed := strings.Trim(outputMatch, "\"'")
		cleaned := strings.TrimPrefix(strings.TrimPrefix(trimmed, "./"), ".\\")
		cleaned = strings.TrimPrefix(cleaned, "../")
		if strings.Contains(cleaned, "/") || strings.Contains(cleaned, "\\") {
			msg := fmt.Sprintf("build-api output must drop %s directly in the api directory so lifecycle.setup.condition can watch api/%s for staleness.", expectedBinary, expectedBinary)
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
		} else if filepath.Base(trimmed) != expectedBinary {
			msg := fmt.Sprintf("build-api output must be %s so start-api executes the same binary that setup built.", expectedBinary)
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
		}
	}

	desc := strings.TrimSpace(toStringOrDefault(step["description"]))
	descLower := strings.ToLower(desc)
	if !(strings.Contains(descLower, "build") && strings.Contains(descLower, "api")) {
		msg := "build-api description must describe building the Go API so agents understand this step produces the binary referenced elsewhere."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
	}

	if cond, ok := step["condition"].(map[string]any); ok {
		fileExists := strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
		if fileExists != "" && fileExists != "api/go.mod" {
			msg := "build-api condition.file_exists should reference 'api/go.mod' so setup reruns whenever Go sources change."
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, rec))
		}
	}

	return violations
}

var buildOutputRegex = regexp.MustCompile(`-o\s+([^\s]+)`)

func buildOutputArgument(run string) string {
	match := buildOutputRegex.FindStringSubmatch(run)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func newSetupStepsViolation(filePath string, line int, message string, recommendation ...string) Violation {
	if line <= 0 {
		line = 1
	}
	rec := "Ensure lifecycle.setup.steps include install-cli, scenario-specific build-api, and show-urls as the final step"
	if len(recommendation) > 0 {
		if custom := strings.TrimSpace(recommendation[0]); custom != "" {
			rec = custom
		}
	}
	return Violation{
		Type:           "config_setup_steps",
		Severity:       "medium",
		Title:          "Setup steps configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: rec,
		Standard:       "configuration-v1",
	}
}

func extractSetupStepServiceName(payload map[string]any) string {
	serviceRaw, ok := payload["service"].(map[string]any)
	if !ok {
		return ""
	}
	name := strings.TrimSpace(toStringOrDefault(serviceRaw["name"]))
	return name
}

func toStringOrDefault(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func shouldCheckSetupStepsJSON(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(trimmed))
	if base == "service.json" {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".json") {
		return true
	}
	return false
}

func findSetupJSONLine(content string, tokens ...string) int {
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

func findMissingStepLine(content, stepName string) int {
	patterns := []string{
		fmt.Sprintf("\"name\": \"%s\"", stepName),
		fmt.Sprintf("\"%s\"", stepName),
	}
	for _, token := range patterns {
		if line := findSetupJSONLine(content, token); line != 1 {
			return line
		}
	}
	fallbacks := []string{"\"steps\"", "\"setup\"", "\"lifecycle\""}
	for _, token := range fallbacks {
		if line := findSetupJSONLine(content, token); line != 1 {
			return line
		}
	}
	return 1
}

func findStepNameLine(content, stepName string) int {
	stepName = strings.TrimSpace(stepName)
	if stepName == "" {
		return 1
	}
	token := fmt.Sprintf("\"name\": \"%s\"", stepName)
	return findSetupJSONLine(content, token)
}

func stepRunsInUIDirectory(runLower string) bool {
	tokens := []string{"cd ui", " ui &&", "ui &&", "--prefix ui", "ui/"}
	return containsAny(runLower, tokens)
}

func stepInstallsUIDependencies(runLower string) bool {
	installTokens := []string{
		"npm install",
		"npm ci",
		"pnpm install",
		"pnpm i",
		"yarn install",
		"bun install",
	}
	if containsAny(runLower, installTokens) {
		return true
	}
	strictTokens := []string{"--frozen-lockfile", "--immutable", "--prefer-offline", "--locked"}
	if containsAny(runLower, strictTokens) {
		return strings.Contains(runLower, "npm") || strings.Contains(runLower, "pnpm") || strings.Contains(runLower, "yarn") || strings.Contains(runLower, "bun")
	}
	return false
}

func stepBuildsUIBundle(runLower string) bool {
	buildTokens := []string{"npm run build", "pnpm run build", "pnpm build", "yarn build", "bun run build", "bun build"}
	return containsAny(runLower, buildTokens)
}

func looksLikeInstallUIDepsStep(step map[string]any) bool {
	if step == nil {
		return false
	}
	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	if run == "" {
		return false
	}
	runLower := strings.ToLower(run)
	return stepRunsInUIDirectory(runLower) && stepInstallsUIDependencies(runLower)
}

func looksLikeBuildUIStep(step map[string]any) bool {
	if step == nil {
		return false
	}
	run := strings.TrimSpace(toStringOrDefault(step["run"]))
	if run == "" {
		return false
	}
	runLower := strings.ToLower(run)
	return stepRunsInUIDirectory(runLower) && stepBuildsUIBundle(runLower)
}

func dedupeSetupStepsViolations(list []Violation) []Violation {
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

func setupScenarioHasUI(payload map[string]any, steps []any) bool {
	if scenarioPortsSuggestUI(payload) {
		return true
	}
	if scenarioComponentsSuggestUI(payload) {
		return true
	}
	return stepsSuggestUI(steps)
}

func scenarioPortsSuggestUI(payload map[string]any) bool {
	portsRaw, ok := payload["ports"].(map[string]any)
	if !ok {
		return false
	}
	for key, entry := range portsRaw {
		if setupIsUIPortKey(key) && setupEntryEnabled(entry) {
			return true
		}
	}
	return false
}

func scenarioComponentsSuggestUI(payload map[string]any) bool {
	componentsRaw, ok := payload["components"].(map[string]any)
	if !ok {
		return false
	}
	for key, entry := range componentsRaw {
		if setupIsUIPortKey(key) && setupEntryEnabled(entry) {
			return true
		}
	}
	return false
}

func stepsSuggestUI(steps []any) bool {
	for _, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		if setupUIStepIndicator(toStringOrDefault(stepMap["name"])) || setupUIStepIndicator(toStringOrDefault(stepMap["run"])) {
			return true
		}
	}
	return false
}

func setupIsUIPortKey(key string) bool {
	if key == "" {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(key))
	tokens := []string{"ui", "web", "frontend", "dashboard", "client", "spa"}
	for _, token := range tokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func setupUIStepIndicator(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	tokens := []string{"install-ui", "build-ui", "start-ui", "start-frontend", "ui server", "frontend", "vite", "next dev", "react", "node server"}
	for _, token := range tokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func setupEntryEnabled(entry any) bool {
	if entry == nil {
		return true
	}
	if m, ok := entry.(map[string]any); ok {
		if enabledRaw, ok := m["enabled"]; ok {
			switch v := enabledRaw.(type) {
			case bool:
				return v
			case string:
				return strings.EqualFold(v, "true")
			}
		}
	}
	return true
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
