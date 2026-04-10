package pipeline

import "path/filepath"

func isStagingLocation(mode string) bool {
	switch mode {
	case "staging", "temp":
		return true
	default:
		return false
	}
}

// resolvePipelineOutputPaths returns the bundle output root and desktop output path.
// For staging/temp location modes, outputs go under scenario-to-desktop/data/staging/<scenario>/<pipelineID>.
func resolvePipelineOutputPaths(config *Config, scenarioPath, pipelineID, framework string) (string, string) {
	if framework == "" {
		framework = FrameworkElectron
	}
	if config != nil && isStagingLocation(config.LocationMode) && scenarioPath != "" && pipelineID != "" {
		scenariosRoot := filepath.Dir(scenarioPath)
		stagingRoot := filepath.Join(scenariosRoot, "scenario-to-desktop", "data", "staging", config.ScenarioName, pipelineID)
		return stagingRoot, filepath.Join(stagingRoot, "platforms", framework)
	}
	return scenarioPath, filepath.Join(scenarioPath, "platforms", framework)
}
