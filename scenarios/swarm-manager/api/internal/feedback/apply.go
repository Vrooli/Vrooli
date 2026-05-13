package feedback

import (
	"context"
	"errors"
	"fmt"
	"time"

	"swarm-manager/internal/proposals"
)

// applyCurrentProposal applies the round's current proposal through the
// shared proposals.ApplyFlow recipe (build state → Normalize → Apply).
// Returns an error when no current proposal exists; accept/reject
// semantics are the caller's to enforce.
func (s *Service) applyCurrentProposal(ctx context.Context, round Round, acceptedIDs []string, decidedBy, decidedAt string) (*proposals.ApplyResult, error) {
	current := round.CurrentProposal()
	if current == nil {
		return nil, errors.New("round has no current proposal to apply")
	}
	if decidedAt == "" {
		decidedAt = s.clock().UTC().Format(time.RFC3339)
	}
	source := proposals.Source{
		InitiativeName:   round.InitiativeName,
		FeedbackRoundID:  fmt.Sprintf("%s/round-%03d", round.InitiativeName, round.Number),
		RoundNumber:      round.Number,
		RoundSlug:        round.Slug,
		Entrypoint:       "initiative.feedback",
		DecidedBy:        decidedBy,
		DecidedAtRFC3339: decidedAt,
	}
	return s.apply.ApplyFlow(ctx, current.Proposal, proposals.StateBuilder(s.StateBuilder), acceptedIDs, source)
}
