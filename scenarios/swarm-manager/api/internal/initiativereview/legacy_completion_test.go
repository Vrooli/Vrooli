package initiativereview

import (
	"context"
	"strings"

	"swarm-manager/internal/review"
)

type legacyCompletionAction struct {
	InitiativeName string
	ExecutionID    string
	RunID          string
	Outcome        string
	Result         []byte
}

// commitInitiativeReview remains test-only while historical operation records
// are exercised. Production applies declared workflow completions through the
// workflow endpoint instead.
func (s *Service) commitInitiativeReview(ctx context.Context, action legacyCompletionAction) error {
	name := strings.TrimSpace(action.InitiativeName)
	if name == "" {
		return nil
	}
	round, err := review.FindGatheringRoundByRunID(s.initStore.InitDir(name), action.RunID)
	if err != nil || round == nil {
		return err
	}
	review.FinalizeRoundFromResult(round, action.Result, action.Outcome)
	if err := review.SaveRound(s.initStore.InitDir(name), *round); err != nil {
		return err
	}
	s.handleTerminalRound(ctx, name, *round)
	return nil
}
