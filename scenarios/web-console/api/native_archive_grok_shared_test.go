package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

func TestGrokSharedACPFixtureFlowsThroughWebConsoleTailAndSearch(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("ensure local FTS: %v", err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	roots := NativeArchiveRootsFromHome(t.TempDir())
	const sessionID = "grok-shared-synthetic-session"
	const clue = "groksharedsearch"
	const appendedClue = "groksharedtail"
	path := filepath.Join(roots.FirstPath("grok"), "synthetic-cwd", sessionID, "updates.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve shared Grok fixture path")
	}
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "../../../packages/cli-core/agenttranscript/testdata/grok-shared-consumer.jsonl")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read shared Grok ACP fixture %q: %v", fixturePath, err)
	}
	if err := os.WriteFile(path, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Harness != "grok" || files[0].NativeSessionID != sessionID {
		t.Fatalf("discovered Grok archive = %+v", files)
	}
	engine.Scan(ctx)
	engine.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 2 {
		t.Fatalf("Grok backfill event count = %d, want user and assistant", count)
	}
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	handler := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	search := func(query string) []*conversationv1.ArchivedSearchMatch {
		t.Helper()
		response, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: query, AgentType: "grok", Limit: 10}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetMatches()
	}
	if matches := search(clue); len(matches) != 1 {
		t.Fatalf("Grok shared fixture search returned %d matches, want one", len(matches))
	}
	appendRecord := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + sessionID + `","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"` + appendedClue + `"}}}}` + "\n"
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(appendRecord); err != nil {
		_ = file.Close()
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
		t.Fatalf("Grok tail event count = %d, want three", count)
	}
	if matches := search(appendedClue); len(matches) != 1 {
		t.Fatalf("Grok appended tail search returned %d matches, want one", len(matches))
	}
}
