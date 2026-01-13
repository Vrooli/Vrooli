package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"scenario-to-desktop-api/signing"
	signinggeneration "scenario-to-desktop-api/signing/generation"
)

// DefaultService is the default implementation of the generation Service.
type DefaultService struct {
	vrooliRoot   string
	templateDir  string
	builds       BuildStore
	records      RecordStore
	analyzer     ScenarioAnalyzer
	iconSyncer   IconSyncer
	bundlePackager BundlePackager
	logger       *slog.Logger
}

// RecordStore persists desktop app records.
type RecordStore interface {
	Upsert(record *DesktopAppRecord) error
}

// DesktopAppRecord represents a persisted desktop generation record.
type DesktopAppRecord struct {
	ID              string
	BuildID         string
	ScenarioName    string
	AppDisplayName  string
	TemplateType    string
	Framework       string
	LocationMode    string
	OutputPath      string
	DestinationPath string
	StagingPath     string
	CustomPath      string
	DeploymentMode  string
	Icon            string
}

// BundlePackager packages bundles for bundled deployment mode.
type BundlePackager interface {
	Package(outputPath, manifestPath string, platforms []string) (*BundlePackageResult, error)
}

// BundlePackageResult contains bundle packaging results.
type BundlePackageResult struct {
	BundleDir       string
	ManifestPath    string
	RuntimeBinaries []string
	TotalSizeBytes  int64
	TotalSizeHuman  string
	SizeWarning     *SizeWarning
}

// SizeWarning indicates a potential issue with bundle size.
type SizeWarning struct {
	Level   string
	Message string
}

// ServiceOption configures a DefaultService.
type ServiceOption func(*DefaultService)

// WithVrooliRoot sets the vrooli root directory.
func WithVrooliRoot(root string) ServiceOption {
	return func(s *DefaultService) {
		s.vrooliRoot = root
	}
}

// WithTemplateDir sets the template directory.
func WithTemplateDir(dir string) ServiceOption {
	return func(s *DefaultService) {
		s.templateDir = dir
	}
}

// WithBuildStore sets the build store.
func WithBuildStore(store BuildStore) ServiceOption {
	return func(s *DefaultService) {
		s.builds = store
	}
}

// WithRecordStore sets the record store.
func WithRecordStore(store RecordStore) ServiceOption {
	return func(s *DefaultService) {
		s.records = store
	}
}

// WithAnalyzer sets the scenario analyzer.
func WithAnalyzer(analyzer ScenarioAnalyzer) ServiceOption {
	return func(s *DefaultService) {
		s.analyzer = analyzer
	}
}

// WithIconSyncer sets the icon syncer.
func WithIconSyncer(syncer IconSyncer) ServiceOption {
	return func(s *DefaultService) {
		s.iconSyncer = syncer
	}
}

// WithBundlePackager sets the bundle packager.
func WithBundlePackager(packager BundlePackager) ServiceOption {
	return func(s *DefaultService) {
		s.bundlePackager = packager
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) ServiceOption {
	return func(s *DefaultService) {
		s.logger = logger
	}
}

// NewService creates a new generation service.
func NewService(opts ...ServiceOption) *DefaultService {
	s := &DefaultService{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}

	// Create default analyzer if not provided
	if s.analyzer == nil && s.vrooliRoot != "" {
		s.analyzer = NewAnalyzer(s.vrooliRoot)
	}

	// Create default icon syncer if not provided
	if s.iconSyncer == nil {
		s.iconSyncer = NewIconSyncer(s.vrooliRoot)
	}

	return s
}

// QueueBuild queues a desktop build and returns the initial status.
func (s *DefaultService) QueueBuild(config *DesktopConfig, metadata *ScenarioMetadata, includeMetadata bool) *BuildStatus {
	buildID := uuid.New().String()

	outputPath, destinationPath := s.resolveOutputPath(config, buildID)
	config.OutputPath = outputPath

	buildStatus := &BuildStatus{
		BuildID:     buildID,
		Status:      "building",
		OutputPath:  config.OutputPath,
		StartedAt:   time.Now(),
		BuildLog:    []string{},
		ErrorLog:    []string{},
		Artifacts:   map[string]string{},
		Metadata:    map[string]interface{}{},
	}

	if metadata != nil {
		buildStatus.Metadata["auto_detected"] = includeMetadata
		buildStatus.Metadata["ui_dist_path"] = metadata.UIDistPath
		buildStatus.Metadata["has_api"] = config.ServerType == "external"
		buildStatus.Metadata["category"] = metadata.Category
		buildStatus.Metadata["source_version"] = metadata.Version
	} else if includeMetadata {
		buildStatus.Metadata["auto_detected"] = true
	}
	if config.LocationMode != "" {
		buildStatus.Metadata["location_mode"] = config.LocationMode
	}
	buildStatus.Metadata["destination_path"] = destinationPath

	if s.builds != nil {
		// Create the build entry first (Update only works for existing builds)
		s.builds.Create(buildID)
		// Then update with the full status including metadata
		s.builds.Update(buildID, func(status *BuildStatus) {
			status.OutputPath = buildStatus.OutputPath
			status.StartedAt = buildStatus.StartedAt
			status.BuildLog = buildStatus.BuildLog
			status.ErrorLog = buildStatus.ErrorLog
			status.Artifacts = buildStatus.Artifacts
			status.Metadata = buildStatus.Metadata
		})
	}

	// Persist record
	if s.records != nil {
		record := &DesktopAppRecord{
			ID:              buildID,
			BuildID:         buildID,
			ScenarioName:    config.AppName,
			AppDisplayName:  config.AppDisplayName,
			TemplateType:    config.TemplateType,
			Framework:       config.Framework,
			LocationMode:    config.LocationMode,
			OutputPath:      outputPath,
			DestinationPath: destinationPath,
			DeploymentMode:  config.DeploymentMode,
			Icon:            config.Icon,
		}
		switch config.LocationMode {
		case "temp", "staging":
			record.StagingPath = outputPath
		case "custom":
			record.CustomPath = outputPath
		}
		if err := s.records.Upsert(record); err != nil {
			s.logger.Warn("failed to persist desktop record", "error", err)
		}
	}

	go s.Generate(buildID, config)

	return buildStatus
}

// Generate generates a desktop application from a config.
func (s *DefaultService) Generate(buildID string, config *DesktopConfig) {
	defer func() {
		if r := recover(); r != nil {
			s.updateBuildStatus(buildID, func(status *BuildStatus) {
				status.Status = "failed"
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic: %v", r))
				now := time.Now()
				status.CompletedAt = &now
			})
		}
	}()

	// Load signing configuration if not already set
	if config.CodeSigning == nil && config.ScenarioName != "" {
		signingConfig, err := s.loadSigningConfig(config.ScenarioName)
		if err != nil {
			s.updateBuildStatus(buildID, func(status *BuildStatus) {
				status.BuildLog = append(status.BuildLog, fmt.Sprintf("Note: Could not load signing config: %v", err))
			})
		} else if signingConfig != nil && signingConfig.Enabled {
			config.CodeSigning = signingConfig
			s.updateBuildStatus(buildID, func(status *BuildStatus) {
				status.BuildLog = append(status.BuildLog, "Loaded signing configuration from scenario")
			})
		}
	}

	// Generate signing artifacts if code signing is enabled
	if config.CodeSigning != nil && config.CodeSigning.Enabled {
		if err := s.generateSigningArtifacts(config); err != nil {
			s.updateBuildStatus(buildID, func(status *BuildStatus) {
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Warning: Failed to generate signing artifacts: %v", err))
			})
		} else {
			s.updateBuildStatus(buildID, func(status *BuildStatus) {
				status.BuildLog = append(status.BuildLog, "Generated signing artifacts")
			})
		}
	}

	// Create configuration JSON file
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		s.updateBuildStatus(buildID, func(status *BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to marshal config: %v", err))
		})
		return
	}

	// Write config to temporary file
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("desktop-config-%s.json", buildID))
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		s.updateBuildStatus(buildID, func(status *BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to write config file: %v", err))
		})
		return
	}
	defer os.Remove(configPath)

	// Execute template generator
	templateGeneratorPath := filepath.Join(s.templateDir, "build-tools", "dist", "template-generator.js")
	cmd := exec.Command("node", templateGeneratorPath, configPath)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	s.updateBuildStatus(buildID, func(status *BuildStatus) {
		status.BuildLog = append(status.BuildLog, outputStr)

		if err != nil {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Generation failed: %v", err))
			status.ErrorLog = append(status.ErrorLog, outputStr)
		} else {
			status.Status = "ready"
			status.Artifacts["config_path"] = configPath
			status.Artifacts["output_path"] = config.OutputPath

			// Sync scenario icons
			if s.iconSyncer != nil {
				if err := s.iconSyncer.SyncIcons(config.ScenarioName, config.OutputPath, func(msg string, fields map[string]interface{}) {
					s.logger.Info(msg, "fields", fields)
				}); err != nil {
					status.BuildLog = append(status.BuildLog, fmt.Sprintf("Icon sync warning: %v", err))
				}
			}

			// Package bundle if bundled deployment mode
			if config.DeploymentMode == "bundled" && config.BundleManifestPath != "" && s.bundlePackager != nil {
				pkgResult, pkgErr := s.bundlePackager.Package(config.OutputPath, config.BundleManifestPath, config.Platforms)
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

// GetAnalyzer returns the scenario analyzer.
func (s *DefaultService) GetAnalyzer() ScenarioAnalyzer {
	return s.analyzer
}

// StandardOutputPath returns the canonical electron output path for a scenario.
func (s *DefaultService) StandardOutputPath(appName string) string {
	return filepath.Join(s.vrooliRoot, "scenarios", appName, "platforms", "electron")
}

// ScenarioRoot returns the absolute path to a scenario directory.
func (s *DefaultService) ScenarioRoot(appName string) string {
	return filepath.Join(s.vrooliRoot, "scenarios", appName)
}

// updateBuildStatus updates the build status in the store.
func (s *DefaultService) updateBuildStatus(buildID string, fn func(status *BuildStatus)) {
	if s.builds == nil {
		return
	}
	s.builds.Update(buildID, fn)
}

// resolveOutputPath determines where to write the generated app.
func (s *DefaultService) resolveOutputPath(config *DesktopConfig, buildID string) (string, string) {
	destinationPath := s.StandardOutputPath(config.AppName)
	mode := config.LocationMode
	if mode == "" {
		mode = "proper"
	}

	switch mode {
	case "temp", "staging":
		return s.stagingOutputPath(config.AppName, buildID), destinationPath
	case "custom":
		if config.OutputPath != "" {
			return config.OutputPath, destinationPath
		}
		return destinationPath, destinationPath
	default: // "proper"
		if config.OutputPath != "" {
			return config.OutputPath, destinationPath
		}
		return destinationPath, destinationPath
	}
}

// stagingOutputPath returns the gitignored staging area for temporary desktop outputs.
func (s *DefaultService) stagingOutputPath(appName, buildID string) string {
	return filepath.Join(
		s.vrooliRoot,
		"scenarios",
		"scenario-to-desktop",
		"data",
		"staging",
		appName,
		buildID,
	)
}

// loadSigningConfig loads the signing configuration from the file repository for a scenario.
func (s *DefaultService) loadSigningConfig(scenarioName string) (*signing.SigningConfig, error) {
	repo := signing.NewFileRepository()
	ctx := context.Background()
	config, err := repo.Get(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing config: %w", err)
	}
	return config, nil
}

// generateSigningArtifacts generates signing-related files in the output directory.
func (s *DefaultService) generateSigningArtifacts(config *DesktopConfig) error {
	if config.CodeSigning == nil || !config.CodeSigning.Enabled {
		return nil
	}

	outputPath := config.OutputPath
	if outputPath == "" {
		return fmt.Errorf("output path not set")
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if config.CodeSigning.MacOS != nil {
		opts := &signinggeneration.Options{
			OutputDir:          outputPath,
			EntitlementsPath:   "entitlements.mac.plist",
			NotarizeScriptPath: "scripts/notarize.js",
		}

		generator := signinggeneration.NewGenerator(opts)
		files, err := generator.GenerateAll(config.CodeSigning)
		if err != nil {
			return fmt.Errorf("failed to generate signing artifacts: %w", err)
		}

		for relPath, content := range files {
			fullPath := filepath.Join(outputPath, relPath)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			if err := os.WriteFile(fullPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", relPath, err)
			}
		}

		if config.CodeSigning.MacOS.EntitlementsFile == "" {
			config.CodeSigning.MacOS.EntitlementsFile = "entitlements.mac.plist"
		}
	}

	return nil
}
