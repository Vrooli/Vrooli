package main

import (
	"context"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-007] PostgreSQL audit logging tests

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
