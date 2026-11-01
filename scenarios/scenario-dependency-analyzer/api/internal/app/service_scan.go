package app

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/app/services"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type scanService struct {
	analysis     services.AnalysisService
	dependencies *dependencyService
}

func (s *scanService) ScanScenario(name string, req types.ScanRequest) (*services.ScanResult, error) {
	selected := map[string]struct{}{}
	if req.OnlyScenarios != nil {
		if len(req.OnlyScenarios) == 0 {
			return nil, fmt.Errorf("%w: only_scenarios cannot be empty", types.ErrInvalidScanRequest)
		}
		if !req.ApplyScenarios || req.Apply || req.ApplyResources {
			return nil, fmt.Errorf("%w: only_scenarios requires apply_scenarios and cannot apply resources", types.ErrInvalidScanRequest)
		}
		for _, target := range req.OnlyScenarios {
			if target == "" || target != strings.TrimSpace(target) {
				return nil, fmt.Errorf("%w: only_scenarios must contain nonempty exact scenario names", types.ErrInvalidScanRequest)
			}
			selected[target] = struct{}{}
		}
	}
	analysis, err := s.analysis.AnalyzeScenario(name)
	if err != nil {
		return nil, err
	}

	applyResources := req.ApplyResources || req.Apply
	applyScenarios := req.ApplyScenarios || req.Apply
	var applySummary map[string]interface{}
	applied := false

	if applyResources || applyScenarios {
		applySummary, err = applyDetectedDiffs(name, analysis, applyResources, applyScenarios, s.dependencies, selected)
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
