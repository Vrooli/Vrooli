package setupsteps

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	installUIDepsRecommendation = "Add install-ui-deps (npm|pnpm|yarn|bun install inside ui/) before build-ui so PRODUCTION_BUNDLES staleness detection can recreate ui/dist when ui/src changes."
	buildUIRecommendation       = "Run build-ui (npm|pnpm|yarn|bun build inside ui/) so develop serves the ui/dist production bundle per docs/scenarios/PRODUCTION_BUNDLES.md."
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

const defaultUIBundlePath = "ui/dist/index.html"

/*
Rule: Setup Steps Configuration
Description: Ensure lifecycle.setup.steps include the steps that actually satisfy setup conditions
Reason: Reliable setup steps prevent missing binaries and inconsistent developer environments
Category: config
Severity: medium
Standard: configuration-v1
Targets: service_json

<test-case id="missing-setup-steps" should-fail="true">
  <description>service.json without lifecycle.setup steps</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {}
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>lifecycle.setup.steps</expected-message>
</test-case>

<test-case id="missing-build-api" should-fail="true">
  <description>Setup steps missing the build-api task</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "cli": {
    "enabled": true,
    "command": "file-tools",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "file-tools"}
  },
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "verify-db", "run": "echo done", "description": "Verify database access"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>build-api</expected-message>
</test-case>

<test-case id="wrong-build-api-output" should-fail="true">
  <description>build-api must emit the scenario-specific binary name</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go build -o tools-api .", "description": "Build Go API"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>output must be file-tools-api</expected-message>
</test-case>

<test-case id="ui-missing-build-step" should-fail="true">
  <description>UI scenarios must build production bundles before develop starts the UI server</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "app-issue-tracker"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go build -o app-issue-tracker-api .", "description": "Build API"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>2</expected-violations>
  <expected-message>install-ui-deps</expected-message>
</test-case>

<test-case id="valid-setup" should-fail="false">
  <description>Setup steps include build-api plus optional CLI-independent steps</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"},
  "cli": {
    "enabled": true,
    "command": "file-tools",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "file-tools"}
  },
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "db-schema", "run": "echo schema", "description": "Prepare the database"},
        {"name": "build-api", "run": "cd api && go mod tidy && go build -o file-tools-api .", "description": "Build Go API binary"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="ui-valid-setup" should-fail="false">
  <description>install-ui-deps and build-ui steps prepare the production bundle</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "system-monitor"},
  "ports": {"ui": {"env_var": "UI_PORT"}},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go build -o system-monitor-api .", "description": "Build API"},
        {"name": "install-ui-deps", "run": "cd ui && npm install", "description": "Install UI dependencies", "condition": {"file_exists": "ui/package.json"}},
        {"name": "build-ui", "run": "cd ui && npm run build", "description": "Build production UI"}
      ]
    }
  }
}
  ]]></input>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule is skipped for files other than service.json</description>
  <input language="json"><![CDATA[
{
  "service": {"name": "file-tools"}
}
  ]]></input>
</test-case>

<test-case id="missing-service-name" should-fail="true">
  <description>build-api exists but service.name is missing</description>
  <input language="json"><![CDATA[
{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go build -o file-tools-api .", "description": "Build Go API"}
      ]
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>service.name</expected-message>
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
		buildMatch     *stepMatch
		uiInstallMatch *stepMatch
		uiBuildMatch   *stepMatch
	)

	for idx, step := range stepsSlice {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(toStringOrDefault(stepMap["name"]))
		switch name {
		case "build-api":
			buildMatch = newStepMatch(stepMap, idx, name, true)
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
	bundlePath := inferUIBundlePath(payload)

	combinedUIWork := uiInstallMatch != nil && uiBuildMatch != nil && uiInstallMatch.index == uiBuildMatch.index
	if combinedUIWork {
		line := findStepNameLine(source, uiBuildMatch.name)
		if line == 1 {
			line = findMissingStepLine(source, "build-ui")
		}
		msg := fmt.Sprintf("Step \"%s\" currently runs npm install and npm run build in a single step. Split build-ui into install-ui-deps (npm install in ui/) followed by build-ui (npm run build) so lifecycle.setup.condition can rerun whichever half went stale.", uiBuildMatch.name)
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
		uiInstallMatch = nil
	}

	if buildMatch == nil {
		line := findMissingStepLine(source, "build-api")
		msg := fmt.Sprintf("lifecycle.setup.steps must include a build-api step so lifecycle.setup.condition's binaries check can detect stale Go code and start-api reuses the %s-api binary instead of rebuilding ad hoc.", serviceName)
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildAPIRecommendation(serviceName)))
	} else {
		violations = append(violations, validateBuildAPI(filePath, source, buildMatch.step, serviceName)...)
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
			violations = append(violations, validateUIBuildStep(filePath, source, uiBuildMatch.step, installIdx, uiBuildMatch.index, bundlePath)...)
		}
	}

	return dedupeSetupStepsViolations(violations)
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

	if cond, ok := step["condition"].(map[string]any); ok {
		fileExists := strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
		if fileExists != "" && fileExists != "ui/package.json" {
			msg := fmt.Sprintf("install-ui-deps condition.file_exists currently points to %s; only 'ui/package.json' is allowed so the step is skipped iff the UI workspace is actually missing.", fileExists)
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, installUIDepsRecommendation))
		}
	}

	return violations
}

func validateUIBuildStep(filePath, source string, step map[string]any, installIndex, buildIndex int, expectedBundlePath string) []Violation {
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

	if installIndex != -1 && stepInstallsUIDependencies(runLower) {
		msg := "Split build-ui into install-ui-deps (npm install in ui/) and build-ui (npm run build) so lifecycle.setup.condition can rerun whichever stage is stale."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}

	if !stepBuildsUIBundle(runLower) {
		msg := "build-ui must run npm|pnpm|yarn|bun build to create the production bundle that develop/start-ui serves."
		violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
	}

	if cond, ok := step["condition"].(map[string]any); ok {
		fileExists := strings.TrimSpace(toStringOrDefault(cond["file_exists"]))
		if fileExists != "" {
			msg := fmt.Sprintf("build-ui must not skip itself based on %s. Remove condition.file_exists so fresh clones always rebuild the production bundle before develop serves it.", fileExists)
			if fileExists == expectedBundlePath {
				msg = fmt.Sprintf("build-ui must not guard on %s. Delete condition.file_exists so the bundle step runs even when the artifact is missing.", expectedBundlePath)
			}
			violations = append(violations, newSetupStepsViolation(filePath, line, msg, buildUIRecommendation))
		}
	}

	if installIndex != -1 && buildIndex != -1 && installIndex > buildIndex {
		msg := "install-ui-deps must run before build-ui so dependencies are in place when the production bundle is generated."
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
	rec := "Ensure lifecycle.setup.steps include scenario-specific build-api work and any required UI bundle steps"
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

func shouldCheckSetupStepsJSON(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(trimmed))
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".json") {
		return true
	}
	if base != "service.json" {
		return false
	}
	normalized := filepath.ToSlash(trimmed)
	if strings.Contains(normalized, "/scenarios/") || strings.HasPrefix(normalized, "scenarios/") {
		return true
	}
	if !strings.Contains(normalized, "/") {
		// Doc tests use synthetic service.json paths without directories; allow them.
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
	return scenarioPortsExposeUIPort(payload)
}

func inferUIBundlePath(payload map[string]any) string {
	lifecycleRaw, ok := payload["lifecycle"].(map[string]any)
	if !ok {
		return defaultUIBundlePath
	}
	setupRaw, ok := lifecycleRaw["setup"].(map[string]any)
	if !ok {
		return defaultUIBundlePath
	}
	conditionRaw, ok := setupRaw["condition"].(map[string]any)
	if !ok {
		return defaultUIBundlePath
	}
	checks, ok := conditionRaw["checks"].([]any)
	if !ok {
		return defaultUIBundlePath
	}
	for _, check := range checks {
		checkMap, ok := check.(map[string]any)
		if !ok {
			continue
		}
		typeVal := strings.ToLower(strings.TrimSpace(toStringOrDefault(checkMap["type"])))
		if typeVal != "ui-bundle" {
			continue
		}
		bundle := strings.TrimSpace(toStringOrDefault(checkMap["bundle_path"]))
		if bundle != "" {
			return bundle
		}
	}
	return defaultUIBundlePath
}

func scenarioPortsExposeUIPort(payload map[string]any) bool {
	portsRaw, ok := payload["ports"].(map[string]any)
	if !ok {
		return false
	}
	for _, entry := range portsRaw {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		envVar := strings.ToUpper(strings.TrimSpace(toStringOrDefault(entryMap["env_var"])))
		if envVar == "UI_PORT" {
			// UI_PORT is the standard environment variable for UI servers, so its presence reliably signals a UI bundle.
			return true
		}
	}
	return false
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
