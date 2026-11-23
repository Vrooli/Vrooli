package transcriptindex_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/transcriptindex"

	"github.com/vrooli/cli-core/agenttranscript"
)

func TestQwenCodeJSONLFlowsThroughAgentManagerIndexSearchAndResumeDescriptor(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	const sessionID = "13c74da8-655c-44f8-9f04-570c0a9571af"
	const clue = "qwen-native-search-521"
	const appendedClue = "qwen-native-tail-884"
	path := filepath.Join(r.home, ".qwen", "projects", "-home-example-project", "chats", sessionID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	records := []string{
		`{"type":"user","sessionId":"` + sessionID + `","timestamp":"2026-11-23T16:01:00.123Z","message":{"role":"user","parts":[{"text":"` + strings.Repeat("x", 5000) + `"}]}}`,
		`{"type":"assistant","sessionId":"` + sessionID + `","timestamp":"2026-11-23T16:01:01.456Z","message":{"role":"model","parts":[{"text":"` + clue + `"},{"text":"hidden thought","thought":true}]}}`,
		`{"type":"system","sessionId":"` + sessionID + `","timestamp":"2026-11-23T16:01:02Z","subtype":"custom_title","systemPayload":{"customTitle":"metadata only"}}`,
	}
	transcript := strings.Join(records, "\n") + "\n"
	if err := os.WriteFile(path, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Read != 1 || len(summary.Failures) != 0 {
		t.Fatalf("Qwen transcript summary = %+v", summary)
	}
	row := r.row(path)
	if row.RootKind != transcriptindex.RootQwenCode || row.Harness != "qwen-code" || row.SessionID != sessionID || row.HeaderSessionID != sessionID || row.Offset != int64(len(transcript)) {
		t.Fatalf("Qwen file row = %+v", row)
	}
	appended := `{"type":"assistant","sessionId":"` + sessionID + `","timestamp":"2026-11-23T16:02:00Z","message":{"role":"model","parts":[{"text":"` + appendedClue + `"}]}}` + "\n"
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
		t.Fatalf("Qwen append tail summary = %+v", summary)
	}
	row = r.row(path)
	if row.Offset != int64(len(transcript)+len(appended)) {
		t.Fatalf("Qwen append offset = %d, want %d", row.Offset, len(transcript)+len(appended))
	}
	if got := r.tokens(row.RunID); got != 0 {
		t.Fatalf("Qwen text transcript unexpectedly imported usage: %d", got)
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
	job, err := indexer.RunOnce(ctx, 0)
	if err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Qwen search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{Query: appendedClue, PageSize: 10, Filters: conversationsearch.SearchFilters{Harnesses: []string{"qwen-code"}}})
	if err != nil || len(response.Hits) == 0 {
		t.Fatalf("Qwen native search hits = %d, err=%v", len(response.Hits), err)
	}
	for _, hit := range response.Hits {
		if !strings.Contains(hit.Document.Content, appendedClue) {
			t.Fatalf("Qwen search returned unrelated document: %+v", hit.Document)
		}
		if strings.Contains(hit.Document.Content, "hidden thought") {
			t.Fatal("Qwen hidden thought content entered searchable documents")
		}
	}
	format, ok := agenttranscript.FormatFor("qwen-code")
	if !ok || format.ResumeFlag != "--resume" {
		t.Fatalf("Qwen resume descriptor = %+v, found=%v", format, ok)
	}
	resumeArgs := []string{format.ResumeFlag, sessionID, "-p", "continue"}
	if got := strings.Join(resumeArgs, " "); got != "--resume "+sessionID+" -p continue" {
		t.Fatalf("Qwen native resume args = %q", got)
	}
}
