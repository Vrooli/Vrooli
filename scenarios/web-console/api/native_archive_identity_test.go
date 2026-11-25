package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

func TestNativeArchiveKeepsIdenticalRecordsAtDistinctOffsets(t *testing.T) {
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
	const marker = "repeated native line marker"
	for _, harness := range []string{"claude", "codex", "grok"} {
		nativeID := "native-" + harness + "-repeat"
		if harness == "grok" {
			nativeID = "native-grok"
		}
		path := nativeArchiveFixturePath(roots, harness, nativeID)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if harness == "codex" {
			if _, err := fmt.Fprintf(file, "{\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"session_id\":%q}}\n", nativeID, nativeID); err != nil {
				file.Close()
				t.Fatal(err)
			}
		}
		record := append(nativeArchiveCorpusLine(harness, "assistant", marker, false), '\n')
		if _, err := file.Write(record); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if _, err := file.Write(record); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for sequence := 1; sequence <= 2; sequence++ {
		writeOpenCodeArchiveRecord(t, roots, "native-opencode-repeat", sequence, nativeArchiveCorpusLine("opencode", "assistant", marker, false))
	}
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	search := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	searchRepeated := func() *conversationv1.SearchArchivedResponse {
		t.Helper()
		response, err := search.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: marker, Limit: 20}))
		if err != nil {
			t.Fatalf("production SearchArchived(%q): %v", marker, err)
		}
		return response.Msg
	}
	assertRepeatedRecords := func(stage string) {
		t.Helper()
		response := searchRepeated()
		if len(response.GetMatches()) != 8 || response.GetDistinctSessions() != 4 {
			t.Fatalf("%s: identical records yielded %d matches across %d sessions, want 8 across 4", stage, len(response.GetMatches()), response.GetDistinctSessions())
		}
		ids := map[string]struct{}{}
		turns := map[string]struct{}{}
		for _, match := range response.GetMatches() {
			if _, duplicate := ids[match.GetEventId()]; duplicate {
				t.Errorf("%s: duplicate event id %q", stage, match.GetEventId())
			}
			ids[match.GetEventId()] = struct{}{}
			event, ok := conversations.GetEvent(ctx, match.GetSessionId(), match.GetEventId())
			if !ok || event.NativeProvenance == nil {
				t.Fatalf("%s: match has no persisted native provenance: %+v", stage, match)
			}
			key := event.SessionID + "\x00" + event.NativeProvenance.TurnID + "\x00" + event.NativeProvenance.MessageID
			if _, duplicate := turns[key]; duplicate {
				t.Errorf("%s: identical records share native turn/message provenance for %s", stage, event.SessionID)
			}
			turns[key] = struct{}{}
		}
	}
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 5 {
		t.Fatalf("discovered archive files = %d, want 5", len(files))
	}
	checkpoints.failNextSave()
	engine.Scan(ctx)
	// The first projected line's checkpoint is rejected. A second scan retries
	// that same byte range, then reads the identical second record at its own
	// source offset.
	engine.Scan(ctx)
	engine.Stop()
	assertRepeatedRecords("initial backfill")

	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertRepeatedRecords("process restart/replay")
}
