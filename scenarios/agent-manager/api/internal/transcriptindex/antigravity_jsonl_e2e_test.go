package transcriptindex_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/transcriptindex"
)

func TestAntigravityNativeTranscriptFlowsThroughAgentManagerIndexSearchAndTail(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	const sessionID = "91a7638e-8aa3-4f7e-a63c-39ee39f9d26e"
	const clue = "antigravity-shared-codec-search"
	const appendedClue = "antigravity-native-tail-729"
	path := filepath.Join(r.home, ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript_full.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve shared Antigravity fixture source path")
	}
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "../../../../../packages/cli-core/agenttranscript/testdata/antigravity-shared-consumer.jsonl")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read shared Antigravity codec fixture %q: %v", fixturePath, err)
	}
	transcript := string(fixture) + "\n"
	if err := os.WriteFile(path, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	reducedPath := filepath.Join(filepath.Dir(path), "transcript.jsonl")
	if err := os.WriteFile(reducedPath, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Discovered != 1 || summary.Read != 1 || len(summary.Failures) != 0 {
		t.Fatalf("Antigravity transcript summary = %+v", summary)
	}
	row := r.row(path)
	if row.RootKind != transcriptindex.RootAntigravity || row.Harness != "antigravity" || row.SessionID != sessionID || row.HeaderSessionID != sessionID || row.Offset != int64(len(transcript)) {
		t.Fatalf("Antigravity file row = %+v", row)
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
		t.Fatalf("Antigravity search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	search := func(query string) conversationsearch.TextSearchResponse {
		t.Helper()
		response, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{Query: query, PageSize: 10, Filters: conversationsearch.SearchFilters{Harnesses: []string{"antigravity"}}})
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	if response := search(clue); len(response.Hits) != 3 {
		t.Fatalf("Antigravity native search returned %d hits, want three original fixture records", len(response.Hits))
	}
	if response := search("antigravity-generic-record-482"); len(response.Hits) != 1 || response.Hits[0].Document.OccurredAt.UTC().Format("2006-01-02T15:04:05Z") != "2026-11-24T10:00:03Z" {
		t.Fatalf("Antigravity generic native search response = %+v", response.Hits)
	}
	if response := search("synthetic hidden thought"); len(response.Hits) != 0 {
		t.Fatalf("Antigravity thinking leaked into search: %+v", response.Hits)
	}
	if response := search("non-message state"); len(response.Hits) != 0 {
		t.Fatalf("Antigravity checkpoint leaked into search: %+v", response.Hits)
	}

	appended := `{"step_index":6,"type":"PLANNER_RESPONSE","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:04Z","content":"` + appendedClue + `"}` + "\n"
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(appended); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Read != 1 || len(summary.Failures) != 0 {
		t.Fatalf("Antigravity append tail summary = %+v", summary)
	}
	row = r.row(path)
	if row.Offset != int64(len(transcript)+len(appended)) {
		t.Fatalf("Antigravity append offset = %d, want %d", row.Offset, len(transcript)+len(appended))
	}
	if job, err := indexer.RunOnce(ctx, 0); err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Antigravity incremental search indexing job = %#v, err=%v", job, err)
	}
	if response := search(appendedClue); len(response.Hits) != 1 {
		t.Fatalf("Antigravity appended search returned %d hits, want one", len(response.Hits))
	}
}
