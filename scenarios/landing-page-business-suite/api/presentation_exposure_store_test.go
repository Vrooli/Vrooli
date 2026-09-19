package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/database"
)

func seedExposureStore(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite %s: %v", path, err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE exposure_probe (value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create exposure probe: %v", err)
	}
}

func countExposureRows(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite %s: %v", path, err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM exposure_probe`).Scan(&count); err != nil {
		t.Fatalf("count exposure rows: %v", err)
	}
	return count
}

func TestStrictPresentationExposureStoreRequiresLiveLeaseBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	testPath := filepath.Join(dir, "test.db")
	seedExposureStore(t, primaryPath)
	seedExposureStore(t, testPath)
	routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: primaryPath})
	if err != nil {
		t.Fatalf("open routed database: %v", err)
	}
	defer routed.Close()
	store := strictPresentationExposureStore{routed: routed}
	testCtx := database.WithTestMode(context.Background())

	if _, err := store.ExecContext(testCtx, `INSERT INTO exposure_probe(value) VALUES (?)`, "missing"); err == nil {
		t.Fatal("missing lease was accepted")
	}
	if got := countExposureRows(t, primaryPath); got != 0 {
		t.Fatalf("missing lease wrote to primary: %d rows", got)
	}

	if err := routed.InstallTestPool(context.Background(), testPath, "presentation-lease", time.Minute); err != nil {
		t.Fatalf("install test pool: %v", err)
	}
	if _, err := store.ExecContext(testCtx, `INSERT INTO exposure_probe(value) VALUES (?)`, "active"); err != nil {
		t.Fatalf("active lease write: %v", err)
	}
	if got := countExposureRows(t, primaryPath); got != 0 {
		t.Fatalf("active lease wrote to primary: %d rows", got)
	}
	if got := countExposureRows(t, testPath); got != 1 {
		t.Fatalf("active lease test-pool rows = %d, want 1", got)
	}
}

func TestStrictPresentationExposureStoreRejectsExpiredLeaseWithoutFallback(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	testPath := filepath.Join(dir, "test.db")
	seedExposureStore(t, primaryPath)
	seedExposureStore(t, testPath)
	routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: primaryPath})
	if err != nil {
		t.Fatalf("open routed database: %v", err)
	}
	defer routed.Close()
	if err := routed.InstallTestPool(context.Background(), testPath, "presentation-lease", time.Nanosecond); err != nil {
		t.Fatalf("install expiring test pool: %v", err)
	}
	time.Sleep(2 * time.Millisecond)

	_, err = (strictPresentationExposureStore{routed: routed}).ExecContext(database.WithTestMode(context.Background()), `INSERT INTO exposure_probe(value) VALUES (?)`, "expired")
	if err == nil {
		t.Fatal("expired lease was accepted")
	}
	if got := countExposureRows(t, primaryPath); got != 0 {
		t.Fatalf("expired lease wrote to primary: %d rows", got)
	}
	if got := countExposureRows(t, testPath); got != 0 {
		t.Fatalf("expired lease wrote to test pool: %d rows", got)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("expired lease returned unrelated context cancellation: %v", err)
	}
}
