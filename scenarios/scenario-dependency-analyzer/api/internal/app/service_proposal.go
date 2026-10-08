package app

import types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"

type proposalService struct{}

func (proposalService) AnalyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error) {
	return analyzeProposedScenario(req)
}
