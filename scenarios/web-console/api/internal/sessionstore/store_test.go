package sessionstore

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// sessionsTable mirrors the current sessions schema (initialization/sqlite/
// schema.sql) closely enough to exercise the store's read/write paths,
// including the provenance columns.
const sessionsTable = `
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    backend TEXT NOT NULL DEFAULT 'standard',
    shell TEXT NOT NULL DEFAULT '/bin/bash',
    cols INTEGER NOT NULL DEFAULT 80,
    rows INTEGER NOT NULL DEFAULT 24,
    policy_mode TEXT NOT NULL DEFAULT 'never',
    policy_duration TEXT,
    created_at TEXT NOT NULL DEFAULT '',
    detached INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'live',
    agent_type TEXT NOT NULL DEFAULT 'none',
    launch_command TEXT NOT NULL DEFAULT '',
    agent_session_id TEXT NOT NULL DEFAULT '',
    cwd TEXT NOT NULL DEFAULT '',
    last_rollout_path TEXT NOT NULL DEFAULT '',
    last_activity_at TEXT NOT NULL DEFAULT '',
    orphaned_at TEXT NOT NULL DEFAULT '',
    recovered_into TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'ui',
    owner TEXT NOT NULL DEFAULT '',
    display_label TEXT NOT NULL DEFAULT ''
);`

func newSQLStore(t *testing.T) *SQLStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(sessionsTable); err != nil {
		t.Fatalf("create sessions table: %v", err)
	}
	return NewSQL(db)
}

// TestSQLStore_ProvenanceRoundTrip proves origin/owner/display_label survive a
// Save then a SetProvenance patch, through both Get and List.
func TestSQLStore_ProvenanceRoundTrip(t *testing.T) {
	s := newSQLStore(t)
	if err := s.Save(Metadata{ID: "sess-1", Shell: "/bin/bash"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SetProvenance("sess-1", OriginProgrammatic, "agent-manager", "Nightly build"); err != nil {
		t.Fatalf("set provenance: %v", err)
	}

	got, err := s.Get("sess-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin != OriginProgrammatic || got.Owner != "agent-manager" || got.DisplayLabel != "Nightly build" {
		t.Fatalf("get provenance = %q/%q/%q, want programmatic/agent-manager/Nightly build",
			got.Origin, got.Owner, got.DisplayLabel)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Origin != OriginProgrammatic || list[0].Owner != "agent-manager" {
		t.Fatalf("list provenance = %+v, want one row origin=programmatic owner=agent-manager", list)
	}
}

// TestSQLStore_SaveWritesProvenance proves Save persists provenance set on the
// Metadata directly (the recovery Save path relies on this).
func TestSQLStore_SaveWritesProvenance(t *testing.T) {
	s := newSQLStore(t)
	if err := s.Save(Metadata{ID: "sess-2", Origin: OriginRemote, Owner: "bridge", DisplayLabel: "mac-mini"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get("sess-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin != OriginRemote || got.Owner != "bridge" || got.DisplayLabel != "mac-mini" {
		t.Fatalf("provenance = %q/%q/%q, want remote/bridge/mac-mini", got.Origin, got.Owner, got.DisplayLabel)
	}
}

func TestInMemoryStore_ProvenanceRoundTrip(t *testing.T) {
	s := NewInMemory()
	if err := s.Save(Metadata{ID: "sess-1"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SetProvenance("sess-1", OriginUI, "", "My tab"); err != nil {
		t.Fatalf("set provenance: %v", err)
	}
	got, err := s.Get("sess-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin != OriginUI || got.Owner != "" || got.DisplayLabel != "My tab" {
		t.Fatalf("provenance = %q/%q/%q, want ui//My tab", got.Origin, got.Owner, got.DisplayLabel)
	}
}
