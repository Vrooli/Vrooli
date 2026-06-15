package app

import (
	"fmt"

	types "scenario-dependency-analyzer/internal/types"
)

type analysisService struct {
	analyzer *Analyzer
}

func (s *analysisService) AnalyzeScenario(name string) (*types.DependencyAnalysisResponse, error) {
	if s == nil || s.analyzer == nil {
		return nil, fmt.Errorf("analysis service not initialized")
	}
	return s.analyzer.AnalyzeScenario(name)
}

func (s *analysisService) AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	if s == nil || s.analyzer == nil {
		return nil, fmt.Errorf("analysis service not initialized")
	}
	return s.analyzer.AnalyzeAllScenarios()
}
