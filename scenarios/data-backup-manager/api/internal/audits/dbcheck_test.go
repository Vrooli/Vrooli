package audits

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteChecker_ValidDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "good.db")
	makeSQLiteDB(t, path, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT); CREATE INDEX idx_name ON users(name);")

	inv := NewSQLiteChecker().Check(context.Background(), path, "good.db")
	if inv.IntegrityStatus != "ok" {
		t.Errorf("integrity = %q, want ok", inv.IntegrityStatus)
	}
	if inv.Path != "good.db" {
		t.Errorf("path = %q, want good.db", inv.Path)
	}
	if inv.PageCount <= 0 || inv.PageSize <= 0 {
		t.Errorf("expected positive page_count/page_size, got %d/%d", inv.PageCount, inv.PageSize)
	}
	if inv.SchemaSHA256 == "" {
		t.Errorf("expected non-empty schema hash")
	}
	// table (users) + index (idx_name) = 2 schema objects.
	if inv.TableCount != 2 {
		t.Errorf("table_count = %d, want 2", inv.TableCount)
	}
}

func TestSQLiteChecker_CorruptFileReportsFailedWithoutPanic(t *testing.T) {
	// A file with the SQLite magic header but garbage body.
	path := filepath.Join(t.TempDir(), "corrupt.db")
	body := append([]byte("SQLite format 3\x00"), []byte("totally not a valid b-tree page")...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	inv := NewSQLiteChecker().Check(context.Background(), path, "corrupt.db")
	if inv.IntegrityStatus != "failed" {
		t.Errorf("integrity = %q, want failed", inv.IntegrityStatus)
	}
}

func TestSQLiteChecker_SchemaHashStableAcrossCreationOrder(t *testing.T) {
	a := filepath.Join(t.TempDir(), "a.db")
	b := filepath.Join(t.TempDir(), "b.db")
	// Same DDL, different statement order — schema hash must match (sorted).
	makeSQLiteDB(t, a, "CREATE TABLE t1 (x INTEGER); CREATE TABLE t2 (y TEXT);")
	makeSQLiteDB(t, b, "CREATE TABLE t2 (y TEXT); CREATE TABLE t1 (x INTEGER);")

	checker := NewSQLiteChecker()
	ia := checker.Check(context.Background(), a, "a.db")
	ib := checker.Check(context.Background(), b, "b.db")
	if ia.SchemaSHA256 != ib.SchemaSHA256 {
		t.Errorf("schema hash differs across creation order: %s vs %s", ia.SchemaSHA256, ib.SchemaSHA256)
	}
}

func TestSQLiteChecker_DifferentSchemaDifferentHash(t *testing.T) {
	a := filepath.Join(t.TempDir(), "a.db")
	b := filepath.Join(t.TempDir(), "b.db")
	makeSQLiteDB(t, a, "CREATE TABLE t1 (x INTEGER);")
	makeSQLiteDB(t, b, "CREATE TABLE t1 (x INTEGER, extra TEXT);")

	checker := NewSQLiteChecker()
	ia := checker.Check(context.Background(), a, "a.db")
	ib := checker.Check(context.Background(), b, "b.db")
	if ia.SchemaSHA256 == ib.SchemaSHA256 {
		t.Errorf("expected different schema hashes for different DDL")
	}
}
