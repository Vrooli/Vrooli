package main

import (
	"context"
	"testing"
	"time"

	testdb "github.com/vrooli/api-core/databasetest"
)

// [REQ:GCT-OT-P0-007] SQLite audit logging tests

func newTestSQLiteAuditLogger(t *testing.T) *SQLiteAuditLogger {
	t.Helper()

	db := testdb.OpenSQLiteFile(t, "audit.db")
	if err := ensureAuditSchema(db); err != nil {
		t.Fatalf("ensure audit schema: %v", err)
	}
	return NewSQLiteAuditLogger(db)
}

func TestSQLiteAuditLogger_LogAndQueryFilters(t *testing.T) {
	logger := newTestSQLiteAuditLogger(t)
	ctx := context.Background()
	baseTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	entries := []AuditEntry{
		{
			Operation:     AuditOpStage,
			RepoDir:       "/work/repo",
			Branch:        "main",
			Paths:         []string{"a.go", "b.go"},
			CommitHash:    "abc123",
			CommitMessage: "stage changes",
			Success:       true,
			Timestamp:     baseTime,
			Metadata:      map[string]interface{}{"source": "test"},
		},
		{
			Operation: AuditOpCommit,
			RepoDir:   "/work/repo",
			Branch:    "feature",
			Success:   false,
			Error:     "commit failed",
			Timestamp: baseTime.Add(time.Minute),
		},
		{
			Operation: AuditOpStage,
			RepoDir:   "/work/repo",
			Branch:    "feature",
			Success:   true,
			Timestamp: baseTime.Add(2 * time.Minute),
		},
	}

	for _, entry := range entries {
		if err := logger.Log(ctx, entry); err != nil {
			t.Fatalf("log %s: %v", entry.Operation, err)
		}
	}

	resp, err := logger.Query(ctx, AuditQueryRequest{Operation: AuditOpStage})
	if err != nil {
		t.Fatalf("query stage entries: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("Total = %d, want 2", resp.Total)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(resp.Entries))
	}
	if resp.Entries[0].Branch != "feature" {
		t.Fatalf("first Branch = %q, want newest feature entry", resp.Entries[0].Branch)
	}
	if resp.Entries[1].Paths[0] != "a.go" {
		t.Fatalf("oldest stage paths = %#v, want persisted paths", resp.Entries[1].Paths)
	}
	if resp.Entries[1].Metadata["source"] != "test" {
		t.Fatalf("metadata = %#v, want source=test", resp.Entries[1].Metadata)
	}

	resp, err = logger.Query(ctx, AuditQueryRequest{
		Branch: "feature",
		Since:  baseTime.Add(30 * time.Second),
		Until:  baseTime.Add(90 * time.Second),
	})
	if err != nil {
		t.Fatalf("query time window: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("Total = %d, want 1", resp.Total)
	}
	if resp.Entries[0].Operation != AuditOpCommit {
		t.Fatalf("Operation = %q, want commit", resp.Entries[0].Operation)
	}
	if resp.Entries[0].Error != "commit failed" {
		t.Fatalf("Error = %q, want commit failed", resp.Entries[0].Error)
	}
}

func TestSQLiteAuditLogger_QueryPaginationAndGracefulDegradation(t *testing.T) {
	logger := newTestSQLiteAuditLogger(t)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		if err := logger.Log(ctx, AuditEntry{
			Operation: AuditOpStage,
			RepoDir:   "/work/repo",
			Success:   true,
			Timestamp: time.Date(2026, 5, 1, 12, i, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("log entry %d: %v", i, err)
		}
	}

	resp, err := logger.Query(ctx, AuditQueryRequest{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("query paginated entries: %v", err)
	}
	if resp.Total != 4 {
		t.Fatalf("Total = %d, want 4", resp.Total)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(resp.Entries))
	}

	if NewSQLiteAuditLogger(nil) != nil {
		t.Fatal("NewSQLiteAuditLogger(nil) returned logger, want nil")
	}
	var unconfigured *SQLiteAuditLogger
	if unconfigured.IsConfigured() {
		t.Fatal("nil SQLiteAuditLogger IsConfigured = true, want false")
	}
	if err := unconfigured.Log(ctx, AuditEntry{Operation: AuditOpStage}); err != nil {
		t.Fatalf("nil SQLiteAuditLogger Log() error = %v, want nil", err)
	}
	empty, err := unconfigured.Query(ctx, AuditQueryRequest{})
	if err != nil {
		t.Fatalf("nil SQLiteAuditLogger Query() error = %v, want nil", err)
	}
	if len(empty.Entries) != 0 {
		t.Fatalf("nil SQLiteAuditLogger entries = %d, want 0", len(empty.Entries))
	}
}

func TestAuditTimestampAndJSONHelpers(t *testing.T) {
	ts := time.Date(2026, 5, 1, 12, 13, 14, 15, time.FixedZone("offset", -4*60*60))
	formatted := formatTimestamp(ts)
	parsed := parseTimestamp(formatted)
	if !parsed.Equal(ts.UTC()) {
		t.Fatalf("parseTimestamp(formatTimestamp(ts)) = %s, want %s", parsed, ts.UTC())
	}
	if !parseTimestamp("").IsZero() {
		t.Fatal("parseTimestamp(empty) = non-zero, want zero")
	}
	if !parseTimestamp("not a timestamp").IsZero() {
		t.Fatal("parseTimestamp(invalid) = non-zero, want zero")
	}
	if got := nullableJSON(nil); got != nil {
		t.Fatalf("nullableJSON(nil) = %#v, want nil", got)
	}
	if got := nullableJSON([]byte(`{"ok":true}`)); got != `{"ok":true}` {
		t.Fatalf("nullableJSON(payload) = %#v, want JSON string", got)
	}
}

func TestFakeAuditLogger_Log(t *testing.T) {
	logger := NewFakeAuditLogger()

	entry := AuditEntry{
		Operation: AuditOpStage,
		RepoDir:   "/test/repo",
		Paths:     []string{"file1.go", "file2.go"},
		Success:   true,
	}

	err := logger.Log(context.Background(), entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger.EntryCount() != 1 {
		t.Errorf("expected 1 entry, got %d", logger.EntryCount())
	}

	last := logger.LastEntry()
	if last == nil {
		t.Fatal("expected last entry to exist")
	}
	if last.Operation != AuditOpStage {
		t.Errorf("expected operation 'stage', got %q", last.Operation)
	}
	if last.RepoDir != "/test/repo" {
		t.Errorf("expected repo_dir '/test/repo', got %q", last.RepoDir)
	}
	if len(last.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(last.Paths))
	}
}

func TestFakeAuditLogger_Query(t *testing.T) {
	logger := NewFakeAuditLogger()

	// Log some entries
	_ = logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpStage,
		RepoDir:   "/test/repo",
		Success:   true,
	})
	_ = logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpCommit,
		RepoDir:   "/test/repo",
		Success:   true,
	})
	_ = logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpStage,
		RepoDir:   "/test/repo",
		Success:   false,
	})

	// Query all
	resp, err := logger.Query(context.Background(), AuditQueryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(resp.Entries))
	}

	// Query by operation
	resp, err = logger.Query(context.Background(), AuditQueryRequest{
		Operation: AuditOpStage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Errorf("expected 2 stage entries, got %d", len(resp.Entries))
	}
}

func TestFakeAuditLogger_QueryPagination(t *testing.T) {
	logger := NewFakeAuditLogger()

	// Log 5 entries
	for i := 0; i < 5; i++ {
		_ = logger.Log(context.Background(), AuditEntry{
			Operation: AuditOpStage,
			RepoDir:   "/test/repo",
			Success:   true,
		})
	}

	// Query with limit
	resp, err := logger.Query(context.Background(), AuditQueryRequest{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(resp.Entries))
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}

	// Query with offset
	resp, err = logger.Query(context.Background(), AuditQueryRequest{
		Offset: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Errorf("expected 2 entries with offset, got %d", len(resp.Entries))
	}
}

func TestFakeAuditLogger_Unconfigured(t *testing.T) {
	logger := NewFakeAuditLogger().WithUnconfigured()

	if logger.IsConfigured() {
		t.Error("expected IsConfigured to return false")
	}

	// Log should succeed but not store entry
	err := logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpStage,
		Success:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger.EntryCount() != 0 {
		t.Errorf("expected 0 entries when unconfigured, got %d", logger.EntryCount())
	}
}

func TestFakeAuditLogger_HasOperation(t *testing.T) {
	logger := NewFakeAuditLogger()

	_ = logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpStage,
		Success:   true,
	})
	_ = logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpCommit,
		Success:   true,
	})

	if !logger.HasOperation(AuditOpStage) {
		t.Error("expected HasOperation(stage) to return true")
	}
	if !logger.HasOperation(AuditOpCommit) {
		t.Error("expected HasOperation(commit) to return true")
	}
	if logger.HasOperation(AuditOpUnstage) {
		t.Error("expected HasOperation(unstage) to return false")
	}
}

func TestFakeAuditLogger_CountOperation(t *testing.T) {
	logger := NewFakeAuditLogger()

	_ = logger.Log(context.Background(), AuditEntry{Operation: AuditOpStage, Success: true})
	_ = logger.Log(context.Background(), AuditEntry{Operation: AuditOpStage, Success: true})
	_ = logger.Log(context.Background(), AuditEntry{Operation: AuditOpCommit, Success: true})

	if count := logger.CountOperation(AuditOpStage); count != 2 {
		t.Errorf("expected 2 stage operations, got %d", count)
	}
	if count := logger.CountOperation(AuditOpCommit); count != 1 {
		t.Errorf("expected 1 commit operation, got %d", count)
	}
	if count := logger.CountOperation(AuditOpUnstage); count != 0 {
		t.Errorf("expected 0 unstage operations, got %d", count)
	}
}

func TestNoOpAuditLogger(t *testing.T) {
	logger := &NoOpAuditLogger{}

	if logger.IsConfigured() {
		t.Error("expected NoOpAuditLogger.IsConfigured() to return false")
	}

	err := logger.Log(context.Background(), AuditEntry{
		Operation: AuditOpStage,
		Success:   true,
	})
	if err != nil {
		t.Errorf("expected no error from NoOp logger, got: %v", err)
	}

	resp, err := logger.Query(context.Background(), AuditQueryRequest{})
	if err != nil {
		t.Errorf("expected no error from NoOp query, got: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected empty entries from NoOp query, got %d", len(resp.Entries))
	}
}

func TestAuditEntry_TimestampAutoSet(t *testing.T) {
	logger := NewFakeAuditLogger()

	entry := AuditEntry{
		Operation: AuditOpStage,
		Success:   true,
		// No timestamp set
	}

	before := time.Now().UTC()
	_ = logger.Log(context.Background(), entry)
	after := time.Now().UTC()

	last := logger.LastEntry()
	if last.Timestamp.Before(before) || last.Timestamp.After(after) {
		t.Errorf("expected timestamp to be auto-set between %v and %v, got %v",
			before, after, last.Timestamp)
	}
}

func TestAuditEntry_IDAutoIncrement(t *testing.T) {
	logger := NewFakeAuditLogger()

	_ = logger.Log(context.Background(), AuditEntry{Operation: AuditOpStage, Success: true})
	_ = logger.Log(context.Background(), AuditEntry{Operation: AuditOpStage, Success: true})
	_ = logger.Log(context.Background(), AuditEntry{Operation: AuditOpStage, Success: true})

	if logger.Entries[0].ID != 1 {
		t.Errorf("expected first entry ID=1, got %d", logger.Entries[0].ID)
	}
	if logger.Entries[1].ID != 2 {
		t.Errorf("expected second entry ID=2, got %d", logger.Entries[1].ID)
	}
	if logger.Entries[2].ID != 3 {
		t.Errorf("expected third entry ID=3, got %d", logger.Entries[2].ID)
	}
}
