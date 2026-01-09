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
)

var (
	cachedName string
	detectOnce sync.Once

	// For testing
	getwd     = os.Getwd
	envGetter = os.Getenv
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

	// Normalize path separators
	cwd = filepath.ToSlash(cwd)

	// Look for /api in the path
	// Handle both: .../scenarios/foo/api and .../scenarios/foo/api/subdir
	parts := strings.Split(cwd, "/")
	for i := len(parts) - 1; i >= 1; i-- {
		if parts[i] == "api" {
			// Parent of "api" is the scenario name
			return parts[i-1]
		}
	}

	// Also check if cwd itself is the api directory
	if filepath.Base(cwd) == "api" {
		parent := filepath.Dir(cwd)
		return filepath.Base(parent)
	}

	return ""
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

	getwd = getwdFn
	envGetter = envGetterFn
	Reset()

	return func() {
		getwd = oldGetwd
		envGetter = oldEnvGetter
		Reset()
	}
}
