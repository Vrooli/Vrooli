package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	types "scenario-dependency-analyzer/internal/types"
)

func loadServiceConfigFromFile(scenarioPath string) (*types.ServiceConfig, error) {
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
