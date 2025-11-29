package collectors

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const defaultCategory = "utility"

// loadServiceConfig loads service.json to get scenario category
// [REQ:SCS-CORE-001] Service configuration loading
func loadServiceConfig(scenarioRoot string) ServiceConfig {
	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")

	data, err := os.ReadFile(servicePath)
	if err != nil {
		return ServiceConfig{
			Category: defaultCategory,
			Name:     filepath.Base(scenarioRoot),
			Version:  "0.0.0",
		}
	}

	var config ServiceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ServiceConfig{
			Category: defaultCategory,
			Name:     filepath.Base(scenarioRoot),
			Version:  "0.0.0",
		}
	}

	// Apply defaults
	if config.Category == "" {
		config.Category = defaultCategory
	}
	if config.Name == "" {
		config.Name = filepath.Base(scenarioRoot)
	}
	if config.Version == "" {
		config.Version = "0.0.0"
	}

	return config
}
