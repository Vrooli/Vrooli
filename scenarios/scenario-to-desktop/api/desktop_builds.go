package main

import (
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// standardOutputPath returns the canonical electron output path for a scenario.
func (s *Server) standardOutputPath(appName string) string {
	return filepath.Join(
		detectVrooliRoot(),
		"scenarios",
		appName,
		"platforms",
		"electron",
	)
}

// stagingOutputPath returns the gitignored staging area for temporary desktop outputs.
func (s *Server) stagingOutputPath(appName, buildID string) string {
	return filepath.Join(
		detectVrooliRoot(),
		"scenarios",
		"scenario-to-desktop",
		"data",
		"staging",
		appName,
		buildID,
	)
}

// scenarioRoot returns the absolute path to a scenario directory.
func (s *Server) scenarioRoot(appName string) string {
	return filepath.Join(detectVrooliRoot(), "scenarios", appName)
}

// queueDesktopBuild centralizes build status creation and kicks off generation.
// Handlers own HTTP concerns; this keeps the domain flow (status setup, metadata,
// background launch) in one place so we don't duplicate it across entrypoints.
func (s *Server) queueDesktopBuild(config *DesktopConfig, metadata *ScenarioMetadata, autoDetected bool) *BuildStatus {
	buildID := uuid.New().String()

	outputPath, destinationPath := s.resolveOutputPath(config, buildID)
	config.OutputPath = outputPath

	buildStatus := &BuildStatus{
		BuildID:      buildID,
		ScenarioName: config.AppName,
		Status:       "building",
		Framework:    config.Framework,
		TemplateType: config.TemplateType,
		Platforms:    config.Platforms,
		OutputPath:   config.OutputPath,
		CreatedAt:    time.Now(),
		BuildLog:     []string{},
		ErrorLog:     []string{},
		Artifacts:    map[string]string{},
		Metadata:     map[string]interface{}{},
	}

	if metadata != nil {
		buildStatus.Metadata["auto_detected"] = autoDetected
		buildStatus.Metadata["ui_dist_path"] = metadata.UIDistPath
		buildStatus.Metadata["has_api"] = config.ServerType == "external"
		buildStatus.Metadata["category"] = metadata.Category
		buildStatus.Metadata["source_version"] = metadata.Version
	} else if autoDetected {
		buildStatus.Metadata["auto_detected"] = true
	}
	if config.LocationMode != "" {
		buildStatus.Metadata["location_mode"] = config.LocationMode
	}
	buildStatus.Metadata["destination_path"] = destinationPath
	buildStatus.Metadata["destination_path"] = destinationPath

	s.builds.Save(buildStatus)
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
		// Only set staging/custom hints when applicable to keep records clean.
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
	go s.performDesktopGeneration(buildID, config)

	return buildStatus
}

func (s *Server) resolveOutputPath(config *DesktopConfig, buildID string) (string, string) {
	destinationPath := s.standardOutputPath(config.AppName)
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
		// Fall back to the destination to avoid nil paths even if caller forgot output.
		return destinationPath, destinationPath
	default: // "proper"
		if config.OutputPath != "" {
			return config.OutputPath, destinationPath
		}
		return destinationPath, destinationPath
	}
}
