package app

// detection_bridge.go provides a functional API for detector operations.
//
// These functions manage access to the dependency detector, which scans
// scenarios and resources for dependency relationships. The bridge handles
// fallback creation and caching of the detector instance.
//
// Key functions:
// - detectorInstance(): Get or create the detector
// - isKnownScenario/isKnownResource: Check catalog membership
// - scanForResourceUsage, scanForScenarioDependencies, scanForSharedWorkflows: Detection ops
// - refreshDependencyCatalogs: Invalidate cached catalogs after filesystem changes

import (
	"fmt"

	"scenario-dependency-analyzer/internal/detection"
	types "scenario-dependency-analyzer/internal/types"
)

// detectorInstance returns the detector from the active analyzer, constructing via runtime when absent.
func detectorInstance() *detection.Detector {
	if analyzer := analyzerInstance(); analyzer != nil && analyzer.Detector() != nil {
		return analyzer.Detector()
	}
	rt := ensureRuntime(loadConfig(), db)
	if rt != nil && rt.Analyzer() != nil {
		return rt.Analyzer().Detector()
	}
	return nil
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
