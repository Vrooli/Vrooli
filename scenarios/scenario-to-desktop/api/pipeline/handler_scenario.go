package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sharedpath "scenario-to-desktop-api/shared/path"
)

type bundleCleanResult struct {
	ScenarioName string
	LocationMode string
	PipelineID   string
	Path         string
	Removed      bool
}

func (h *Handler) cleanBundle(scenarioName, framework, locationMode, pipelineID string) (*bundleCleanResult, error) {
	if framework == "" {
		framework = FrameworkElectron
	}
	if framework != FrameworkElectron {
		return nil, fmt.Errorf("unsupported framework %q: only %q is supported", framework, FrameworkElectron)
	}
	if locationMode == "" {
		locationMode = "proper"
	}
	if isStagingLocation(locationMode) && strings.TrimSpace(pipelineID) == "" {
		return nil, fmt.Errorf("pipeline_id is required for staging/temp location_mode")
	}
	scenarioPath := sharedpath.ResolveScenarioRoot(scenarioName)
	config := &Config{ScenarioName: scenarioName, LocationMode: locationMode, Framework: framework}
	_, desktopPath := resolvePipelineOutputPaths(config, scenarioPath, pipelineID, framework)
	if desktopPath == "" || !strings.Contains(desktopPath, filepath.Join("platforms", framework)) {
		return nil, fmt.Errorf("refusing to clean: computed output path is unsafe")
	}
	bundlePath := filepath.Join(desktopPath, "bundle")
	removed := false
	if _, err := os.Stat(bundlePath); err == nil {
		if err := removeAllRobust(bundlePath); err != nil {
			return nil, fmt.Errorf("failed to clean bundle directory: %w", err)
		}
		removed = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat bundle directory: %w", err)
	}
	return &bundleCleanResult{ScenarioName: scenarioName, LocationMode: locationMode, PipelineID: pipelineID, Path: bundlePath, Removed: removed}, nil
}
