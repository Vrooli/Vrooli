package journal

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the journal domain's SQL contribution.
func Schema() string { return schemaSQL }

type schemaMigrator interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// EnsureSchema upgrades the additive journal columns required by provenance
// and reversal reasons. Fresh databases already receive them from schema.sql;
// this path preserves live append-only rows created before Phase 10.
func EnsureSchema(ctx context.Context, db schemaMigrator) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(journal_events)`)
	if err != nil {
		return fmt.Errorf("inspect journal event schema: %w", err)
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan journal event schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close journal event schema rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate journal event schema: %w", err)
	}
	additions := []struct {
		name string
		ddl  string
	}{
		{"reason", `ALTER TABLE journal_events ADD COLUMN reason TEXT NOT NULL DEFAULT ''`},
		{"actor_kind", `ALTER TABLE journal_events ADD COLUMN actor_kind TEXT NOT NULL DEFAULT 'operator' CHECK (actor_kind IN ('operator', 'agent'))`},
		{"actor_verification_status", `ALTER TABLE journal_events ADD COLUMN actor_verification_status TEXT NOT NULL DEFAULT 'absent' CHECK (actor_verification_status IN ('verified', 'unavailable', 'invalid', 'absent'))`},
		{"actor_run_id", `ALTER TABLE journal_events ADD COLUMN actor_run_id TEXT NOT NULL DEFAULT ''`},
	}
	for _, addition := range additions {
		if columns[addition.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, addition.ddl); err != nil {
			return fmt.Errorf("add journal event column %s: %w", addition.name, err)
		}
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_events_one_reversal ON journal_events(cause_reference) WHERE kind = 'reversal'`); err != nil {
		return fmt.Errorf("ensure one-reversal index: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS journal_reversal_receipts (
		idempotency_key TEXT PRIMARY KEY CHECK (length(trim(idempotency_key)) > 0),
		original_event_id TEXT NOT NULL REFERENCES journal_events(id),
		reversal_event_id TEXT NOT NULL UNIQUE REFERENCES journal_events(id),
		created_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("ensure journal reversal receipts: %w", err)
	}
	return nil
}
