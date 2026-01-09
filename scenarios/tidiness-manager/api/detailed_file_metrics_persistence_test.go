package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// [REQ:TM-DA-001] Test persistDetailedFileMetrics with valid data
func TestPersistDetailedFileMetrics_Success(t *testing.T) {
	srv, db := setupDBTest(t)
	defer db.Close()
	ctx := context.Background()

	metrics := []DetailedFileMetrics{
		{
			FilePath:      "api/main.go",
			Language:      "go",
			FileExtension: ".go",
			LineCount:     100,
			TodoCount:     2,
			FixmeCount:    1,
			HackCount:     0,
			ImportCount:   3,
			FunctionCount: 5,
			CodeLines:     80,
			CommentLines:  20,
			CommentRatio:  0.25,
			HasTestFile:   true,
		},
	}

	err := srv.persistDetailedFileMetrics(ctx, "test-scenario", metrics)
	if err != nil {
		t.Fatalf("persistDetailedFileMetrics failed: %v", err)
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM file_metrics WHERE scenario = $1", "test-scenario").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query persisted metrics: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 persisted metric, found %d", count)
	}

	cleanupDBTest(db, "test-scenario")
}

// [REQ:TM-DA-001] Test persistDetailedFileMetrics with empty metrics
func TestPersistDetailedFileMetrics_EmptyMetrics(t *testing.T) {
	srv := &Server{
		config: &Config{},
		db:     nil,
	}

	ctx := context.Background()

	err := srv.persistDetailedFileMetrics(ctx, "test-scenario", []DetailedFileMetrics{})
	if err != nil {
		t.Errorf("persistDetailedFileMetrics should handle empty metrics: %v", err)
	}
}

// [REQ:TM-DA-001] Test persistDetailedFileMetrics upsert behavior
func TestPersistDetailedFileMetrics_Upsert(t *testing.T) {
	srv, db := setupDBTest(t)
	defer db.Close()
	ctx := context.Background()
	cleanupDBTest(db, "upsert-test")

	metrics1 := []DetailedFileMetrics{
		{
			FilePath:      "api/test.go",
			Language:      "go",
			FileExtension: ".go",
			LineCount:     50,
			TodoCount:     1,
		},
	}

	err := srv.persistDetailedFileMetrics(ctx, "upsert-test", metrics1)
	if err != nil {
		t.Fatalf("First persist failed: %v", err)
	}

	metrics2 := []DetailedFileMetrics{
		{
			FilePath:      "api/test.go",
			Language:      "go",
			FileExtension: ".go",
			LineCount:     100,
			TodoCount:     3,
		},
	}

	err = srv.persistDetailedFileMetrics(ctx, "upsert-test", metrics2)
	if err != nil {
		t.Fatalf("Second persist (upsert) failed: %v", err)
	}

	var count int
	var lineCount int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*), MAX(line_count) FROM file_metrics WHERE scenario = $1 AND file_path = $2",
		"upsert-test", "api/test.go",
	).Scan(&count, &lineCount)
	if err != nil {
		t.Fatalf("Failed to query metrics: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row after upsert, got %d", count)
	}

	if lineCount != 100 {
		t.Errorf("Expected line count to be updated to 100, got %d", lineCount)
	}

	cleanupDBTest(db, "upsert-test")
}

// [REQ:TM-DA-001] Test persistDetailedFileMetrics with database connection failure
func TestPersistDetailedFileMetrics_DBConnectionFailure(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Could not open database")
	}
	db.Close()

	srv := &Server{
		config: &Config{DatabaseURL: dbURL},
		db:     db,
	}

	ctx := context.Background()
	metrics := []DetailedFileMetrics{
		{FilePath: "test.go", Language: "go", LineCount: 10},
	}

	err = srv.persistDetailedFileMetrics(ctx, "test", metrics)
	if err == nil {
		t.Error("Expected error when persisting with closed database")
	}

	t.Logf("Got expected error: %v", err)
}

// [REQ:TM-DA-001] Test persistDetailedFileMetrics with context cancellation
func TestPersistDetailedFileMetrics_ContextCancellation(t *testing.T) {
	srv, db := setupDBTest(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	metrics := []DetailedFileMetrics{
		{FilePath: "test.go", Language: "go", LineCount: 10},
	}

	err := srv.persistDetailedFileMetrics(ctx, "test", metrics)
	if err != nil {
		t.Logf("Handled cancelled context with error: %v", err)
	}
}
