package opencodearchive

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshReadsV1ArchiveAndUpdatesStableSessionSnapshot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	databasePath := filepath.Join(root, "opencode.db")
	cacheRoot := CacheDirectory(databasePath, filepath.Join(root, "cache"))
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE session (id TEXT PRIMARY KEY)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, seq INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE part (message_id TEXT NOT NULL, session_id TEXT NOT NULL, ordinal INTEGER NOT NULL, data TEXT NOT NULL)`,
		`INSERT INTO session VALUES ('ses_v1')`,
		`INSERT INTO message VALUES ('msg_v1', 'ses_v1', 1, '{"id":"msg_v1","sessionID":"ses_v1","role":"assistant"}')`,
		`INSERT INTO part VALUES ('msg_v1', 'ses_v1', 0, '{"type":"text","text":"first"}')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	exists, err := SessionExists(ctx, databasePath, "session", "ses_v1")
	if err != nil || !exists {
		t.Fatalf("SessionExists() = %t, %v", exists, err)
	}
	if count, err := Refresh(ctx, databasePath, cacheRoot); err != nil || count != 1 {
		t.Fatalf("Refresh() = %d, %v", count, err)
	}
	entries, err := os.ReadDir(cacheRoot)
	if err != nil || len(entries) != 1 {
		t.Fatalf("snapshot entries = %v, %v", entries, err)
	}
	path := filepath.Join(cacheRoot, entries[0].Name())
	first, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(first), `"text":"first"`) {
		t.Fatalf("first snapshot = %s, %v", first, err)
	}

	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE part SET data = '{"type":"text","text":"updated"}' WHERE message_id = 'msg_v1'`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if count, err := Refresh(ctx, databasePath, cacheRoot); err != nil || count != 1 {
		t.Fatalf("second Refresh() = %d, %v", count, err)
	}
	updated, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(updated), `"text":"updated"`) {
		t.Fatalf("updated snapshot = %s, %v", updated, err)
	}
}
