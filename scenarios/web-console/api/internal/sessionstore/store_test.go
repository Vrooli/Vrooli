package sessionstore

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newSQLStore(t *testing.T) *SQLStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	schema, err := os.ReadFile("../sessions/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
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

func TestSQLStore_NativeOwnershipMetadataRoundTrip(t *testing.T) {
	s := newSQLStore(t)
	want := Metadata{ID: "managed", Created: time.Now().UTC(), LaunchMode: LaunchModeCodexAppServer, ControlMode: ControlModeNativeCapable, NativeOwner: "web-console:managed", NativeTransport: "stdio", ProviderVersion: "0.153.4", NativeThreadID: "thread-1", LastVerifiedTurnID: "turn-4", ForkedFromNativeSession: "thread-0", LaunchDescriptorJSON: `{"launchMode":"codex_app_server"}`}
	if err := s.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LaunchMode != want.LaunchMode || got.ControlMode != want.ControlMode || got.NativeOwner != want.NativeOwner || got.NativeThreadID != want.NativeThreadID || got.LastVerifiedTurnID != want.LastVerifiedTurnID || got.ForkedFromNativeSession != want.ForkedFromNativeSession {
		t.Fatalf("native metadata = %+v", got)
	}
}

func TestParseLaunchDescriptorDefaultsAndRejectsUnverifiedShape(t *testing.T) {
	d, err := ParseLaunchDescriptor("")
	if err != nil || d.LaunchMode != string(LaunchModeTerminalPTY) || d.ControlMode != string(ControlModeTranscriptOnly) {
		t.Fatalf("default descriptor = %+v, err=%v", d, err)
	}
	if _, err := ParseLaunchDescriptor(`{"agent":"claude","launchMode":"codex_app_server"}`); err == nil {
		t.Fatal("non-Codex app-server descriptor was accepted")
	}
	d, err = ParseLaunchDescriptor(`{"agent":"codex","launchMode":"codex_app_server","serverTransport":"stdio"}`)
	if err != nil || d.LaunchMode != string(LaunchModeCodexAppServer) {
		t.Fatalf("managed descriptor = %+v, err=%v", d, err)
	}
}

func TestSQLStore_LegacyCodexRowNormalizesToTerminalOwnership(t *testing.T) {
	s := newSQLStore(t)
	if _, err := s.db.ExecContext(context.Background(), `INSERT INTO sessions(id, agent_type, launch_command) VALUES ('legacy-codex', 'codex', 'codex --yolo')`); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), "legacy-codex")
	if err != nil {
		t.Fatal(err)
	}
	if got.LaunchMode != LaunchModeTerminalPTY || got.ControlMode != ControlModeTranscriptOnly || got.NativeOwner != "" {
		t.Fatalf("legacy row = %+v", got)
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

func TestSQLStore_SaveCannotResurrectArchivedMetadata(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	s := newSQLStore(t)
	if err := s.Save(ctx, Metadata{ID: "preserve", Shell: "/bin/bash"}); err != nil {
		t.Fatal(err)
	}
	archivedAt := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if err := s.MarkArchived(ctx, "preserve", archivedAt); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, Metadata{ID: "preserve", Shell: "/bin/zsh", AgentType: AgentCodex}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "preserve")
	if err != nil {
		t.Fatal(err)
	}
	if !got.ArchivedAt.Equal(archivedAt) {
		t.Fatalf("late save cleared archive marker: %+v", got)
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

func TestSQLStore_ReservedFingerprintSurvivesRuntimeAdoption(t *testing.T) {
	s := newSQLStore(t)
	ctx := t.Context()
	id := "reserved"
	fingerprint := "immutable-request-hash"
	if err := s.Save(ctx, Metadata{ID: id, CreateFingerprint: fingerprint, Owner: "agent-manager", Origin: OriginProgrammatic}); err != nil {
		t.Fatal(err)
	}
	adopted, err := s.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	adopted.Shell = "/bin/sh"
	if err := s.Save(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, id)
	if err != nil || got.CreateFingerprint != fingerprint || got.Owner != "agent-manager" {
		t.Fatalf("row=%+v err=%v", got, err)
	}
	rows, err := s.List(ctx)
	if err != nil || len(rows) != 1 || rows[0].CreateFingerprint != fingerprint {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}
