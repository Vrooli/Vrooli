package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"experience-manager/internal/reconcile"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

func TestRunEvidenceRetentionCycleOpensReaderOnLiveWALDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "experience-manager.db")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	writer.SetMaxOpenConns(1)
	if _, err := writer.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatal(err)
	}
	if err := apidb.EnsureSchemas(ctx, writer, apidb.SchemaProviderFunc(reconcile.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := reconcile.NewSQLiteRepository(writer)
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		ev := reconcile.Evidence{
			ID: fmt.Sprintf("ev-%d", i), Scenario: "demo", DocumentKind: "page", PageID: "home", Route: "/", StateID: "default",
			ClaimID: "cta", ClaimType: "x", Verdict: "passed", CaptureRef: "r", AXNodeJSON: "{}", Message: "m",
			CheckedAt: now.Add(-time.Duration(30+i) * 24 * time.Hour).Format(time.RFC3339Nano),
		}
		if err := repo.SaveEvidence(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}

	res, err := runEvidenceRetentionCycle(ctx, path, writer, reconcile.DefaultEvidenceRetentionPolicy(), now)
	if err != nil {
		t.Fatalf("runEvidenceRetentionCycle: %v", err)
	}
	if res.DeletedRows != 2 {
		t.Errorf("DeletedRows = %d, want 2 (keep newest 3 of 5)", res.DeletedRows)
	}
}
