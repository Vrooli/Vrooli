package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/workshop"
)

// migrationNameBackfillExecCompletedV1 is the sentinel key recorded in the
// event log when the v1 execution-completed backfill has finished. Never
// change this constant: the presence of an event with this name gates
// re-running the migration.
const migrationNameBackfillExecCompletedV1 = "backfill_execution_completed_v1"

// migrationNameBackfillRecommendationAcceptanceV1 is the sentinel for the
// one-time backfill that scans every workshop/round-NNN.json on disk and
// emits enriched decision.workshop_round_completed events for rounds whose
// per-item counters are not yet represented in the event log. Gated on the
// presence of a system.migration_applied event with this name.
const migrationNameBackfillRecommendationAcceptanceV1 = "backfill_recommendation_acceptance_v1"

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

	if !migrationApplied(events, migrationNameBackfillRecommendationAcceptanceV1) {
		enriched := enrichedWorkshopRounds(events)
		affected, err := s.backfillRecommendationAcceptance(enriched)
		if err != nil {
			slog.Error("migrations: backfill_recommendation_acceptance_v1 failed", "err", err)
			return
		}
		s.emitter.EmitMigrationApplied(
			migrationNameBackfillRecommendationAcceptanceV1,
			"Emit enriched decision.workshop_round_completed events from on-disk round files so recommendation-acceptance stats reflect historical data.",
			affected,
		)
		slog.Info("migrations: backfill_recommendation_acceptance_v1 applied", "affected", affected)

		if s.statsEngine != nil {
			if err := s.statsEngine.Rebuild(ctx); err != nil {
				slog.Error("migrations: stats rebuild after recommendation-acceptance backfill failed", "err", err)
			}
		}
	}
}

// enrichedWorkshopRounds returns the set of (entityID, roundNumber) pairs
// already represented by an enriched (per-item populated) workshop event.
// Pre-schema events with only round_number contribute nothing — they are
// not "enriched" — so the backfill replaces their signal with one that
// carries item-level data.
func enrichedWorkshopRounds(events []eventlog.Event) map[string]struct{} {
	seen := make(map[string]struct{})
	for _, e := range events {
		if e.EventType != eventlog.EventWorkshopRoundCompleted {
			continue
		}
		if len(e.Metadata) == 0 {
			continue
		}
		var p eventlog.WorkshopRoundPayload
		if err := json.Unmarshal(e.Metadata, &p); err != nil {
			continue
		}
		if p.ItemsTotal == 0 {
			continue
		}
		seen[fmt.Sprintf("%s#%d", e.EntityID, p.RoundNumber)] = struct{}{}
	}
	return seen
}

// backfillRecommendationAcceptance walks the workshop round files for every
// kind under the scenario root and emits an enriched workshop-round event
// for each round not already represented. Returns the number of events
// emitted. Errors during a single file are logged and skipped so a
// malformed round does not abort the whole backfill.
func (s *Server) backfillRecommendationAcceptance(alreadyEnriched map[string]struct{}) (int, error) {
	kinds := []backlog.BacklogKind{
		backlog.KindIdea, backlog.KindResearch, backlog.KindFix,
		backlog.KindExecute, backlog.KindChore,
	}
	store := backlog.NewFileStore(s.scenarioRoot)
	emitted := 0

	for _, kind := range kinds {
		kindDir := store.KindDir(kind)
		entries, err := os.ReadDir(kindDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return emitted, fmt.Errorf("read %s: %w", kindDir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			workshopDir := filepath.Join(kindDir, name, "workshop")
			roundFiles, err := os.ReadDir(workshopDir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				slog.Warn("backfill_recommendation_acceptance_v1: read workshop dir", "dir", workshopDir, "err", err)
				continue
			}
			for _, rf := range roundFiles {
				if rf.IsDir() {
					continue
				}
				if !strings.HasPrefix(rf.Name(), "round-") || !strings.HasSuffix(rf.Name(), ".json") {
					continue
				}
				roundPath := filepath.Join(workshopDir, rf.Name())
				data, err := os.ReadFile(roundPath)
				if err != nil {
					slog.Warn("backfill_recommendation_acceptance_v1: read round", "path", roundPath, "err", err)
					continue
				}
				var round workshop.Round
				if err := json.Unmarshal(data, &round); err != nil {
					slog.Warn("backfill_recommendation_acceptance_v1: parse round", "path", roundPath, "err", err)
					continue
				}
				entityID := string(kind) + "/" + name
				key := fmt.Sprintf("%s#%d", entityID, round.RoundNum)
				if _, present := alreadyEnriched[key]; present {
					continue
				}
				summary := workshop.SummarizeRound(&round)
				s.emitter.EmitWorkshopRoundCompleted(entityID, eventlog.WorkshopRoundPayload{
					RoundNumber:            round.RoundNum,
					Kind:                   string(kind),
					ItemsTotal:             summary.ItemsTotal,
					ItemsAnswered:          summary.ItemsAnswered,
					ItemsRecommendedChosen: summary.ItemsRecommendedChosen,
					ItemsFreeformChosen:    summary.ItemsFreeformChosen,
				})
				emitted++
			}
		}
	}
	return emitted, nil
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
