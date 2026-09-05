package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"web-console/internal/dbx"
	"web-console/internal/grouptemplates"
	"web-console/internal/handoffrules"
	"web-console/internal/modules"
	intsessions "web-console/internal/sessions"
	"web-console/internal/snippets"

	"github.com/vrooli/api-core/database"
)

// schemaProviders returns the scenario's embedded schema + seed SQL as api-core
// schema providers. Exposing them as providers (rather than executing inline)
// lets the same SQL be applied to the primary pool at boot AND to every leased
// test pool installed at runtime, which is what makes a test-mode request find
// tables.
func schemaProviders() ([]database.SchemaProvider, error) {
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(intsessions.Schema),
		database.SchemaProviderFunc(intsessions.Seed),
	}
	providers = append(providers, modules.AllSchemas()...)
	return providers, nil
}

// initSchema applies the schema, seed, and forward-only migrations to db.
// db is whichever pool the caller wants initialized: RoutedDB.Primary() at
// boot, or a freshly leased test pool via the test-pool initializer.
func initSchema(ctx context.Context, db dbx.Handle) error {
	providers, err := schemaProviders()
	if err != nil {
		return err
	}
	return initSchemaWithProviders(ctx, db, providers)
}

// initSchemaWithProviders is initSchema's body with the schema source injected,
// so the apply/migrate/verify ordering can be exercised against SQL loaded from
// the repo rather than from a path relative to the running executable.
func initSchemaWithProviders(ctx context.Context, db dbx.Handle, providers []database.SchemaProvider) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	// Apply, migrate, THEN verify. The declared-column drift check has to run
	// last: schema.sql declares every current column, but on an existing DB
	// `CREATE TABLE IF NOT EXISTS` is a no-op, so a newly declared column only
	// lands via applyColumnMigrations below. Checking before migrating fails
	// boot on precisely the drift the next few lines repair — which is exactly
	// what adding workspace_panes.manually_unread hit.
	if err := database.ApplySchemas(ctx, db, providers...); err != nil {
		return err
	}
	log.Println("Schema initialized successfully")

	if err := applyColumnMigrations(ctx, db); err != nil {
		return err
	}
	if err := migrateCodexCheckpoints(ctx, db); err != nil {
		return fmt.Errorf("migration: codex checkpoints: %w", err)
	}
	if err := ensureConversationFTS(ctx, db); err != nil {
		return fmt.Errorf("migration: conversation FTS: %w", err)
	}
	if err := ensureActivationEvents(ctx, db); err != nil {
		return fmt.Errorf("migration: activation events: %w", err)
	}

	if err := migrateSessionsAgentTypeConstraint(ctx, db); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	if err := reconcileDefaultShortcutProfile(ctx, db); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	// EnsureSchemas is deliberately last: it reapplies the idempotent domain
	// providers and reconciles additive columns after the forward-only
	// migrations above have repaired existing databases.
	if err := database.EnsureSchemas(ctx, db, providers...); err != nil {
		return err
	}

	if err := seedExampleContent(ctx, db); err != nil {
		return err
	}

	return nil
}

// seedExampleContent writes the shipped example group template and capture
// rule through their domains' ordinary public write paths — the same calls the
// UI makes. Shipped content is data, not behaviour: there is no is-builtin
// column and no delete guard anywhere in either domain.
//
// Each seeder writes only into an EMPTY store, so an operator who deletes the
// example does not get it back on the next boot.
//
// A seeding failure is logged, not fatal. An example is a convenience; losing
// it must never keep the console from starting.
func seedExampleContent(ctx context.Context, db dbx.Handle) error {
	if err := grouptemplates.SeedExamples(ctx, grouptemplates.NewSQLStore(db)); err != nil {
		log.Printf("seed: group template example: %v", err)
	}
	if err := handoffrules.SeedExamples(ctx, handoffrules.NewSQLStore(db)); err != nil {
		log.Printf("seed: handoff rule example: %v", err)
	}
	if err := snippets.SeedExamples(ctx, snippets.NewSQLStore(db)); err != nil {
		log.Printf("seed: snippet examples: %v", err)
	}
	return nil
}

// migrateCodexCheckpoints folds the pre-existing Codex-only cursor table into
// the source-scoped checkpoint table. The migration is idempotent and drops
// the old table only after its rows have been copied.
func migrateCodexCheckpoints(ctx context.Context, db dbx.Handle) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'codex_rollout_checkpoints'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO agent_transcript_checkpoints(source, source_key, web_console_session_id, cursor, updated_at)
		SELECT 'codex_rollout', path, session_id, CAST(offset_bytes AS TEXT), updated_at
		FROM codex_rollout_checkpoints`); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `DROP TABLE codex_rollout_checkpoints`)
	return err
}

const conversationFTSMigration = "conversation_events_fts_v2_external_content"

// ensureConversationFTS creates the archive-search index and backfills old
// events in bounded transactions. Progress is durable, so an interrupted boot
// resumes rather than duplicating index rows or restarting the full backfill.
func ensureConversationFTS(ctx context.Context, db dbx.Handle) error {
	var ftsSQL string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(sql, '') FROM sqlite_master WHERE type = 'table' AND name = 'conversation_events_fts'`).Scan(&ftsSQL); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	ftsDefinition := strings.ToLower(ftsSQL)
	if ftsSQL != "" && !strings.Contains(ftsDefinition, "content='conversation_events'") && !strings.Contains(ftsDefinition, `content="conversation_events"`) {
		for _, statement := range []string{
			`DROP TRIGGER IF EXISTS conversation_events_fts_insert`,
			`DROP TRIGGER IF EXISTS conversation_events_fts_delete`,
			`DROP TRIGGER IF EXISTS conversation_events_fts_text_update`,
			`DROP TABLE conversation_events_fts`,
		} {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
	}
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS web_console_migrations (
			name TEXT PRIMARY KEY,
			last_rowid INTEGER NOT NULL DEFAULT 0,
			completed_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid', tokenize='unicode61')`,
		`DROP TRIGGER IF EXISTS conversation_events_fts_insert`,
		`DROP TRIGGER IF EXISTS conversation_events_fts_delete`,
		`DROP TRIGGER IF EXISTS conversation_events_fts_text_update`,
		`CREATE TRIGGER conversation_events_fts_insert AFTER INSERT ON conversation_events BEGIN
			INSERT INTO conversation_events_fts(rowid, text) VALUES (new.rowid, new.text);
		END`,
		`CREATE TRIGGER conversation_events_fts_delete AFTER DELETE ON conversation_events BEGIN
			INSERT INTO conversation_events_fts(conversation_events_fts, rowid, text) VALUES ('delete', old.rowid, old.text);
		END`,
		`CREATE TRIGGER conversation_events_fts_text_update AFTER UPDATE OF text ON conversation_events BEGIN
			INSERT INTO conversation_events_fts(conversation_events_fts, rowid, text) VALUES ('delete', old.rowid, old.text);
			INSERT INTO conversation_events_fts(rowid, text) VALUES (new.rowid, new.text);
		END`,
	}
	for _, statement := range ddl {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	var cursor int64
	var completed string
	err := db.QueryRowContext(ctx, `SELECT last_rowid, completed_at FROM web_console_migrations WHERE name = ?`, conversationFTSMigration).Scan(&cursor, &completed)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if completed != "" {
		return nil
	}
	// FTS5's external-content table has no stored text to delete row by row.
	// Its rebuild command asks the content table for the authoritative rows and
	// is both safer and faster than replaying a second text copy through SQL.
	if _, err := db.ExecContext(ctx, `INSERT INTO conversation_events_fts(conversation_events_fts) VALUES ('rebuild')`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO web_console_migrations(name, last_rowid, completed_at)
		VALUES (?, 0, ?)`, conversationFTSMigration, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return nil
}

// applyColumnMigrations adds columns to existing tables. ALTER TABLE ADD COLUMN
// errors if the column already exists, so we ignore that specific error. New
// columns declare their DEFAULT so pre-existing rows are backfilled by SQLite
// as part of the ADD COLUMN — origin backfills to 'ui' because every historical
// session was opened from the web UI.
func applyColumnMigrations(ctx context.Context, db dbx.Handle) error {
	migrations := []string{
		`ALTER TABLE workspace_panes ADD COLUMN supports_messages_view INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE workspace_panes ADD COLUMN manually_unread INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sessions ADD COLUMN backend TEXT NOT NULL DEFAULT 'standard'`,
		`ALTER TABLE sessions ADD COLUMN detached INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sessions ADD COLUMN status TEXT NOT NULL DEFAULT 'live'`,
		`ALTER TABLE sessions ADD COLUMN agent_type TEXT NOT NULL DEFAULT 'none'`,
		`ALTER TABLE sessions ADD COLUMN launch_command TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN agent_session_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN cwd TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN last_rollout_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN last_activity_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN orphaned_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN recovered_into TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN origin TEXT NOT NULL DEFAULT 'ui'`,
		`ALTER TABLE sessions ADD COLUMN owner TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN display_label TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_type, agent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_origin ON sessions(origin)`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_events_session_sequence ON conversation_events(session_id, sequence)`,
	}
	for _, m := range migrations {
		if _, err := db.ExecContext(ctx, m); err != nil {
			// "duplicate column name" means the column already exists — safe to ignore.
			if !isDuplicateColumnError(err) {
				return fmt.Errorf("migration: %w", err)
			}
		}
	}
	return nil
}

// migrateSessionsAgentTypeConstraint relaxes the sessions.agent_type CHECK
// constraint to admit 'opencode' and 'grok'. SQLite cannot ALTER a CHECK
// constraint in place, so a DB created before these agent types carries the old
// constraint and would reject inserts for the new runtimes. The fix is the
// canonical SQLite table-rebuild, guarded so it only runs when the live
// constraint predates these values — making it a no-op on fresh DBs (which get
// the up-to-date constraint from schema.sql) and idempotent on re-run.
//
// The column list is enumerated explicitly rather than `SELECT *` so the copy
// is insensitive to physical column ordering (an incrementally ALTER-migrated
// DB can order columns differently from schema.sql).
func migrateSessionsAgentTypeConstraint(ctx context.Context, db dbx.Handle) error {
	var tableSQL string
	if err := db.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='sessions'`,
	).Scan(&tableSQL); err != nil {
		if err == sql.ErrNoRows {
			return nil // no sessions table yet; schema.sql will create it current
		}
		return fmt.Errorf("inspect sessions table: %w", err)
	}
	// Already admits opencode → nothing to do. Also skip tables with no
	// agent_type CHECK at all (older shapes get columns added by the ALTER
	// block above and never carried the constraint).
	if strings.Contains(tableSQL, "'opencode'") || !strings.Contains(tableSQL, "CHECK(agent_type") {
		return nil
	}

	const cols = `id, backend, shell, cols, rows, policy_mode, policy_duration,
		created_at, detached, status, agent_type, launch_command, agent_session_id,
		cwd, last_rollout_path, last_activity_at, orphaned_at, recovered_into,
		origin, owner, display_label`
	stmts := []string{
		`PRAGMA foreign_keys=off`,
		`ALTER TABLE sessions RENAME TO sessions_legacy_agentcheck`,
		intsessions.AgentTypeMigrationSchema(),
		`INSERT INTO sessions (` + cols + `) SELECT ` + cols + ` FROM sessions_legacy_agentcheck`,
		`DROP TABLE sessions_legacy_agentcheck`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_type, agent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_origin ON sessions(origin)`,
		`PRAGMA foreign_keys=on`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("rebuild sessions constraint (%.40s): %w", stmt, err)
		}
	}
	log.Println("migration: relaxed sessions.agent_type CHECK to include opencode/grok")
	return nil
}

// isDuplicateColumnError checks if a SQLite error is a "duplicate column name" error.
func isDuplicateColumnError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// Server wires the HTTP router, database connection, and session manager.
