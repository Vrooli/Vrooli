// Declarative-operation completion handler for the execution flow.
//
// In the PULL->PUSH model an execution run is an operation the runner starts
// (execution-run / execution-retry against a plan-execution target;
// execution-followup / execution-fixup against a backlog-item target). Its
// terminal outcome arrives as the operation's validated result on completion.
// commit-execution-round is the SINGLE completion authority for a runner-owned
// execution Record: it correlates the completing operation execution back to the
// Record via the OpExecutionID the reroute stamped at start and applies the SAME
// terminal transition the legacy inspector poll (polling.go applyTerminalTransition)
// applied — completed -> validating+finalization, blocked/abstain -> in_review +
// circuit-breaker, spec-sync archive-on-complete. The poller DEFERS runner-owned
// records (inspectRunningRecordsLocked skips any Record with OpExecutionID set), so
// the bridge is the sole driver and there is no double-drive; the startup
// reconciliation sweep is the backstop for a lost delivery.
package execution

import (
	"context"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
)

// RegisterOpsHandlers binds the execution completion handler onto the runner's
// action registry, overriding the registry's pre-registered no-op. Registered
// from the api wiring layer alongside the backlog/review handlers so the
// dependency edge to the execution domain flows through the caller (opsbridge
// stays domain-free).
func (s *Service) RegisterOpsHandlers(reg *opsrunner.ActionRegistry) {
	reg.Register(agentops.ActionCommitExecutionRound, s.commitExecutionRound)
}

// commitExecutionRound finalizes a completing execution operation by driving its
// correlated execution Record through the same terminal transition the poller
// performed. A "continue" outcome is not terminal (the operation loops another
// round); the Record stays running and the handler is a no-op. Idempotent: a
// re-delivered round whose Record is already terminal/finalizing is a no-op.
func (s *Service) commitExecutionRound(ctx context.Context, ac opsrunner.ActionContext) error {
	next, terminal := executionRecordStatusForOutcome(ac.Outcome)
	if !terminal {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return err
	}
	idx := -1
	for i := range records {
		if records[i].OpExecutionID != "" && records[i].OpExecutionID == ac.ExecutionID {
			idx = i
			break
		}
	}
	if idx == -1 {
		// No execution Record correlates to this operation execution (a plan-first
		// run with no backing Record, or one already reaped/migrated). Benign no-op
		// so the coordination transition commits.
		return nil
	}

	record := &records[idx]
	switch record.Status {
	case StatusCompleted, StatusFailed, StatusCanceled, StatusValidating, StatusNeedsFixup, StatusNeedsReview:
		return nil // idempotent replay: already finalized, finalizing, or past terminal handoff
	}

	prev := record.Status
	record.Status = next
	record.UpdatedAt = nowRFC3339()
	if next == StatusFailed {
		record.FailureReason = executionCommitAbstainReason(ac.Outcome)
	}

	// Apply the byte-for-byte terminal transition the poller applied. The empty
	// RunState means "no agent-manager finished-at" so applyTerminalTransition
	// stamps FinishedAt=now, exactly as it did for a poll with no finished-at.
	// finalizationCandidates is a throwaway: the next ProcessActiveExecutions tick
	// re-collects StatusValidating records via collectValidatingCandidatesLocked
	// and runs finalization, so the poll loop still owns finalization scheduling.
	var finalizationCandidates []string
	s.applyTerminalTransition(ctx, record, agentmanager.RunState{}, next, &finalizationCandidates)

	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusAndLog(*record, prev)
	return nil
}

// executionRecordStatusForOutcome maps a completing execution operation's outcome
// onto the execution Record's terminal status. It mirrors the poll-time mapping:
// a successful completion becomes StatusCompleted (which applyTerminalTransition
// routes into finalization/review); a blocked or abstaining round parks the
// Record as StatusFailed so the review agent documents it and the operator
// decides. "continue" is not terminal — the operation loops. Cancellation never
// reaches here (Cancel reaps the operation directly).
func executionRecordStatusForOutcome(outcome string) (status Status, terminal bool) {
	switch outcome {
	case "completed":
		return StatusCompleted, true
	case "blocked", "needs-attention":
		return StatusFailed, true
	default: // "continue" or an unmapped outcome: not a terminal Record transition
		return "", false
	}
}

func executionCommitAbstainReason(outcome string) string {
	if outcome == "blocked" {
		return "execution run reported blocked; parked for operator review"
	}
	return "execution run could not derive an honest outcome; parked for operator review"
}
