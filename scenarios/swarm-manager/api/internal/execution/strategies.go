package execution

import (
	"strings"

	"swarm-manager/internal/transitions"
)

// StrategySummary is the API projection of a declared strategy. Cost is
// calculated from the same governance inputs used by queue admission, so the
// run sheet cannot promise a different number than the backend enforces.
type StrategySummary struct {
	ID              string  `json:"id"`
	WorkflowKey     string  `json:"workflow_key"`
	DisplayName     string  `json:"display_name"`
	Description     string  `json:"description"`
	WhenToUse       string  `json:"when_to_use"`
	CostBand        string  `json:"cost_band"`
	CostEstimate    float64 `json:"cost_estimate"`
	RemainingTurns  int     `json:"remaining_turns,omitempty"`
	RemainingTokens int64   `json:"remaining_tokens,omitempty"`
	RemainingWall   int64   `json:"remaining_wall_seconds,omitempty"`
	RemainingCharge int64   `json:"remaining_charge_micro_usd,omitempty"`
	UsageKnown      bool    `json:"usage_known"`
}

func (s *Service) ExecutionStrategies() ([]StrategySummary, error) {
	return s.executionStrategiesForItem("", "")
}

func (s *Service) ExecutionStrategiesForItem(backlogKind, backlogName string) ([]StrategySummary, error) {
	return s.executionStrategiesForItem(backlogKind, backlogName)
}

func (s *Service) executionStrategiesForItem(backlogKind, backlogName string) ([]StrategySummary, error) {
	governance, err := s.governanceProvider.LoadGovernance()
	if err != nil {
		return nil, err
	}
	turns := governance.AgentMaxTurns
	if turns <= 0 {
		turns = 600
	}
	remainingTurns, remainingTokens, remainingWall, remainingCharge, usageKnown := turns, int64(0), int64(0), int64(0), true
	if strings.TrimSpace(backlogKind) != "" || strings.TrimSpace(backlogName) != "" {
		item, itemErr := s.loadBacklogItem(strings.TrimSpace(backlogKind), strings.TrimSpace(backlogName))
		if itemErr != nil {
			return nil, itemErr
		}
		if item.ExecutionLimits != nil {
			remainingTurns = item.ExecutionLimits.MaxTurns
			remainingTokens = item.ExecutionLimits.MaxTokens
			remainingWall = item.ExecutionLimits.MaxWallSeconds
			remainingCharge = item.ExecutionLimits.MaxChargeMicroUSD
			approvalDigest := ""
			if item.PlanAcceptance != nil {
				approvalDigest = digestStrings(item.PlanAcceptance.SubjectVersion, item.PlanAcceptance.PlanContentHash)
			}
			records, recordsErr := s.store.Load()
			if recordsErr != nil {
				return nil, recordsErr
			}
			for _, record := range records {
				if record.BacklogKind != strings.TrimSpace(backlogKind) || record.BacklogName != strings.TrimSpace(backlogName) || record.ApprovalDigest != approvalDigest || record.SettledUsage == nil {
					continue
				}
				usage := record.SettledUsage
				if !usage.TokensKnown || !usage.ChargeMeasured {
					usageKnown = false
					continue
				}
				remainingTurns -= int(usage.Turns)
				remainingTokens -= usage.Tokens
				remainingWall -= usage.WallSeconds
				remainingCharge -= usage.ChargeMicroUSD
			}
			if remainingTurns < 0 {
				remainingTurns = 0
			}
			if remainingTokens < 0 {
				remainingTokens = 0
			}
			if remainingWall < 0 {
				remainingWall = 0
			}
			if remainingCharge < 0 {
				remainingCharge = 0
			}
		}
	}
	estimate := governance.CostPerTurnEstimate * float64(remainingTurns)
	if strings.TrimSpace(backlogKind) != "" || strings.TrimSpace(backlogName) != "" {
		if remainingCharge > 0 {
			estimate = float64(remainingCharge) / 1_000_000
		} else if remainingTokens > 0 || remainingWall > 0 {
			estimate = 0
		}
	}
	declared := s.declaredExecutionStrategies()
	items := make([]StrategySummary, 0, len(declared))
	for _, strategy := range declared {
		items = append(items, strategySummary(strategy, estimate, remainingTurns, remainingTokens, remainingWall, remainingCharge, usageKnown))
	}
	return items, nil
}

func strategySummary(strategy transitions.ExecutionStrategy, estimate float64, remainingTurns int, remainingTokens, remainingWall, remainingCharge int64, usageKnown bool) StrategySummary {
	return StrategySummary{
		ID: strategy.ID, WorkflowKey: strategy.WorkflowKey, DisplayName: strategy.DisplayName,
		Description: strategy.Description, WhenToUse: strategy.WhenToUse, CostBand: strategy.CostBand,
		CostEstimate: estimate, RemainingTurns: remainingTurns, RemainingTokens: remainingTokens, RemainingWall: remainingWall, RemainingCharge: remainingCharge, UsageKnown: usageKnown,
	}
}
