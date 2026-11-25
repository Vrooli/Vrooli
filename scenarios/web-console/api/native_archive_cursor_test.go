package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	_ "modernc.org/sqlite"
	conversationH "web-console/handlers/conversation"
)

func TestCursorStateDatabaseTailsIntoOwnStoreSearch(t *testing.T) {
	ctx := context.Background()
	storeDB := setupTestDB(t)
	if err := ensureConversationFTS(ctx, storeDB); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(storeDB))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(storeDB)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	search := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	dir := t.TempDir()
	roots := NativeArchiveRootsFromHome(dir)
	path := filepath.Join(roots.FirstPath("cursor"), "workspace-hash", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;
	CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB);
	INSERT INTO cursorDiskKV(key,value) VALUES
	('composerData:composer-native-42', '{"composerId":"composer-native-42","fullConversationHeadersOnly":[{"bubbleId":"h1","type":1},{"bubbleId":"a1","type":2},{"bubbleId":"a2","type":2}]}'),
	('bubbleId:composer-native-42:h1', '{"text":"Where is the cursor transcript?"}'),
	('bubbleId:composer-native-42:a1', '{"text":"Cursor native search marker amber-river-845."}'),
	('bubbleId:composer-native-42:a2', '{"text":"Cursor native search marker amber-river-845."}');`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := DiscoverNativeArchiveFiles(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := indexer.NewLazyDiscoveredEngine(roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Harness != "cursor" {
		t.Fatalf("discovered Cursor archives = %+v", files)
	}
	source, err := newNativeArchiveSource(files)
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.DiscoverFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	document, version, err := source.ReadSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	if current, err := source.SnapshotVersion(path); err != nil || current != version {
		t.Fatalf("Cursor snapshot version changed unexpectedly: current=%q snapshot=%q err=%v", current, version, err)
	}
	events, complete, err := source.DecodeDocument(path, document)
	if err != nil || !complete || len(events) != 3 {
		t.Fatalf("Cursor snapshot decode: complete=%v events=%d, want 3; err=%v", complete, len(events), err)
	}
	engine.Scan(ctx)
	firstComposerSession := cursorComposerSessionID(files[0].WebConsoleSessionID, "composer-native-42")
	secondComposerSession := cursorComposerSessionID(files[0].WebConsoleSessionID, "composer-native-43")
	if got := conversations.CountSessionEvents(ctx, firstComposerSession); got != 3 {
		t.Fatalf("indexed Cursor event count = %d, want 3", got)
	}
	initialResults, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "amber-river-845", Limit: 5}})
	if err != nil || len(initialResults.Matches) != 2 {
		t.Fatalf("initial Cursor search = %+v, err=%v", initialResults.Matches, err)
	}
	initialProvenance := map[string]struct{}{}
	for _, match := range initialResults.Matches {
		event, ok := conversations.GetEvent(ctx, match.SessionID, match.EventID)
		if !ok || event.NativeProvenance == nil {
			t.Fatalf("initial Cursor event has no native provenance: %+v", match)
		}
		initialProvenance[event.NativeProvenance.TurnID+"\x00"+event.NativeProvenance.MessageID] = struct{}{}
	}
	db, err = sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cursorDiskKV(key,value) VALUES
	('composerData:composer-native-43', '{"composerId":"composer-native-43","fullConversationHeadersOnly":[{"bubbleId":"h2","type":1},{"bubbleId":"a2","type":2}]}'),
	('bubbleId:composer-native-43:h2', '{"text":"Followup question"}'),
	('bubbleId:composer-native-43:a2', '{"text":"Cursor incremental marker blue-harbor-216."}');`); err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if got := conversations.CountSessionEvents(ctx, firstComposerSession); got != 3 {
		t.Fatalf("first Cursor conversation changed during second composer append: %d events", got)
	}
	if got := conversations.CountSessionEvents(ctx, secondComposerSession); got != 2 {
		t.Fatalf("incremental second Cursor conversation event count = %d, want 2", got)
	}
	appendedResults, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "amber-river-845", Limit: 5}})
	if err != nil || len(appendedResults.Matches) != 2 {
		t.Fatalf("Cursor search after composer append = %+v, err=%v", appendedResults.Matches, err)
	}
	appendedProvenance := map[string]struct{}{}
	for _, match := range appendedResults.Matches {
		event, ok := conversations.GetEvent(ctx, match.SessionID, match.EventID)
		if !ok || event.NativeProvenance == nil {
			t.Fatalf("Cursor event after composer append has no native provenance: %+v", match)
		}
		appendedProvenance[event.NativeProvenance.TurnID+"\x00"+event.NativeProvenance.MessageID] = struct{}{}
	}
	if len(initialProvenance) != len(appendedProvenance) {
		t.Fatalf("Cursor composer append changed native identities: before=%v after=%v", initialProvenance, appendedProvenance)
	}
	for identity := range initialProvenance {
		if _, ok := appendedProvenance[identity]; !ok {
			t.Fatalf("Cursor composer append changed existing native identity %q: after=%v", identity, appendedProvenance)
		}
	}
	route, err := search.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "marker", Limit: 10}))
	if err != nil {
		t.Fatalf("production SearchArchived route for Cursor composers: %v", err)
	}
	routeSessions := map[string]struct{}{}
	for _, match := range route.Msg.GetMatches() {
		routeSessions[match.GetSessionId()] = struct{}{}
	}
	if route.Msg.GetDistinctSessions() != 2 || len(routeSessions) != 2 {
		t.Fatalf("production SearchArchived route merged Cursor composers: distinct=%d sessions=%v", route.Msg.GetDistinctSessions(), routeSessions)
	}
	result, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "amber-river-845", Limit: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 2 || result.Matches[0].SessionID != firstComposerSession || result.Matches[1].SessionID != firstComposerSession {
		t.Fatalf("Cursor archive search = %+v", result.Matches)
	}
	messageIDs := map[string]struct{}{}
	for _, match := range result.Matches {
		event, ok := conversations.GetEvent(ctx, match.SessionID, match.EventID)
		if !ok || event.NativeProvenance == nil {
			t.Fatalf("Cursor result has no native provenance: %+v", match)
		}
		messageIDs[event.NativeProvenance.MessageID] = struct{}{}
	}
	if len(messageIDs) != 2 {
		t.Fatalf("identical Cursor messages share native message identity: %+v", messageIDs)
	}
	result, err = conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "blue-harbor-216", Limit: 5}})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].SessionID != secondComposerSession {
		t.Fatalf("incremental Cursor archive search = %+v, err=%v", result.Matches, err)
	}
	priorInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE cursorDiskKV SET value='{"text":"Cursor composer rewritten marker amber-river-846."}' WHERE key IN ('bubbleId:composer-native-42:a1','bubbleId:composer-native-42:a2')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if info.Size() != priorInfo.Size() {
		t.Fatalf("Cursor rewrite changed database size: before=%d after=%d", priorInfo.Size(), info.Size())
	}
	restarted, err := indexer.NewLazyDiscoveredEngine(roots)
	if err != nil {
		t.Fatal(err)
	}
	restarted.Scan(ctx)
	restarted.Stop()
	if got := conversations.CountSessionEvents(ctx, firstComposerSession); got != 3 {
		t.Fatalf("rewritten Cursor conversation event count = %d, want 3", got)
	}
	stale, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "amber-river-845", Limit: 5}})
	if err != nil || len(stale.Matches) != 0 {
		t.Fatalf("rewritten Cursor conversation retained stale search result: matches=%+v err=%v", stale.Matches, err)
	}
	current, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "amber-river-846", Limit: 5}})
	if err != nil || len(current.Matches) != 2 || current.Matches[0].SessionID != firstComposerSession || current.Matches[1].SessionID != firstComposerSession {
		t.Fatalf("rewritten Cursor conversation search = %+v, err=%v", current.Matches, err)
	}
	db, err = sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM cursorDiskKV WHERE key LIKE 'composerData:composer-native-43' OR key LIKE 'bubbleId:composer-native-43:%'`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	removed, err := indexer.NewLazyDiscoveredEngine(roots)
	if err != nil {
		t.Fatal(err)
	}
	removed.Scan(ctx)
	removed.Stop()
	if got := conversations.CountSessionEvents(ctx, secondComposerSession); got != 0 {
		t.Fatalf("removed Cursor composer retained %d events", got)
	}
	stale, err = conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "blue-harbor-216", Limit: 5}})
	if err != nil || len(stale.Matches) != 0 {
		t.Fatalf("removed Cursor composer retained stale search result: matches=%+v err=%v", stale.Matches, err)
	}
}
