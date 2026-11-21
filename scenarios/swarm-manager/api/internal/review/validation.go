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

func normalizeRound(round Round) (Round, error) {
	if round.Status != RoundStatusComplete {
		return round, nil
	}
	if err := validateCompletedRound(round); err != nil {
		if transitionErr := round.TransitionStatus(RoundStatusFailed); transitionErr != nil {
			return Round{}, transitionErr
		}
		round.FailureReason = err.Error()
	}
	return round, nil
}

func finalizeRoundFromRunState(round Round, state agentmanager.RunState) (Round, error) {
	next := mapRunStatusToRoundStatus(state.Status)
	switch next {
	case RoundStatusComplete:
		if err := validateCompletedRound(round); err != nil {
			if transitionErr := round.TransitionStatus(RoundStatusFailed); transitionErr != nil {
				return Round{}, transitionErr
			}
			round.FailureReason = err.Error()
			return round, nil
		}
		round.FailureReason = ""
	case RoundStatusFailed:
		round.FailureReason = deriveRoundFailureReason(state)
	}
	if next != "" {
		if err := round.TransitionStatus(next); err != nil {
			return Round{}, err
		}
	}
	return round, nil
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
