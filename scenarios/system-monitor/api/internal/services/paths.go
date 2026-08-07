package services

import (
	"path/filepath"

	"github.com/vrooli/api-core/storage"
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
	return filepath.Join(scenarioRoot, ".vrooli")
}

// ResolveRuntimeStateBasePath returns the api-core storage directory for
// mutable system-monitor settings. Runtime state must not be written into the
// repository working tree.
func ResolveRuntimeStateBasePath() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return ""
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "system-monitor"})
	if err != nil {
		return ""
	}
	return paths.StateDir
}

// ResolvePromptBasePath returns the base path for prompt templates.
func ResolvePromptBasePath() string {
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		return ""
	}
	return filepath.Join(scenarioRoot, "prompts")
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
