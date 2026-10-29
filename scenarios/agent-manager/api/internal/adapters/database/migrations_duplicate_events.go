// Responsibility: the one-time background repair of run events the transcript
// indexer appended twice before events took their identity from their source
// line. It scans the whole event table, so it runs after startup rather than
// inside schema initialisation.
package database

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"

	"github.com/sirupsen/logrus"
)

const (
	migrationDuplicateTranscriptEvents = "2026-10-29-drop-duplicate-transcript-events"
	// migrationLegacyTwinEvents removes the copies the shift rule cannot see:
	// doubled rows with no repeated tool call, and v5 rows re-read over v4 ones.
	migrationLegacyTwinEvents = "2026-10-29-drop-legacy-twin-transcript-events"
)

// duplicateRepairRuns lists the runs whose events come from transcripts.
const duplicateRepairRuns = `SELECT id FROM runs WHERE execution_mode IN ('imported', 'interactive')`

// A duplicateRepair is one marker-gated background pass: for each run it
// selects the later copies (victimSQL, run id bound to every placeholder) and
// deletes them.
type duplicateRepair struct {
	marker    string
	victimSQL string
	args      int
}

var (
	shiftRepair = duplicateRepair{marker: migrationDuplicateTranscriptEvents, args: 2, victimSQL: duplicateShiftsCTE + `
		SELECT DISTINCT e.id FROM run_events e JOIN shifts s ON s.shift > 0
		JOIN run_events o ON o.run_id = e.run_id AND o.sequence = e.sequence - s.shift
			AND o.event_type = e.event_type AND o.timestamp = e.timestamp AND o.data = e.data
		WHERE e.run_id = ?`}
	// An event is a later copy when an earlier event of the same run has the
	// same type, timestamp and payload and at least one of the two carries a
	// pre-deterministic (non-v5) id. Two v5 twins are distinct source lines.
	legacyTwinRepair = duplicateRepair{marker: migrationLegacyTwinEvents, args: 1, victimSQL: `WITH g AS (
		SELECT id, ROW_NUMBER() OVER (PARTITION BY timestamp, event_type, data ORDER BY sequence) AS rn,
			SUM(substr(id, 15, 1) <> '5') OVER (PARTITION BY timestamp, event_type, data) AS legacy
		FROM run_events WHERE run_id = ?) SELECT id FROM g WHERE rn > 1 AND legacy > 0`}
)

// A tool call that repeats in one run with the same toolCallId and timestamp
// proves a copy; its sequence distance is the copy's shift. An event is the
// later copy when the event that many sequences earlier is identical in type,
// timestamp and payload. The first copy and every event with no earlier twin
// stay.
const duplicateShiftsCTE = `WITH calls AS (
	SELECT sequence, json_extract(data,'$.toolCallId') AS tool_call_id, timestamp
	FROM run_events WHERE run_id = ? AND event_type = 'tool_call'
), shifts AS (
	SELECT DISTINCT sequence - FIRST_VALUE(sequence) OVER (PARTITION BY tool_call_id, timestamp ORDER BY sequence) AS shift
	FROM calls WHERE tool_call_id IS NOT NULL
)`

// Retry cadence: contention inside one run's transaction backs off a few
// times, and runs still unrepaired after a pass wait for the next pass. Tests
// shorten them.
var (
	duplicateRepairBusyAttempts = 8
	duplicateRepairBusyBackoff  = 200 * time.Millisecond
	duplicateRepairPassInterval = 10 * time.Minute
)

const duplicateRepairChunk = 400

// RepairDuplicateTranscriptEvents repairs duplicated events run by run until
// every run was repaired, then writes the marker. A run that stays locked or
// fails is skipped and the pass repeats every duplicateRepairPassInterval; it
// never waits for a restart, and a restart resumes it because the marker is
// written only on completion. Each run's work is idempotent.
func (db *DB) RepairDuplicateTranscriptEvents(ctx context.Context) error {
	for _, repair := range []duplicateRepair{shiftRepair, legacyTwinRepair} {
		if err := db.runDuplicateRepair(ctx, repair); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) runDuplicateRepair(ctx context.Context, repair duplicateRepair) error {
	for {
		done, err := db.duplicateRepairDone(ctx, repair.marker)
		if err != nil || done {
			return err
		}
		failed, err := db.duplicateRepairPass(ctx, repair)
		if err != nil {
			return err
		}
		if failed == 0 {
			return db.retryBusy(ctx, func() error {
				_, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations_done (name, completed_at) VALUES (?, ?)`, repair.marker, time.Now().UTC().Format(time.RFC3339Nano))
				return err
			})
		}
		if db.log != nil {
			db.log.WithFields(logrus.Fields{"migration": repair.marker, "runs_unrepaired": failed, "retry_in": duplicateRepairPassInterval.String()}).Warn("duplicate transcript event repair incomplete")
		}
		select {
		case <-time.After(duplicateRepairPassInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (db *DB) duplicateRepairDone(ctx context.Context, marker string) (bool, error) {
	var done int
	err := db.retryBusy(ctx, func() error {
		return db.GetContext(ctx, &done, `SELECT COUNT(*) FROM schema_migrations_done WHERE name = ?`, marker)
	})
	if err != nil {
		return false, &domain.DatabaseError{Operation: "migration_marker_read", EntityType: "Schema", Cause: err}
	}
	return done > 0, nil
}

// duplicateRepairPass visits every run with tool calls and returns how many it
// could not repair.
func (db *DB) duplicateRepairPass(ctx context.Context, repair duplicateRepair) (int, error) {
	var runIDs []string
	if err := db.retryBusy(ctx, func() error {
		return db.SelectContext(ctx, &runIDs, duplicateRepairRuns)
	}); err != nil {
		return 0, &domain.DatabaseError{Operation: "duplicate_events_runs", EntityType: "RunEvent", Cause: err}
	}
	started, repaired, deleted, failed := time.Now(), 0, 0, 0
	for _, runID := range runIDs {
		if err := ctx.Err(); err != nil {
			return failed, err
		}
		n, err := db.repairDuplicateEventsOfRun(ctx, repair, runID)
		if err != nil {
			failed++
			if db.log != nil {
				db.log.WithError(err).WithField("run_id", runID).Warn("duplicate transcript event repair skipped a run")
			}
			continue
		}
		if n > 0 {
			repaired++
			deleted += n
		}
	}
	if db.log != nil {
		db.log.WithFields(logrus.Fields{"migration": repair.marker, "runs_scanned": len(runIDs), "runs_repaired": repaired, "events_deleted": deleted, "runs_failed": failed, "elapsed": time.Since(started).String()}).Info("duplicate transcript event pass")
	}
	return failed, nil
}

// repairDuplicateEventsOfRun reads the run's later copies outside any write
// lock, then deletes them in one short BEGIN IMMEDIATE transaction that also
// queues the run for read-model and conversation-search re-projection.
func (db *DB) repairDuplicateEventsOfRun(ctx context.Context, repair duplicateRepair, runID string) (int, error) {
	var victims []string
	if err := db.retryBusy(ctx, func() error {
		victims = victims[:0]
		args := make([]any, repair.args)
		for i := range args {
			args[i] = runID
		}
		return db.SelectContext(ctx, &victims, repair.victimSQL, args...)
	}); err != nil {
		return 0, &domain.DatabaseError{Operation: "duplicate_events_select", EntityType: "RunEvent", EntityID: runID, Cause: err}
	}
	if len(victims) == 0 {
		return 0, nil
	}
	err := db.retryBusy(ctx, func() error { return db.deleteDuplicateEvents(ctx, runID, victims) })
	if err != nil {
		return 0, &domain.DatabaseError{Operation: "duplicate_events_delete", EntityType: "RunEvent", EntityID: runID, Cause: err}
	}
	return len(victims), nil
}

func (db *DB) deleteDuplicateEvents(ctx context.Context, runID string, victims []string) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if _, err = conn.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_pending (run_id, generation, attempts, next_attempt_unix) VALUES (?, 1, 0, 0)`, runID); err != nil {
		return err
	}
	// The search projection still holds the doubled messages; an upsert_run
	// change makes the conversation-search worker rebuild it.
	if _, err = conn.ExecContext(ctx, `INSERT INTO conversation_search_changes (operation, source_run_id, created_at)
		SELECT 'upsert_run', ?, ? WHERE EXISTS (SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'conversation_search_changes')`, runID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	for start := 0; start < len(victims); start += duplicateRepairChunk {
		chunk := victims[start:min(start+duplicateRepairChunk, len(victims))]
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		if _, err = conn.ExecContext(ctx, `DELETE FROM run_events WHERE id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")+`)`, args...); err != nil {
			return err
		}
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

// retryBusy repeats fn with backoff while SQLite reports contention, using the
// transcript indexer's definition of a transient store failure.
func (db *DB) retryBusy(ctx context.Context, fn func() error) error {
	delay := duplicateRepairBusyBackoff
	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil || attempt >= duplicateRepairBusyAttempts || !transcriptindex.TransientStoreFailure(err.Error()) {
			return err
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
		delay = min(delay*2, 10*time.Second)
	}
}
