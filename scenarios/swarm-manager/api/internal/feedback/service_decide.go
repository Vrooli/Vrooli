package feedback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/proposals"
)

// ErrRoundAlreadyTerminal is returned by Cancel when the round has already
// reached a terminal status (applied/rejected/dismissed). The HTTP layer
// maps this to 409 Conflict.
var ErrRoundAlreadyTerminal = errors.New("round is already terminal")

// ErrRoundNotTerminal is returned by Delete when the round still has an
// active agent run. Awaiting-user rounds are deletable: the lock has already
// been released and there is no live agent to orphan.
var ErrRoundNotTerminal = errors.New("round cannot be deleted while the agent is running")

// Delete permanently removes a feedback round from disk. Allowed whenever the
// round is not agent_thinking so users can discard invalid/unwanted rounds
// without forcing them into a terminal decision first. The disk row is the
// source of truth — once the dir is gone, the round is gone.
func (s *Service) Delete(initiativeName string, roundNumber int) error {
	round, err := s.store.LoadRound(initiativeName, roundNumber)
	if err != nil {
		return err
	}
	if round.Status == RoundStatusAgentThinking {
		return ErrRoundNotTerminal
	}
	return s.store.DeleteRound(initiativeName, roundNumber)
}

// CancelRequest is the user-supplied input for cancelling an in-flight
// feedback round. Both fields are optional — Cancel is the "I want this
// stuck spinner gone" escape hatch and shouldn't fail on missing context.
type CancelRequest struct {
	InitiativeName string
	RoundNumber    int
	Rationale      string
	DecidedBy      string
}

// Cancel forces a round out of agent_thinking, stops the agent-manager run
// (best-effort), releases the lock, and lands the round in dismissed. It
// is the user-facing escape hatch when the agent has crashed or the user
// no longer wants to wait. Idempotent on terminal rounds: returns
// ErrRoundAlreadyTerminal so the caller can decide whether that's an error.
//
// Cancel is intentionally permissive about failures. A dead agent-manager
// run, a missing lock file, an empty RunID — none of those should block a
// user from escaping a stuck UI. We log and continue.
func (s *Service) Cancel(ctx context.Context, req CancelRequest) (Round, error) {
	round, err := s.store.LoadRound(req.InitiativeName, req.RoundNumber)
	if err != nil {
		return Round{}, err
	}
	if round.Status.IsTerminal() {
		return round, ErrRoundAlreadyTerminal
	}

	// Best-effort: stop the agent-manager run. Failures here are logged
	// but don't block local cancellation — the user already wants out.
	if s.canceller != nil && strings.TrimSpace(round.RunID) != "" {
		if stopErr := s.canceller.StopRun(ctx, round.RunID); stopErr != nil {
			slog.Warn("feedback: cancel: stop run failed",
				"err", stopErr,
				"initiative", req.InitiativeName,
				"round", req.RoundNumber,
				"run_id", round.RunID)
		}
	}

	now := s.clock().UTC().Format(time.RFC3339)
	rationale := strings.TrimSpace(req.Rationale)
	if rationale == "" {
		rationale = "cancelled by user"
	}
	round.Thread = append(round.Thread, Message{
		Role:      "agent",
		Content:   "agent run cancelled: " + rationale,
		RunID:     round.RunID,
		CreatedAt: now,
	})
	previousRunID := round.RunID
	round.Status = RoundStatusDismissed
	round.Decision = &Decision{
		Kind:      DecisionDismiss,
		Rationale: rationale,
		DecidedAt: now,
		DecidedBy: req.DecidedBy,
	}
	round.RunID = ""
	round.NeedsRevision = false
	round.LastParseWarnings = nil
	round.LastValidationErrors = nil
	round.LastPollError = ""
	round.PollFailureCount = 0
	round.UpdatedAt = now
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, fmt.Errorf("save round: %w", err)
	}
	// Release the lock keyed on the previous RunID — Release is idempotent
	// when the lock holder doesn't match, so a parallel preempt or sweeper
	// taking the lock first won't fail us.
	if relErr := s.lock.Release(req.InitiativeName, previousRunID); relErr != nil {
		slog.Warn("feedback: release lock failed", "err", relErr, "initiative", req.InitiativeName, "run_id", previousRunID)
	}
	return round, nil
}

// DecideRequest is the user's terminal choice on the current proposal.
// Mutation IDs in AcceptedMutationIDs are applied (in order); anything not
// listed is dropped. For DecisionReject the applier is not invoked.
type DecideRequest struct {
	InitiativeName      string
	RoundNumber         int
	Kind                DecisionKind
	AcceptedMutationIDs []string
	Rationale           string
	DecidedBy           string
}

// Decide resolves the round to a terminal state. For DecisionAccept /
// DecisionPartialAccept the apply layer is invoked against the current
// proposal; the applied/failed counts are persisted on the decision's
// rationale for audit.
func (s *Service) Decide(ctx context.Context, req DecideRequest) (Round, *proposals.ApplyResult, error) {
	if req.Kind == "" {
		return Round{}, nil, errors.New("decision kind is required")
	}
	round, err := s.store.LoadRound(req.InitiativeName, req.RoundNumber)
	if err != nil {
		return Round{}, nil, err
	}
	// Dismiss-while-active: if the user calls Decide(kind=dismiss) on an
	// agent_thinking round (e.g. via the legacy UI Dismiss path), route
	// through Cancel so the agent run is stopped and the lock released.
	// Other decisions still require awaiting_user — accepting/rejecting a
	// proposal that doesn't exist yet would be nonsensical.
	if round.Status == RoundStatusAgentThinking && req.Kind == DecisionDismiss {
		cancelled, cErr := s.Cancel(ctx, CancelRequest{
			InitiativeName: req.InitiativeName,
			RoundNumber:    req.RoundNumber,
			Rationale:      req.Rationale,
			DecidedBy:      req.DecidedBy,
		})
		return cancelled, nil, cErr
	}
	if round.Status != RoundStatusAwaitingUser {
		return Round{}, nil, fmt.Errorf("round is in status %q; decide requires %q", round.Status, RoundStatusAwaitingUser)
	}
	round = s.normalizeRoundProposalState(round)
	if round.CurrentProposalID == "" && round.NeedsRevision && len(round.LastValidationErrors) > 0 {
		return Round{}, nil, &ProposalValidationError{
			ValidationErrors: append([]string(nil), round.LastValidationErrors...),
		}
	}

	now := s.clock().UTC().Format(time.RFC3339)
	decision := &Decision{
		Kind:                req.Kind,
		AcceptedMutationIDs: append([]string(nil), req.AcceptedMutationIDs...),
		Rationale:           req.Rationale,
		DecidedAt:           now,
		DecidedBy:           req.DecidedBy,
	}

	var applyResult *proposals.ApplyResult
	switch req.Kind {
	case DecisionAccept, DecisionPartialAccept:
		current := round.CurrentProposal()
		if current == nil {
			return Round{}, nil, errors.New("round has no current proposal to accept")
		}
		var err error
		applyResult, err = s.applyCurrentProposal(ctx, round, req.AcceptedMutationIDs, req.DecidedBy, now)
		if err != nil {
			return Round{}, nil, fmt.Errorf("apply proposal: %w", err)
		}
		decision.RejectedMutationIDs = computeRejected(current.Proposal, req.AcceptedMutationIDs)
		round.Status = RoundStatusApplied
	case DecisionReject, DecisionDismiss:
		round.Status = decisionToStatus(req.Kind)
	case DecisionRevise:
		// Revise isn't a terminal decision — it's handled via ContinueRound.
		// Reject here to keep the state machine legible.
		return Round{}, nil, errors.New("use ContinueRound for revisions; Decide requires a terminal decision")
	default:
		return Round{}, nil, fmt.Errorf("unknown decision kind %q", req.Kind)
	}

	round.Decision = decision
	round.UpdatedAt = now
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, applyResult, fmt.Errorf("save round: %w", err)
	}
	if relErr := s.lock.Release(req.InitiativeName, round.RunID); relErr != nil {
		slog.Warn("feedback: release lock failed", "err", relErr, "initiative", req.InitiativeName, "run_id", round.RunID)
	}
	return round, applyResult, nil
}

func decisionToStatus(k DecisionKind) RoundStatus {
	switch k {
	case DecisionAccept, DecisionPartialAccept:
		return RoundStatusApplied
	case DecisionReject:
		return RoundStatusRejected
	case DecisionDismiss:
		return RoundStatusDismissed
	}
	return RoundStatusDismissed
}

func computeRejected(p proposals.Proposal, accepted []string) []string {
	set := make(map[string]struct{}, len(accepted))
	for _, id := range accepted {
		set[id] = struct{}{}
	}
	out := make([]string, 0)
	for _, m := range p.Mutations {
		if _, ok := set[m.ID]; ok {
			continue
		}
		out = append(out, m.ID)
	}
	return out
}

func isValidRoundType(t RoundType) bool {
	switch t {
	case RoundTypeFeedback, RoundTypeResearch, RoundTypeNote:
		return true
	}
	return false
}
