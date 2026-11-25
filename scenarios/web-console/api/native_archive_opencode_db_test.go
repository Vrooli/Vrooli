package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenCodeV2SQLiteArchiveTailsAndSearchesThroughSharedCodec(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	storeDB := setupTestDB(t)
	if err := ensureConversationFTS(ctx, storeDB); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(storeDB))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(storeDB)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))

	databasePath := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+databasePath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE session_message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, seq INTEGER NOT NULL, data TEXT NOT NULL);
	CREATE TABLE session_part (message_id TEXT NOT NULL, session_id TEXT NOT NULL, ordinal INTEGER NOT NULL, data TEXT NOT NULL);
	INSERT INTO session_message VALUES
	('message-user','ses_fixture_61',1,'{"id":"message-user","sessionID":"ses_fixture_61","role":"user","time":{"created":1782095815107}}'),
	('message-assistant','ses_fixture_61',2,'{"id":"message-assistant","sessionID":"ses_fixture_61","role":"assistant","time":{"created":1782095815108}}');
	INSERT INTO session_part VALUES
	('message-user','ses_fixture_61',0,'{"id":"part-user","type":"text","text":"Which database path should the fixture use?"}'),
	('message-assistant','ses_fixture_61',0,'{"id":"part-assistant","type":"text","text":"OpenCode SQLite marker cedar-lantern-641."}');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	roots := NativeArchiveRootsFromHome(home)
	roots[openCodeDatabaseRoot] = []string{databasePath}
	roots[openCodeSnapshotRoot] = []string{filepath.Join(home, "wc-snapshots")}
	engine, err := indexer.NewLazyDiscoveredEngine(roots)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	engine.Scan(ctx)
	if got := conversations.CountSessionEvents(ctx, "wc-archive-opencode-ses_fixture_61"); got != 2 {
		t.Fatalf("OpenCode v2 archive event count = %d, want 2", got)
	}
	results, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "cedar-lantern-641", Limit: 5}})
	if err != nil || len(results.Matches) != 1 {
		t.Fatalf("OpenCode v2 archive search = %+v, err=%v", results.Matches, err)
	}
	match := results.Matches[0]
	if match.SessionID != "wc-archive-opencode-ses_fixture_61" {
		t.Fatalf("OpenCode v2 archive identity = %q", match.SessionID)
	}
	event, ok := conversations.GetEvent(ctx, match.SessionID, match.EventID)
	if !ok || event.OccurredAt.IsZero() || event.Text != "OpenCode SQLite marker cedar-lantern-641." {
		t.Fatalf("OpenCode v2 archive timestamp/text = %+v, found=%v", event, ok)
	}

	// Appending a later native message rebuilds the owned snapshot; the same
	// Web Console engine discovers it and the shared parser makes it searchable.
	db, err = sql.Open("sqlite", "file:"+databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session_message VALUES ('message-followup','ses_fixture_61',3,'{"id":"message-followup","sessionID":"ses_fixture_61","role":"assistant","time":{"created":1782095815109}}');
	INSERT INTO session_part VALUES ('message-followup','ses_fixture_61',0,'{"id":"part-followup","type":"text","text":"OpenCode SQLite tail marker violet-bridge-315."}');`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	if got := conversations.CountSessionEvents(ctx, "wc-archive-opencode-ses_fixture_61"); got != 3 {
		t.Fatalf("OpenCode v2 append event count = %d, want 3", got)
	}
	updated, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "violet-bridge-315", Limit: 5}})
	if err != nil || len(updated.Matches) != 1 {
		t.Fatalf("OpenCode v2 appended tail search = %+v, err=%v", updated.Matches, err)
	}
}

func TestOpenCodeNativeImportRejectsUnknownDatabaseSchema(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	databasePath := filepath.Join(home, "opencode.db")
	db, err := sql.Open("sqlite", "file:"+databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE unrelated (id TEXT PRIMARY KEY);`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	roots := NativeArchiveRoots{
		openCodeDatabaseRoot: {databasePath},
		openCodeSnapshotRoot: {filepath.Join(home, "snapshots")},
	}
	if _, err := DiscoverNativeArchiveFiles(ctx, roots); err == nil {
		t.Fatal("OpenCode archive without a supported message/part schema passed import")
	}
	if _, err := os.Stat(filepath.Join(home, "snapshots")); !os.IsNotExist(err) {
		t.Fatalf("failed OpenCode import left a snapshot path: err=%v", err)
	}
}
