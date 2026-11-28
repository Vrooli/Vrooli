package transcriptindex_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
)

func TestGrokSharedACPFixtureFlowsThroughAgentManagerTailSearchAndResume(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	const sessionID = "grok-shared-synthetic-session"
	const clue = "groksharedsearch"
	const appendedClue = "groksharedtail"
	path := filepath.Join(r.home, ".grok", "sessions", "synthetic-cwd", sessionID, "updates.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve shared Grok fixture path")
	}
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "../../../../../packages/cli-core/agenttranscript/testdata/grok-shared-consumer.jsonl")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read shared Grok ACP fixture %q: %v", fixturePath, err)
	}
	if err := os.WriteFile(path, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Discovered != 1 || summary.Read != 1 || len(summary.Failures) != 0 {
		t.Fatalf("Grok shared fixture tail summary = %+v", summary)
	}
	row := r.row(path)
	if row.Harness != transcriptindex.HarnessGrok || row.SessionID != sessionID {
		t.Fatalf("Grok shared fixture identity = harness %q session %q", row.Harness, row.SessionID)
	}

	ctx := context.Background()
	normalizer, err := conversationsearch.NewNormalizer(conversationsearch.NormalizerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	source, err := conversationsearch.NewSQLiteSource(r.db.DB, normalizer)
	if err != nil {
		t.Fatal(err)
	}
	repository := conversationsearch.NewSQLiteRepository(r.db.DB)
	indexer, err := conversationsearch.NewIndexer(conversationsearch.IndexerOptions{Source: source, Repository: repository})
	if err != nil {
		t.Fatal(err)
	}
	if job, err := indexer.RunOnce(ctx, 0); err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Grok search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	search := func(query string) bool {
		t.Helper()
		response, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{Query: query, PageSize: 10, Filters: conversationsearch.SearchFilters{Harnesses: []string{transcriptindex.HarnessGrok}}})
		if err != nil {
			t.Fatal(err)
		}
		for _, hit := range response.Hits {
			if hit.Document.SourceRunID == row.RunID && hit.Document.SourceSessionID == sessionID && strings.Contains(strings.ToLower(hit.Document.Content), strings.ToLower(query)) {
				return true
			}
		}
		return false
	}
	if !search(clue) {
		t.Fatal("Grok shared fixture assistant message was not searchable")
	}
	appendRecord := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + sessionID + `","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"` + appendedClue + `"}}}}` + "\n" + `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + sessionID + `","update":{"sessionUpdate":"turn_completed","stop_reason":"end_turn"}}}` + "\n"
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
	if summary := r.tick(); summary.Read != 1 || len(summary.Failures) != 0 {
		t.Fatalf("Grok shared fixture tail summary after append = %+v", summary)
	}
	if job, err := indexer.RunOnce(ctx, 0); err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Grok incremental search indexing job = %#v, err=%v", job, err)
	}
	if !search(appendedClue) {
		t.Fatal("Grok appended shared fixture message was not searchable")
	}

	prompt := `continue "the Grok ACP session"; $() stays literal`
	args := codecs.NewGrokForTest().BuildContinueArgs(codecs.NewGrokForTest().NewState(), runner.ContinueRequest{
		SessionID: sessionID, Prompt: prompt,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeGrok},
	})
	wantPrefix := []string{"-p", prompt, "--resume", sessionID}
	if len(args) < len(wantPrefix) {
		t.Fatalf("Grok native resume args = %#v, want prefix %#v", args, wantPrefix)
	}
	for i := range wantPrefix {
		if args[i] != wantPrefix[i] {
			t.Fatalf("Grok native resume args = %#v, want descriptor-backed prefix %#v", args, wantPrefix)
		}
	}
}
