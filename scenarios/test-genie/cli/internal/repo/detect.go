// Package repo provides repository detection and scenario path discovery.
package repo

import (
	"os"
	"path/filepath"
	"sync"
)

// Paths holds discovered scenario directory paths.
type Paths struct {
	ScenarioDir string
	TestDir     string
}

var (
	rootOnce sync.Once
	rootPath string
)

// Root returns the repository root directory, caching the result.
func Root() string {
	rootOnce.Do(func() {
		dir, _ := os.Getwd()
		rootPath = locateRoot(dir)
	})
	return rootPath
}

// DiscoverScenarioPaths locates scenario and test directories for the given scenario name.
func DiscoverScenarioPaths(scenario string) Paths {
	root := Root()
	if root == "" {
		return Paths{}
	}
	scenarioDir := filepath.Join(root, "scenarios", scenario)
	info, err := os.Stat(scenarioDir)
	if err != nil || !info.IsDir() {
		return Paths{}
	}
	testDir := filepath.Join(scenarioDir, "test")
	if _, err := os.Stat(testDir); err != nil {
		testDir = ""
	}
	return Paths{ScenarioDir: scenarioDir, TestDir: testDir}
}

// Exists checks if a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FileState returns whether a file exists and whether it is empty.
func FileState(path string) (exists bool, empty bool) {
	tryPath := func(candidate string) (bool, bool) {
		info, err := os.Stat(candidate)
		if err != nil {
			return false, false
		}
		if info.IsDir() {
			return true, true
		}
		return true, info.Size() == 0
	}
	if exists, empty := tryPath(path); exists {
		return exists, empty
	}
	if filepath.IsAbs(path) {
		return false, false
	}
	if root := Root(); root != "" {
		return tryPath(filepath.Join(root, path))
	}
	return false, false
}

func locateRoot(start string) string {
	dir := start
	for i := 0; i < 8 && dir != "" && dir != string(filepath.Separator); i++ {
		if dir == "" {
			break
		}
		if Exists(filepath.Join(dir, ".git")) {
			return dir
		}
		if Exists(filepath.Join(dir, "pnpm-workspace.yaml")) {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
