package feedback

import (
	"context"
	"errors"
	"fmt"
	"time"

	"swarm-manager/internal/proposals"
)

// applyCurrentProposal runs the round's current proposal through the
// proposals.Applier with the given accepted mutation IDs.
//
// Extracted from Service.Decide so the bridge to proposals lives in one
// place, and so future callers (e.g. the review flow's auto-apply path)
// can reuse it without reimplementing the source/state bookkeeping.
//
// Returns (nil, nil) when there is no current proposal; the caller is
// expected to interpret that as "nothing to apply" rather than an error
// because accept/reject semantics are the caller's to enforce.
func (s *Service) applyCurrentProposal(ctx context.Context, round Round, acceptedIDs []string, decidedBy, decidedAt string) (*proposals.ApplyResult, error) {
	current := round.CurrentProposal()
	if current == nil {
		return nil, errors.New("round has no current proposal to apply")
	}
	state, err := s.StateBuilder(round.InitiativeName)
	if err != nil {
		return nil, fmt.Errorf("build state: %w", err)
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
	// Normalize before Apply so agent-produced whitespace/casing quirks
	// (e.g. "  ready  ", "EXECUTE/Foo") are canonicalized. Apply's
	// contract expects pre-normalized input; without this the defensive
	// Validate inside Apply rejects values the agent intended correctly.
	normalized, err := proposals.Normalize(current.Proposal, state)
	if err != nil {
		return nil, fmt.Errorf("normalize proposal: %w", err)
	}
	return s.apply.Apply(ctx, normalized, state, acceptedIDs, source)
}
