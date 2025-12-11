package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/signing"
	"scenario-to-desktop-api/signing/generation"
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

	// Load signing configuration from file repository if not already set
	if config.CodeSigning == nil && config.ScenarioName != "" {
		signingConfig, err := s.loadSigningConfig(config.ScenarioName)
		if err != nil {
			s.builds.Update(buildID, func(status *BuildStatus) {
				status.BuildLog = append(status.BuildLog, fmt.Sprintf("Note: Could not load signing config: %v", err))
			})
		} else if signingConfig != nil && signingConfig.Enabled {
			config.CodeSigning = signingConfig
			s.builds.Update(buildID, func(status *BuildStatus) {
				status.BuildLog = append(status.BuildLog, "Loaded signing configuration from scenario")
			})
		}
	}

	// Generate signing artifacts if code signing is enabled
	if config.CodeSigning != nil && config.CodeSigning.Enabled {
		if err := s.generateSigningArtifacts(config); err != nil {
			s.builds.Update(buildID, func(status *BuildStatus) {
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Warning: Failed to generate signing artifacts: %v", err))
			})
		} else {
			s.builds.Update(buildID, func(status *BuildStatus) {
				status.BuildLog = append(status.BuildLog, "Generated signing artifacts")
			})
		}
	}

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
					status.Metadata["total_size_bytes"] = pkgResult.TotalSizeBytes
					status.Metadata["total_size_human"] = pkgResult.TotalSizeHuman
					if pkgResult.SizeWarning != nil {
						status.Metadata["size_warning"] = pkgResult.SizeWarning
						// Log size warnings
						status.BuildLog = append(status.BuildLog,
							fmt.Sprintf("[%s] %s", pkgResult.SizeWarning.Level, pkgResult.SizeWarning.Message))
					}
				}
			}
		}

		now := time.Now()
		status.CompletedAt = &now
	})
}

// loadSigningConfig loads the signing configuration from the file repository for a scenario.
func (s *Server) loadSigningConfig(scenarioName string) (*signing.SigningConfig, error) {
	// Create a file repository to load the signing config
	repo := signing.NewFileRepository()

	ctx := context.Background()
	config, err := repo.Get(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing config: %w", err)
	}

	return config, nil
}

// generateSigningArtifacts generates signing-related files in the output directory.
// This includes:
// - entitlements.mac.plist for macOS (if macOS signing is configured)
// - scripts/notarize.js for macOS notarization (if notarization is enabled)
func (s *Server) generateSigningArtifacts(config *DesktopConfig) error {
	if config.CodeSigning == nil || !config.CodeSigning.Enabled {
		return nil
	}

	outputPath := config.OutputPath
	if outputPath == "" {
		return fmt.Errorf("output path not set")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate macOS signing artifacts
	if config.CodeSigning.MacOS != nil {
		// Generate entitlements.plist and notarize script
		opts := &generation.Options{
			OutputDir:          outputPath,
			EntitlementsPath:   "entitlements.mac.plist",
			NotarizeScriptPath: "scripts/notarize.js",
		}

		generator := generation.NewGenerator(opts)
		files, err := generator.GenerateAll(config.CodeSigning)
		if err != nil {
			return fmt.Errorf("failed to generate signing artifacts: %w", err)
		}

		// Write all generated files
		for relPath, content := range files {
			fullPath := filepath.Join(outputPath, relPath)

			// Ensure parent directory exists
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			if err := os.WriteFile(fullPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", relPath, err)
			}
		}

		// Update config to point to the generated entitlements file if not already set
		if config.CodeSigning.MacOS.EntitlementsFile == "" {
			config.CodeSigning.MacOS.EntitlementsFile = opts.EntitlementsPath
		}
	}

	return nil
}
