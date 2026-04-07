// Package path provides utilities for detecting and working with Vrooli paths.
//
// This package centralizes Vrooli root detection logic that was previously
// duplicated across the main package and distribution packages.
package path

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Env abstracts environment variable access for testing.
type Env interface {
	Getenv(key string) string
	UserHomeDir() (string, error)
	Getwd() (string, error)
}

// FS abstracts filesystem stat operations for testing.
type FS interface {
	Stat(name string) (fs.FileInfo, error)
}

// realEnv implements Env using the os package.
type realEnv struct{}

func (realEnv) Getenv(key string) string     { return os.Getenv(key) }
func (realEnv) UserHomeDir() (string, error) { return os.UserHomeDir() }
func (realEnv) Getwd() (string, error)       { return os.Getwd() }

// realFS implements FS using the os package.
type realFS struct{}

func (realFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

// DetectVrooliRoot finds the Vrooli root directory using multiple strategies:
//  1. VROOLI_ROOT environment variable (highest priority)
//  2. Default home directory location (~/{vrooli_home})
//  3. Walk up from current directory looking for .vrooli marker
//  4. Fallback to relative path from current directory
//
// Returns empty string if no valid root can be found.
func DetectVrooliRoot() string {
	return detectRoot(realEnv{}, realFS{})
}

// detectRoot is the testable implementation of DetectVrooliRoot.
func detectRoot(env Env, fsys FS) string {
	// 1. Check environment variable (highest priority)
	if root := env.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	// 2. Try default home directory location
	if homeDir, err := env.UserHomeDir(); err == nil {
		defaultRoot := filepath.Join(homeDir, "Vrooli")
		if info, err := fsys.Stat(defaultRoot); err == nil && info.IsDir() {
			return defaultRoot
		}
	}

	// 3. Walk up from current directory looking for .vrooli marker
	cwd, err := env.Getwd()
	if err != nil {
		return ""
	}

	for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		vrooliDir := filepath.Join(dir, ".vrooli")
		if info, err := fsys.Stat(vrooliDir); err == nil && info.IsDir() {
			return dir
		}
	}

	// 4. Fallback to relative path (assumes running from api/ directory)
	return filepath.Clean(filepath.Join(cwd, "../../.."))
}
