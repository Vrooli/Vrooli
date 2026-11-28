package transcriptindex_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
)

func TestGrokRealCorpusFlowsThroughTailIndexSearchAndResume(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	fixturePath := filepath.Join("..", "adapters", "runner", "codecs", "testdata", "corpus", "grok-updates.jsonl")
	corpus, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var searchTerm string
	for _, line := range strings.Split(string(corpus), "\n") {
		for _, message := range agenttranscript.ParseMessages(agenttranscript.HarnessGrok, []byte(line)) {
			if message.Role != "assistant" || strings.HasPrefix(message.Text, "[Tool ") {
				continue
			}
			for _, word := range strings.Fields(message.Text) {
				word = strings.Trim(word, ".,!?;:()[]{}\"'")
				if len(word) >= 7 {
					searchTerm = word
					break
				}
			}
			if searchTerm != "" {
				break
			}
		}
		if searchTerm != "" {
			break
		}
	}
	if searchTerm == "" {
		t.Fatal("redacted Grok ACP corpus has no assistant prose term for the lexical search check")
	}

	path := filepath.Join(r.home, ".grok", "sessions", "fixture-cwd", "019f1023-e563-7513-a64e-96165fba4be6", "updates.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, corpus, 0o600); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Read != 1 {
		t.Fatalf("real-corpus transcript tail summary = %+v", summary)
	}
	row := r.row(path)
	if row.Harness != transcriptindex.HarnessGrok || row.SessionID != "019f1023-e563-7513-a64e-96165fba4be6" {
		t.Fatalf("real-corpus identity = harness %q session %q", row.Harness, row.SessionID)
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
		t.Fatalf("search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(searchRepo, searchRepo, searchRepo, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{
		Query: searchTerm, PageSize: 10,
		Filters: conversationsearch.SearchFilters{Harnesses: []string{transcriptindex.HarnessGrok}},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, hit := range response.Hits {
		if hit.Document.SourceRunID == row.RunID && hit.Document.SourceSessionID == row.SessionID && strings.Contains(strings.ToLower(hit.Document.Content), strings.ToLower(searchTerm)) {
			found = true
		}
	}
	if !found {
		t.Fatalf("real Grok corpus search for selected assistant term produced %d hits without the indexed session", len(response.Hits))
	}

	format, ok := agenttranscript.FormatFor(agenttranscript.HarnessGrok)
	if !ok {
		t.Fatal("shared Grok format descriptor is unavailable")
	}
	resume := codecs.NewGrokForTest().BuildContinueArgs(codecs.NewGrokForTest().NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: row.SessionID, Prompt: "continue the indexed conversation",
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeGrok},
	})
	if !hasArgPair(resume, format.ResumeFlag, row.SessionID) {
		t.Fatalf("shared Grok resume flag %q was not used: %#v", format.ResumeFlag, resume)
	}
}

func hasArgPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
