package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/agenttranscript"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

func TestGeminiCLIArchiveTailsIntoOwnStoreSearchAndResumeMetadata(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := &archiveFailOnceCheckpointStore{inner: NewSQLAgentTranscriptCheckpointStore(db)}
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	root := t.TempDir()
	roots := NativeArchiveRootsFromHome(root)
	path := filepath.Join(roots.FirstPath("gemini-cli"), "project-hash", "chats", "session-fixture.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	document := []byte(`{"sessionId":"gemini-native-session","messages":[{"id":"u1","type":"user","content":"Which route has the local archive evidence?"},{"id":"a1","type":"gemini","timestamp":"2026-10-01T12:00:00Z","content":"The native Gemini archive has searchable evidence marker violet-bridge-731."},{"id":"a2","type":"gemini","timestamp":"2026-10-02T12:00:00Z","content":"The native Gemini archive has searchable evidence marker violet-bridge-731."},{"id":"i1","type":"info","content":"violet-bridge-731"}]}`)
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Harness != "gemini-cli" || files[0].NativeSessionID != "gemini-native-session" {
		t.Fatalf("discovered Gemini archive = %+v", files)
	}
	engine.Scan(ctx)
	engine.Stop()
	if got := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); got != 3 {
		t.Fatalf("indexed event count = %d, want user and both assistant messages only", got)
	}
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	handler := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	response, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "violet-bridge-731", Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetMatches()) != 2 || response.Msg.GetMatches()[0].GetSessionId() != files[0].WebConsoleSessionID || response.Msg.GetMatches()[1].GetSessionId() != files[0].WebConsoleSessionID {
		t.Fatalf("Gemini archive search results = %+v", response.Msg.GetMatches())
	}
	gotTimes := map[string]bool{}
	for _, match := range response.Msg.GetMatches() {
		gotTimes[match.GetOccurredAt()] = true
	}
	if !gotTimes["2026-10-01T12:00:00Z"] || !gotTimes["2026-10-02T12:00:00Z"] {
		t.Fatalf("Gemini source timestamps = %+v", gotTimes)
	}
	dateFiltered, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "violet-bridge-731", Limit: 5, CreatedAfter: "2026-11-01T00:00:00Z"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(dateFiltered.Msg.GetMatches()) != 0 {
		t.Fatalf("newly indexed old Gemini messages passed source-time filter: %+v", dateFiltered.Msg.GetMatches())
	}
	provenance := map[string]struct{}{}
	for _, match := range response.Msg.GetMatches() {
		event, ok := conversations.GetEvent(ctx, match.GetSessionId(), match.GetEventId())
		if !ok || event.NativeProvenance == nil {
			t.Fatalf("Gemini result has no native provenance: %+v", match)
		}
		provenance[event.NativeProvenance.MessageID] = struct{}{}
	}
	if len(provenance) != 2 {
		t.Fatalf("identical Gemini messages share native identity: %+v", provenance)
	}
	initialGeneration := files[0].Generation
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.ReplaceAll(document, []byte("violet-bridge-731"), []byte("violet-bridge-732"))
	if len(updated) != len(document) {
		t.Fatal("same-length Gemini rewrite fixture changed size")
	}
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	engine, rewritten, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(rewritten) != 1 || rewritten[0].Generation == initialGeneration {
		t.Fatalf("same-length Gemini rewrite generation = %+v, initial %q", rewritten, initialGeneration)
	}
	if got := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); got != 0 {
		t.Fatalf("same-length Gemini rewrite left stale projection before backfill: %d", got)
	}
	checkpoints.failNextSave()
	engine.Scan(ctx)
	engine.Stop()
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	if got := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); got != 3 {
		t.Fatalf("Gemini rewrite replay event count = %d, want 3 without duplicates", got)
	}
	response, err = handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "violet-bridge-731", Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetMatches()) != 0 {
		t.Fatalf("Gemini rewrite retained stale search results: %+v", response.Msg.GetMatches())
	}
	response, err = handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "violet-bridge-732", Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetMatches()) != 2 {
		t.Fatalf("Gemini rewritten search results = %+v", response.Msg.GetMatches())
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	if got := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); got != 0 {
		t.Fatalf("removed Gemini archive retained %d events", got)
	}
	response, err = handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "violet-bridge-732", Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetMatches()) != 0 {
		t.Fatalf("removed Gemini archive retained stale search results: %+v", response.Msg.GetMatches())
	}
	format, ok := agenttranscript.FormatFor("gemini-cli")
	if !ok || format.ResumeFlag != "--resume" || format.ResumeSelection != "session-id-or-index-or-latest" {
		t.Fatalf("shared Gemini resume format = %+v, found=%v", format, ok)
	}
}

func TestGeminiCLIArchiveAppendReplacesDocumentProjectionWithoutDuplicates(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	roots := NativeArchiveRootsFromHome(t.TempDir())
	path := filepath.Join(roots.FirstPath("gemini-cli"), "project-hash", "chats", "session-append-fixture.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	initial := []byte(`{"sessionId":"gemini-append-session","messages":[{"type":"user","content":"first message violet-fern-1"},{"type":"gemini","content":"first reply amber-lake-2"}]}`)
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	if len(files) != 1 {
		t.Fatalf("Gemini archive files = %d, want 1", len(files))
	}
	sessionID := files[0].WebConsoleSessionID
	initialGeneration := files[0].Generation
	if got := conversations.CountSessionEvents(ctx, sessionID); got != 2 {
		t.Fatalf("initial Gemini projection has %d events, want 2", got)
	}
	initialMatch, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "violet-fern-1", Limit: 10}})
	if err != nil || len(initialMatch.Matches) != 1 {
		t.Fatalf("initial Gemini message search = %+v, err=%v", initialMatch.Matches, err)
	}
	initialEvent, ok := conversations.GetEvent(ctx, sessionID, initialMatch.Matches[0].EventID)
	if !ok || initialEvent.NativeProvenance == nil {
		t.Fatalf("initial Gemini event has no native provenance: %+v", initialEvent)
	}
	initialProvenance := *initialEvent.NativeProvenance
	appended := bytes.Replace(initial, []byte("]}"), []byte(`,{"type":"gemini","content":"appended reply copper-pine-3"}]}`), 1)
	if len(appended) <= len(initial) || bytes.Equal(appended, initial) {
		t.Fatal("Gemini append fixture did not grow the document")
	}
	if err := os.WriteFile(path, appended, 0o600); err != nil {
		t.Fatal(err)
	}
	engine, appendedFiles, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(appendedFiles) != 1 || appendedFiles[0].Generation == initialGeneration {
		t.Fatalf("Gemini message append generation = %+v, initial %q", appendedFiles, initialGeneration)
	}
	if got := conversations.CountSessionEvents(ctx, sessionID); got != 0 {
		t.Fatalf("Gemini message append left %d stale projected events before backfill", got)
	}
	engine.Scan(ctx)
	engine.Stop()
	if got := conversations.CountSessionEvents(ctx, sessionID); got != 3 {
		t.Fatalf("Gemini append projection has %d events, want 3 without duplicates", got)
	}
	appendedMatch, err := conversations.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "violet-fern-1", Limit: 10}})
	if err != nil || len(appendedMatch.Matches) != 1 {
		t.Fatalf("appended Gemini message search = %+v, err=%v", appendedMatch.Matches, err)
	}
	appendedEvent, ok := conversations.GetEvent(ctx, sessionID, appendedMatch.Matches[0].EventID)
	if !ok || appendedEvent.NativeProvenance == nil || *appendedEvent.NativeProvenance != initialProvenance {
		t.Fatalf("unchanged Gemini message provenance changed after append: before=%+v after=%+v", initialProvenance, appendedEvent.NativeProvenance)
	}
	handler := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(&Server{conversations: conversations, agentCheckpointStore: checkpoints}), Logger: log.New(io.Discard, "", 0)})
	for query, want := range map[string]int{"violet-fern-1": 1, "amber-lake-2": 1, "copper-pine-3": 1} {
		response, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: query, Limit: 10}))
		if err != nil {
			t.Fatalf("search appended Gemini document for %q: %v", query, err)
		}
		if got := len(response.Msg.GetMatches()); got != want {
			t.Fatalf("Gemini query %q returned %d matches, want %d", query, got, want)
		}
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	if got := conversations.CountSessionEvents(ctx, sessionID); got != 3 {
		t.Fatalf("Gemini append replay has %d events, want 3 without loss/duplicates", got)
	}
}
