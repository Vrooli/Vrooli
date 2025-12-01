package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var baselineCommands = []string{"bash", "curl", "jq"}

// runDependenciesPhase inspects runtime/tool requirements without invoking bash helpers.
func runDependenciesPhase(ctx context.Context, env PhaseEnvironment, logWriter io.Writer) PhaseRunReport {
	if err := ctx.Err(); err != nil {
		return PhaseRunReport{Err: err, FailureClassification: failureClassSystem}
	}

	var observations []string
	tracker := &missingDependencyTracker{}

	checkCommand := func(name, reason string) {
		if err := ensureCommandAvailable(name); err != nil {
			logPhaseWarn(logWriter, "command missing: %s (%v)", name, err)
			tracker.Add(name, reason)
			return
		}
		logPhaseStep(logWriter, "command verified: %s", name)
		observations = append(observations, fmt.Sprintf("command available: %s", name))
	}

	for _, cmd := range baselineCommands {
		checkCommand(cmd, "baseline tool required to run local phases")
	}

	runtimes := detectRuntimes(env)
	if len(runtimes) == 0 {
		logPhaseWarn(logWriter, "no language runtimes detected for this scenario")
		observations = append(observations, "no runtime-specific checks detected")
	}
	for _, runtime := range runtimes {
		checkCommand(runtime, fmt.Sprintf("%s runtime required to compile or test scenario code", runtime))
	}

	managers := detectPackageManagers(env)
	nodeWorkspace := hasNodeWorkspace(env)
	if len(managers) == 0 && nodeWorkspace {
		logPhaseWarn(logWriter, "node workspace detected without lockfile; assuming pnpm is required")
		managers = append(managers, "pnpm")
	}
	seenManagers := map[string]struct{}{}
	for _, manager := range managers {
		if _, exists := seenManagers[manager]; exists {
			continue
		}
		seenManagers[manager] = struct{}{}
		checkCommand(manager, fmt.Sprintf("%s package manager required to install JavaScript dependencies", manager))
	}
	if len(managers) == 0 && nodeWorkspace {
		observations = append(observations, "JavaScript workspace detected but package manager requirement defaulted to pnpm")
	} else if len(managers) == 0 {
		observations = append(observations, "no JavaScript package managers required")
	}

	requiredResources, err := reportResourceExpectations(env, logWriter)
	if err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Fix .vrooli/service.json so required resources can be read.",
			Observations:          observations,
		}
	}
	if len(requiredResources) == 0 {
		observations = append(observations, "manifest declares no required resources")
	} else {
		for _, resource := range requiredResources {
			observations = append(observations, fmt.Sprintf("requires resource: %s", resource))
		}
	}

	if tracker.HasAny() {
		return PhaseRunReport{
			Err:                   errors.New(tracker.Summary()),
			FailureClassification: failureClassMissingDependency,
			Remediation:           tracker.Remediation(),
			Observations:          observations,
		}
	}

	logPhaseStep(logWriter, "dependency validation complete")
	return PhaseRunReport{Observations: observations}
}

func detectRuntimes(env PhaseEnvironment) []string {
	var runtimes []string

	if hasGoSources(env) {
		runtimes = append(runtimes, "go")
	}
	if hasNodeWorkspace(env) {
		runtimes = append(runtimes, "node")
	}
	if hasPythonWorkspace(env) {
		runtimes = append(runtimes, "python3")
	}
	return runtimes
}

func hasGoSources(env PhaseEnvironment) bool {
	candidates := []string{
		filepath.Join(env.ScenarioDir, "api", "go.mod"),
		filepath.Join(env.ScenarioDir, "cli", "go.mod"),
	}
	for _, path := range candidates {
		if fileExists(path) {
			return true
		}
	}
	globPatterns := []string{
		filepath.Join(env.ScenarioDir, "api", "*.go"),
		filepath.Join(env.ScenarioDir, "cli", "*.go"),
	}
	for _, pattern := range globPatterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return true
		}
	}
	return false
}

func hasNodeWorkspace(env PhaseEnvironment) bool {
	candidates := []string{
		filepath.Join(env.ScenarioDir, "package.json"),
		filepath.Join(env.ScenarioDir, "ui", "package.json"),
	}
	for _, path := range candidates {
		if fileExists(path) {
			return true
		}
	}
	return false
}

func hasPythonWorkspace(env PhaseEnvironment) bool {
	candidates := []string{
		filepath.Join(env.ScenarioDir, "requirements.txt"),
		filepath.Join(env.ScenarioDir, "pyproject.toml"),
	}
	for _, path := range candidates {
		if fileExists(path) {
			return true
		}
	}
	return false
}

func detectPackageManagers(env PhaseEnvironment) []string {
	var managers []string
	lockFiles := []struct {
		Path    string
		Command string
	}{
		{filepath.Join(env.ScenarioDir, "pnpm-lock.yaml"), "pnpm"},
		{filepath.Join(env.ScenarioDir, "ui", "pnpm-lock.yaml"), "pnpm"},
		{filepath.Join(env.ScenarioDir, "package-lock.json"), "npm"},
		{filepath.Join(env.ScenarioDir, "ui", "package-lock.json"), "npm"},
		{filepath.Join(env.ScenarioDir, "yarn.lock"), "yarn"},
		{filepath.Join(env.ScenarioDir, "ui", "yarn.lock"), "yarn"},
	}
	for _, entry := range lockFiles {
		if fileExists(entry.Path) && !contains(managers, entry.Command) {
			managers = append(managers, entry.Command)
		}
	}
	if len(managers) == 0 && hasNodeWorkspace(env) {
		// Default to pnpm if the scenario has a Node workspace but no lock file yet.
		managers = append(managers, "pnpm")
	}
	return managers
}

func reportResourceExpectations(env PhaseEnvironment, logWriter io.Writer) ([]string, error) {
	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	manifest, err := loadServiceManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	required := manifest.requiredResources()
	if len(required) == 0 {
		logPhaseWarn(logWriter, "no required resources declared in %s", manifestPath)
		return nil, nil
	}
	for _, resource := range required {
		logPhaseStep(logWriter, "resource requirement detected: %s", resource)
	}
	return required, nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

type missingDependencyTracker struct {
	details []string
	names   []string
}

func (m *missingDependencyTracker) Add(name, reason string) {
	m.details = append(m.details, fmt.Sprintf("%s (%s)", name, reason))
	m.names = append(m.names, name)
}

func (m *missingDependencyTracker) HasAny() bool {
	return len(m.details) > 0
}

func (m *missingDependencyTracker) Summary() string {
	if len(m.details) == 0 {
		return ""
	}
	ordered := append([]string(nil), m.details...)
	sort.Strings(ordered)
	return fmt.Sprintf("missing required tooling: %s", strings.Join(ordered, "; "))
}

func (m *missingDependencyTracker) Remediation() string {
	if len(m.names) == 0 {
		return ""
	}
	names := dedupeStrings(m.names)
	sort.Strings(names)
	return fmt.Sprintf("Install or expose these commands to PATH: %s", strings.Join(names, ", "))
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var result []string
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
