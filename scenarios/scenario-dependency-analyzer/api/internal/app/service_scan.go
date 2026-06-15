package app

import (
	"scenario-dependency-analyzer/internal/app/services"
	types "scenario-dependency-analyzer/internal/types"
)

type scanService struct {
	analysis     services.AnalysisService
	dependencies *dependencyService
}

func (s *scanService) ScanScenario(name string, req types.ScanRequest) (*services.ScanResult, error) {
	analysis, err := s.analysis.AnalyzeScenario(name)
	if err != nil {
		return nil, err
	}

	applyResources := req.ApplyResources || req.Apply
	applyScenarios := req.ApplyScenarios || req.Apply
	var applySummary map[string]interface{}
	applied := false

	if applyResources || applyScenarios {
		applySummary, err = applyDetectedDiffs(name, analysis, applyResources, applyScenarios, s.dependencies)
		if err != nil {
			return nil, err
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = s.analysis.AnalyzeScenario(name)
			if err != nil {
				return nil, err
			}
		}
	}

	return &services.ScanResult{
		Analysis:     analysis,
		Applied:      applied,
		ApplySummary: applySummary,
	}, nil
}
