package execution

import (
	"context"
	"log/slog"
)

// MigrationBackfillReport captures the result of a one-time backfill run so
// callers can log or surface it.
type MigrationBackfillReport struct {
	Affected         int
	StuckValidating  int
	MissingTerminal  int
	AlreadyCompleted int
}

// BackfillStuckTerminalEvents scans the execution store for records whose
// in-memory status is already terminal (Completed/Failed/Canceled) but whose
// terminal event never made it to the event log, and emits the missing
// events. It also promotes records stuck at StatusValidating with a
// Completed finalization to StatusCompleted.
//
// The caller is responsible for gating this with a sentinel so it runs only
// once per install. Emission is idempotent per run (we never re-emit for a
// record that already has a terminal event in the log).
func (s *Service) BackfillStuckTerminalEvents(ctx context.Context, alreadyEmittedExecIDs map[string]struct{}) (MigrationBackfillReport, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.eventLogger == nil {
		return MigrationBackfillReport{}, nil
	}

	records, err := s.store.Load()
	if err != nil {
		return MigrationBackfillReport{}, err
	}

	var rep MigrationBackfillReport
	dirty := false

	for i := range records {
		r := &records[i]

		// Promote records stuck in validating when finalization already succeeded.
		if r.Status == StatusValidating && r.Finalization != nil && r.Finalization.Status == FinalizationStatusCompleted {
			rep.StuckValidating++
			prev := r.Status
			r.Status = StatusCompleted
			if r.FinishedAt == "" {
				r.FinishedAt = nowRFC3339()
			}
			r.UpdatedAt = nowRFC3339()
			dirty = true
			s.eventLogger.EmitExecutionStatusChanged(r.ExecutionID, string(prev), string(r.Status))
		}

		// Emit missing terminal event for records whose status is terminal
		// but whose terminal event never made it to the log.
		if _, hasEvent := alreadyEmittedExecIDs[r.ExecutionID]; hasEvent {
			continue
		}
		switch r.Status {
		case StatusCompleted:
			rep.MissingTerminal++
			dur := executionDuration(*r)
			if r.ManuallyAccepted {
				s.eventLogger.EmitExecutionManuallyAccepted(
					r.ExecutionID,
					r.AcceptedBy,
					r.AcceptedReason,
					string(r.AcceptedPreviousStatus),
				)
			}
			s.eventLogger.EmitExecutionCompleted(r.ExecutionID, dur, r.FixupAttempt > 0)
			rep.Affected++
		case StatusFailed:
			rep.MissingTerminal++
			dur := executionDuration(*r)
			s.eventLogger.EmitExecutionFailed(r.ExecutionID, r.FailureReason, dur)
			rep.Affected++
		case StatusCanceled:
			rep.MissingTerminal++
			s.eventLogger.EmitExecutionCanceled(r.ExecutionID, "user canceled")
			rep.Affected++
		}
	}

	if dirty {
		if err := s.store.Save(records); err != nil {
			slog.Error("backfill: save failed", "err", err)
			return rep, err
		}
	}

	return rep, nil
}
