package opsbridge

import (
	"context"
	"log/slog"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opsrunner"
)

// WorkflowLoader loads a domain workflow instance by target, or by the live agent
// run id when the round's scope id is not the workflow key. Satisfied by
// *opsrunner.WorkflowRepo.
type WorkflowLoader interface {
	Load(kind agentops.TargetKind, id string) (agentops.WorkflowInstance, bool, error)
	// FindByRunID correlates a delivered round back to its owning workflow by the
	// globally-unique agent run id, for targets whose round scope id differs from
	// the workflow key (plan-execution: round keyed by resolved plan execution id,
	// workflow keyed by the plan handle).
	FindByRunID(runID string) (agentops.WorkflowInstance, bool, error)
}

// ResultCommitter finalizes a running operation execution with a delivered
// result. Satisfied by *opsrunner.Runner.
type ResultCommitter interface {
	CommitResult(ctx context.Context, req opsrunner.CommitRequest) (opsrunner.OperationResult, error)
}

// CompletionRouter is the terminal-round observer wired into the operating-mode
// engine (Service.SetRoundObserver). It routes a runner-OWNED round's completion
// into Runner.CommitResult and leaves every other round — a legacy initiative
// round, or a target round no runner operation started — untouched. Ownership is
// established by correlating the round's agent run id back to a running operation
// record on the target's workflow, the linkage the runner persists at start.
//
// It is fail-soft by contract: it never propagates an error into the refresh
// path, because a lost delivery is recoverable (the refresh driver re-observes the
// still-running operation and CommitResult is idempotent) while an error-
// propagating or panicking observer would wedge the engine's round refresh.
type CompletionRouter struct {
	repo      WorkflowLoader
	committer ResultCommitter
	log       *slog.Logger
}

// NewCompletionRouter builds the router. A nil logger defaults to slog.Default.
func NewCompletionRouter(repo WorkflowLoader, committer ResultCommitter, log *slog.Logger) *CompletionRouter {
	if log == nil {
		log = slog.Default()
	}
	return &CompletionRouter{repo: repo, committer: committer, log: log}
}

// Observe implements operatingmode.RoundObserver: it is called once when a round
// reaches a terminal status. Non-owned rounds are silently ignored.
func (r *CompletionRouter) Observe(ctx context.Context, round operatingmode.RoundEnvelope) {
	kind := agentops.TargetKind(round.ScopeKind)
	if kind == "" || round.ScopeID == "" || round.RunID == "" {
		return // not a keyed target round with a live run association
	}
	// Fast path: for a target whose round scope id IS the workflow key (backlog-
	// item, initiative), load the workflow directly and correlate the round's run
	// id to a running operation. For a plan-execution round the scope id is the
	// engine's resolved plan execution id, which differs from the workflow key (the
	// plan handle the runner targeted), so this misses and the run-id fallback below
	// finds the owning workflow.
	w, found, err := r.repo.Load(kind, round.ScopeID)
	if err != nil {
		r.log.Warn("opsbridge: load workflow for round completion failed",
			"err", err, "scope", round.ScopeID, "run_id", round.RunID)
		return
	}
	var op agentops.OperationExecutionRecord
	var ok bool
	if found {
		op, ok = opsrunner.FindOperationByRunID(w, round.RunID)
	}
	if !ok {
		w, found, err = r.repo.FindByRunID(round.RunID)
		if err != nil {
			r.log.Warn("opsbridge: correlate round to workflow by run id failed",
				"err", err, "scope", round.ScopeID, "run_id", round.RunID)
			return
		}
		if !found {
			return // no runner workflow owns this round's run: not runner-owned
		}
		op, ok = opsrunner.FindOperationByRunID(w, round.RunID)
		if !ok {
			return // the round's run is owned by no runner operation (legacy round)
		}
	}
	// Review operations classify with the verdict vocabulary, which the handoff
	// delivery mapper does not read; routing them through it would abstain every
	// review round. Branch on the owning operation so review-round /
	// initiative-review resolve their verdict-derived outcome, while every
	// handoff-style operation (evidence-request, revision, and the rest) keeps the
	// shared handoff mapping.
	delivery, err := roundDeliveryFor(string(op.Operation), round)
	if err != nil {
		r.log.Warn("opsbridge: map round to delivery failed",
			"err", err, "scope", round.ScopeID, "run_id", round.RunID)
		return
	}
	if !delivery.Deliver {
		return
	}
	if _, err := r.committer.CommitResult(ctx, opsrunner.CommitRequest{
		// Key the commit by the workflow's OWN domain id, not the round scope id:
		// they are equal for backlog-item/initiative but diverge for plan-execution
		// (round keyed by resolved exec id, workflow keyed by the plan handle), and
		// CommitResult loads the workflow by this id.
		Target:          opsrunner.TargetRef{Kind: kind, ID: w.Domain.ID},
		ExecutionID:     op.ExecutionID,
		Outcome:         delivery.Outcome,
		DeliveredResult: delivery.Result,
		RequestedBy:     "operating-mode-round-refresh",
	}); err != nil {
		// Fail-soft: a delivery error (e.g. ErrInvalidResult) leaves the operation
		// running with its round artifacts intact; the refresh driver re-observes and
		// the idempotent CommitResult retries. Surface it for diagnostics only.
		r.log.Warn("opsbridge: commit round result failed",
			"err", err, "scope", round.ScopeID, "execution_id", op.ExecutionID, "outcome", delivery.Outcome)
	}
}
