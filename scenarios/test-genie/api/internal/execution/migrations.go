package execution

import (
	"context"
	"fmt"

	"test-genie/internal/dbexec"
)

// Migrate evolves an existing suite_executions table to the compact runtime
// shape without reading legacy result documents. It is guarded and idempotent:
// every step introspects current state before acting, so it is safe to run on
// every boot. Historical evidence remains an operator archive/cutover concern,
// not a runtime compatibility reader.
//
// This is the execution domain's column-evolution hook — the minimal,
// data-preserving substrate aligned with storage-steer's deferred
// MigrationProvider direction (PRAGMA introspect → ALTER ADD COLUMN → backfill).
func Migrate(ctx context.Context, db dbexec.Executor) error {
	// The normalized phase projection was introduced after suite_executions.
	// Applying declarative DDL here creates it for an existing database without
	// reading or expanding the retired phases JSON column.
	if _, err := db.ExecContext(ctx, Schema()); err != nil {
		return fmt.Errorf("ensure execution phase schema: %w", err)
	}
	// requested_at is nullable on purpose: rows recorded before it existed
	// genuinely do not know when their run was requested, and inventing a value
	// (started_at, say) would report zero queue latency as though it were
	// measured. A NULL reads as "unknown" everywhere downstream.
	for _, column := range []string{"terminal_outcome", "run_id", "phase_set_digest", "descriptor_snapshot_digest", "configuration_fingerprint", "host_os", "host_arch", "host_node", "host_fact_digest", "target_kind", "target_id", "requested_at"} {
		hasColumn, err := columnExists(ctx, db, "suite_executions", column)
		if err != nil {
			return fmt.Errorf("introspect suite_executions: %w", err)
		}
		if !hasColumn {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE suite_executions ADD COLUMN %s TEXT", column)); err != nil {
				return fmt.Errorf("add suite_executions.%s: %w", column, err)
			}
		}
	}
	for _, column := range []struct {
		name string
		ddl  string
	}{
		{name: "started_at", ddl: "TEXT"},
		{name: "completed_at", ddl: "TEXT"},
		{name: "duration_ms", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "predicted_duration_ms", ddl: "INTEGER"},
		{name: "wall_clock_ms", ddl: "INTEGER"},
		{name: "cpu_user_ms", ddl: "INTEGER"},
		{name: "cpu_sys_ms", ddl: "INTEGER"},
		{name: "peak_rss_bytes", ddl: "INTEGER"},
		{name: "cpu_reliability", ddl: "TEXT"},
		{name: "memory_reliability", ddl: "TEXT"},
		{name: "gpu_reliability", ddl: "TEXT"},
		{name: "measurement_scope", ddl: "TEXT"},
		{name: "cache_hit", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "cache_source_run_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "cache_audit", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "cache_audit_mismatch", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "cache_no_saving", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "classification_source", ddl: "TEXT NOT NULL DEFAULT ''"},
	} {
		hasColumn, err := columnExists(ctx, db, "suite_execution_phases", column.name)
		if err != nil {
			return fmt.Errorf("introspect suite_execution_phases: %w", err)
		}
		if !hasColumn {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE suite_execution_phases ADD COLUMN %s %s", column.name, column.ddl)); err != nil {
				return fmt.Errorf("add suite_execution_phases.%s: %w", column.name, err)
			}
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE suite_execution_phases SET duration_ms = duration_seconds * 1000 WHERE duration_ms = 0 AND duration_seconds > 0`); err != nil {
		return fmt.Errorf("backfill suite_execution_phases.duration_ms: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE suite_executions SET target_kind = 'scenario' WHERE target_kind IS NULL OR target_kind = ''`); err != nil {
		return fmt.Errorf("backfill suite_executions.target_kind: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE suite_executions SET target_id = scenario_name WHERE target_id IS NULL OR target_id = ''`); err != nil {
		return fmt.Errorf("backfill suite_executions.target_id: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_suite_executions_target ON suite_executions (target_kind, target_id)`); err != nil {
		return fmt.Errorf("index suite_executions target identity: %w", err)
	}
	// Backfill rows that predate the column (or were written before it was
	// populated): derive the run-level outcome from the success flag. The
	// IS NULL guard makes this a no-op once every row carries a value.
	if _, err := db.ExecContext(ctx, `
UPDATE suite_executions
SET terminal_outcome = CASE WHEN success = 1 THEN ? ELSE ? END
WHERE terminal_outcome IS NULL OR terminal_outcome = ''`,
		TerminalOutcomePassed.String(), TerminalOutcomeFailed.String()); err != nil {
		return fmt.Errorf("backfill suite_executions.terminal_outcome: %w", err)
	}
	return nil
}

// columnExists reports whether a table already has the named column, via
// PRAGMA table_info introspection (SQLite-portable, no information_schema).
func columnExists(ctx context.Context, db dbexec.Executor, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
