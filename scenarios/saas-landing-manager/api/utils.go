package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateScenarioPath validates and sanitizes a scenario path to prevent path traversal attacks.
// Returns the cleaned path and an error if the path contains potentially malicious patterns.
func validateScenarioPath(scenarioPath string) (string, error) {
	cleanPath := filepath.Clean(scenarioPath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid scenario path: contains path traversal pattern")
	}
	return cleanPath, nil
}
