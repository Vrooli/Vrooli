package services

import (
	"path/filepath"

	"github.com/vrooli/api-core/storage"
)

func resolveSystemMonitorStorage() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return storage.Paths{}, err
	}
	return resolver.Resolve(storage.Options{ScenarioID: "system-monitor"})
}

// ResolveConfigBasePath returns the base path for configuration files.
func ResolveConfigBasePath() string {
	paths, err := resolveSystemMonitorStorage()
	if err != nil {
		return ""
	}
	return paths.ConfigDir
}

// ResolveRuntimeStateBasePath returns the api-core storage directory for
// mutable system-monitor settings. Runtime state must not be written into the
// repository working tree.
func ResolveRuntimeStateBasePath() string {
	paths, err := resolveSystemMonitorStorage()
	if err != nil {
		return ""
	}
	return paths.StateDir
}

// ResolvePromptBasePath returns the base path for prompt templates.
func ResolvePromptBasePath() string {
	paths, err := resolveSystemMonitorStorage()
	if err != nil {
		return ""
	}
	return filepath.Join(paths.ConfigDir, "prompts")
}
