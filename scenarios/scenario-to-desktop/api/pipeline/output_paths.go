package pipeline

import (
	"path/filepath"
	"scenario-to-desktop-api/storagepaths"
)

func isStagingLocation(mode string) bool {
	switch mode {
	case "staging", "temp":
		return true
	default:
		return false
	}
}

// resolvePipelineOutputPaths returns the bundle output root and desktop output path.
// For staging/temp location modes, outputs go under the scenario-to-desktop cache root.
func resolvePipelineOutputPaths(config *Config, scenarioPath, pipelineID, framework string) (string, string) {
	if framework == "" {
		framework = FrameworkElectron
	}
	if config != nil && isStagingLocation(config.LocationMode) && scenarioPath != "" && pipelineID != "" {
		if locator, err := storagepaths.NewLocator(); err == nil {
			if stagingRoot, err := locator.StagingRoot(); err == nil {
				outputRoot := filepath.Join(stagingRoot, config.ScenarioName, pipelineID)
				return outputRoot, filepath.Join(outputRoot, "platforms", framework)
			}
		}
	}
	return scenarioPath, filepath.Join(scenarioPath, "platforms", framework)
}
