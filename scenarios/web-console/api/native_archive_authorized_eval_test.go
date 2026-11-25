package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

// This opt-in integration test reads the current user's native archive roots.
// Do not enable until WC4-OPERATOR-CORPUS-ACCESS-001 is explicitly answered.
const authorizedCorpusOptIn = "I_HAVE_OPERATOR_AUTHORIZATION"

type wc4LabeledCorpus struct {
	Version int                     `json:"version"`
	Queries []wc4LabeledCorpusQuery `json:"queries"`
}

type wc4LabeledCorpusQuery struct {
	Query            string   `json:"query"`
	Harness          string   `json:"harness,omitempty"`
	Role             string   `json:"role,omitempty"`
	RelevantEventIDs []string `json:"relevant_event_ids"`
}

func TestWC4AuthorizedOperatorCorpusEvaluation(t *testing.T) {
	if os.Getenv("WC4_OPERATOR_CORPUS_EVAL") != authorizedCorpusOptIn {
		t.Skip("requires explicit authorization recorded for WC4-OPERATOR-CORPUS-ACCESS-001")
	}
	manifestPath := strings.TrimSpace(os.Getenv("WC4_OPERATOR_LABEL_MANIFEST"))
	if manifestPath == "" {
		t.Fatal("WC4_OPERATOR_LABEL_MANIFEST must point to an operator-maintained labeled query manifest")
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal("could not load labeled query manifest")
	}
	var corpus wc4LabeledCorpus
	if err := json.Unmarshal(manifestBytes, &corpus); err != nil {
		t.Fatal("labeled query manifest is invalid JSON")
	}
	if corpus.Version != 1 || len(corpus.Queries) == 0 {
		t.Fatal("labeled query manifest requires version 1 and at least one query")
	}
	for index, item := range corpus.Queries {
		if strings.TrimSpace(item.Query) == "" || len(item.RelevantEventIDs) == 0 {
			t.Fatalf("labeled query row %d is missing query text or relevant event IDs", index+1)
		}
		seen := make(map[string]struct{}, len(item.RelevantEventIDs))
		for _, id := range item.RelevantEventIDs {
			if strings.TrimSpace(id) == "" {
				t.Fatalf("labeled query row %d contains an empty relevant event ID", index+1)
			}
			if _, duplicate := seen[id]; duplicate {
				t.Fatalf("labeled query row %d repeats a relevant event ID", index+1)
			}
			seen[id] = struct{}{}
		}
	}

	ctx := context.Background()
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		t.Fatal("current user home is unavailable")
	}
	roots := NativeArchiveRootsFromHome(home)
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("prepare local search index: %v", err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatalf("discover authorized archive roots: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no supported native archives were discovered")
	}
	engine.Scan(ctx)
	engine.Stop()
	wc4AssertArchiveCheckpoints(t, ctx, checkpoints, files)
	initialCount := wc4ArchiveEventCount(t, conversations, files)
	if initialCount == 0 {
		t.Fatal("archive backfill produced no searchable events")
	}

	// A fresh engine against the same local store verifies replay idempotence
	// without changing or transferring source archives.
	replay, replayFiles, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatalf("rediscover archives for replay: %v", err)
	}
	replay.Scan(ctx)
	replay.Stop()
	wc4AssertArchiveCheckpoints(t, ctx, checkpoints, replayFiles)
	if replayCount := wc4ArchiveEventCount(t, conversations, replayFiles); replayCount != initialCount {
		t.Fatalf("unchanged-source replay event count changed from %d to %d", initialCount, replayCount)
	}

	// The local server has no Agent Manager integration configured. Only the
	// local Connect SearchArchived handler is measured below.
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	if server.integrationHubURL != "" || server.backendRegistry != nil || server.managedCodex != nil {
		t.Fatal("authorized corpus evaluation requires Agent Manager to remain stopped")
	}
	search := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})
	var latencies []time.Duration
	var relevantTotal, retrievedTotal int
	for queryIndex, item := range corpus.Queries {
		relevant := make(map[string]struct{}, len(item.RelevantEventIDs))
		for _, id := range item.RelevantEventIDs {
			relevant[id] = struct{}{}
		}
		request := &conversationv1.SearchArchivedRequest{
			Query: item.Query, AgentType: item.Harness, Role: item.Role, Limit: 500,
			Mode: conversationv1.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT,
		}
		for repeat := 0; repeat < 10; repeat++ {
			started := time.Now()
			response, err := search.SearchArchived(ctx, connect.NewRequest(request))
			latencies = append(latencies, time.Since(started))
			if err != nil || response == nil || response.Msg == nil {
				t.Fatalf("local search failed for labeled query %d", queryIndex+1)
			}
			if response.Msg.GetError() != "" || response.Msg.GetTruncated() {
				t.Fatalf("local search was incomplete for labeled query %d", queryIndex+1)
			}
			if repeat != 0 {
				continue
			}
			found := make(map[string]struct{}, len(response.Msg.GetMatches()))
			for _, match := range response.Msg.GetMatches() {
				found[match.GetEventId()] = struct{}{}
			}
			relevantTotal += len(relevant)
			for id := range relevant {
				if _, ok := found[id]; ok {
					retrievedTotal++
				}
			}
		}
	}
	p50, p95 := durationPercentiles(latencies)
	if relevantTotal == 0 {
		t.Fatal("labeled corpus contains no relevant judgments")
	}
	if p95 > time.Second {
		t.Fatalf("local archive search p95 %s exceeds 1s target", p95)
	}
	t.Logf("WC4_AUTHORIZED_METRICS files=%d backfilled_events=%d labeled_queries=%d recall=%.3f search_p50_ms=%.3f search_p95_ms=%.3f agent_manager=stopped",
		len(files), initialCount, len(corpus.Queries), float64(retrievedTotal)/float64(relevantTotal), float64(p50.Microseconds())/1000, float64(p95.Microseconds())/1000)
}

func wc4ArchiveEventCount(t *testing.T, conversations *ConversationStore, files []NativeArchiveFile) int64 {
	t.Helper()
	seen := make(map[string]struct{})
	var count int64
	for _, file := range files {
		if _, ok := seen[file.WebConsoleSessionID]; ok {
			continue
		}
		seen[file.WebConsoleSessionID] = struct{}{}
		count += conversations.CountSessionEvents(context.Background(), file.WebConsoleSessionID)
	}
	return count
}

func wc4AssertArchiveCheckpoints(t *testing.T, ctx context.Context, checkpoints AgentTranscriptCheckpointStore, files []NativeArchiveFile) {
	t.Helper()
	for index, file := range files {
		checkpoint, ok, err := checkpoints.Get(ctx, "native-archive-index", file.SourceKey)
		if err != nil || !ok || checkpoint.SessionID != file.WebConsoleSessionID {
			t.Fatalf("discovered archive %d lacks its production ingestion checkpoint", index+1)
		}
	}
}
