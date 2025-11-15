package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findBuiltPackage finds the built package file for a specific platform
func (s *Server) findBuiltPackage(distPath, platform string) (string, error) {
	// Check if dist-electron directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dist-electron directory not found at %s", distPath)
	}

	// Platform-specific file patterns
	var patterns []string
	switch platform {
	case "win":
		patterns = []string{"*.exe"}
	case "mac":
		// Accept both DMG (macOS builds) and ZIP (Linux cross-compilation)
		patterns = []string{"*.dmg", "*.zip"}
	case "linux":
		patterns = []string{"*.AppImage", "*.deb"}
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}

	// Search for matching files
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(distPath, pattern))
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			// Return the first match (usually there's only one)
			// Prefer Setup.exe over portable.exe for Windows
			if platform == "win" && len(matches) > 1 {
				for _, match := range matches {
					if strings.Contains(strings.ToLower(match), "setup") {
						return match, nil
					}
				}
			}
			// For Mac, prefer x64 over arm64 when both exist, and non-arm64 zip files
			if platform == "mac" && len(matches) > 1 {
				// First, try to find x64-specific file (not arm64)
				for _, match := range matches {
					lowerMatch := strings.ToLower(match)
					if !strings.Contains(lowerMatch, "arm64") && !strings.Contains(lowerMatch, "blockmap") {
						return match, nil
					}
				}
				// If only arm64 exists, return the first non-blockmap one
				for _, match := range matches {
					if !strings.Contains(strings.ToLower(match), "blockmap") {
						return match, nil
					}
				}
			}
			// Filter out blockmap files for Mac
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
