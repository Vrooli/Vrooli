package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

func TestQwenCodeArchiveBackfillTailReplayAndSourceTimeSearch(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	roots := NativeArchiveRootsFromHome(t.TempDir())
	path := filepath.Join(roots.FirstPath("qwen-code"), "project-fixture", "chats", "native-qwen-session.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	initial := strings.Join([]string{
		`{"sessionId":"native-qwen-session","timestamp":"2026-10-01T12:00:00Z","type":"user","message":{"role":"user","parts":[{"text":"Question about qwen source-time marker amber-lantern-991."}]}}`,
		`{"sessionId":"native-qwen-session","timestamp":"2026-10-02T12:00:00.123Z","type":"assistant","message":{"role":"model","parts":[{"text":"Thoughts must stay hidden.","thought":true},{"text":"Qwen assistant source-time marker amber-lantern-991."}]}}`,
		`{"sessionId":"native-qwen-session","timestamp":"2026-10-03T12:00:00Z","type":"system","message":{"role":"system","parts":[{"text":"system marker amber-lantern-991"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Harness != "qwen-code" || files[0].NativeSessionID != "native-qwen-session" {
		t.Fatalf("discovered Qwen archive = %+v", files)
	}
	engine.Scan(ctx)
	engine.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 2 {
		t.Fatalf("backfilled event count = %d, want user and assistant only", count)
	}

	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	handler := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	response, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{
		Query: "amber-lantern-991", AgentType: "qwen-code", Limit: 10,
	}))
	if err != nil {
		t.Fatal(err)
	}
	matches := response.Msg.GetMatches()
	if len(matches) != 2 {
		t.Fatalf("Qwen archive search returned %d hits, want two user/assistant hits", len(matches))
	}
	times := make(map[string]bool)
	initialEventIDs := make(map[string]bool)
	for _, match := range matches {
		event, ok := conversations.GetEvent(ctx, match.GetSessionId(), match.GetEventId())
		if !ok || event.NativeProvenance == nil || event.NativeProvenance.Provider != "qwen-code" || event.NativeProvenance.SessionID != "native-qwen-session" {
			t.Fatalf("Qwen result lost native provenance: %+v", match)
		}
		if event.OccurredAt.IsZero() || match.GetOccurredAt() == "" {
			t.Fatalf("Qwen source time was dropped from search response: %+v", match)
		}
		times[match.GetOccurredAt()] = true
		initialEventIDs[match.GetEventId()] = true
	}
	if !times["2026-10-01T12:00:00Z"] || !times["2026-10-02T12:00:00.123Z"] {
		t.Fatalf("Qwen source timestamps = %+v", times)
	}
	thoughts, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "Thoughts must stay hidden", AgentType: "qwen-code", Limit: 10}))
	if err != nil || len(thoughts.Msg.GetMatches()) != 0 {
		t.Fatalf("Qwen thought content unexpectedly searchable: %+v err=%v", thoughts, err)
	}
	wrongHarness, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{
		Query: "amber-lantern-991", AgentType: "codex", Limit: 10,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(wrongHarness.Msg.GetMatches()) != 0 {
		t.Fatalf("Qwen results leaked through Codex filter: %+v", wrongHarness.Msg.GetMatches())
	}

	filtered, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{
		Query: "amber-lantern-991", AgentType: "qwen-code", CreatedAfter: "2026-11-01T00:00:00Z", Limit: 10,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Msg.GetMatches()) != 0 {
		t.Fatalf("old Qwen source times passed age filter: %+v", filtered.Msg.GetMatches())
	}

	appendRecord := `{"sessionId":"native-qwen-session","timestamp":"2026-10-04T12:00:00Z","type":"assistant","message":{"role":"model","parts":[{"text":"Incremental qwen tail marker cedar-moon-224."}]}}` + "\n"
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(appendRecord); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 3 {
		t.Fatalf("appended Qwen event count = %d, want 3", count)
	}
	originalAfterAppend, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "amber-lantern-991", AgentType: "qwen-code", Limit: 10}))
	if err != nil || len(originalAfterAppend.Msg.GetMatches()) != 2 {
		t.Fatalf("Qwen original search after append = %+v err=%v", originalAfterAppend, err)
	}
	for _, match := range originalAfterAppend.Msg.GetMatches() {
		if !initialEventIDs[match.GetEventId()] {
			t.Fatalf("Qwen unchanged message received a new event identity: %+v", match)
		}
	}
	replay, _, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	replay.Scan(ctx)
	replay.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 3 {
		t.Fatalf("Qwen restart replay event count = %d, want 3 without duplicates", count)
	}
	appended, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "cedar-moon-224", AgentType: "qwen-code", Limit: 10}))
	if err != nil || len(appended.Msg.GetMatches()) != 1 || appended.Msg.GetMatches()[0].GetOccurredAt() != "2026-10-04T12:00:00Z" {
		t.Fatalf("Qwen append search result = %+v err=%v", appended, err)
	}

	// Qwen chats are JSONL, so a full-file rewrite that changes length but
	// preserves the path must invalidate the old projection before reindexing.
	replacement := strings.Join([]string{
		`{"sessionId":"native-qwen-session","timestamp":"2026-10-06T12:00:00Z","type":"user","message":{"role":"user","parts":[{"text":"Rewritten Qwen record marker violet-branch-441."}]}}`,
		`{"sessionId":"native-qwen-session","timestamp":"2026-10-07T12:00:00Z","type":"assistant","message":{"role":"model","parts":[{"text":"Rewritten Qwen answer marker silver-river-552."}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(replacement), 0o600); err != nil {
		t.Fatal(err)
	}
	rewrite, _, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	rewrite.Scan(ctx)
	rewrite.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 2 {
		t.Fatalf("same-path Qwen rewrite event count = %d, want two replacement events", count)
	}
	old, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "amber-lantern-991", AgentType: "qwen-code", Limit: 10}))
	if err != nil || len(old.Msg.GetMatches()) != 0 {
		t.Fatalf("stale Qwen records survived rewrite: %+v err=%v", old, err)
	}
	newHits, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "Rewritten Qwen", AgentType: "qwen-code", Limit: 10}))
	if err != nil || len(newHits.Msg.GetMatches()) != 2 {
		t.Fatalf("rewritten Qwen records are not searchable: %+v err=%v", newHits, err)
	}
	if err := os.RemoveAll(roots.FirstPath("qwen-code")); err != nil {
		t.Fatal(err)
	}
	removed, _, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	removed.Scan(ctx)
	removed.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 0 {
		t.Fatalf("Qwen root removal left %d events, want zero", count)
	}
}
