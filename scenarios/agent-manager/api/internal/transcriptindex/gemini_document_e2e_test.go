package transcriptindex_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
)

func TestGeminiCLIDocumentFlowsThroughAgentManagerIndexSearchAndResumeDescriptor(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	const sessionID = "d863e471-354a-4d98-a03b-7bd9ac27c66b"
	const clue = "amber-meadow-482"
	const updatedClue = "cider-stream-593"
	path := filepath.Join(r.home, ".gemini", "tmp", "project-hash", "chats", "session-fixture.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	document := `{"sessionId":"` + sessionID + `","startTime":"2025-11-22T00:00:00Z","lastUpdated":"2025-11-22T00:01:00Z","messages":[{"type":"user","content":"` + strings.Repeat("x", 5000) + `"},{"type":"gemini","content":"` + clue + `"}]}`
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Read != 1 {
		t.Fatalf("Gemini transcript summary = %+v", summary)
	}
	row := r.row(path)
	if row.RootKind != transcriptindex.RootGeminiCLI || row.Harness != "gemini-cli" || row.SessionID != sessionID || row.HeaderSessionID != sessionID || row.Offset != int64(len(document)) {
		t.Fatalf("Gemini file row = %+v", row)
	}
	fullHash, err := transcriptindex.FullHash(path)
	if err != nil || row.HeadHash != fullHash {
		t.Fatalf("Gemini full-document checkpoint hash mismatch: row=%q current=%q err=%v", row.HeadHash, fullHash, err)
	}
	updatedDocument := strings.Replace(document, clue, updatedClue, 1)
	if len(updatedDocument) != len(document) {
		t.Fatal("rewrite fixture must retain its original byte length")
	}
	if err := os.WriteFile(path, []byte(updatedDocument), 0o600); err != nil {
		t.Fatal(err)
	}
	updatedHash, err := transcriptindex.FullHash(path)
	if err != nil || updatedHash == row.HeadHash {
		t.Fatalf("fixture mutation was not detected by full hash: old=%q new=%q err=%v", row.HeadHash, updatedHash, err)
	}
	if summary := r.tick(); summary.Read != 1 || len(summary.Failures) != 0 {
		t.Fatalf("Gemini full-document rewrite summary = %+v", summary)
	}
	row = r.row(path)
	if summary := r.tick(); summary.Read != 0 || len(summary.Failures) != 0 {
		t.Fatalf("unchanged Gemini document was reimported: %+v", summary)
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
	searchRepo := conversationsearch.NewSQLiteRepository(r.db.DB)
	searchIndexer, err := conversationsearch.NewIndexer(conversationsearch.IndexerOptions{Source: source, Repository: searchRepo})
	if err != nil {
		t.Fatal(err)
	}
	job, err := searchIndexer.RunOnce(ctx, 0)
	if err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Gemini search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(searchRepo, searchRepo, searchRepo, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{
		Query: updatedClue, PageSize: 10,
		Filters: conversationsearch.SearchFilters{Harnesses: []string{"gemini-cli"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, hit := range response.Hits {
		if hit.Document.SourceRunID == row.RunID && hit.Document.SourceSessionID == sessionID && strings.Contains(hit.Document.Content, updatedClue) {
			found = true
		}
	}
	if !found {
		t.Fatalf("Gemini search did not return the indexed native session (hits=%d)", len(response.Hits))
	}
	oldResponse, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{
		Query: clue, PageSize: 10,
		Filters: conversationsearch.SearchFilters{Harnesses: []string{"gemini-cli"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(oldResponse.Hits) != 0 {
		t.Fatalf("stale Gemini document content remained searchable (%d hits)", len(oldResponse.Hits))
	}
	runID, err := uuid.Parse(row.RunID)
	if err != nil {
		t.Fatal(err)
	}
	run, err := r.repos.Runs.Get(ctx, runID)
	if err != nil || run == nil || run.ResolvedConfig.TranscriptCodec != "gemini-cli" || run.ResolvedConfig.RunnerType != domain.RunnerType("transcript-import") {
		t.Fatalf("Gemini imported run codec = %+v, err=%v", run, err)
	}
	format, ok := agenttranscript.FormatFor("gemini-cli")
	if !ok || format.ResumeFlag != "--resume" || format.ResumeSelection != "session-id-or-index-or-latest" {
		t.Fatalf("Gemini resume descriptor = %+v, found=%v", format, ok)
	}
	resumeArgs := []string{format.ResumeFlag, sessionID}
	if len(resumeArgs) != 2 || resumeArgs[0] != "--resume" || resumeArgs[1] != row.SessionID {
		t.Fatalf("Gemini native resume selection = %#v", resumeArgs)
	}
}
