package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	types "scenario-dependency-analyzer/internal/types"
)

// LoadServiceConfig loads and parses a scenario's service.json file.
func LoadServiceConfig(scenarioPath string) (*types.ServiceConfig, error) {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario %s not found or missing service.json", filepath.Base(scenarioPath))
	}

	data, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service.json: %w", err)
	}

	var cfg types.ServiceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse service.json: %w", err)
	}

	return &cfg, nil
}

// ResolvedResourceMap returns the resources from a service config, preferring
// dependencies.resources over the legacy resources field.
func ResolvedResourceMap(cfg *types.ServiceConfig) map[string]types.Resource {
	if cfg.Dependencies.Resources != nil && len(cfg.Dependencies.Resources) > 0 {
		return cfg.Dependencies.Resources
	}
	if cfg.Resources == nil {
		return map[string]types.Resource{}
	}
	return cfg.Resources
}

// NormalizeName returns a lowercased, trimmed name for consistent comparisons.
func NormalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}
