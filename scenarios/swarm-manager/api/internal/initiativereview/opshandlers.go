// Declarative-operation completion handler for the initiative review flow.
//
// An initiative-review round is an operation the runner starts against the
// initiative target; the agent's evidence + classification arrive as the
// operation's validated result on completion. commit-initiative-review
// materializes that result into the initiative's review round and opens the
// operator review gate (flips the initiative from in_review to review_pending,
// releasing the per-initiative lock) — exactly what the legacy poller's
// handleTerminalRound did.
//
// CRITICAL (recommendation-not-mutation): the agent verdict is a RECOMMENDATION.
// The handler records the review artifacts + recommended verdict and opens the
// gate, but performs NO terminal initiative-status mutation. The operator's
// accept/changes-requested/fail decision (initiativereview Decide) stays the sole
// authority for the terminal status, with its own immutable decision record.
package initiativereview

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/review"
)

// RegisterOpsHandlers binds the initiative-review completion handler onto the
// runner's action registry. It overrides the registry's pre-registered no-op for
// commit-initiative-review.
func (s *Service) RegisterOpsHandlers(reg *opsrunner.ActionRegistry) {
	reg.Register(agentops.ActionCommitInitiativeReview, s.commitInitiativeReview)
}

// commitInitiativeReview materializes the completing initiative-review
// operation's result into the open initiative review round and opens the operator
// review gate. It correlates the round by the live run id the reroute stamped on
// it at start. Idempotent: a re-delivered round whose correlated round is already
// terminal is a no-op.
func (s *Service) commitInitiativeReview(ctx context.Context, ac opsrunner.ActionContext) error {
	initiativeName := strings.TrimSpace(ac.Target.ID)
	if initiativeName == "" {
		return fmt.Errorf("commit-initiative-review: empty initiative target ref")
	}
	itemDir := s.initStore.InitDir(initiativeName)
	runID := review.RunIDForExecution(ac.Workflow, ac.ExecutionID)

	round, err := review.FindGatheringRoundByRunID(itemDir, runID)
	if err != nil {
		return fmt.Errorf("commit-initiative-review: load rounds for %s: %w", initiativeName, err)
	}
	if round == nil {
		return nil // no round correlates to this run: benign no-op
	}
	if round.Status == review.RoundStatusComplete || round.Status == review.RoundStatusFailed {
		return nil // idempotent replay: already finalized
	}

	review.FinalizeRoundFromResult(round, ac.Result, ac.Outcome)

	if err := review.SaveRound(itemDir, *round); err != nil {
		return fmt.Errorf("commit-initiative-review: save round for %s: %w", initiativeName, err)
	}

	// Open the operator review gate: release the per-initiative lock and flip
	// in_review -> review_pending. This is NOT a terminal initiative mutation; the
	// operator still decides the terminal status via Decide.
	s.handleTerminalRound(ctx, initiativeName, *round)

	slog.Info("initiative review ops: round committed", "initiative", initiativeName,
		"round", round.RoundNum, "status", round.Status, "outcome", ac.Outcome, "execution", ac.ExecutionID)
	return nil
}
