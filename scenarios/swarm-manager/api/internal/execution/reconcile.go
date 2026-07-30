package execution

import (
	"context"
	"log/slog"
	"strings"
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
