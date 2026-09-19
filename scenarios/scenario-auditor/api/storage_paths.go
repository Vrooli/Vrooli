package main

import (
	"fmt"

	"github.com/vrooli/api-core/storage"
)

const scenarioAuditorScenarioID = "scenario-auditor"

func resolveScenarioAuditorStoragePath(class storage.Class, rel string) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}

	opts := storage.Options{ScenarioID: scenarioAuditorScenarioID}
	if _, err := storage.EnsureClassDir(resolver, opts, class, 0); err != nil {
		return "", fmt.Errorf("ensure %s storage dir: %w", class, err)
	}

	path, err := resolver.Path(opts, class, rel)
	if err != nil {
		return "", fmt.Errorf("resolve %s storage path %q: %w", class, rel, err)
	}
	return path, nil
}
