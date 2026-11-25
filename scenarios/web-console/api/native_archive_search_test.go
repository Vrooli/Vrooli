package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

func TestNativeArchiveSearchCorpusBackfillTailRestartAndLocalRecall(t *testing.T) {
	const generatedBackgroundEvents = 2500
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("ensure local FTS: %v", err)
	}
	repo := NewSQLConversationRepository(db)
	conversations := NewConversationStoreWithRepository(repo)
	checkpoints := &archiveFailOnceCheckpointStore{inner: NewSQLAgentTranscriptCheckpointStore(db)}
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	root := t.TempDir()
	roots := NativeArchiveRootsFromHome(root)
	harnesses := []string{"claude", "codex", "grok", "opencode"}
	labels := []string{
		"opportunity portfolio outcome",
		"retention milestone cadence",
		"risk followup mitigation",
		"partner signal pipeline",
		"target receipt ledger",
	}
	for _, harness := range harnesses {
		nativeID := "native-" + harness + "-session-1"
		if harness == "grok" {
			nativeID = "native-grok"
		}
		if harness == "opencode" {
			for n := 0; n < generatedBackgroundEvents; n++ {
				role := "assistant"
				if n%2 == 0 {
					role = "user"
				}
				line := nativeArchiveCorpusLine(harness, role, fmt.Sprintf("synthetic project update number %04d", n), false)
				writeOpenCodeArchiveRecord(t, roots, nativeID, n+1, line)
			}
			for n, label := range labels {
				line := nativeArchiveCorpusLine(harness, "assistant", "operator-intent label: "+label, n == len(labels)-1)
				writeOpenCodeArchiveRecord(t, roots, nativeID, generatedBackgroundEvents+1+n, line)
			}
			progress, err := json.Marshal(map[string]any{"type": "progress", "text": "operator-intent label: " + labels[0]})
			if err != nil {
				t.Fatal(err)
			}
			writeOpenCodeArchiveRecord(t, roots, nativeID, generatedBackgroundEvents+len(labels)+1, progress)
			continue
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
			if _, err := file.WriteString(`{"type":"session_meta","payload":{"id":"` + nativeID + `","session_id":"` + nativeID + `"}}` + "\n"); err != nil {
				file.Close()
				t.Fatal(err)
			}
		}
		for n := 0; n < generatedBackgroundEvents; n++ {
			role := "assistant"
			if n%2 == 0 {
				role = "user"
			}
			line := nativeArchiveCorpusLine(harness, role, fmt.Sprintf("synthetic project update number %04d", n), false)
			if _, err := file.Write(append(line, '\n')); err != nil {
				file.Close()
				t.Fatal(err)
			}
		}
		for n, label := range labels {
			line := nativeArchiveCorpusLine(harness, "assistant", "operator-intent label: "+label, n == len(labels)-1)
			if _, err := file.Write(append(line, '\n')); err != nil {
				file.Close()
				t.Fatal(err)
			}
		}
		// Native lifecycle/progress records can repeat prompt-like strings. They
		// must not become searchable conversation events or inflate labeled recall.
		progress, err := json.Marshal(map[string]any{"type": "progress", "text": "operator-intent label: " + labels[0]})
		if err != nil {
			file.Close()
			t.Fatal(err)
		}
		if _, err := file.Write(append(progress, '\n')); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	files, err := DiscoverNativeArchiveFiles(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	wantFiles := len(harnesses) - 1 + generatedBackgroundEvents + len(labels) + 1
	if len(files) != wantFiles {
		t.Fatalf("discovered archive files = %d, want %d", len(files), wantFiles)
	}

	// This is the production Connect handler and local store/search adapter.
	// No Agent Manager client or external service is present in this server.
	server := &Server{conversations: conversations, agentCheckpointStore: checkpoints}
	if server.integrationHubURL != "" || server.backendRegistry != nil || server.managedCodex != nil {
		t.Fatal("fixture unexpectedly wired an optional integration or live backend")
	}
	search := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(server), Logger: log.New(io.Discard, "", 0)})

	started := time.Now()
	engine, discovered, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovered) != len(files) {
		t.Fatalf("indexer admitted %d discovered files, want %d", len(discovered), len(files))
	}
	files = discovered
	engine.Scan(ctx)
	engine.Stop()
	ingestDuration := time.Since(started)
	initialCount := int64(0)
	archiveSessions := uniqueArchiveSessions(files)
	for _, file := range archiveSessions {
		initialCount += conversations.CountSessionEvents(ctx, file.WebConsoleSessionID)
	}
	wantInitial := int64(len(harnesses) * (generatedBackgroundEvents + len(labels)))
	if initialCount != wantInitial {
		t.Fatalf("initial backfill indexed %d events, want %d", initialCount, wantInitial)
	}

	expected := make(map[string]map[string]struct{}, len(labels))
	for _, label := range labels {
		expected[label] = make(map[string]struct{}, len(harnesses))
		for _, file := range archiveSessions {
			for _, event := range conversations.ListSession(ctx, file.WebConsoleSessionID).Events {
				if strings.Contains(event.Text, label) {
					if event.NativeProvenance == nil || event.NativeProvenance.Provider != file.Harness || event.NativeProvenance.SessionID != file.NativeSessionID {
						t.Fatalf("%s native provenance = %+v", file.Harness, event.NativeProvenance)
					}
					expected[label][event.ID] = struct{}{}
				}
			}
		}
		if len(expected[label]) != len(harnesses) {
			t.Fatalf("label %q has %d indexed targets, want %d", label, len(expected[label]), len(harnesses))
		}
	}

	var latencies []time.Duration
	var relevant, retrieved int
	for _, label := range labels {
		for repeat := 0; repeat < 16; repeat++ {
			queryStarted := time.Now()
			response, err := search.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: label, Limit: 20}))
			latencies = append(latencies, time.Since(queryStarted))
			if err != nil {
				t.Fatalf("local SearchArchived query %q: %v", label, err)
			}
			matches := response.Msg.GetMatches()
			if repeat == 0 {
				seen := make(map[string]struct{}, len(matches))
				for _, match := range matches {
					seen[match.GetEventId()] = struct{}{}
					if match.GetSessionId() == "" {
						t.Errorf("search result lost session provenance: %+v", match)
					}
				}
				relevant += len(expected[label])
				for eventID := range expected[label] {
					if _, ok := seen[eventID]; ok {
						retrieved++
					} else {
						t.Errorf("label %q missed expected event %s", label, eventID)
					}
				}
				if len(matches) != len(harnesses) {
					t.Errorf("label %q returned %d hits, want one per harness", label, len(matches))
				}
			}
		}
	}
	if relevant == 0 || retrieved != relevant {
		t.Fatalf("labeled recall = %d/%d, want 1.0", retrieved, relevant)
	}
	median, p95 := durationPercentiles(latencies)
	if p95 > time.Second {
		t.Fatalf("local SearchArchived p95 = %s, exceeds 1s target", p95)
	}
	t.Logf("WC4_METRICS mode=local-sqlite-fts agent_manager=stopped files=%d harnesses=%d backfilled_events=%d labels=%d queries=%d recall=%.3f ingest_ms=%d search_p50_ms=%.3f search_p95_ms=%.3f",
		len(files), len(harnesses), initialCount, len(labels), len(latencies), float64(retrieved)/float64(relevant), ingestDuration.Milliseconds(), float64(median.Microseconds())/1000, float64(p95.Microseconds())/1000)

	// Append one native record to every archive, rebuild the tail engine from
	// persisted SQLite cursors, and verify incremental visibility after restart.
	const incrementalQuery = "incremental archive cursor"
	for _, file := range archiveSessions {
		line := nativeArchiveCorpusLine(file.Harness, "user", "incremental archive cursor seen", false)
		if file.Harness == "opencode" {
			writeOpenCodeArchiveRecord(t, roots, file.NativeSessionID, 9001, line)
		} else {
			appendArchiveLine(t, file.Path, line)
		}
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, incrementalQuery, len(harnesses))
	for _, file := range archiveSessions {
		want := int64(generatedBackgroundEvents + len(labels) + 1)
		if got := conversations.CountSessionEvents(ctx, file.WebConsoleSessionID); got != want {
			t.Fatalf("%s incremental count = %d, want %d", file.Harness, got, want)
		}
	}

	// An incomplete final JSONL record is held back until its terminating
	// newline arrives. A fresh catalog/indexer process must then consume it.
	partialLine := nativeArchiveCorpusLine("claude", "assistant", "partial archive boundary", false)
	claudePath := archiveForHarness(t, archiveSessions, "claude").Path
	partialFile, err := os.OpenFile(claudePath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := partialFile.Write(partialLine); err != nil {
		partialFile.Close()
		t.Fatal(err)
	}
	if err := partialFile.Close(); err != nil {
		t.Fatal(err)
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "partial archive boundary", 0)
	appendArchiveLine(t, claudePath, nil)
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	assertArchiveSearchCount(t, ctx, search, "partial archive boundary", 1)

	// Replace one discovered OpenCode message-file generation in the active
	// catalog. The removed file's reader is stopped and the replacement is
	// admitted without disturbing other files' persisted byte cursors.
	openCodeFile := writeOpenCodeArchiveRecord(t, roots, "native-opencode-session-1", 9002, nativeArchiveCorpusLine("opencode", "assistant", "catalog old generation", false))
	engine.Scan(ctx)
	assertArchiveSearchCount(t, ctx, search, "catalog old generation", 1)
	if err := os.Remove(openCodeFile); err != nil {
		t.Fatal(err)
	}
	writeOpenCodeArchiveRecord(t, roots, "native-opencode-session-1", 9003, nativeArchiveCorpusLine("opencode", "assistant", "catalog replacement generation", false))
	engine.Scan(ctx)
	engine.Stop()
	refreshed, err := DiscoverNativeArchiveFiles(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed) != len(files)+2 {
		t.Fatalf("refreshed catalog files=%d, want %d", len(refreshed), len(files)+2)
	}
	assertArchiveSearchCount(t, ctx, search, "catalog old generation", 0)
	assertArchiveSearchCount(t, ctx, search, "catalog replacement generation", 1)

	// A projected event followed by a failed checkpoint save is replayed after
	// process restart. Native provenance suppresses duplicate projection rows.
	const replayQuery = "checkpoint replay safety"
	appendArchiveLine(t, files[1].Path, nativeArchiveCorpusLine(files[1].Harness, "assistant", replayQuery, false))
	checkpoints.failNextSave()
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, replayQuery, 1)
	if got := conversations.CountSessionEvents(ctx, files[1].WebConsoleSessionID); got != int64(generatedBackgroundEvents+len(labels)+2) {
		t.Fatalf("checkpoint replay duplicated/lost events: count=%d", got)
	}

	// Repeating a full catalog backfill with cursors removed is safe because
	// each imported event carries the same canonical native identity.
	if _, err := db.ExecContext(ctx, `DELETE FROM agent_transcript_checkpoints WHERE source = 'native-archive-index'`); err != nil {
		t.Fatalf("clear synthetic checkpoints: %v", err)
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, incrementalQuery, len(harnesses))
	if got := conversations.CountSessionEvents(ctx, files[0].WebConsoleSessionID); got != int64(generatedBackgroundEvents+len(labels)+2) {
		t.Fatalf("full replay duplicated events: count=%d", got)
	}

}

func TestNativeJSONLArchiveCompletesPartialRecordAcrossTailAndRestart(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	roots := NativeArchiveRootsFromHome(t.TempDir())
	path := nativeArchiveFixturePath(roots, "claude", "native-claude-partial")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	complete := append(nativeArchiveCorpusLine("claude", "assistant", "complete prefix marker", false), '\n')
	incoming := append(nativeArchiveCorpusLine("claude", "assistant", "partial completion marker", false), '\n')
	if err := os.WriteFile(path, complete, 0o600); err != nil {
		t.Fatal(err)
	}
	appendRaw := func(content []byte) {
		t.Helper()
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	search := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(&Server{conversations: conversations, agentCheckpointStore: checkpoints}), Logger: log.New(io.Discard, "", 0)})
	engine, _, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	appendRaw(incoming[:len(incoming)-1]) // a line without its newline is not yet committed
	engine.Scan(ctx)
	assertArchiveSearchCount(t, ctx, search, "complete prefix marker", 1)
	assertArchiveSearchCount(t, ctx, search, "partial completion marker", 0)
	appendRaw(incoming[len(incoming)-1:])
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "partial completion marker", 1)
	if got := conversations.CountSessionEvents(ctx, "wc-archive-claude-native-claude-partial"); got != 2 {
		t.Fatalf("completed JSONL projection has %d events, want 2", got)
	}
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "complete prefix marker", 1)
	assertArchiveSearchCount(t, ctx, search, "partial completion marker", 1)
	if got := conversations.CountSessionEvents(ctx, "wc-archive-claude-native-claude-partial"); got != 2 {
		t.Fatalf("replayed JSONL projection has %d events, want 2 without loss/duplicates", got)
	}
}

func archiveForHarness(t *testing.T, files []NativeArchiveFile, harness string) NativeArchiveFile {
	t.Helper()
	for _, file := range files {
		if file.Harness == harness {
			return file
		}
	}
	t.Fatalf("archive catalog has no %s file", harness)
	return NativeArchiveFile{}
}

func uniqueArchiveSessions(files []NativeArchiveFile) []NativeArchiveFile {
	seen := make(map[string]struct{}, len(files))
	var out []NativeArchiveFile
	for _, file := range files {
		if _, exists := seen[file.WebConsoleSessionID]; exists {
			continue
		}
		seen[file.WebConsoleSessionID] = struct{}{}
		out = append(out, file)
	}
	return out
}

func writeOpenCodeArchiveRecord(t *testing.T, roots NativeArchiveRoots, nativeID string, sequence int, content []byte) string {
	t.Helper()
	dir := filepath.Dir(nativeArchiveFixturePath(roots, "opencode", nativeID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fmt.Sprintf("message-%04d.json", sequence))
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNativeArchiveCatalogSamePathReplacementAndRootRemoval(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	conversations := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := &archiveFailOnceCheckpointStore{inner: NewSQLAgentTranscriptCheckpointStore(db)}
	root := t.TempDir()
	roots := NativeArchiveRootsFromHome(root)
	for _, harness := range []string{"claude", "codex", "grok", "opencode"} {
		nativeID := "native-" + harness
		if harness == "grok" {
			nativeID = "native-grok"
		}
		path := nativeArchiveFixturePath(roots, harness, nativeID)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		content := nativeArchiveCorpusLine(harness, "assistant", "old marker", false)
		if harness != "grok" {
			var record map[string]any
			if err := json.Unmarshal(content, &record); err != nil {
				t.Fatal(err)
			}
			if harness == "opencode" {
				info, ok := record["info"].(map[string]any)
				if !ok {
					t.Fatalf("OpenCode fixture info has type %T", record["info"])
				}
				info["time"] = map[string]any{"created": "2026-10-01T12:00:00Z"}
			} else {
				record["timestamp"] = "2026-10-01T12:00:00Z"
			}
			encoded, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			content = encoded
		}
		if harness != "opencode" {
			content = append(content, '\n')
		}
		if harness == "codex" {
			content = append([]byte(`{"type":"session_meta","payload":{"id":"`+nativeID+`","session_id":"`+nativeID+`"}}`+"\n"), content...)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	indexer := NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, files, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	search := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(&Server{conversations: conversations, agentCheckpointStore: checkpoints}), Logger: log.New(io.Discard, "", 0)})
	assertArchiveSearchCount(t, ctx, search, "old marker", 4)
	initialResults, err := search.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "old marker", Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	var initialOpenCodeProvenance *NativeProvenance
	occurrenceTimes := make(map[string]string)
	for _, match := range initialResults.Msg.GetMatches() {
		event, ok := conversations.GetEvent(ctx, match.GetSessionId(), match.GetEventId())
		if !ok {
			t.Fatalf("search result event %q disappeared from the own store", match.GetEventId())
		}
		occurrenceTimes[event.Source] = match.GetOccurredAt()
		if match.GetSessionId() != "wc-archive-opencode-native-opencode" {
			continue
		}
		if ok && event.NativeProvenance != nil {
			copy := *event.NativeProvenance
			initialOpenCodeProvenance = &copy
		}
	}
	for _, source := range []string{"claude_archive", "codex_archive", "opencode_archive"} {
		if got := occurrenceTimes[source]; got != "2026-10-01T12:00:00Z" {
			t.Fatalf("%s occurrence time = %q", source, got)
		}
	}
	if got := occurrenceTimes["grok_archive"]; got != "" {
		t.Fatalf("unverified Grok source time must stay unknown, got %q", got)
	}
	if initialOpenCodeProvenance == nil {
		t.Fatal("initial OpenCode event has no native provenance")
	}
	claude := archiveForHarness(t, files, "claude")
	openCodePartial := filepath.Join(roots.FirstPath("opencode"), "native-opencode", "message-partial.json")
	completedOpenCode := nativeArchiveCorpusLine("opencode", "assistant", "completed eof message", false)
	if err := os.WriteFile(openCodePartial, completedOpenCode[:8], 0o600); err != nil {
		t.Fatal(err)
	}
	indexer = NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "completed eof message", 0)
	if err := os.WriteFile(openCodePartial, completedOpenCode, 0o600); err != nil {
		t.Fatal(err)
	}
	indexer = NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "completed eof message", 1)
	updatedResults, err := search.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "old marker", Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	var updatedOpenCodeProvenance *NativeProvenance
	for _, match := range updatedResults.Msg.GetMatches() {
		if match.GetSessionId() != "wc-archive-opencode-native-opencode" {
			continue
		}
		event, ok := conversations.GetEvent(ctx, match.GetSessionId(), match.GetEventId())
		if ok && event.NativeProvenance != nil {
			copy := *event.NativeProvenance
			updatedOpenCodeProvenance = &copy
		}
	}
	if updatedOpenCodeProvenance == nil || *updatedOpenCodeProvenance != *initialOpenCodeProvenance {
		t.Fatalf("adding an OpenCode message changed existing event provenance: before=%+v after=%+v", initialOpenCodeProvenance, updatedOpenCodeProvenance)
	}
	oldGenerations := make(map[string]string, len(files))
	for _, file := range files {
		oldGenerations[file.Harness] = file.Generation
		info, err := os.Stat(file.Path)
		if err != nil {
			t.Fatal(err)
		}
		before := readArchive(t, file.Path)
		updated := bytes.Replace(before, []byte("old marker"), []byte("new marker"), 1)
		if len(updated) != len(before) || bytes.Equal(updated, before) {
			t.Fatalf("%s fixture did not produce a same-size rewrite", file.Harness)
		}
		if err := os.WriteFile(file.Path, updated, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(file.Path, info.ModTime(), info.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	// A fresh indexer proves each source generation survives process restart.
	indexer = NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, replaced, err := indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range replaced {
		if file.Generation == oldGenerations[file.Harness] {
			t.Fatalf("%s same-path, same-size rewrite did not change generation", file.Harness)
		}
		if got := conversations.CountSessionEvents(ctx, file.WebConsoleSessionID); got != 0 {
			t.Fatalf("%s same-size rewrite left %d stale projected events before backfill", file.Harness, got)
		}
	}
	checkpoints.failNextSave()
	engine.Scan(ctx)
	engine.Scan(ctx) // retry the rejected source checkpoint before restart
	engine.Stop()
	indexer = NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "old marker", 0)
	assertArchiveSearchCount(t, ctx, search, "new marker", 4)
	for _, file := range replaced {
		want := int64(1)
		if file.Harness == "opencode" {
			want = 2 // the completed EOF message added above remains in this session
		}
		if got := conversations.CountSessionEvents(ctx, file.WebConsoleSessionID); got != want {
			t.Fatalf("rewritten %s projection has %d events, want %d", file.Harness, got, want)
		}
	}

	// A larger overwrite is not mistaken for append-only growth because its
	// prior content prefix no longer matches the persisted generation digest.
	claude = archiveForHarness(t, replaced, "claude")
	var updated []byte
	priorInfo, err := os.Stat(claude.Path)
	if err != nil {
		t.Fatal(err)
	}
	updated = bytes.Replace(readArchive(t, claude.Path), []byte("new marker"), []byte("replacement marker grew substantially"), 1)
	if err := os.WriteFile(claude.Path, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(claude.Path, priorInfo.ModTime(), priorInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	indexer = NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "new marker", 3)
	assertArchiveSearchCount(t, ctx, search, "replacement marker grew", 1)

	if err := os.RemoveAll(roots.FirstPath("opencode")); err != nil {
		t.Fatal(err)
	}
	indexer = NewNativeArchiveIndexer(conversations, checkpoints, log.New(io.Discard, "", 0))
	engine, _, err = indexer.NewDiscoveredEngine(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	engine.Scan(ctx)
	engine.Stop()
	assertArchiveSearchCount(t, ctx, search, "old marker", 0)
	assertArchiveSearchCount(t, ctx, search, "new marker", 2)
	if got := conversations.CountSessionEvents(ctx, "wc-archive-opencode-native-opencode"); got != 0 {
		t.Fatalf("removed OpenCode root left %d searchable projection events", got)
	}
}

func readArchive(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func writeArchiveGeneration(t *testing.T, path string, content []byte, seconds int) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(time.Duration(seconds) * time.Second)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
}

func nativeArchiveFixturePath(roots NativeArchiveRoots, harness, nativeID string) string {
	switch harness {
	case "claude":
		return filepath.Join(roots.FirstPath("claude"), "synthetic-project", nativeID+".jsonl")
	case "codex":
		return filepath.Join(roots.Paths("codex")[0], "2026", "11", "22", "rollout-synthetic-"+nativeID+".jsonl")
	case "grok":
		return filepath.Join(roots.FirstPath("grok"), "encoded-synthetic-cwd", nativeID, "updates.jsonl")
	case "opencode":
		return filepath.Join(roots.FirstPath("opencode"), nativeID, "message-0001.json")
	default:
		panic("unsupported synthetic harness " + harness)
	}
}

type archiveFailOnceCheckpointStore struct {
	mu      sync.Mutex
	inner   AgentTranscriptCheckpointStore
	failOne bool
}

func (s *archiveFailOnceCheckpointStore) nativeArchiveCatalog() *nativeArchiveCatalog {
	if provider, ok := s.inner.(interface{ nativeArchiveCatalog() *nativeArchiveCatalog }); ok {
		return provider.nativeArchiveCatalog()
	}
	return newMemoryNativeArchiveCatalog()
}

func (s *archiveFailOnceCheckpointStore) Get(ctx context.Context, source, key string) (AgentTranscriptCheckpoint, bool, error) {
	return s.inner.Get(ctx, source, key)
}

func (s *archiveFailOnceCheckpointStore) Save(ctx context.Context, checkpoint AgentTranscriptCheckpoint) error {
	s.mu.Lock()
	if s.failOne {
		s.failOne = false
		s.mu.Unlock()
		return fmt.Errorf("injected checkpoint write failure")
	}
	s.mu.Unlock()
	return s.inner.Save(ctx, checkpoint)
}

func (s *archiveFailOnceCheckpointStore) DeleteSession(ctx context.Context, sessionID string) error {
	return s.inner.DeleteSession(ctx, sessionID)
}

func (s *archiveFailOnceCheckpointStore) failNextSave() {
	s.mu.Lock()
	s.failOne = true
	s.mu.Unlock()
}

func nativeArchiveCorpusLine(harness, role, text string, tool bool) []byte {
	var record any
	switch harness {
	case "claude":
		if tool {
			record = map[string]any{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "id": "wc4-tool", "name": "Search", "input": map[string]string{"query": text}}}}}
		} else {
			record = map[string]any{"type": role, "message": map[string]any{"content": []any{map[string]string{"type": "text", "text": text}}}}
		}
	case "codex":
		if tool {
			record = map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "wc4-tool", "name": "search", "arguments": `{"query":"` + text + `"}`}}
		} else if role == "user" {
			record = map[string]any{"type": "event_msg", "payload": map[string]string{"type": "user_message", "text": text}}
		} else {
			record = map[string]any{"type": "response_item", "payload": map[string]string{"type": "agent_message", "text": text}}
		}
	case "grok":
		kind := "agent_message_chunk"
		if role == "user" {
			kind = "user_message_chunk"
		}
		update := map[string]any{"sessionUpdate": kind, "content": map[string]string{"type": "text", "text": text}}
		if tool {
			update = map[string]any{"sessionUpdate": "tool_call", "toolCallId": "wc4-tool", "title": "Search", "rawInput": map[string]string{"query": text}}
		}
		record = map[string]any{"method": "session/update", "params": map[string]any{"sessionId": "native-grok", "update": update}}
	case "opencode":
		part := map[string]any{"type": "text", "text": text}
		if tool {
			role = "assistant"
			part = map[string]any{"type": "tool", "callID": "wc4-tool", "tool": "search", "input": map[string]string{"query": text}}
		}
		record = map[string]any{"info": map[string]string{"role": role}, "parts": []any{part}}
	default:
		panic("unsupported synthetic harness " + harness)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	return encoded
}

func appendArchiveLine(t *testing.T, path string, line []byte) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertArchiveSearchCount(t *testing.T, ctx context.Context, search interface {
	SearchArchived(context.Context, *connect.Request[conversationv1.SearchArchivedRequest]) (*connect.Response[conversationv1.SearchArchivedResponse], error)
}, query string, want int) {
	t.Helper()
	response, err := search.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: query, Limit: 20}))
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}
	if got := len(response.Msg.GetMatches()); got != want {
		t.Fatalf("search %q returned %d matches, want %d", query, got, want)
	}
}

func durationPercentiles(values []time.Duration) (median, p95 time.Duration) {
	if len(values) == 0 {
		return 0, 0
	}
	ordered := append([]time.Duration(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	median = ordered[int(math.Ceil(float64(len(ordered))*0.50))-1]
	p95 = ordered[int(math.Ceil(float64(len(ordered))*0.95))-1]
	return median, p95
}
