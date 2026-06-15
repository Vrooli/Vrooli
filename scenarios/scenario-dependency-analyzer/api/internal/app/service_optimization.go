package app

import (
	"fmt"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/app/services"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

type optimizationService struct {
	analysis     services.AnalysisService
	workspace    *scenarioWorkspace
	detector     scenarioDetector
	dependencies *dependencyService
	store        *store.Store
}

func (s *optimizationService) RunOptimization(req types.OptimizationRequest) (map[string]*types.OptimizationResult, error) {
	if s.workspace == nil {
		return nil, fmt.Errorf("optimization service not initialized")
	}
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		scenario = "all"
	}

	var targets []string
	if scenario == "all" {
		names, err := s.workspace.listScenarioNames()
		if err != nil {
			return nil, err
		}
		targets = names
	} else {
		if !s.workspace.hasServiceConfig(scenario) {
			return nil, fmt.Errorf("%w: %s", errScenarioNotFound, scenario)
		}
		if s.detector != nil && !s.detector.KnownScenario(scenario) {
			return nil, fmt.Errorf("%w: %s", errScenarioNotFound, scenario)
		}
		targets = []string{scenario}
	}

	results := make(map[string]*types.OptimizationResult, len(targets))
	for _, target := range targets {
		result, err := s.runScenarioOptimization(target, req)
		if err != nil {
			results[target] = &types.OptimizationResult{
				Scenario:          target,
				Recommendations:   nil,
				Summary:           types.OptimizationSummary{},
				Applied:           false,
				AnalysisTimestamp: time.Now().UTC(),
				Error:             err.Error(),
			}
			continue
		}
		results[target] = result
	}

	return results, nil
}
