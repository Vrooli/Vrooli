package filepreview

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func sqliteFixture(t *testing.T) (string, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "active.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE items(id INTEGER PRIMARY KEY, title TEXT); INSERT INTO items VALUES(1,'in WAL')"); err != nil {
		t.Fatal(err)
	}
	return path, db
}
func snapshot(t *testing.T, path string) ([]byte, error) {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return SQLiteSnapshot(context.Background(), path, stat.Size(), stat.ModTime().UnixNano())
}
func TestSQLiteSnapshotIncludesWALWithoutSourceWrites(t *testing.T) {
	path, db := sqliteFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	walBefore, err := os.ReadFile(path + "-wal")
	if err != nil {
		t.Fatal(err)
	}
	data, err := snapshot(t, path)
	if err != nil {
		t.Fatal(err)
	}
	if data[18] != 1 || data[19] != 1 {
		t.Fatalf("snapshot is not standalone: %d %d", data[18], data[19])
	}
	copyPath := filepath.Join(t.TempDir(), "snapshot.sqlite")
	if err = os.WriteFile(copyPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	copyDB, err := sql.Open("sqlite", copyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	var title string
	if err = copyDB.QueryRow("SELECT title FROM items").Scan(&title); err != nil || title != "in WAL" {
		t.Fatalf("missing WAL row %q: %v", title, err)
	}
	after, _ := os.ReadFile(path)
	walAfter, _ := os.ReadFile(path + "-wal")
	if string(before) != string(after) || string(walBefore) != string(walAfter) {
		t.Fatal("source bytes changed")
	}
	if _, err = db.Exec("INSERT INTO items VALUES(2,'new row')"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = copyDB.QueryRow("SELECT count(*) FROM items").Scan(&count); err != nil || count != 1 {
		t.Fatal("snapshot changed with live source")
	}
	next, err := snapshot(t, path)
	if err != nil || string(next) == string(data) {
		t.Fatal("refresh did not include new commit", err)
	}
}
func TestSQLiteSnapshotRejectsStaleInvalidOversizeAndCancellation(t *testing.T) {
	path, _ := sqliteFixture(t)
	stat, _ := os.Stat(path)
	if _, err := SQLiteSnapshot(context.Background(), path, stat.Size()+1, stat.ModTime().UnixNano()); err == nil {
		t.Fatal("stale source admitted")
	}
	invalid := filepath.Join(t.TempDir(), "bad.db")
	if err := os.WriteFile(invalid, make([]byte, 100), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot(t, invalid); !errors.Is(err, ErrSQLiteInvalid) {
		t.Fatal(err)
	}
	oversized := filepath.Join(t.TempDir(), "huge.db")
	f, err := os.Create(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(MaxSQLiteBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := snapshot(t, oversized); !errors.Is(err, ErrSQLiteLimit) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SQLiteSnapshot(ctx, path, stat.Size(), stat.ModTime().UnixNano()); err == nil {
		t.Fatal("cancellation ignored")
	}
	if _, err := SQLiteSnapshot(context.Background(), filepath.Join(t.TempDir(), "missing.db"), 0, 0); !errors.Is(err, ErrSQLiteInvalid) {
		t.Fatal(err)
	}
}
func TestSQLiteSnapshotBoundsWALGrowth(t *testing.T) {
	path, db := sqliteFixture(t)
	if _, err := db.Exec("INSERT INTO items VALUES(2,zeroblob(?))", MaxSQLiteBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot(t, path); !errors.Is(err, ErrSQLiteLimit) {
		t.Fatal("oversize logical database accepted", err)
	}
}
func TestSQLiteClassification(t *testing.T) {
	for _, name := range []string{"a.db", "a.SQLITE", "a.sqlite3", "extensionless"} {
		t.Run(name, func(t *testing.T) {
			c := classify(name, func() ([]byte, error) { return []byte("SQLite format 3\x00xxxxxxxxxxxxxxxxxxxxxxxx"), nil })
			if c.kind != KindDatabase {
				t.Fatal(c)
			}
		})
	}
	if !KindDatabase.UsesBlob() || !KindDatabase.CanPreview() || KindDatabase.TextContentAvailable() {
		t.Fatal("bad capabilities")
	}
}

func TestSQLiteSnapshotReportsBusyWithoutWaitingForUnboundedWork(t *testing.T) {
	path, db := sqliteFixture(t)
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE); PRAGMA journal_mode=DELETE; BEGIN EXCLUSIVE"); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("ROLLBACK")
	if _, err := snapshot(t, path); !errors.Is(err, ErrSQLiteBusy) {
		t.Fatalf("expected busy database: %v", err)
	}
}
func TestSQLiteSnapshotBoundsConcurrency(t *testing.T) {
	sqliteSlots <- struct{}{}
	sqliteSlots <- struct{}{}
	defer func() { <-sqliteSlots; <-sqliteSlots }()
	if _, err := SQLiteSnapshot(context.Background(), "unused", 0, 0); !errors.Is(err, ErrSQLiteBusy) {
		t.Fatal(err)
	}
}

func TestSQLiteReadOnlyURIEncodesPortablePaths(t *testing.T) {
	for _, filename := range []string{"/tmp/report ?#.sqlite", "C:/Users/Example/report ?#.sqlite", "//server/share/report.sqlite"} {
		t.Run(filename, func(t *testing.T) {
			u, err := url.Parse(sqliteReadOnlyURI(filename))
			if err != nil {
				t.Fatal(err)
			}
			if u.Host != "" || u.Scheme != "file" || u.Query().Get("mode") != "ro" {
				t.Fatalf("unsafe or nonportable URI: %s", u)
			}
			expected := filepath.ToSlash(filename)
			if len(expected) > 2 && expected[1] == ':' {
				expected = "/" + expected
			}
			if u.Path != expected {
				t.Fatalf("path mismatch %q != %q", u.Path, expected)
			}
		})
	}
}
