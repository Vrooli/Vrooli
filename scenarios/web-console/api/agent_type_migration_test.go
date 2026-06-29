package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// oldSessionsTable is the sessions schema as it existed before opencode/grok —
// the CHECK constraint that a pre-migration DB carries and that would reject the
// new agent types.
const oldSessionsTable = `
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    backend TEXT NOT NULL DEFAULT 'standard',
    shell TEXT NOT NULL DEFAULT '/bin/bash',
    cols INTEGER NOT NULL DEFAULT 80,
    rows INTEGER NOT NULL DEFAULT 24,
    policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never', 'preset', 'custom')),
    policy_duration TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    detached INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'live'
        CHECK(status IN ('live','awaiting_recovery','dismissed')),
    agent_type TEXT NOT NULL DEFAULT 'none'
        CHECK(agent_type IN ('none','codex','claude')),
    launch_command TEXT NOT NULL DEFAULT '',
    agent_session_id TEXT NOT NULL DEFAULT '',
    cwd TEXT NOT NULL DEFAULT '',
    last_rollout_path TEXT NOT NULL DEFAULT '',
    last_activity_at TEXT NOT NULL DEFAULT '',
    orphaned_at TEXT NOT NULL DEFAULT '',
    recovered_into TEXT NOT NULL DEFAULT ''
);`

func openMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateSessionsAgentTypeConstraint_RelaxesOldCheck(t *testing.T) {
	db := openMigrationDB(t)
	if _, err := db.Exec(oldSessionsTable); err != nil {
		t.Fatal(err)
	}
	// Seed a pre-existing row so we can prove data survives the rebuild.
	if _, err := db.Exec(`INSERT INTO sessions (id, agent_type, agent_session_id) VALUES ('keep', 'codex', 'cid')`); err != nil {
		t.Fatal(err)
	}
	// Sanity: the old constraint rejects opencode before migration.
	if _, err := db.Exec(`INSERT INTO sessions (id, agent_type) VALUES ('pre', 'opencode')`); err == nil {
		t.Fatal("expected old CHECK to reject opencode before migration")
	}

	if err := migrateSessionsAgentTypeConstraint(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Old row preserved.
	var agent, sid string
	if err := db.QueryRow(`SELECT agent_type, agent_session_id FROM sessions WHERE id='keep'`).Scan(&agent, &sid); err != nil {
		t.Fatalf("pre-existing row lost: %v", err)
	}
	if agent != "codex" || sid != "cid" {
		t.Fatalf("row corrupted: agent=%q sid=%q", agent, sid)
	}
	// New agent types now accepted.
	for _, a := range []string{"opencode", "grok"} {
		if _, err := db.Exec(`INSERT INTO sessions (id, agent_type) VALUES (?, ?)`, "new-"+a, a); err != nil {
			t.Fatalf("insert %s after migration: %v", a, err)
		}
	}
	// The legacy temp table must be gone.
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='sessions_legacy_agentcheck'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("legacy temp table was not dropped")
	}
}

func TestMigrateSessionsAgentTypeConstraint_NoopOnCurrentSchema(t *testing.T) {
	db := setupTestDB(t) // schema.sql already carries the relaxed constraint
	// Running the migration must be a clean no-op (idempotent).
	if err := migrateSessionsAgentTypeConstraint(db); err != nil {
		t.Fatalf("no-op migration errored: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, agent_type) VALUES ('oc', 'opencode')`); err != nil {
		t.Fatalf("current schema should accept opencode: %v", err)
	}
}

func TestMigrateSessionsAgentTypeConstraint_NoTableIsNoop(t *testing.T) {
	db := openMigrationDB(t)
	if err := migrateSessionsAgentTypeConstraint(db); err != nil {
		t.Fatalf("missing-table migration should be a no-op, got %v", err)
	}
}
