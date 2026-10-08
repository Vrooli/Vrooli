package config

import (
	"fmt"
	"path/filepath"
	"strings"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// LoadServiceConfig loads and parses a scenario's service.json file.
func LoadServiceConfig(scenarioPath string) (*types.Manifest, error) {
	scenarioPath = filepath.Clean(scenarioPath)
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	cfg, err := scenariomodel.ReadService(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("scenario %s has no readable canonical service manifest: %w", filepath.Base(scenarioPath), err)
	}

	return &cfg, nil
}

// ResolvedResourceMap returns the canonical dependencies.resources declaration.
func ResolvedResourceMap(cfg *types.Manifest) map[string]types.Resource {
	if cfg == nil || cfg.Dependencies.Resources == nil {
		return map[string]types.Resource{}
	}
	return cfg.Dependencies.Resources
}

// NormalizeName returns a lowercased, trimmed name for consistent comparisons.
func NormalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}
