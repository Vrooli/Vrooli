package execution

import (
	"context"
	"log/slog"
	"strings"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"
)

// ReconcileReport summarizes a stranded-record reconciliation sweep: the
// execution ids it swept to a terminal failed status and, for diagnostics, how
// many records it examined. OpReapsAttempted lists the execution ids of
// terminal records whose durable operation execution was offered a reap (the
// reap itself is idempotent — an already-terminal operation is a no-op).
type ReconcileReport struct {
	Scanned          int      `json:"scanned"`
	Stranded         []string `json:"stranded"`
	OpReapsAttempted []string `json:"op_reaps_attempted,omitempty"`
}

// WorkflowReconcileReport records a callback-loss repair sweep.
type WorkflowReconcileReport struct {
	Scanned    int      `json:"scanned"`
	Observed   int      `json:"terminal_workflows_observed"`
	Reconciled []string `json:"reconciled,omitempty"`
	Skipped    []string `json:"skipped,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

// ReconcileWorkflowExecutions projects Agent Manager's authoritative terminal
// workflow state into Swarm execution records when the normal completion
// callback was lost. It is safe at startup and on every background cycle.
func (s *Service) ReconcileWorkflowExecutions(ctx context.Context) (WorkflowReconcileReport, error) {
	s.mu.Lock()
	reader := s.workflowStateReader
	if s.reconcileTracker == nil {
		s.reconcileTracker = newReconcileTracker()
	}
	tracker := s.reconcileTracker
	records, err := s.store.Load()
	s.mu.Unlock()
	if err != nil {
		return WorkflowReconcileReport{}, err
	}
	report := WorkflowReconcileReport{Scanned: len(records)}
	if reader == nil {
		return report, nil
	}
	// Several records can share one workflow (retries, follow-ups); trace each
	// workflow at most once per pass and let every record read the same answer.
	type observed struct {
		state agentmanager.WorkflowExecutionState
		err   error
	}
	traced := make(map[string]observed)
	live := make(map[string]struct{})
	for _, candidate := range records {
		if !isInspectableStatus(candidate.Status) {
			continue
		}
		live[candidate.ExecutionID] = struct{}{}
		// Goal-mode records have no workflow: read the run's typed terminal
		// fields directly.
		if candidate.ExecutionMode == transitions.ExecutionModeGoal {
			if s.goalRunReader == nil || strings.TrimSpace(candidate.RunID) == "" {
				report.Skipped = append(report.Skipped, candidate.ExecutionID)
				continue
			}
			changed, goalErr := s.applyReconciledGoalRun(ctx, candidate.ExecutionID)
			if goalErr != nil {
				report.Errors = append(report.Errors, candidate.ExecutionID+": "+goalErr.Error())
				continue
			}
			if changed {
				report.Reconciled = append(report.Reconciled, candidate.ExecutionID)
			}
			continue
		}
		workflowID := strings.TrimSpace(candidate.OpWorkflowID)
		if workflowID == "" {
			if correlation, correlationErr := s.transitionCorrelation(candidate); correlationErr == nil {
				workflowID = strings.TrimSpace(correlation.ExecutionID)
			}
		}
		if workflowID == "" {
			report.Skipped = append(report.Skipped, candidate.ExecutionID)
			continue
		}
		if !tracker.due(candidate, workflowID) {
			continue
		}
		result, ok := traced[workflowID]
		if !ok {
			stateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			result.state, result.err = reader.GetWorkflowExecutionState(stateCtx, workflowID)
			cancel()
			traced[workflowID] = result
		}
		tracker.observe(candidate, workflowID, result.state, result.err)
		state, stateErr := result.state, result.err
		if stateErr != nil {
			report.Errors = append(report.Errors, candidate.ExecutionID+": "+stateErr.Error())
			continue
		}
		if !terminalWorkflowStatus(state.Status) {
			continue
		}
		report.Observed++
		changed, err := s.applyReconciledWorkflowState(candidate.ExecutionID, workflowID, state)
		if err != nil {
			report.Errors = append(report.Errors, candidate.ExecutionID+": "+err.Error())
			continue
		}
		if changed {
			report.Reconciled = append(report.Reconciled, candidate.ExecutionID)
		}
	}
	tracker.retain(live)
	return report, nil
}

// applyReconciledGoalRun reads a goal run's typed terminal fields and projects
// the terminal status onto the record. It is idempotent on an already-terminal
// record and makes the network read before taking the store lock.
func (s *Service) applyReconciledGoalRun(ctx context.Context, executionID string) (bool, error) {
	s.mu.Lock()
	records, err := s.store.Load()
	s.mu.Unlock()
	if err != nil {
		return false, err
	}
	var target *Record
	for i := range records {
		if records[i].ExecutionID == executionID {
			if !isInspectableStatus(records[i].Status) {
				return false, nil
			}
			target = &records[i]
			break
		}
	}
	if target == nil {
		return false, nil
	}
	state, err := s.goalRunReader.GetGoalRunState(ctx, target.RunID)
	if err != nil {
		return false, err
	}
	status, reason, terminal := goalRunStatus(state)
	if !terminal {
		return false, nil
	}
	target.Status = status
	target.FailureReason = reason
	target.StopReason = strings.TrimSpace(state.StopReason)
	target.LastHandoff = state.LastHandoff
	target.FinishedAt = nowRFC3339()
	target.UpdatedAt = nowRFC3339()
	if status == StatusValidating {
		// A goal verdict enters finalization exactly like a workflow success:
		// the validating poller only picks records that already carry a
		// finalization record, so initialize it here or the execution sits in
		// validating forever.
		ensureFinalization(target)
	}
	// Settle the run's owner accounting with its terminal status so a resume or
	// re-queue can admit against the remaining allowance. A pending or
	// unavailable receipt leaves the reservation held for the grant path to
	// collect later; it never blocks the status transition.
	if target.SettledUsage == nil {
		if usage, usageErr := s.settledGoalUsage(ctx, *target); usageErr == nil && usage != nil {
			target.SettledUsage = usage
		}
	}
	s.mu.Lock()
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return false, err
	}
	s.mu.Unlock()
	s.dispatchStatusUpdate(*target)
	return true, nil
}

// goalRunStatus maps an Agent Manager goal run's typed terminal pair onto a
// Swarm execution status. An empty terminal_class is not terminal.
func goalRunStatus(state agentmanager.GoalRunState) (Status, string, bool) {
	switch strings.TrimSpace(state.TerminalClass) {
	case "verdict":
		switch strings.TrimSpace(state.StopReason) {
		case "complete":
			return StatusValidating, "goal verdict complete", true
		case "blocked":
			return StatusNeedsAttention, "goal verdict blocked", true
		case "abstained":
			return StatusNeedsReview, "goal verdict abstained", true
		default:
			return StatusNeedsReview, "goal verdict " + state.StopReason, true
		}
	case "interruption":
		// An involuntary stop is resumable under until-allowance; nothing
		// finalizes it until the sweeper resumes or the chain halts.
		return StatusInterrupted, "goal interruption: " + state.StopReason, true
	}
	// A run can end before its harness reports a typed terminal — a launch
	// failure, or a cancel before start. Treating that as live leaves the
	// execution starting forever with its lane held. A failure is final:
	// the until-allowance sweeper must not relaunch a run that cannot start.
	switch strings.TrimSpace(state.Status) {
	case "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED":
		reason := "goal run ended " + strings.TrimPrefix(state.Status, "RUN_STATUS_") + " without a verdict or interruption"
		if msg := strings.TrimSpace(state.ErrorMessage); msg != "" {
			reason += ": " + msg
		}
		return StatusFailed, reason, true
	case "RUN_STATUS_COMPLETE":
		return StatusNeedsReview, "goal run completed without a verdict", true
	default:
		return "", "", false
	}
}

func terminalWorkflowStatus(status domainpb.WorkflowExecutionStatus) bool {
	switch status {
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

func reconciledStatus(state agentmanager.WorkflowExecutionState) (Status, string) {
	switch state.Status {
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED:
		if !state.TerminalEvidence {
			return StatusNeedsReview, "workflow succeeded without terminal result evidence; inspect and apply explicitly"
		}
		return StatusCompleted, "workflow terminal state reconciled: succeeded"
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
		return StatusCanceled, "workflow terminal state reconciled: cancelled"
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED:
		return StatusNeedsAttention, "workflow terminal state reconciled: blocked"
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED:
		return StatusAbstained, "workflow terminal state reconciled: abstained"
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED:
		return StatusBudgetExhausted, "workflow terminal state reconciled: budget_exhausted"
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED:
		return StatusFailed, "workflow terminal state reconciled: failed"
	default:
		return StatusFailed, "workflow terminal state reconciled: unsupported terminal outcome"
	}
}

func (s *Service) applyReconciledWorkflowState(executionID, workflowID string, state agentmanager.WorkflowExecutionState) (bool, error) {
	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return false, err
	}
	for i := range records {
		current := &records[i]
		if current.ExecutionID != executionID || !isInspectableStatus(current.Status) {
			continue
		}
		currentWorkflowID := strings.TrimSpace(current.OpWorkflowID)
		var correlation transitionrun.Correlation
		correlationResolved := false
		if currentWorkflowID == "" {
			if resolved, resolvedErr := s.transitionCorrelation(*current); resolvedErr == nil {
				correlation = resolved
				correlationResolved = true
				currentWorkflowID = strings.TrimSpace(resolved.ExecutionID)
			}
		}
		if currentWorkflowID != workflowID {
			s.mu.Unlock()
			return false, nil
		}
		if correlationResolved && correlation.ApplyState == transitionrun.ApplyStateClaimed {
			// The transition sweeper has claimed this correlation and owns the
			// idempotent terminal apply. Reconciling the same terminal here
			// would be a second completion authority racing it.
			s.mu.Unlock()
			return false, nil
		}
		previous := current.Status
		targetStatus, targetReason := reconciledStatus(state)
		if current.Status == targetStatus && strings.HasPrefix(current.FailureReason, targetReason) {
			s.mu.Unlock()
			return false, nil
		}
		current.Status, current.FailureReason = targetStatus, targetReason
		current.FinishedAt = firstNonEmpty(state.UpdatedAt, nowRFC3339())
		current.UpdatedAt = nowRFC3339()
		if state.TerminalCode != "" {
			current.FailureReason += " (" + state.TerminalCode + ")"
		}
		if err := s.store.Save(records); err != nil {
			s.mu.Unlock()
			return false, err
		}
		changed := *current
		s.mu.Unlock()
		candidates := []string{}
		if item, itemErr := s.loadBacklogItemByRecord(&changed); itemErr == nil {
			switch changed.Status {
			case StatusCompleted:
				// Reuse the canonical completion projection so a recovered
				// callback enters validation/finalization exactly like the
				// normal completion bridge.
				s.applyCompletedTransition(&changed, item, &candidates)
			case StatusNeedsReview, StatusFailed:
				_ = s.updateBacklogStatus(item, backlogStatusInReview)
			case StatusCanceled:
				_ = s.updateBacklogStatus(item, restoreBacklogStatus(changed))
			}
		}
		if changed.Status != StatusCompleted || len(candidates) > 0 || changed.Finalization != nil {
			if saveErr := s.saveReconciledProjection(changed); saveErr != nil {
				return false, saveErr
			}
		}
		s.dispatchStatusAndLog(changed, previous)
		return true, nil
	}
	s.mu.Unlock()
	return false, nil
}

func (s *Service) saveReconciledProjection(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range records {
		if records[i].ExecutionID != record.ExecutionID {
			continue
		}
		// Completion projection can run outside the mutex. A cancellation
		// accepted since that snapshot revoked its authority to replace state.
		if records[i].Cancellation != nil {
			return nil
		}
		record.UpdatedAt = nowRFC3339()
		records[i] = record
		return s.store.Save(records)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ReconcileStrandedRecords sweeps execution records that can never reach a
// terminal status on their own and marks them failed (retryable), restoring the
// backlog item so the work can be re-queued.
//
// A record is stranded when it is in an inspectable status (starting / running /
// needs_review) but carries no agent run id: the poller skips run-id-less records
// (nothing advances it) and Cancel refuses them (it requires a run id), so the
// record would linger forever. This is the deterministic recovery for the class
// of pre-fix artifacts the empty-run-id root-cause left behind (plan-manager
// finding 98911a67); the three fail-closed guards shipped in slice A prevent new
// ones, but existing records need an explicit sweep because Cancel cannot clear
// them.
//
// It is idempotent and safe to run at startup or on demand: a healthy record
// (any record with a run id, or any already-terminal record) is untouched.
func (s *Service) ReconcileStrandedRecords() (ReconcileReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return ReconcileReport{}, err
	}

	report := ReconcileReport{Scanned: len(records)}
	type swept struct {
		record Record
		prev   Status
	}
	var changes []swept
	for i := range records {
		r := &records[i]
		if !isInspectableStatus(r.Status) || strings.TrimSpace(r.RunID) != "" {
			continue
		}
		if _, correlationErr := s.transitionCorrelation(*r); correlationErr == nil {
			continue
		}
		prev := r.Status
		r.Status = StatusFailed
		r.FailureReason = "reconciled: execution had no agent run id (stranded before a trackable run started); retry to re-run"
		r.FinishedAt = nowRFC3339()
		r.UpdatedAt = r.FinishedAt
		report.Stranded = append(report.Stranded, r.ExecutionID)
		changes = append(changes, swept{record: *r, prev: prev})
		slog.Warn("execution: reconciled stranded run-id-less record to failed",
			"execution_id", r.ExecutionID,
			"backlog_ref", r.BacklogKind+"/"+r.BacklogName,
			"previous_status", string(prev))
	}
	if len(changes) > 0 {
		if err := s.store.Save(records); err != nil {
			return ReconcileReport{}, err
		}
		// Restore each stranded item's backlog status (best-effort) so it is
		// re-queueable, and emit status events for observers.
		for _, c := range changes {
			if err := s.restoreBacklogStatusForRecord(c.record); err != nil {
				slog.Warn("execution: reconcile could not restore backlog status",
					"execution_id", c.record.ExecutionID, "err", err)
			}
			s.dispatchStatusAndLog(c.record, c.prev)
		}
	}

	return report, nil
}
