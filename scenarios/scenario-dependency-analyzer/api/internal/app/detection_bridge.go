package app

import (
	"fmt"
	"sync"

	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/detection"
	types "scenario-dependency-analyzer/internal/types"
)

var (
	fallbackDetector     *detection.Detector
	fallbackDetectorOnce sync.Once
)

func detectorInstance() *detection.Detector {
	if analyzer := analyzerInstance(); analyzer != nil && analyzer.Detector() != nil {
		return analyzer.Detector()
	}
	fallbackDetectorOnce.Do(func() {
		cfg := appconfig.Load()
		fallbackDetector = detection.New(cfg)
	})
	return fallbackDetector
}

func refreshDependencyCatalogs() {
	if det := detectorInstance(); det != nil {
		det.RefreshCatalogs()
	}
}

func isKnownScenario(name string) bool {
	det := detectorInstance()
	if det == nil {
		return true
	}
	return det.KnownScenario(name)
}

func isKnownResource(name string) bool {
	det := detectorInstance()
	if det == nil {
		return true
	}
	return det.KnownResource(name)
}

func scanForScenarioDependencies(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	det := detectorInstance()
	if det == nil {
		return nil, fmt.Errorf("detector not initialized")
	}
	return det.ScanScenarioDependencies(scenarioPath, scenarioName)
}

func scanForSharedWorkflows(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	det := detectorInstance()
	if det == nil {
		return nil, fmt.Errorf("detector not initialized")
	}
	return det.ScanSharedWorkflows(scenarioPath, scenarioName)
}

func scanForResourceUsage(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	return scanForResourceUsageWithConfig(scenarioPath, scenarioName, nil)
}

func scanForResourceUsageWithConfig(scenarioPath, scenarioName string, cfg *types.ServiceConfig) ([]types.ScenarioDependency, error) {
	det := detectorInstance()
	if det == nil {
		return nil, fmt.Errorf("detector not initialized")
	}
	return det.ScanResources(scenarioPath, scenarioName, cfg)
}
