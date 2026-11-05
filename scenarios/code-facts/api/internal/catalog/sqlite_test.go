package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fixedClock struct{ value time.Time }

func (c fixedClock) Now() time.Time { return c.value }

func openCatalogDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:catalog-%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrateIsRestartSafeAndFailedMigrationRollsBack(t *testing.T) {
	ctx := context.Background()
	db := openCatalogDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("restart migration must be idempotent: %v", err)
	}
	err := ApplyMigrations(ctx, db, []Migration{{Version: 99, SQL: `CREATE TABLE should_roll_back(id INTEGER); this is invalid SQL;`}})
	if err == nil {
		t.Fatal("expected migration failure")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='should_roll_back'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed migration left its partial table behind")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM code_facts_catalog_migrations WHERE version=99`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed migration left a version marker behind")
	}
}

func TestSQLiteRepositoryPersistsGenerationRevision(t *testing.T) {
	ctx := context.Background()
	db := openCatalogDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	repository := NewSQLiteRepository(db, fixedClock{value: time.Unix(100, 0)})
	if _, err := repository.GenerationRevision(ctx, "g1"); !errors.Is(err, ErrRevisionNotFound) {
		t.Fatalf("missing revision error = %v, want %v", err, ErrRevisionNotFound)
	}
	if err := repository.BeginGeneration(ctx, Generation{ID: "g1", Policy: "v1"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.SetGenerationRevision(ctx, "g1", "abc123"); err != nil {
		t.Fatal(err)
	}
	if revision, err := repository.GenerationRevision(ctx, "g1"); err != nil || revision != "abc123" {
		t.Fatalf("revision = %q, err=%v", revision, err)
	}
	if err := repository.SetGenerationRevision(ctx, "g1", "def456"); err != nil {
		t.Fatal(err)
	}
	if revision, err := repository.GenerationRevision(ctx, "g1"); err != nil || revision != "def456" {
		t.Fatalf("updated revision = %q, err=%v", revision, err)
	}
	if err := repository.RecordGenerationDirtyPaths(ctx, "g1", []string{"packages/b.go", "packages/a.go"}); err != nil {
		t.Fatal(err)
	}
	if paths, err := repository.GenerationDirtyPaths(ctx, "g1"); err != nil || fmt.Sprint(paths) != "[packages/a.go packages/b.go]" {
		t.Fatalf("dirty paths = %v, err=%v", paths, err)
	}
	if err := repository.CompleteGenerationAudit(ctx, "g1", "fed987", []string{"packages/b.go"}); err != nil {
		t.Fatal(err)
	}
	if paths, err := repository.GenerationDirtyPaths(ctx, "g1"); err != nil || fmt.Sprint(paths) != "[packages/b.go]" {
		t.Fatalf("retained dirty paths = %v, err=%v", paths, err)
	}
}

func TestSQLiteRepositoryPromotesOnlyCompleteShadowGeneration(t *testing.T) {
	ctx := context.Background()
	db := openCatalogDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	clock := fixedClock{value: time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)}
	repository := NewSQLiteRepository(db, clock)
	if err := repository.BeginGeneration(ctx, Generation{ID: "g1", Policy: "corpus-v1"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpsertFiles(ctx, "g1", []SourceFile{fixtureSource("a.go")}); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	if err := repository.BeginGeneration(ctx, Generation{ID: "g2", Policy: "corpus-v1"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(ctx, "g2"); err == nil {
		t.Fatal("empty shadow generation must not replace the serving generation")
	}
	active, err := repository.Active(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != "g1" {
		t.Fatalf("failed promotion changed serving generation to %q", active.ID)
	}
	if err := repository.UpsertFiles(ctx, "g2", []SourceFile{fixtureSource("renamed.go")}); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(ctx, "g2"); err != nil {
		t.Fatal(err)
	}
	active, err = repository.Active(ctx)
	if err != nil || active.ID != "g2" {
		t.Fatalf("new complete generation was not promoted: %+v err=%v", active, err)
	}
	if err := repository.UpsertFiles(ctx, "g1", []SourceFile{fixtureSource("late.go")}); err == nil {
		t.Fatal("retired generation must be immutable")
	}
	if err := repository.Rollback(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	active, err = repository.Active(ctx)
	if err != nil || active.ID != "g1" {
		t.Fatalf("rollback did not restore prior complete generation: %+v err=%v", active, err)
	}
}

func TestSQLiteRepositoryUsesStableKeysetPages(t *testing.T) {
	ctx := context.Background()
	db := openCatalogDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	repository := NewSQLiteRepository(db, fixedClock{value: time.Unix(100, 0)})
	if err := repository.BeginGeneration(ctx, Generation{ID: "g", Policy: "v1"}); err != nil {
		t.Fatal(err)
	}
	files := []SourceFile{fixtureSource("z.go"), fixtureSource("a.go"), fixtureSource("m.go")}
	if err := repository.UpsertFiles(ctx, "g", files); err != nil {
		t.Fatal(err)
	}
	first, err := repository.PageFiles(ctx, "g", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Files) != 2 || first.Files[0].Path != "a.go" || first.Files[1].Path != "m.go" || first.NextToken == "" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	second, err := repository.PageFiles(ctx, "g", first.NextToken, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Files) != 1 || second.Files[0].Path != "z.go" || second.NextToken != "" {
		t.Fatalf("unexpected second page: %+v", second)
	}
	if _, err := repository.PageFiles(ctx, "g", "not base64!", 2); err == nil {
		t.Fatal("invalid token must fail closed")
	}
}

func TestSQLiteRepositoryRejectsUnknownGeneration(t *testing.T) {
	db := openCatalogDB(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository := NewSQLiteRepository(db, fixedClock{value: time.Unix(100, 0)})
	err := repository.UpsertFiles(context.Background(), "missing", []SourceFile{fixtureSource("a.go")})
	if !errors.Is(err, ErrGenerationNotFound) {
		t.Fatalf("expected generation-not-found, got %v", err)
	}
}

func TestSQLiteRepositoryConvergesChangedRenamedAndDeletedFiles(t *testing.T) {
	ctx := context.Background()
	db := openCatalogDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	repository := NewSQLiteRepository(db, fixedClock{value: time.Unix(100, 0)})
	if err := repository.BeginGeneration(ctx, Generation{ID: "g", Policy: "v1"}); err != nil {
		t.Fatal(err)
	}
	original := fixtureSource("old.go")
	deleted := fixtureSource("deleted.go")
	if err := repository.UpsertFiles(ctx, "g", []SourceFile{original, deleted}); err != nil {
		t.Fatal(err)
	}
	original.Hash = "sha256:changed"
	if err := repository.UpsertFiles(ctx, "g", []SourceFile{original}); err != nil {
		t.Fatal(err)
	}
	renamed := fixtureSource("new.go")
	if err := repository.UpsertFiles(ctx, "g", []SourceFile{renamed}); err != nil {
		t.Fatal(err)
	}
	if err := repository.DeleteFiles(ctx, "g", []string{original.ID, deleted.ID}); err != nil {
		t.Fatal(err)
	}
	page, err := repository.PageFiles(ctx, "g", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Files) != 1 || page.Files[0].Path != "new.go" {
		t.Fatalf("rename/delete left stale catalog rows: %+v", page.Files)
	}
}

func TestSQLiteRepositoryBatchRollsBackOnDuplicatePath(t *testing.T) {
	ctx := context.Background()
	db := openCatalogDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	repository := NewSQLiteRepository(db, fixedClock{value: time.Unix(100, 0)})
	if err := repository.BeginGeneration(ctx, Generation{ID: "g", Policy: "v1"}); err != nil {
		t.Fatal(err)
	}
	first := fixtureSource("duplicate.go")
	second := first
	second.ID = "file:different-id"
	if err := repository.UpsertFiles(ctx, "g", []SourceFile{first, second}); err == nil {
		t.Fatal("duplicate path must fail the transaction")
	}
	page, err := repository.PageFiles(ctx, "g", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Files) != 0 {
		t.Fatalf("failed batch partially committed: %+v", page.Files)
	}
}

func fixtureSource(path string) SourceFile {
	return SourceFile{
		ID: StableFileID(path), Path: path, Language: "go", Role: RoleImplementation,
		Scope: "repository", Authority: "authoritative", Owner: "vrooli",
		Hash: "sha256:fixture", Size: 10, ModTime: time.Unix(50, 0), Searchable: true,
	}
}
