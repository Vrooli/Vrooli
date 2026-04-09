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
	if _, err := fs.Stat(distPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dist-electron directory not found at %s", distPath)
	}

	patterns, err := platformPatterns(platform)
	if err != nil {
		return "", err
	}

	for _, pattern := range patterns {
		matches, err := fs.Glob(filepath.Join(distPath, pattern))
		if err != nil || len(matches) == 0 {
			continue
		}
		if preferred := selectPreferred(platform, matches); preferred != "" {
			return preferred, nil
		}
		return matches[0], nil
	}

	return "", fmt.Errorf("no built package found for platform %s in %s", platform, distPath)
}

// platformPatterns returns the file glob patterns for a given platform.
func platformPatterns(platform string) ([]string, error) {
	switch platform {
	case "win":
		return []string{"*.msi", "*Setup.exe", "*.exe"}, nil
	case "mac":
		return []string{"*.pkg", "*.dmg", "*.zip"}, nil
	case "linux":
		return []string{"*.AppImage", "*.deb"}, nil
	default:
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}
}

// selectPreferred picks the best match from multiple candidates using
// platform-specific preference rules. Returns "" if no preference applies.
func selectPreferred(platform string, matches []string) string {
	switch platform {
	case "win":
		return selectPreferredWin(matches)
	case "mac":
		return selectPreferredMac(matches)
	default:
		return ""
	}
}

// selectPreferredWin prefers .msi, then *setup*.exe among Windows matches.
func selectPreferredWin(matches []string) string {
	if len(matches) <= 1 {
		return ""
	}
	for _, match := range matches {
		if strings.HasSuffix(strings.ToLower(match), ".msi") {
			return match
		}
	}
	for _, match := range matches {
		if strings.Contains(strings.ToLower(match), "setup") {
			return match
		}
	}
	return ""
}

// selectPreferredMac prefers .pkg, then non-arm64/non-blockmap, then non-blockmap.
func selectPreferredMac(matches []string) string {
	if len(matches) > 1 {
		for _, match := range matches {
			if strings.HasSuffix(strings.ToLower(match), ".pkg") {
				return match
			}
		}
		for _, match := range matches {
			lowerMatch := strings.ToLower(match)
			if !strings.Contains(lowerMatch, "arm64") && !strings.Contains(lowerMatch, "blockmap") {
				return match
			}
		}
	}
	// For any number of mac matches, skip blockmaps
	for _, match := range matches {
		if !strings.Contains(strings.ToLower(match), "blockmap") {
			return match
		}
	}
	return ""
}
