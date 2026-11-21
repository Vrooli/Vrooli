// Package scenario provides automatic scenario name detection for Vrooli APIs.
//
// The package determines the scenario name from the directory structure or
// environment variables, eliminating the need to manually specify the scenario
// name in multiple places.
//
// Detection order:
//  1. SCENARIO_NAME environment variable (if set)
//  2. Directory structure: parent of "api/" directory
//
// Example directory structure:
//
//	scenarios/chart-generator/api/main.go
//	         └── scenario name ──┘
//
// Usage:
//
//	name := scenario.Name()           // "chart-generator"
//	svc := scenario.ServiceName()     // "chart-generator-api"
package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"
)

var (
	cachedName string
	detectOnce sync.Once

	// For testing
	getwd        = os.Getwd
	envGetter    = os.Getenv
	findRepoRoot = repocontract.FindRepoRootFromPath
	loadContract = repocontract.LoadDefault
	statPath     = os.Stat
)

// Name returns the scenario name.
//
// Detection priority:
//  1. SCENARIO_NAME environment variable
//  2. Parent directory of "api/" in current working directory
//  3. "unknown" if detection fails
//
// The result is cached after first call.
func Name() string {
	detectOnce.Do(func() {
		cachedName = detect()
	})
	return cachedName
}

// ServiceName returns the API service name ("<scenario>-api").
func ServiceName() string {
	return Name() + "-api"
}

// detect determines the scenario name from environment or directory structure.
func detect() string {
	// Priority 1: Environment variable
	if name := strings.TrimSpace(envGetter("SCENARIO_NAME")); name != "" {
		return name
	}

	// Priority 2: Directory structure
	if name := detectFromDirectory(); name != "" {
		return name
	}

	return "unknown"
}

// detectFromDirectory attempts to detect scenario name from current working directory.
//
// Expected structure: .../scenarios/<scenario-name>/api/...
// The function finds "api" in the path and returns its parent directory name.
func detectFromDirectory() string {
	cwd, err := getwd()
	if err != nil {
		return ""
	}

	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return ""
	}
	contract, err := loadContract(repoRoot)
	if err != nil {
		return ""
	}
	scenarioDir, err := contract.TopLevelDir(repoRoot, "scenarios")
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(scenarioDir, cwd)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") {
		return ""
	}

	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return ""
	}
	scenarioName := strings.TrimSpace(parts[0])
	if scenarioName == "" {
		return ""
	}

	apiPath, ok := contract.Scenario().WellKnownPaths["api"]
	if !ok {
		return ""
	}
	apiPath = strings.Trim(strings.TrimSpace(filepath.ToSlash(apiPath)), "/")
	if apiPath == "" {
		return ""
	}
	remaining := strings.Join(parts[1:], "/")
	if remaining != apiPath && !strings.HasPrefix(remaining, apiPath+"/") {
		return ""
	}

	servicePath, err := contract.ScenarioFile(repoRoot, scenarioName, "service")
	if err != nil {
		return ""
	}
	if _, err := statPath(servicePath); err != nil {
		return ""
	}
	return scenarioName
}

// Reset clears the cached name, forcing re-detection on next call.
// This is primarily useful for testing.
func Reset() {
	detectOnce = sync.Once{}
	cachedName = ""
}

// SetTestHooks allows tests to override detection functions.
// Returns a cleanup function that restores the original functions.
func SetTestHooks(getwdFn func() (string, error), envGetterFn func(string) string) func() {
	oldGetwd := getwd
	oldEnvGetter := envGetter
	oldFindRepoRoot := findRepoRoot
	oldLoadContract := loadContract
	oldStatPath := statPath

	getwd = getwdFn
	envGetter = envGetterFn
	findRepoRoot = repocontract.FindRepoRootFromPath
	loadContract = repocontract.LoadDefault
	statPath = os.Stat
	Reset()

	return func() {
		getwd = oldGetwd
		envGetter = oldEnvGetter
		findRepoRoot = oldFindRepoRoot
		loadContract = oldLoadContract
		statPath = oldStatPath
		Reset()
	}
}
