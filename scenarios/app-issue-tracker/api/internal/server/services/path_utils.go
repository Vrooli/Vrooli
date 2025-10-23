package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveScenarioPath ensures raw references stay within the scenario root or
// explicitly permitted subdirectories and returns the normalized absolute path.
func ResolveScenarioPath(scenarioRoot, raw string, allowedSubdirs ...string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("empty path")
	}

	candidate := trimmed
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(scenarioRoot, candidate)
	}
	candidate = filepath.Clean(candidate)

	allowedBases := make([]string, 0, len(allowedSubdirs)+1)
	allowedBases = append(allowedBases, filepath.Clean(scenarioRoot))
	for _, subdir := range allowedSubdirs {
		allowedBases = append(allowedBases, filepath.Clean(filepath.Join(scenarioRoot, subdir)))
	}

	for _, base := range allowedBases {
		rel, err := filepath.Rel(base, candidate)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !strings.HasPrefix(rel, fmt.Sprintf("..%c", os.PathSeparator))) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("path %s is outside the allowed scenario directories", candidate)
}
