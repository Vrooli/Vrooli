package graph_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"architecture-cartographer/internal/clock"
	localdb "architecture-cartographer/internal/database"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/testutil/db"

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

func TestSQLiteRepository_SaveAndGetSnapshot(t *testing.T) {
	d := newSchemaDB(t)
	repo := graph.NewSQLiteRepository(d, clock.System{})

	snap := graph.GraphSnapshot{
		Scenario:    "demo",
		ContentHash: "h1",
		Languages:   []graph.Language{graph.LanguageGo},
		Files: []graph.FileNode{
			{ID: "file:a.go", Path: "a.go", PackageID: "pkg:demo", Language: graph.LanguageGo},
		},
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
	if len(got.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got.Files))
	}
}

func TestSQLiteRepository_FindByHashAndClear(t *testing.T) {
	d := newSchemaDB(t)
	repo := graph.NewSQLiteRepository(d, clock.System{})
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
