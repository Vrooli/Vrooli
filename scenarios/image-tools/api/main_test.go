package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
)

func TestConfigureTestPoolSchemasInitializesOnlyLeasedPool(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	configureTestPoolSchemas(db)
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), "schema-test", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.ClearTestPool("schema-test") })

	var testTables int
	if err := db.QueryRowContext(database.WithTestMode(ctx), "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='jobs'").Scan(&testTables); err != nil {
		t.Fatal(err)
	}
	if testTables != 1 {
		t.Fatalf("leased pool jobs table count = %d, want 1", testTables)
	}
	var operationIndex int
	if err := db.QueryRowContext(database.WithTestMode(ctx), "SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_jobs_client_operation'").Scan(&operationIndex); err != nil {
		t.Fatal(err)
	}
	if operationIndex != 1 {
		t.Fatalf("leased pool idempotency index count = %d, want 1", operationIndex)
	}
	var primaryTables int
	if err := db.Primary().QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='jobs'").Scan(&primaryTables); err != nil {
		t.Fatal(err)
	}
	if primaryTables != 0 {
		t.Fatalf("test-pool initialization touched primary jobs table; count=%d", primaryTables)
	}
}
