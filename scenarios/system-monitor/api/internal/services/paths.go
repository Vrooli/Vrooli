package services

import (
	"os"
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
		return filepath.Join(".", "initialization", "configuration")
	}
	return filepath.Join(scenarioRoot, "initialization", "configuration")
}

// ResolvePromptBasePath returns the base path for prompt templates.
func ResolvePromptBasePath() string {
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		return filepath.Join(".", "initialization", "claude-code")
	}
	return filepath.Join(scenarioRoot, "initialization", "claude-code")
}

// ResolveScriptsDir finds the investigations/active directory on disk.
func ResolveScriptsDir() string {
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, "investigations", "active")
		} else {
			return "."
		}
	}

	scriptsPath := filepath.Join(scenarioRoot, "investigations", "active")
	if info, err := os.Stat(scriptsPath); err == nil && info.IsDir() {
		return scriptsPath
	}

	repoRoot := filepath.Dir(filepath.Dir(scenarioRoot))
	return filepath.Join(repoRoot, "investigations", "active")
}
