package execution

import (
	"context"
	"log/slog"
	"strings"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
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
	records, err := s.store.Load()
	s.mu.Unlock()
	if err != nil {
		return WorkflowReconcileReport{}, err
	}
	report := WorkflowReconcileReport{Scanned: len(records)}
	if reader == nil {
		return report, nil
	}
	for _, candidate := range records {
		if !isInspectableStatus(candidate.Status) {
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
		stateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		state, stateErr := reader.GetWorkflowExecutionState(stateCtx, workflowID)
		cancel()
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
	return report, nil
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
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED:
		return StatusNeedsReview, "workflow terminal state reconciled: " + strings.ToLower(state.Status.String())
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
		if currentWorkflowID == "" {
			if correlation, correlationErr := s.transitionCorrelation(*current); correlationErr == nil {
				currentWorkflowID = strings.TrimSpace(correlation.ExecutionID)
			}
		}
		if currentWorkflowID != workflowID {
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

	// Second pass: deliver missed operation reaps. A failed/canceled record that
	// still carries an operation-execution correlation may have left its durable
	// workflow operation "running" — e.g. the stranded sweep above marked the
	// record failed without reaping, or a cancel-time reap was lost before it
	// committed. Cancel's reap (reapOperationForRecord) is the honest terminal
	// for such an operation: no result was ever delivered, so the operation is
	// administratively reaped rather than given a fabricated outcome. The reap is
	// best-effort and idempotent (an already-terminal or untracked operation
	// execution is a no-op), so re-offering it on every sweep is safe. Completed
	// records are excluded: their operation should have been driven terminal by
	// the completion bridge, and reaping one as canceled would be dishonest —
	// such a divergence must surface for operator attention instead.
	if s.operationStarter != nil {
		for i := range records {
			r := records[i]
			if strings.TrimSpace(r.OpExecutionID) == "" {
				continue
			}
			if r.Status != StatusFailed && r.Status != StatusCanceled {
				continue
			}
			s.reapOperationForRecord(context.Background(), r)
			report.OpReapsAttempted = append(report.OpReapsAttempted, r.ExecutionID)
		}
	}
	return report, nil
}
