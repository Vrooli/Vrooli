package app

import (
	"fmt"
	"time"

	appoptimization "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/app/optimization"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

func (s *optimizationService) runScenarioOptimization(scenarioName string, req types.OptimizationRequest) (*types.OptimizationResult, error) {
	if s == nil || s.workspace == nil || s.analysis == nil {
		return nil, fmt.Errorf("optimization service not initialized")
	}

	analysis, err := s.analysis.AnalyzeScenario(scenarioName)
	if err != nil {
		return nil, err
	}

	recommendations := appoptimization.GenerateRecommendations(scenarioName, analysis)
	if err := s.persistOptimizationResults(scenarioName, recommendations); err != nil {
		return nil, err
	}

	depSvc := s.dependencies
	if depSvc == nil {
		depSvc = defaultDependencyService()
	}

	var applySummary map[string]interface{}
	applied := false
	if req.Apply && len(recommendations) > 0 {
		applySummary, err = applyOptimizationRecommendations(scenarioName, recommendations, depSvc)
		if err != nil {
			return nil, err
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = s.analysis.AnalyzeScenario(scenarioName)
			if err != nil {
				return nil, err
			}
			recommendations = appoptimization.GenerateRecommendations(scenarioName, analysis)
			if err := s.persistOptimizationResults(scenarioName, recommendations); err != nil {
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

func (s *optimizationService) persistOptimizationResults(scenarioName string, recs []types.OptimizationRecommendation) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.PersistOptimizationRecommendations(scenarioName, recs)
}
