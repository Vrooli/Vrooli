package graph_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	localdb "architecture-cartographer/internal/database"
	"architecture-cartographer/internal/graph"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(graph.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return d
}

func TestSchema_AllowsExistingDatabaseBeforeSourceFingerprintColumn(t *testing.T) {
	d := db.NewSQLite(t)
	_, err := d.ExecContext(context.Background(), `
CREATE TABLE graph_snapshots (
  id            TEXT PRIMARY KEY,
  scenario      TEXT NOT NULL,
  content_hash  TEXT NOT NULL,
  payload       BLOB NOT NULL,
  extracted_at  TEXT NOT NULL,
  extraction_ms INTEGER NOT NULL DEFAULT 0
)`)
	if err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	if err := graph.MigrateSchema(context.Background(), d); err != nil {
		t.Fatalf("MigrateSchema: %v", err)
	}
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(graph.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas after graph migration: %v", err)
	}

	repo := graph.NewSQLiteRepository(d, schedule.System())
	if _, err := repo.SaveSnapshot(context.Background(), graph.GraphSnapshot{
		Scenario:          "demo",
		ContentHash:       "h1",
		SourceFingerprint: "src:h1",
	}); err != nil {
		t.Fatalf("lazy repository migration should add source_fingerprint: %v", err)
	}
}

func TestMigrateSchema_AllowsFreshDatabaseBeforeSchemaCreation(t *testing.T) {
	d := db.NewSQLite(t)
	if err := graph.MigrateSchema(context.Background(), d); err != nil {
		t.Fatalf("MigrateSchema on fresh database: %v", err)
	}
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(graph.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas after fresh migration: %v", err)
	}
}

func TestSQLiteRepository_SaveAndGetSnapshot(t *testing.T) {
	d := newSchemaDB(t)
	repo := graph.NewSQLiteRepository(d, schedule.System())

	snap := graph.GraphSnapshot{
		Scenario:          "demo",
		ContentHash:       "h1",
		SourceFingerprint: "src:h1",
		Languages:         []graph.Language{graph.LanguageGo},
		Files: []graph.FileNode{
			{ID: "file:a.go", Path: "a.go", PackageID: "pkg:demo", Language: graph.LanguageGo},
		},
		SkippedAdapters: []string{"typescript"},
	}
	persisted, err := repo.SaveSnapshot(context.Background(), snap)
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	got, err := repo.GetSnapshot(context.Background(), persisted.ID)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if got.ContentHash != "h1" {
		t.Fatalf("content_hash mismatch: %q", got.ContentHash)
	}
	if got.SourceFingerprint != "src:h1" {
		t.Fatalf("source_fingerprint mismatch: %q", got.SourceFingerprint)
	}
	if len(got.SkippedAdapters) != 1 || got.SkippedAdapters[0] != "typescript" {
		t.Fatalf("skipped adapters not round-tripped: %+v", got.SkippedAdapters)
	}
	if len(got.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got.Files))
	}
}

func TestSQLiteRepository_FindBySourceFingerprint(t *testing.T) {
	d := newSchemaDB(t)
	repo := graph.NewSQLiteRepository(d, schedule.System())
	if _, err := repo.SaveSnapshot(context.Background(), graph.GraphSnapshot{
		Scenario: "demo", ContentHash: "h1", SourceFingerprint: "src:demo",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := repo.FindBySourceFingerprint(context.Background(), "demo", "src:demo")
	if err != nil {
		t.Fatalf("FindBySourceFingerprint: %v", err)
	}
	if got.ContentHash != "h1" {
		t.Fatalf("unexpected: %+v", got)
	}

	_, err = repo.FindBySourceFingerprint(context.Background(), "demo", "missing")
	var notFound graph.ErrSnapshotNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("want ErrSnapshotNotFound, got %v", err)
	}
}

func TestSQLiteRepository_LatestSnapshotMetaDoesNotDecodePayload(t *testing.T) {
	d := newSchemaDB(t)
	repo := graph.NewSQLiteRepository(d, schedule.System())
	_, err := d.ExecContext(context.Background(), `
INSERT INTO graph_snapshots (id, scenario, content_hash, source_fingerprint, payload, extracted_at, extraction_ms)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"snap:bad", "demo", "h1", "src:h1", []byte("{not-json"), time.Now().UTC().Format(time.RFC3339Nano), 12,
	)
	if err != nil {
		t.Fatalf("seed invalid payload: %v", err)
	}

	meta, err := repo.LatestSnapshotMeta(context.Background(), "demo")
	if err != nil {
		t.Fatalf("LatestSnapshotMeta: %v", err)
	}
	if meta.ID != "snap:bad" || meta.PayloadBytes == 0 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	if _, err := repo.GetSnapshot(context.Background(), "snap:bad"); err == nil {
		t.Fatal("GetSnapshot should still reject invalid payload")
	}
}

func TestSQLiteRepository_FindByHashAndClear(t *testing.T) {
	d := newSchemaDB(t)
	repo := graph.NewSQLiteRepository(d, schedule.System())
	if _, err := repo.SaveSnapshot(context.Background(), graph.GraphSnapshot{
		Scenario: "demo", ContentHash: "h1",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := repo.FindByHash(context.Background(), "demo", "h1")
	if err != nil {
		t.Fatalf("FindByHash: %v", err)
	}
	if got.ContentHash != "h1" {
		t.Fatalf("unexpected: %+v", got)
	}

	_, err = repo.FindByHash(context.Background(), "demo", "missing")
	var notFound graph.ErrSnapshotNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("want ErrSnapshotNotFound, got %v", err)
	}

	n, err := repo.ClearSnapshots(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 deleted, got %d", n)
	}
}
