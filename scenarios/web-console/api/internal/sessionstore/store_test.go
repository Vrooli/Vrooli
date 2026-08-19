package sessionstore

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// sessionsTable mirrors the current sessions schema closely enough to exercise
// the store's read/write paths,
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
    archived_at TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'ui',
    owner TEXT NOT NULL DEFAULT '',
    display_label TEXT NOT NULL DEFAULT ''
);
CREATE TABLE conversation_events (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    text TEXT NOT NULL DEFAULT ''
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
	if err := s.Save(context.Background(), Metadata{ID: "sess-1", Shell: "/bin/bash"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SetProvenance(context.Background(), "sess-1", OriginProgrammatic, "agent-manager", "Nightly build"); err != nil {
		t.Fatalf("set provenance: %v", err)
	}

	got, err := s.Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin != OriginProgrammatic || got.Owner != "agent-manager" || got.DisplayLabel != "Nightly build" {
		t.Fatalf("get provenance = %q/%q/%q, want programmatic/agent-manager/Nightly build",
			got.Origin, got.Owner, got.DisplayLabel)
	}

	list, err := s.List(context.Background())
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
	if err := s.Save(context.Background(), Metadata{ID: "sess-2", Origin: OriginRemote, Owner: "bridge", DisplayLabel: "mac-mini"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(context.Background(), "sess-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin != OriginRemote || got.Owner != "bridge" || got.DisplayLabel != "mac-mini" {
		t.Fatalf("provenance = %q/%q/%q, want remote/bridge/mac-mini", got.Origin, got.Owner, got.DisplayLabel)
	}
}

func TestInMemoryStore_ProvenanceRoundTrip(t *testing.T) {
	s := NewInMemory()
	if err := s.Save(context.Background(), Metadata{ID: "sess-1"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SetProvenance(context.Background(), "sess-1", OriginUI, "", "My tab"); err != nil {
		t.Fatalf("set provenance: %v", err)
	}
	got, err := s.Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin != OriginUI || got.Owner != "" || got.DisplayLabel != "My tab" {
		t.Fatalf("provenance = %q/%q/%q, want ui//My tab", got.Origin, got.Owner, got.DisplayLabel)
	}
}

func TestSQLStore_ArchiveRoundTrip(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	s := newSQLStore(t)
	if err := s.Save(ctx, Metadata{ID: "sess-archive", Shell: "/bin/bash"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	archivedAt := time.Date(2026, 8, 18, 19, 0, 0, 0, time.UTC)
	if err := s.MarkArchived(ctx, "sess-archive", archivedAt); err != nil {
		t.Fatalf("mark archived: %v", err)
	}
	rows, err := s.ListArchived(ctx)
	if err != nil {
		t.Fatalf("list archived: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "sess-archive" || !rows[0].ArchivedAt.Equal(archivedAt) {
		t.Fatalf("archived rows = %+v", rows)
	}
	if err := s.MarkUnarchived(ctx, "sess-archive"); err != nil {
		t.Fatalf("mark unarchived: %v", err)
	}
	rows, err = s.ListArchived(ctx)
	if err != nil {
		t.Fatalf("list after unarchive: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("archived rows after unarchive = %+v, want empty", rows)
	}
	if _, err := s.Get(ctx, "sess-archive"); err != nil {
		t.Fatalf("session row was destroyed: %v", err)
	}
}

func TestSQLStore_MarkDismissedAfterReopenClearsArchiveAtomically(t *testing.T) {
	ctx := context.Background()
	s := newSQLStore(t)
	if err := s.Save(ctx, Metadata{ID: "old", ArchivedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkDismissed(ctx, "old", "replacement"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "old")
	if err != nil {
		t.Fatal(err)
	}
	if !got.ArchivedAt.IsZero() || got.RecoveredInto != "replacement" || got.Status != StatusDismissed {
		t.Fatalf("reopened archive metadata = %+v", got)
	}
}

func TestSQLStore_RetentionCandidatesRequireExplicitArchiveTimestamp(t *testing.T) {
	ctx := context.Background()
	s := newSQLStore(t)
	ancient := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	archived := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, meta := range []Metadata{
		{ID: "live-five-years-old", Status: StatusLive, Created: ancient, AgentType: AgentCodex},
		{ID: "legacy-dismissed", Status: StatusDismissed, Created: ancient, AgentType: AgentCodex},
		{ID: "explicitly-archived", Status: StatusLive, Created: ancient, ArchivedAt: archived, AgentType: AgentCodex},
	} {
		if err := s.Save(ctx, meta); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListRetentionCandidates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "explicitly-archived" {
		t.Fatalf("retention candidates = %+v, want only explicitly-archived", got)
	}
}
