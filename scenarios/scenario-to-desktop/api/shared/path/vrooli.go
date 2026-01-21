// Package path provides utilities for detecting and working with Vrooli paths.
//
// This package centralizes Vrooli root detection logic that was previously
// duplicated across the main package and distribution packages.
package path

import (
	"os"
	"path/filepath"
)

// DetectVrooliRoot finds the Vrooli root directory using multiple strategies:
//  1. VROOLI_ROOT environment variable (highest priority)
//  2. Default home directory location (~/{vrooli_home})
//  3. Walk up from current directory looking for .vrooli marker
//  4. Fallback to relative path from current directory
//
// Returns empty string if no valid root can be found.
func DetectVrooliRoot() string {
	// 1. Check environment variable (highest priority)
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	// 2. Try default home directory location
	if homeDir, err := os.UserHomeDir(); err == nil {
		defaultRoot := filepath.Join(homeDir, "Vrooli")
		if info, err := os.Stat(defaultRoot); err == nil && info.IsDir() {
			return defaultRoot
		}
	}

	// 3. Walk up from current directory looking for .vrooli marker
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		vrooliDir := filepath.Join(dir, ".vrooli")
		if info, err := os.Stat(vrooliDir); err == nil && info.IsDir() {
			return dir
		}
	}

	// 4. Fallback to relative path (assumes running from api/ directory)
	return filepath.Clean(filepath.Join(cwd, "../../.."))
}
