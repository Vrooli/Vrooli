package execution

import "swarm-manager/internal/transitions"

// StrategySummary is the API projection of a declared strategy. Cost is
// calculated from the same governance inputs used by queue admission, so the
// run sheet cannot promise a different number than the backend enforces.
type StrategySummary struct {
	ID           string  `json:"id"`
	WorkflowKey  string  `json:"workflow_key"`
	DisplayName  string  `json:"display_name"`
	Description  string  `json:"description"`
	WhenToUse    string  `json:"when_to_use"`
	CostBand     string  `json:"cost_band"`
	CostEstimate float64 `json:"cost_estimate"`
}

func (s *Service) ExecutionStrategies() ([]StrategySummary, error) {
	governance, err := s.governanceProvider.LoadGovernance()
	if err != nil {
		return nil, err
	}
	turns := governance.AgentMaxTurns
	if turns <= 0 {
		turns = 600
	}
	estimate := governance.CostPerTurnEstimate * float64(turns)
	declared := s.declaredExecutionStrategies()
	items := make([]StrategySummary, 0, len(declared))
	for _, strategy := range declared {
		items = append(items, strategySummary(strategy, estimate))
	}
	return items, nil
}

func strategySummary(strategy transitions.ExecutionStrategy, estimate float64) StrategySummary {
	return StrategySummary{
		ID: strategy.ID, WorkflowKey: strategy.WorkflowKey, DisplayName: strategy.DisplayName,
		Description: strategy.Description, WhenToUse: strategy.WhenToUse, CostBand: strategy.CostBand,
		CostEstimate: estimate,
	}
}
