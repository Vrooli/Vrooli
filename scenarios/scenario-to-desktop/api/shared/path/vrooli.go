// Package path provides utilities for detecting and working with Vrooli paths.
package path

import (
	"io/fs"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Env abstracts environment variable access for testing.
type Env interface {
	Getenv(key string) string
	Getwd() (string, error)
}

// FS abstracts filesystem stat operations for testing.
type FS interface {
	Stat(name string) (fs.FileInfo, error)
}

// realEnv implements Env using the os package.
type realEnv struct{}

func (realEnv) Getenv(key string) string { return os.Getenv(key) }
func (realEnv) Getwd() (string, error)   { return os.Getwd() }

// realFS implements FS using the os package.
type realFS struct{}

func (realFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

// DetectVrooliRoot finds the canonical Vrooli repo root.
func DetectVrooliRoot() string {
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	return detectRoot(realEnv{}, realFS{})
}

// DetectScenariosRoot returns the contract-defined scenarios root.
func DetectScenariosRoot() string {
	root := DetectVrooliRoot()
	if root == "" {
		return ""
	}
	if contract, err := repocontract.LoadDefault(root); err == nil {
		if scenariosDir, err := contract.TopLevelDir(root, "scenarios"); err == nil {
			return scenariosDir
		}
	}
	return filepath.Join(root, "scenarios")
}

// ResolveScenarioRoot returns the canonical root for a scenario.
func ResolveScenarioRoot(scenario string) string {
	root := DetectVrooliRoot()
	if root == "" {
		return ""
	}
	if resolved, err := repocontract.ResolveScenarioPath(root, scenario); err == nil {
		return resolved
	}
	return filepath.Join(DetectScenariosRoot(), filepath.Clean(scenario))
}

// detectRoot is the testable fallback implementation used when repo-contract
// resolution is unavailable.
func detectRoot(env Env, fsys FS) string {
	if root := env.Getenv("VROOLI_ROOT"); root != "" {
		if resolved, err := repocontract.FindRepoRootFromPath(root); err == nil {
			return resolved
		}
		return filepath.Clean(root)
	}

	cwd, err := env.Getwd()
	if err != nil {
		return ""
	}

	for dir := filepath.Clean(cwd); ; dir = filepath.Dir(dir) {
		vrooliDir := filepath.Join(dir, ".vrooli")
		if info, err := fsys.Stat(vrooliDir); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == "." {
			break
		}
	}

	return filepath.Clean(filepath.Join(cwd, "../../.."))
}
