package main

import (
	"context"
	"strings"
	"testing"
)

// A sessions table created before the idle mode rejects policy_mode='idle'.
// The migration must widen only that CHECK and keep every row, column and
// index.
func TestMigrateSessionsPolicyModeConstraintAdmitsIdle(t *testing.T) {
	db := openMigrationDB(t)
	ctx := context.Background()
	for _, stmt := range []string{
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never', 'preset', 'custom')),
			policy_duration TEXT,
			agent_type TEXT NOT NULL DEFAULT 'none'
				CHECK(agent_type IN ('none','codex','claude','opencode','grok')),
			archived_at TEXT NOT NULL DEFAULT '',
			archive_note TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX idx_sessions_agent ON sessions(agent_type)`,
		`INSERT INTO sessions (id, policy_mode, policy_duration, agent_type, archived_at, archive_note) VALUES ('keep', 'preset', '8h', 'codex', '2026-11-21T05:47:44Z', 'note')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, policy_mode) VALUES ('pre', 'idle')`); err == nil {
		t.Fatal("the old CHECK should reject idle before migration")
	}

	if err := migrateSessionsPolicyModeConstraint(ctx, db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, policy_mode, policy_duration) VALUES ('idle', 'idle', '6h')`); err != nil {
		t.Fatalf("idle rejected after migration: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, policy_mode) VALUES ('bad', 'bogus')`); err == nil {
		t.Fatal("the widened CHECK must still reject unknown modes")
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, agent_type) VALUES ('bad-agent', 'bogus')`); err == nil {
		t.Fatal("other constraints must survive the rebuild")
	}
	var mode, duration, agent, archivedAt, note string
	if err := db.QueryRow(`SELECT policy_mode, policy_duration, agent_type, archived_at, archive_note FROM sessions WHERE id='keep'`).Scan(&mode, &duration, &agent, &archivedAt, &note); err != nil {
		t.Fatalf("existing row lost: %v", err)
	}
	if mode != "preset" || duration != "8h" || agent != "codex" || archivedAt != "2026-11-21T05:47:44Z" || note != "note" {
		t.Fatalf("existing row changed: %s %s %s %s %s", mode, duration, agent, archivedAt, note)
	}
	var index string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_sessions_agent' AND tbl_name='sessions'`).Scan(&index); err != nil {
		t.Fatalf("index not recreated: %v", err)
	}
	var tableSQL string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE name='sessions'`).Scan(&tableSQL); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tableSQL, "policy_rebuild") {
		t.Fatalf("rebuilt table kept its temporary name: %s", tableSQL)
	}
	// Idempotent.
	if err := migrateSessionsPolicyModeConstraint(ctx, db); err != nil {
		t.Fatalf("second run: %v", err)
	}
}
