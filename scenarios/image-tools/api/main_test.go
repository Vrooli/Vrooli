package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/modules"
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

func TestHostModelRuntimeReadinessIgnoresRoutedTestDatabase(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	configureTestPoolSchemas(db)
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "test.db"), "readiness-test", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.ClearTestPool("readiness-test") })

	registry, err := internalmodels.Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := registry.ByID("sd-1.5")
	if !ok {
		t.Fatal("sd-1.5 missing from seed registry")
	}
	modelPath := t.TempDir()
	state := internalmodels.NewInstallStore(db)
	if err := state.Set(ctx, internalmodels.InstallRecord{ID: model.ID, Installed: true, Path: modelPath, Checksum: "host-artifact"}); err != nil {
		t.Fatal(err)
	}
	installer := &internalmodels.Installer{Root: t.TempDir(), Reg: registry, State: state}

	ready, stateName := hostModelRuntimeReadiness(installer)(database.WithTestMode(ctx), model.ID)
	if !ready || stateName != "ready" {
		t.Fatalf("host runtime readiness under routed test context = %t/%q, want true/ready", ready, stateName)
	}
	if ready, stateName := installer.RuntimeReadiness(database.WithTestMode(ctx), model.ID); ready || stateName != "needs_model_install" {
		t.Fatalf("direct routed read = %t/%q, want isolated test database state", ready, stateName)
	}
}
