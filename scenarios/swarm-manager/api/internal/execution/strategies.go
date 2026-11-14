package execution

import (
	"strings"

	"swarm-manager/internal/transitions"
)

// ExecutionModeSummary is the API projection of a declared strategy. Cost is
// calculated from the same governance inputs used by queue admission, so the
// run sheet cannot promise a different number than the backend enforces.
type ExecutionModeSummary struct {
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

func (s *Service) ExecutionModes() ([]ExecutionModeSummary, error) {
	return s.executionModesForItem("", "")
}

func (s *Service) ExecutionModesForItem(backlogKind, backlogName string) ([]ExecutionModeSummary, error) {
	return s.executionModesForItem(backlogKind, backlogName)
}

func (s *Service) executionModesForItem(backlogKind, backlogName string) ([]ExecutionModeSummary, error) {
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
	declared, err := s.declaredExecutionModes()
	if err != nil {
		return nil, err
	}
	items := make([]ExecutionModeSummary, 0, len(declared))
	for _, mode := range declared {
		modeCost := executionModeEstimate(mode.ID, governance.CostPerTurnEstimate, remainingTurns, remainingCharge, remainingTokens, remainingWall)
		items = append(items, executionModeSummary(mode, modeCost, remainingTurns, remainingTokens, remainingWall, remainingCharge, usageKnown))
	}
	return items, nil
}

// executionModeEstimate gives each mode an honest cost estimate: sliced is the
// per-turn estimate over the remaining turn budget, bounded by the remaining
// charge; goal is the item's remaining charge allowance.
func executionModeEstimate(modeID string, costPerTurn float64, remainingTurns int, remainingCharge, remainingTokens, remainingWall int64) float64 {
	if modeID == transitions.ExecutionModeGoal {
		if remainingCharge > 0 {
			return float64(remainingCharge) / 1_000_000
		}
		return 0
	}
	estimate := costPerTurn * float64(remainingTurns)
	if remainingCharge > 0 && estimate > float64(remainingCharge)/1_000_000 {
		estimate = float64(remainingCharge) / 1_000_000
	}
	if remainingTokens == 0 && remainingWall == 0 && remainingTurns == 0 {
		return 0
	}
	return estimate
}

func executionModeSummary(mode transitions.ExecutionMode, estimate float64, remainingTurns int, remainingTokens, remainingWall, remainingCharge int64, usageKnown bool) ExecutionModeSummary {
	return ExecutionModeSummary{
		ID: mode.ID, WorkflowKey: mode.WorkflowKey, DisplayName: mode.DisplayName,
		Description: mode.Description, WhenToUse: mode.WhenToUse, CostBand: mode.CostBand,
		CostEstimate: estimate, RemainingTurns: remainingTurns, RemainingTokens: remainingTokens, RemainingWall: remainingWall, RemainingCharge: remainingCharge, UsageKnown: usageKnown,
	}
}
