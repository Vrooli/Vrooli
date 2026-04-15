package services

import (
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

func resolveSystemMonitorScenarioRoot() (string, error) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return "", err
	}
	return repocontract.ResolveScenarioPath(repoRoot, "system-monitor")
}

// ResolveConfigBasePath returns the base path for configuration files.
func ResolveConfigBasePath() string {
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		return ""
	}
	return filepath.Join(scenarioRoot, "initialization", "configuration")
}

// ResolvePromptBasePath returns the base path for prompt templates.
func ResolvePromptBasePath() string {
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		return ""
	}
	return filepath.Join(scenarioRoot, "initialization", "claude-code")
}

// ResolveScriptsDir finds the investigations/active directory on disk.
func ResolveScriptsDir() string {
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		return ""
	}

	scriptsPath := filepath.Join(scenarioRoot, "investigations", "active")
	return scriptsPath
}
