package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDatabaseIncludesCommittedWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{"PRAGMA journal_mode=WAL", "PRAGMA wal_autocheckpoint=0", "CREATE TABLE measurements (id INTEGER)", "INSERT INTO measurements VALUES (1)"} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path + "-wal"); err != nil {
		t.Fatalf("expected an open WAL: %v", err)
	}
	copyPath, err := copyDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(copyPath) })
	copyDB, err := sql.Open("sqlite", copyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	var count int
	if err := copyDB.QueryRow("SELECT count(*) FROM measurements").Scan(&count); err != nil || count != 1 {
		t.Fatalf("snapshot count = %d, error = %v; want committed WAL row", count, err)
	}
}

func TestCopyDatabaseRemovesTemporaryFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	if _, err := copyDatabase(filepath.Join(dir, "missing.db")); err == nil {
		t.Fatal("expected an error for a missing source database")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary directory contains %v, error %v", entries, err)
	}
}
