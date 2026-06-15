package app

import types "scenario-dependency-analyzer/internal/types"

type proposalService struct{}

func (proposalService) AnalyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error) {
	return analyzeProposedScenario(req)
}
