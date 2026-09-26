// Package repo provides repository detection and scenario path discovery.
package repo

import (
	"os"
	"path/filepath"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/artifactpaths"
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

func Root() string {
	rootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			return
		}
		rootPath, _ = repocontract.FindRepoRootFromPath(dir)
	})
	return rootPath
}

// DiscoverScenarioPaths locates scenario and test directories for the given scenario name.
func DiscoverScenarioPaths(scenario string) Paths {
	root := Root()
	if root == "" {
		return Paths{}
	}
	scenarioDir, err := repocontract.ResolveScenarioPath(root, scenario)
	if err != nil {
		return Paths{}
	}
	info, err := os.Stat(scenarioDir)
	if err != nil || !info.IsDir() {
		return Paths{}
	}
	testDir := artifactpaths.ScenarioPath(scenarioDir, artifactpaths.CoverageRoot)
	if _, err := os.Stat(testDir); err != nil {
		testDir = ""
	}
	return Paths{ScenarioDir: scenarioDir, TestDir: testDir}
}

// AbsPath converts a path to an absolute path relative to the repository root.
// If the path is already absolute or the root cannot be determined, the input
// path is returned.
func AbsPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	if root := Root(); root != "" {
		return filepath.Join(root, path)
	}
	return path
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
