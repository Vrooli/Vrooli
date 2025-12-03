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
	_, exists := s.builds.Get(buildID)
	if !exists {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			s.builds.Update(buildID, func(status *BuildStatus) {
				status.Status = "failed"
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic: %v", r))
				now := time.Now()
				status.CompletedAt = &now
			})
		}
	}()

	// Create configuration JSON file
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		s.builds.Update(buildID, func(status *BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to marshal config: %v", err))
		})
		return
	}

	// Write config to temporary file
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("desktop-config-%s.json", buildID))
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		s.builds.Update(buildID, func(status *BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to write config file: %v", err))
		})
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

	s.builds.Update(buildID, func(status *BuildStatus) {
		status.BuildLog = append(status.BuildLog, outputStr)

		if err != nil {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Generation failed: %v", err))
			status.ErrorLog = append(status.ErrorLog, outputStr)
		} else {
			status.Status = "ready"
			status.Artifacts["config_path"] = configPath
			status.Artifacts["output_path"] = config.OutputPath
			if config.DeploymentMode == "bundled" && config.BundleManifestPath != "" {
				pkgResult, pkgErr := packageBundle(config.OutputPath, config.BundleManifestPath, config.Platforms)
				if pkgErr != nil {
					status.Status = "failed"
					status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Bundle packaging failed: %v", pkgErr))
				} else {
					status.Metadata["bundle_dir"] = pkgResult.BundleDir
					status.Metadata["bundle_manifest"] = pkgResult.ManifestPath
					status.Metadata["runtime_binaries"] = pkgResult.RuntimeBinaries
				}
			}
		}

		now := time.Now()
		status.CompletedAt = &now
	})
}
