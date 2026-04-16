package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

func discoverScenarios(orchestrator *Orchestrator, logger *log.Logger) {
	scenariosPath, err := resolveScenariosPath()
	if err != nil {
		logger.Printf("Error resolving scenarios directory: %v", err)
		return
	}

	entries, err := ioutil.ReadDir(scenariosPath)
	if err != nil {
		logger.Printf("Error reading scenarios directory: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serviceJsonPath, err := resolveServicePath(scenariosPath, entry.Name())
		if err != nil {
			continue
		}
		data, err := ioutil.ReadFile(serviceJsonPath)
		if err != nil {
			continue
		}

		var service map[string]interface{}
		if err := json.Unmarshal(data, &service); err != nil {
			continue
		}

		// Extract the service object first
		serviceData, ok := service["service"].(map[string]interface{})
		if !ok {
			continue
		}

		tags, ok := serviceData["tags"].([]interface{})
		if !ok {
			continue
		}

		hasMaintenance := false
		stringTags := make([]string, 0)
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				stringTags = append(stringTags, tagStr)
				if tagStr == "maintenance" {
					hasMaintenance = true
				}
			}
		}

		if !hasMaintenance {
			continue
		}

		scenario := &MaintenanceScenario{
			ID:          entry.Name(),
			Name:        entry.Name(),
			DisplayName: getStringField(serviceData, "displayName", entry.Name()),
			Description: getStringField(serviceData, "description", ""),
			IsActive:    false,
			Tags:        stringTags,
		}

		// Ports are at the root level, not in service object
		if ports, ok := service["ports"].(map[string]interface{}); ok {
			// Look for the api port configuration
			if apiPortConfig, ok := ports["api"].(map[string]interface{}); ok {
				// Check for static port first, then range
				if apiPort, ok := apiPortConfig["port"].(float64); ok {
					scenario.Port = int(apiPort)
					scenario.Endpoint = fmt.Sprintf("http://localhost:%d", int(apiPort))
				} else if rangeStr, ok := apiPortConfig["range"].(string); ok {
					// For now, we'll skip dynamic ports since we need the actual allocated port
					// The scenario would need to be running and we'd need to query its actual port
					logger.Printf("Scenario %s uses dynamic port range: %s", entry.Name(), rangeStr)
				}
			}
		}

		orchestrator.AddScenario(scenario)
		logger.Printf("Discovered maintenance scenario: %s", scenario.ID)
	}

	orchestrator.UpdatePresetStates()
}

func resolveScenariosPath() (string, error) {
	if envPath := strings.TrimSpace(os.Getenv("VROOLI_SCENARIOS_PATH")); envPath != "" {
		return filepath.Abs(envPath)
	}

	if rootOverride := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); rootOverride != "" {
		if root, err := repocontract.FindRepoRoot(rootOverride); err == nil {
			if path, resolveErr := resolveScenariosPathFromRepoRoot(root); resolveErr == nil {
				return path, nil
			}
		}
		return filepath.Join(filepath.Clean(rootOverride), "scenarios"), nil
	}

	if cwd, err := os.Getwd(); err == nil {
		if _, findErr := repocontract.FindRepoRoot(cwd); findErr != nil {
			return filepath.Join(cwd, "scenarios"), nil
		}
	}

	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		if path, resolveErr := resolveScenariosPathFromRepoRoot(root); resolveErr == nil {
			return path, nil
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "scenarios"), nil
	}

	return "", fmt.Errorf("resolve scenarios path: unable to determine repo root or working directory")
}

func resolveScenariosPathFromRepoRoot(root string) (string, error) {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", err
	}
	return contract.TopLevelDir(root, "scenarios")
}

func resolveServicePath(scenariosPath, scenarioName string) (string, error) {
	repoRoot := filepath.Dir(filepath.Clean(scenariosPath))
	if path, err := repocontract.ResolveScenarioFile(repoRoot, scenarioName, "service"); err == nil {
		return path, nil
	}
	return filepath.Join(scenariosPath, scenarioName, ".vrooli", "service.json"), nil
}

func getStringField(m map[string]interface{}, field, defaultValue string) string {
	if val, ok := m[field].(string); ok {
		return val
	}
	return defaultValue
}
