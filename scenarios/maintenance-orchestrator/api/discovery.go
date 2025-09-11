package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func discoverScenarios(orchestrator *Orchestrator, logger *log.Logger) {
	// Use project root-relative path or environment variable
	scenariosPath := "scenarios"
	if envPath := os.Getenv("VROOLI_SCENARIOS_PATH"); envPath != "" {
		scenariosPath = envPath
	} else {
		// Try to find scenarios directory relative to project root
		if cwd, err := os.Getwd(); err == nil {
			if _, err := os.Stat(filepath.Join(cwd, "scenarios")); err == nil {
				scenariosPath = filepath.Join(cwd, "scenarios")
			}
		}
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

		serviceJsonPath := filepath.Join(scenariosPath, entry.Name(), ".vrooli", "service.json")
		data, err := ioutil.ReadFile(serviceJsonPath)
		if err != nil {
			continue
		}

		var service map[string]interface{}
		if err := json.Unmarshal(data, &service); err != nil {
			continue
		}

		tags, ok := service["tags"].([]interface{})
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
			DisplayName: getStringField(service, "displayName", entry.Name()),
			Description: getStringField(service, "description", ""),
			IsActive:    false,
			Tags:        stringTags,
		}

		if ports, ok := service["ports"].(map[string]interface{}); ok {
			if apiPort, ok := ports["api"].(float64); ok {
				scenario.Port = int(apiPort)
				scenario.Endpoint = fmt.Sprintf("http://localhost:%d", int(apiPort))
			}
		}

		orchestrator.AddScenario(scenario)
		logger.Printf("Discovered maintenance scenario: %s", scenario.ID)
	}

	orchestrator.UpdatePresetStates()
}

func getStringField(m map[string]interface{}, field, defaultValue string) string {
	if val, ok := m[field].(string); ok {
		return val
	}
	return defaultValue
}