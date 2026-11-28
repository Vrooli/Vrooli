package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	"web-console/internal/continuity"
)

// writeNativeArchive writes one native transcript of lines for harness.
func writeNativeArchive(t *testing.T, roots NativeArchiveRoots, harness, nativeID string, lines ...[]byte) {
	t.Helper()
	path := nativeArchiveFixturePath(roots, harness, nativeID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	var content []byte
	if harness == "codex" {
		content = append(content, `{"type":"session_meta","payload":{"id":"`+nativeID+`","session_id":"`+nativeID+`"}}`+"\n"...)
	}
	for _, line := range lines {
		content = append(append(content, line...), '\n')
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Importing native archive conversations catalogs each one in the transaction
// that creates its projection, so continuity health stays healthy and the
// missing-catalog repair finds nothing to do. One import mirrors a
// conversation a live pane already owns under the same native id; that must
// not leave the import uncataloged either.
func TestNativeArchiveImportKeepsContinuityHealthy(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	const mirrored = "native-claude-mirrored"
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, backend, agent_type, agent_session_id) VALUES ('live-pane', 'persistent', 'claude', ?)`, mirrored); err != nil {
		t.Fatal(err)
	}
	store := continuity.NewSQLCatalogStore(db)
	live, err := continuity.BuildCatalogRecord(continuity.Evidence{SessionID: "live-pane", AgentType: "claude", AgentSessionID: mirrored, HasConversation: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(ctx, live); err != nil {
		t.Fatal(err)
	}
	if err := store.AddAliases(ctx, live); err != nil {
		t.Fatal(err)
	}

	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	indexer := NewNativeArchiveIndexer(conversations, NewSQLAgentTranscriptCheckpointStore(db), log.New(io.Discard, "", 0))
	roots := NativeArchiveRootsFromHome(t.TempDir())
	writeNativeArchive(t, roots, "claude", mirrored, nativeArchiveCorpusLine("claude", "user", "mirrored pane question", false))
	writeNativeArchive(t, roots, "codex", "native-codex-fresh",
		nativeArchiveCorpusLine("codex", "user", "fresh archive question", false),
		nativeArchiveCorpusLine("codex", "assistant", "fresh archive answer", false))
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()

	srv := &Server{db: database.NewFromPrimary(db)}
	health, err := srv.observeContinuityHealth(ctx)
	if err != nil || health.UncatalogedConversations != 0 {
		t.Fatalf("after import, health=%+v err=%v; want no uncataloged conversations", health, err)
	}
	existing, err := store.Existing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range uniqueArchiveSessions(files) {
		if conversations.CountSessionEvents(ctx, file.WebConsoleSessionID) == 0 {
			t.Fatalf("%s was not imported", file.WebConsoleSessionID)
		}
		record, ok := existing[file.WebConsoleSessionID]
		if !ok || record.LifecycleState != continuity.StateRecoverable || record.AgentType != file.Harness {
			t.Fatalf("%s catalog record = %+v (present %v); want a recoverable %s record", file.WebConsoleSessionID, record, ok, file.Harness)
		}
	}
	repair, err := srv.reconcileCatalog(uncatalogedRepairContext(), false, "", "", "", 0, 0, true)
	if err != nil || len(repair.Items) != 0 {
		t.Fatalf("missing-catalog repair still has work after import: %+v err=%v", repair, err)
	}
}
