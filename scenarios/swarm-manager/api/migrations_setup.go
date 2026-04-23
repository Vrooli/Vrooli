package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"swarm-manager/internal/eventlog"
)

// migrationNameBackfillExecCompletedV1 is the sentinel key recorded in the
// event log when the v1 execution-completed backfill has finished. Never
// change this constant: the presence of an event with this name gates
// re-running the migration.
const migrationNameBackfillExecCompletedV1 = "backfill_execution_completed_v1"

// runMigrationsOnce runs any pending one-time migrations against the event
// log and the execution store. Each migration is gated on a sentinel event of
// type system.migration_applied so a restart never re-applies them.
func (s *Server) runMigrationsOnce() {
	if s.eventDB == nil || s.emitter == nil || s.executionSvc == nil {
		return
	}
	ctx := context.Background()

	// Always-run cleanup: drop pending records whose backlog item has been
	// removed (or whose record leaked in from a stray test run). This is
	// unconditional — fast, idempotent, and keeps the queue-depth budget
	// from being permanently consumed by orphaned entries.
	if pruned, err := s.executionSvc.PruneOrphanedPending(); err != nil {
		slog.Error("migrations: prune orphaned pending executions failed", "err", err)
	} else if pruned > 0 {
		slog.Info("migrations: pruned orphaned pending executions", "count", pruned)
	}

	// Fetch existing events once so migrations can check sentinels and the
	// per-execution emit history without issuing extra queries per migration.
	repo := eventlog.NewSQLiteRepository(s.eventDB)
	events, err := repo.All(ctx)
	if err != nil {
		slog.Error("migrations: failed to read event log", "err", err)
		return
	}

	if !migrationApplied(events, migrationNameBackfillExecCompletedV1) {
		emitted := executionsWithTerminalEvents(events)
		report, err := s.executionSvc.BackfillStuckTerminalEvents(ctx, emitted)
		if err != nil {
			slog.Error("migrations: backfill_execution_completed_v1 failed", "err", err)
			return
		}
		s.emitter.EmitMigrationApplied(
			migrationNameBackfillExecCompletedV1,
			"Emit missing execution.completed/failed/canceled events and promote validating-with-finalization-completed records to completed.",
			report.Affected,
		)
		slog.Info("migrations: backfill_execution_completed_v1 applied",
			"affected", report.Affected,
			"stuck_validating", report.StuckValidating,
			"missing_terminal", report.MissingTerminal)

		// Rebuild stats so the freshly-emitted events are reflected without
		// waiting for the first GET /stats call.
		if s.statsEngine != nil {
			if err := s.statsEngine.Rebuild(ctx); err != nil {
				slog.Error("migrations: stats rebuild after backfill failed", "err", err)
			}
		}
	}
}

// migrationApplied returns true if a system.migration_applied event with the
// given name exists in the event log.
func migrationApplied(events []eventlog.Event, name string) bool {
	for _, e := range events {
		if e.EventType != eventlog.EventSystemMigrationApplied {
			continue
		}
		var p eventlog.MigrationAppliedPayload
		if len(e.Metadata) == 0 {
			continue
		}
		if err := json.Unmarshal(e.Metadata, &p); err != nil {
			continue
		}
		if p.Name == name {
			return true
		}
	}
	return false
}

// executionsWithTerminalEvents returns the set of execution IDs that already
// have a terminal event in the log. The backfill skips these to avoid
// double-emission even on a partial prior run.
func executionsWithTerminalEvents(events []eventlog.Event) map[string]struct{} {
	seen := make(map[string]struct{})
	for _, e := range events {
		switch e.EventType {
		case eventlog.EventExecutionCompleted,
			eventlog.EventExecutionFailed,
			eventlog.EventExecutionCanceled:
			seen[e.EntityID] = struct{}{}
		}
	}
	return seen
}
