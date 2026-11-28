package pipeline

import (
	"path/filepath"

	"scenario-to-desktop-api/generation"
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
func resolvePipelineOutputPaths(config *PipelineConfig, scenarioPath, pipelineID, strategyID string) (string, string) {
	if strategyID == "" && config != nil {
		strategyID = config.StrategyID
	}
	if strategyID == "" {
		strategyID = generation.DefaultStrategyID
	}
	outputName, err := generation.StrategyOutputName(strategyID)
	if err != nil {
		return "", ""
	}
	if config != nil && isStagingLocation(config.LocationMode) && scenarioPath != "" && pipelineID != "" {
		if locator, err := storagepaths.NewLocator(); err == nil {
			if stagingRoot, err := locator.StagingRoot(); err == nil {
				outputRoot := filepath.Join(stagingRoot, config.ScenarioName, pipelineID)
				return outputRoot, filepath.Join(outputRoot, "platforms", outputName)
			}
		}
	}
	return scenarioPath, filepath.Join(scenarioPath, "platforms", outputName)
}
