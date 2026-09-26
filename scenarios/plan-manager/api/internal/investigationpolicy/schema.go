package investigationpolicy

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

// EnsureMigrations upgrades the incident ledger in-place. The policy domain
// was introduced after some local stores already existed, so new dispatch
// identity columns must be added before repositories scan the table.
func EnsureMigrations(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(plan_investigation_incidents)")
	if err != nil {
		return fmt.Errorf("inspect investigation incident columns: %w", err)
	}
	defer rows.Close()
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primary int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primary); err != nil {
			return fmt.Errorf("scan investigation incident columns: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read investigation incident columns: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close investigation incident columns: %w", err)
	}
	if len(existing) == 0 {
		return nil
	}
	for _, column := range []struct{ name, definition string }{
		{name: "program_id", definition: "program_id TEXT NOT NULL DEFAULT ''"},
		{name: "family_id", definition: "family_id TEXT NOT NULL DEFAULT ''"},
		{name: "program_status", definition: "program_status TEXT NOT NULL DEFAULT ''"},
		{name: "dispatch_error", definition: "dispatch_error TEXT NOT NULL DEFAULT ''"},
		{name: "dispatch_claim_key", definition: "dispatch_claim_key TEXT NOT NULL DEFAULT ''"},
		{name: "dispatch_started_at", definition: "dispatch_started_at TEXT NOT NULL DEFAULT ''"},
	} {
		if existing[column.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE plan_investigation_incidents ADD COLUMN "+column.definition); err != nil {
			return fmt.Errorf("add plan_investigation_incidents.%s: %w", column.name, err)
		}
	}
	if err := ensureOccurrenceColumns(ctx, db); err != nil {
		return err
	}
	if err := ensureOverrideColumns(ctx, db); err != nil {
		return err
	}
	return nil
}

func ensureOccurrenceColumns(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(plan_investigation_occurrences)")
	if err != nil {
		return fmt.Errorf("inspect investigation occurrence columns: %w", err)
	}
	defer rows.Close()
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primary int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primary); err != nil {
			return fmt.Errorf("scan investigation occurrence columns: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read investigation occurrence columns: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close investigation occurrence columns: %w", err)
	}
	if len(existing) == 0 {
		return nil
	}
	for _, column := range []struct{ name, definition string }{
		{name: "execution_id", definition: "execution_id TEXT NOT NULL DEFAULT ''"},
		{name: "family_id", definition: "family_id TEXT NOT NULL DEFAULT ''"},
		{name: "shared_failure_ref", definition: "shared_failure_ref TEXT NOT NULL DEFAULT ''"},
		{name: "phase_id", definition: "phase_id TEXT NOT NULL DEFAULT ''"},
		{name: "phase_generation", definition: "phase_generation TEXT NOT NULL DEFAULT ''"},
		{name: "policy_version", definition: "policy_version TEXT NOT NULL DEFAULT ''"},
		{name: "eligible", definition: "eligible INTEGER NOT NULL DEFAULT 1"},
	} {
		if existing[column.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE plan_investigation_occurrences ADD COLUMN "+column.definition); err != nil {
			return fmt.Errorf("add plan_investigation_occurrences.%s: %w", column.name, err)
		}
	}
	return nil
}

func ensureOverrideColumns(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(plan_investigation_policy_overrides)")
	if err != nil {
		return fmt.Errorf("inspect investigation policy override columns: %w", err)
	}
	defer rows.Close()
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primary int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primary); err != nil {
			return fmt.Errorf("scan investigation policy override columns: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read investigation policy override columns: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close investigation policy override columns: %w", err)
	}
	if len(existing) == 0 || existing["active"] {
		return nil
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE plan_investigation_policy_overrides ADD COLUMN active INTEGER NOT NULL DEFAULT 1"); err != nil {
		return fmt.Errorf("add plan_investigation_policy_overrides.active: %w", err)
	}
	return nil
}
