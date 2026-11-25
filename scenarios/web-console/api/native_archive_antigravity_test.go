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

func TestAntigravityCLIArchiveBackfillTailAndSearch(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	const sessionID = "3c09df9e-61ac-46bb-b1a7-bfb5ddbd7c11"
	const clue = "antigravity-shared-codec-search"
	const appendedClue = "antigravity-archive-tail-518"
	roots := NativeArchiveRootsFromHome(t.TempDir())
	path := filepath.Join(roots.FirstPath("antigravity"), sessionID, ".system_generated", "logs", "transcript_full.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve shared Antigravity fixture source path")
	}
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "../../../packages/cli-core/agenttranscript/testdata/antigravity-shared-consumer.jsonl")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read shared Antigravity codec fixture %q: %v", fixturePath, err)
	}
	initial := string(fixture) + "\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	duplicate := filepath.Join(filepath.Dir(path), "transcript.jsonl")
	if err := os.WriteFile(duplicate, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Harness != "antigravity" || files[0].NativeSessionID != sessionID {
		t.Fatalf("discovered Antigravity archives = %+v", files)
	}
	engine.Scan(ctx)
	engine.Stop()
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 4 {
		t.Fatalf("Antigravity backfill event count = %d, want user, assistant, generic assistant and tool result", count)
	}
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	handler := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	search := func(query string) []*conversationv1.ArchivedSearchMatch {
		t.Helper()
		response, err := handler.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: query, AgentType: "antigravity", Limit: 10}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetMatches()
	}
	if matches := search(clue); len(matches) != 3 {
		t.Fatalf("Antigravity archive search returned %d hits, want three original fixture messages", len(matches))
	}
	if matches := search("antigravity-generic-record-482"); len(matches) != 1 || matches[0].GetOccurredAt() != "2026-11-24T10:00:03Z" {
		t.Fatalf("Antigravity generic archive search matches = %+v", matches)
	}
	appendRecord := `{"step_index":6,"type":"PLANNER_RESPONSE","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:04Z","content":"` + appendedClue + `"}` + "\n"
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
	if count := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); count != 5 {
		t.Fatalf("Antigravity append event count = %d, want five", count)
	}
	if matches := search(appendedClue); len(matches) != 1 || matches[0].GetOccurredAt() != "2026-11-24T10:00:04Z" {
		t.Fatalf("Antigravity appended search matches = %+v", matches)
	}
}
