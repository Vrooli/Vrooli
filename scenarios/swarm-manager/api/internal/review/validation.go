package review

import (
	"fmt"
	"strings"
	"swarm-manager/internal/agentmanager"
)

var validRoundClassifications = map[string]struct{}{
	"ready":            {},
	"ready_with_notes": {},
	"needs_work":       {},
	"not_assessable":   {},
}

func normalizeRound(round Round) Round {
	if round.Status != RoundStatusComplete {
		return round
	}
	if err := validateCompletedRound(round); err != nil {
		round.Status = RoundStatusFailed
		round.FailureReason = err.Error()
	}
	return round
}

func finalizeRoundFromRunState(round Round, state agentmanager.RunState) Round {
	switch mapRunStatusToRoundStatus(state.Status) {
	case RoundStatusComplete:
		if err := validateCompletedRound(round); err != nil {
			round.Status = RoundStatusFailed
			round.FailureReason = err.Error()
			return round
		}
		round.Status = RoundStatusComplete
		round.FailureReason = ""
	case RoundStatusFailed:
		round.Status = RoundStatusFailed
		round.FailureReason = deriveRoundFailureReason(state)
	}
	return round
}

func validateCompletedRound(round Round) error {
	if strings.TrimSpace(round.AgentAssessment) == "" {
		return fmt.Errorf("review run completed without agent_assessment")
	}

	classification := strings.TrimSpace(round.Classification)
	if classification == "" {
		return fmt.Errorf("review run completed without classification")
	}
	if _, ok := validRoundClassifications[classification]; !ok {
		return fmt.Errorf("review run completed with invalid classification %q", classification)
	}

	return nil
}

func deriveRoundFailureReason(state agentmanager.RunState) string {
	status := strings.ToLower(strings.TrimSpace(state.Status))
	if status == "cancelled" {
		return "review agent run was cancelled"
	}
	if msg := strings.TrimSpace(state.ErrorMsg); msg != "" {
		return msg
	}
	return "review agent run failed"
}
