package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// performDesktopGeneration performs desktop generation asynchronously
func (s *Server) performDesktopGeneration(buildID string, config *DesktopConfig) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			s.buildMutex.Lock()
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic: %v", r))
			s.buildMutex.Unlock()
			now := time.Now()
			status.CompletedAt = &now
		}
	}()

	// Create configuration JSON file
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		s.buildMutex.Lock()
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to marshal config: %v", err))
		s.buildMutex.Unlock()
		return
	}

	// Write config to temporary file
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("desktop-config-%s.json", buildID))
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		s.buildMutex.Lock()
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to write config file: %v", err))
		s.buildMutex.Unlock()
		return
	}
	defer os.Remove(configPath)

	// Execute template generator
	// Path is relative to scenario root, not api directory
	templateGeneratorPath := filepath.Join("..", "templates", "build-tools", "dist", "template-generator.js")
	cmd := exec.Command("node",
		templateGeneratorPath,
		configPath)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	s.buildMutex.Lock()
	status.BuildLog = append(status.BuildLog, outputStr)

	if err != nil {
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Generation failed: %v", err))
		status.ErrorLog = append(status.ErrorLog, outputStr)
	} else {
		status.Status = "ready"
		status.Artifacts["config_path"] = configPath
		status.Artifacts["output_path"] = config.OutputPath
	}

	now := time.Now()
	status.CompletedAt = &now
	s.buildMutex.Unlock()
}
