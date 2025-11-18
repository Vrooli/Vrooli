package app

import (
	"path/filepath"
	"time"

	appoptimization "scenario-dependency-analyzer/internal/app/optimization"
	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

func runScenarioOptimization(scenarioName string, req types.OptimizationRequest) (*types.OptimizationResult, error) {
	analysis, err := analyzeScenario(scenarioName)
	if err != nil {
		return nil, err
	}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	svcCfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		svcCfg = nil
	}
	recommendations := appoptimization.GenerateRecommendations(scenarioName, analysis, svcCfg)
	if err := persistOptimizationResults(scenarioName, recommendations); err != nil {
		return nil, err
	}
	var applySummary map[string]interface{}
	applied := false
	if req.Apply && len(recommendations) > 0 {
		applySummary, err = applyOptimizationRecommendations(scenarioName, recommendations)
		if err != nil {
			return nil, err
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = analyzeScenario(scenarioName)
			if err != nil {
				return nil, err
			}
			recommendations = appoptimization.GenerateRecommendations(scenarioName, analysis, svcCfg)
			if err := persistOptimizationResults(scenarioName, recommendations); err != nil {
				return nil, err
			}
		}
	}
	return &types.OptimizationResult{
		Scenario:          scenarioName,
		Recommendations:   recommendations,
		Summary:           appoptimization.BuildSummary(recommendations),
		Applied:           applied,
		ApplySummary:      applySummary,
		AnalysisTimestamp: time.Now().UTC(),
	}, nil
}

func persistOptimizationResults(scenarioName string, recs []types.OptimizationRecommendation) error {
	analyzer := analyzerInstance()
	if analyzer == nil || analyzer.Store() == nil {
		return nil
	}
	return analyzer.Store().PersistOptimizationRecommendations(scenarioName, recs)
}
