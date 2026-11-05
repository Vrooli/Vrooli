// Responsibility: data migrations that run exactly once per database, gated by
// a durable marker written in the migration's own transaction.
package database

import (
	"context"
	"encoding/json"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"

	"github.com/google/uuid"
)

// Migration names recorded in schema_migrations_done.
const (
	migrationQuotaEvents         = "2026-10-29-quota-event-type"
	migrationAttachedExitStatus  = "2026-10-29-attached-exit-status"
	migrationDropRetiredBookkeep = "2026-10-29-drop-stats-checkpoint-and-usage-repairs"
	// The error-code and file-reread projections served only the retired
	// error-patterns and file-reread-rate measures (tool friction profiles
	// replace both).
	migrationDropRetiredSignals = "2026-10-30-drop-error-and-reread-projections"
	// Friction episodes were deleted with their classifier; the watermark's
	// episode version and the self-report operator-correction flag went with
	// them.
	migrationDropEpisodes = "2026-10-30-drop-friction-episodes"
)

// attachedExitReason is the error the reconciler wrote when it marked every
// normally exited attached run failed (D-7).
const attachedExitReason = "attached harness process is no longer present"

// legacyQuotaEventPredicate matches provider quota frames stored as error
// events before EventTypeQuota existed: payloads tagged payloadKind=quota and
// the older untagged shape (a nonempty limitType and no generic error field).
const legacyQuotaEventPredicate = `event_type = 'error' AND CASE WHEN json_valid(data) THEN
	json_extract(data,'$.payloadKind') = 'quota' OR (json_type(data,'$.payloadKind') IS NULL
		AND trim(coalesce(json_extract(data,'$.limitType'),'')) <> ''
		AND json_type(data,'$.code') IS NULL AND json_type(data,'$.retryable') IS NULL
		AND json_type(data,'$.recovery') IS NULL AND json_type(data,'$.stackTrace') IS NULL
		AND json_type(data,'$.details') IS NULL)
	ELSE 0 END`

// runOnce runs fn in a transaction unless the named migration's marker exists,
// and records the marker in that same transaction. A completed migration costs
// one indexed lookup on later starts.
func (db *DB) runOnce(ctx context.Context, name string, fn func(*Tx) error) error {
	return db.WithTransaction(ctx, func(tx *Tx) error {
		var done int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations_done WHERE name = ?`, name).Scan(&done); err != nil {
			return &domain.DatabaseError{Operation: "migration_marker_read", EntityType: "Schema", Cause: err}
		}
		if done > 0 {
			return nil
		}
		if err := fn(tx); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations_done (name, completed_at) VALUES (?, ?)`, name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return &domain.DatabaseError{Operation: "migration_marker_write", EntityType: "Schema", Cause: err}
		}
		return nil
	})
}

// runOnceMigrations applies the one-shot data migrations in order.
func (db *DB) runOnceMigrations(ctx context.Context) error {
	if err := db.runOnce(ctx, migrationQuotaEvents, func(tx *Tx) error { return db.migrateQuotaEvents(ctx, tx) }); err != nil {
		return err
	}
	if err := db.runOnce(ctx, migrationAttachedExitStatus, func(tx *Tx) error { return migrateAttachedExitStatus(ctx, tx) }); err != nil {
		return err
	}
	for _, drop := range []struct {
		name   string
		tables []string
	}{
		{migrationDropRetiredBookkeep, []string{"stats_checkpoint", "transcript_usage_repairs"}},
		{migrationDropRetiredSignals, []string{"invocation_read_model_errors", "invocation_read_model_run_signals"}},
	} {
		if err := db.runOnce(ctx, drop.name, func(tx *Tx) error { return dropTables(ctx, tx, drop.tables) }); err != nil {
			return err
		}
	}
	return db.runOnce(ctx, migrationDropEpisodes, func(tx *Tx) error { return dropEpisodes(ctx, tx) })
}

// dropEpisodes removes the episode projection and the two columns that went
// with it.
func dropEpisodes(ctx context.Context, tx *Tx) error {
	if err := dropTables(ctx, tx, []string{"invocation_read_model_episodes"}); err != nil {
		return err
	}
	return dropColumns(ctx, tx, map[string][]string{
		"invocation_read_model_watermarks":        {"episode_classifier_version"},
		"invocation_read_model_self_report_spans": {"operator_correction"},
	})
}

// migrateQuotaEvents moves legacy quota frames out of the error type and
// back-fills quota_observations from them. Observation ids are stable. New
// quota events record their observation through the run-event sink hooks.
func (db *DB) migrateQuotaEvents(ctx context.Context, tx *Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, run_id, timestamp, data FROM run_events WHERE `+legacyQuotaEventPredicate)
	if err != nil {
		return &domain.DatabaseError{Operation: "quota_event_migration_query", EntityType: "RunEvent", Cause: err}
	}
	now := time.Now().UTC()
	var observations []*pricing.QuotaObservation
	for rows.Next() {
		var id, runID, data string
		var at SQLiteTime
		if err := rows.Scan(&id, &runID, &at, &data); err != nil {
			rows.Close()
			return &domain.DatabaseError{Operation: "quota_event_migration_scan", EntityType: "RunEvent", Cause: err}
		}
		eventID, idErr := uuid.Parse(id)
		runUUID, runErr := uuid.Parse(runID)
		payload := &domain.RateLimitEventData{}
		if idErr != nil || runErr != nil || json.Unmarshal([]byte(data), payload) != nil {
			continue
		}
		observation, err := pricing.ObservationFromEvent(&domain.RunEvent{ID: eventID, RunID: runUUID, EventType: domain.EventTypeQuota, Timestamp: at.Time(), Data: payload}, now)
		if err == nil && observation != nil {
			observations = append(observations, observation)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return &domain.DatabaseError{Operation: "quota_event_migration_iterate", EntityType: "RunEvent", Cause: err}
	}
	rows.Close()
	for _, o := range observations {
		var reset any
		if o.ResetAt != nil {
			reset = SQLiteTime(o.ResetAt.UTC())
		}
		if _, err := tx.ExecContext(ctx, insertQuotaObservationSQL, o.AccountScope, o.ID, o.Provider, o.Pool, o.Window, SQLiteTime(o.ObservedAt.UTC()), o.Standing, o.Used, o.Limit, o.Remaining, o.UsedPercent, o.WindowMinutes, reset, o.SourceRunID, o.Freshness, o.Provenance, o.Uncertainty, o.EvidenceRef); err != nil {
			return &domain.DatabaseError{Operation: "quota_observation_backfill", EntityType: "QuotaObservation", Cause: err}
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE run_events SET event_type = 'quota' WHERE `+legacyQuotaEventPredicate)
	if err != nil {
		return &domain.DatabaseError{Operation: "quota_event_migration_update", EntityType: "RunEvent", Cause: err}
	}
	if moved, _ := result.RowsAffected(); moved > 0 && db.log != nil {
		db.log.WithField("events", moved).WithField("observations", len(observations)).Info("Moved provider quota frames out of error events")
	}
	return nil
}

// migrateAttachedExitStatus corrects attached runs the reconciler marked
// failed only because their harness process exited: a normal exit is
// complete. It queues them for re-projection so the read model follows.
func migrateAttachedExitStatus(ctx context.Context, tx *Tx) error {
	match := `execution_mode = 'attached' AND status = 'failed' AND error_msg = ?`
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_pending (run_id, generation, attempts, next_attempt_unix)
		SELECT id, 1, 0, 0 FROM runs WHERE `+match, attachedExitReason); err != nil {
		return &domain.DatabaseError{Operation: "attached_exit_requeue", EntityType: "Run", Cause: err}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE runs SET status = 'complete', phase = 'completed', error_msg = '' WHERE `+match, attachedExitReason); err != nil {
		return &domain.DatabaseError{Operation: "attached_exit_correct", EntityType: "Run", Cause: err}
	}
	return nil
}

// dropTables drops retired tables (their indexes go with them).
func dropTables(ctx context.Context, tx *Tx, tables []string) error {
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			return &domain.DatabaseError{Operation: "schema_migrate", EntityType: "Schema", Cause: err}
		}
	}
	return nil
}

// dropColumns drops retired columns that still exist; a fresh database never
// had them.
func dropColumns(ctx context.Context, tx *Tx, columns map[string][]string) error {
	for table, retired := range columns {
		existing, err := columnsOf(ctx, tx, table)
		if err != nil {
			return err
		}
		for _, column := range retired {
			if _, ok := existing[column]; !ok {
				continue
			}
			if _, err := tx.ExecContext(ctx, "ALTER TABLE "+table+" DROP COLUMN "+column); err != nil {
				return &domain.DatabaseError{Operation: "schema_migrate", EntityType: "Schema", Cause: err}
			}
		}
	}
	return nil
}
