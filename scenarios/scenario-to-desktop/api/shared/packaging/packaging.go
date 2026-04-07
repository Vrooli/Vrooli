// Package packaging provides utilities for finding and identifying built packages.
package packaging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FS abstracts filesystem operations needed by this package.
// The default implementation uses the real os/filepath functions.
type FS interface {
	Stat(name string) (os.FileInfo, error)
	Glob(pattern string) ([]string, error)
}

// realFS implements FS using the standard library.
type realFS struct{}

func (realFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (realFS) Glob(pattern string) ([]string, error) { return filepath.Glob(pattern) }

// defaultFS is the filesystem used when no override is provided.
var defaultFS FS = realFS{}

// FindBuiltPackage finds the built package file for a specific platform.
// It searches the given directory for platform-specific installer files
// and returns the path to the most preferred match.
func FindBuiltPackage(distPath, platform string) (string, error) {
	return FindBuiltPackageWith(defaultFS, distPath, platform)
}

// FindBuiltPackageWith finds a built package using the provided filesystem.
// This is the testable core; FindBuiltPackage delegates here with the real FS.
func FindBuiltPackageWith(fs FS, distPath, platform string) (string, error) {
	// Check if dist-electron directory exists
	if _, err := fs.Stat(distPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dist-electron directory not found at %s", distPath)
	}

	// Platform-specific file patterns
	var patterns []string
	switch platform {
	case "win":
		patterns = []string{"*.msi", "*Setup.exe", "*.exe"}
	case "mac":
		patterns = []string{"*.pkg", "*.dmg", "*.zip"}
	case "linux":
		patterns = []string{"*.AppImage", "*.deb"}
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}

	// Search for matching files
	for _, pattern := range patterns {
		matches, err := fs.Glob(filepath.Join(distPath, pattern))
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			// Return the first match with platform-specific preferences
			if platform == "win" && len(matches) > 1 {
				for _, match := range matches {
					if strings.HasSuffix(strings.ToLower(match), ".msi") {
						return match, nil
					}
				}
				for _, match := range matches {
					if strings.Contains(strings.ToLower(match), "setup") {
						return match, nil
					}
				}
			}
			if platform == "mac" && len(matches) > 1 {
				for _, match := range matches {
					if strings.HasSuffix(strings.ToLower(match), ".pkg") {
						return match, nil
					}
				}
				for _, match := range matches {
					lowerMatch := strings.ToLower(match)
					if !strings.Contains(lowerMatch, "arm64") && !strings.Contains(lowerMatch, "blockmap") {
						return match, nil
					}
				}
				for _, match := range matches {
					if !strings.Contains(strings.ToLower(match), "blockmap") {
						return match, nil
					}
				}
			}
			if platform == "mac" {
				for _, match := range matches {
					if !strings.Contains(strings.ToLower(match), "blockmap") {
						return match, nil
					}
				}
			}
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no built package found for platform %s in %s", platform, distPath)
}
