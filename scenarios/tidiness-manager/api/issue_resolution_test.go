package main

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// countOpenIssues returns the number of open issues for a scenario+category.
func countOpenIssues(t *testing.T, db *sql.DB, scenario, category string) int {
	t.Helper()
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM issues
		WHERE scenario = $1 AND category = $2 AND status = 'open'
	`, scenario, category).Scan(&count)
	if err != nil {
		t.Fatalf("countOpenIssues: %v", err)
	}
	return count
}

// countResolvedIssues returns the number of resolved issues for a scenario+category.
func countResolvedIssues(t *testing.T, db *sql.DB, scenario, category string) int {
	t.Helper()
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM issues
		WHERE scenario = $1 AND category = $2 AND status = 'resolved'
	`, scenario, category).Scan(&count)
	if err != nil {
		t.Fatalf("countResolvedIssues: %v", err)
	}
	return count
}

// cleanupTestIssues removes all test issues for the given scenario.
func cleanupTestIssues(t *testing.T, db *sql.DB, scenario string) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM issues WHERE scenario = $1`, scenario)
}

func TestResolveStaleMetricIssues_ResolvesOutdated(t *testing.T) {
	srv := setupTestServerOrSkip(t)
	scenario := "test-resolve-outdated"
	defer cleanupTestIssues(t, srv.store.db, scenario)

	// Insert open issues for two files
	insertTestIssue(t, srv.store.db, scenario, "api/big_file.go", "length", "high",
		"File has 800 lines", "File has 800 lines, exceeds threshold of 500 lines")
	insertTestIssue(t, srv.store.db, scenario, "api/dup_file.go", "duplication", "medium",
		"File has 25% dup", "File has 25.0% duplicated code, exceeds threshold of 10.0%")

	// Fresh issues only include the length issue (duplication was fixed)
	freshIssues := []Issue{
		{Scenario: scenario, File: "api/big_file.go", Category: "length"},
	}

	resolved, err := srv.store.ResolveStaleMetricIssues(context.Background(), scenario, freshIssues)
	if err != nil {
		t.Fatalf("ResolveStaleMetricIssues: %v", err)
	}
	if resolved != 1 {
		t.Errorf("expected 1 resolved, got %d", resolved)
	}

	// Length issue should still be open
	if got := countOpenIssues(t, srv.store.db, scenario, "length"); got != 1 {
		t.Errorf("expected 1 open length issue, got %d", got)
	}
	// Duplication issue should be resolved
	if got := countResolvedIssues(t, srv.store.db, scenario, "duplication"); got != 1 {
		t.Errorf("expected 1 resolved duplication issue, got %d", got)
	}
}

func TestResolveStaleMetricIssues_KeepsValidIssues(t *testing.T) {
	srv := setupTestServerOrSkip(t)
	scenario := "test-resolve-keeps"
	defer cleanupTestIssues(t, srv.store.db, scenario)

	insertTestIssue(t, srv.store.db, scenario, "api/big.go", "length", "high",
		"File has 800 lines", "File has 800 lines")
	insertTestIssue(t, srv.store.db, scenario, "api/complex.go", "complexity", "medium",
		"Complexity 20", "File has max complexity of 20")

	// Fresh issues match both existing issues
	freshIssues := []Issue{
		{Scenario: scenario, File: "api/big.go", Category: "length"},
		{Scenario: scenario, File: "api/complex.go", Category: "complexity"},
	}

	resolved, err := srv.store.ResolveStaleMetricIssues(context.Background(), scenario, freshIssues)
	if err != nil {
		t.Fatalf("ResolveStaleMetricIssues: %v", err)
	}
	if resolved != 0 {
		t.Errorf("expected 0 resolved, got %d", resolved)
	}

	if got := countOpenIssues(t, srv.store.db, scenario, "length"); got != 1 {
		t.Errorf("expected 1 open length issue, got %d", got)
	}
	if got := countOpenIssues(t, srv.store.db, scenario, "complexity"); got != 1 {
		t.Errorf("expected 1 open complexity issue, got %d", got)
	}
}

func TestResolveStaleMetricIssues_ResolvesDeletedFiles(t *testing.T) {
	srv := setupTestServerOrSkip(t)
	scenario := "test-resolve-deleted"
	defer cleanupTestIssues(t, srv.store.db, scenario)

	insertTestIssue(t, srv.store.db, scenario, "api/gone.go", "length", "low",
		"File has 600 lines", "File has 600 lines")

	// Fresh issues are empty (file was deleted, no more violations)
	freshIssues := []Issue{}

	resolved, err := srv.store.ResolveStaleMetricIssues(context.Background(), scenario, freshIssues)
	if err != nil {
		t.Fatalf("ResolveStaleMetricIssues: %v", err)
	}
	if resolved != 1 {
		t.Errorf("expected 1 resolved, got %d", resolved)
	}

	if got := countOpenIssues(t, srv.store.db, scenario, "length"); got != 0 {
		t.Errorf("expected 0 open length issues, got %d", got)
	}
}

func TestResolveStaleMetricIssues_Idempotent(t *testing.T) {
	srv := setupTestServerOrSkip(t)
	scenario := "test-resolve-idempotent"
	defer cleanupTestIssues(t, srv.store.db, scenario)

	insertTestIssue(t, srv.store.db, scenario, "api/old.go", "duplication", "low",
		"File has 15% dup", "File has 15.0% dup")

	freshIssues := []Issue{} // no fresh issues → resolve all

	resolved1, err := srv.store.ResolveStaleMetricIssues(context.Background(), scenario, freshIssues)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if resolved1 != 1 {
		t.Errorf("first call: expected 1 resolved, got %d", resolved1)
	}

	// Second call should resolve 0 (already resolved)
	resolved2, err := srv.store.ResolveStaleMetricIssues(context.Background(), scenario, freshIssues)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if resolved2 != 0 {
		t.Errorf("second call: expected 0 resolved, got %d", resolved2)
	}
}

func TestResolveStaleMetricIssues_IgnoresNonMetricCategories(t *testing.T) {
	srv := setupTestServerOrSkip(t)
	scenario := "test-resolve-nonmetric"
	defer cleanupTestIssues(t, srv.store.db, scenario)

	// Insert lint and type issues (non-metric categories)
	insertTestIssue(t, srv.store.db, scenario, "api/file.go", "lint", "warning",
		"unused import", "unused import bytes")

	// Use a unique timestamp to insert a second issue on the same file
	_, err := srv.store.db.Exec(`
		INSERT INTO issues (scenario, file_path, category, severity, title, description, line_number, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, scenario, "ui/src/App.tsx", "type", "error", "TS2345", "Type error", 42, "open", time.Now())
	if err != nil {
		t.Fatalf("insert type issue: %v", err)
	}

	// Resolve with empty fresh set — should NOT touch lint/type issues
	resolved, err := srv.store.ResolveStaleMetricIssues(context.Background(), scenario, []Issue{})
	if err != nil {
		t.Fatalf("ResolveStaleMetricIssues: %v", err)
	}
	if resolved != 0 {
		t.Errorf("expected 0 resolved (lint/type untouched), got %d", resolved)
	}

	if got := countOpenIssues(t, srv.store.db, scenario, "lint"); got != 1 {
		t.Errorf("expected 1 open lint issue, got %d", got)
	}
}
