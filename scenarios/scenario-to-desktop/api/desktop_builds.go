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

// scenarioRoot returns the absolute path to a scenario directory.
func (s *Server) scenarioRoot(appName string) string {
	return filepath.Join(detectVrooliRoot(), "scenarios", appName)
}

// queueDesktopBuild centralizes build status creation and kicks off generation.
// Handlers own HTTP concerns; this keeps the domain flow (status setup, metadata,
// background launch) in one place so we don't duplicate it across entrypoints.
func (s *Server) queueDesktopBuild(config *DesktopConfig, metadata *ScenarioMetadata, autoDetected bool) *BuildStatus {
	if config.OutputPath == "" {
		config.OutputPath = s.standardOutputPath(config.AppName)
		s.logger.Info("using standard output path",
			"scenario", config.AppName,
			"path", config.OutputPath)
	}

	buildID := uuid.New().String()

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

	s.builds.Save(buildStatus)
	go s.performDesktopGeneration(buildID, config)

	return buildStatus
}
