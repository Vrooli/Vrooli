package main

import (
	"path/filepath"
	"strings"

	docshandler "landing-page-business-suite-api/handlers/docs"
	"landing-page-business-suite-api/internal/envx"
)

// getDocsRoot returns the absolute path to the docs directory
func getDocsRoot() string {
	// Check for explicit override first
	if override := strings.TrimSpace(envx.Get("DOCS_ROOT")); override != "" {
		if abs, err := filepath.Abs(override); err == nil {
			return abs
		}
		return override
	}

	// Check for scenario root
	if scenarioRoot := strings.TrimSpace(envx.Get("SCENARIO_ROOT")); scenarioRoot != "" {
		return filepath.Join(scenarioRoot, "docs")
	}

	// Fallback: API runs from api/ subdirectory, so go up one level
	// Always resolve to absolute path for consistent behavior
	if abs, err := filepath.Abs(filepath.Join("..", "docs")); err == nil {
		return abs
	}
	return filepath.Join("..", "docs")
}

func docsConnectDependencies() docshandler.ConnectDependencies {
	return docshandler.ConnectDependencies{DocsRoot: getDocsRoot, Log: logStructured}
}
