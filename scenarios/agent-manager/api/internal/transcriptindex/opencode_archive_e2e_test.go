package transcriptindex_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/runner/opencodearchive"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
	_ "modernc.org/sqlite"
)

// This uses OpenCode's durable session_message/session_part table shape in a
// temporary DB. It proves the adapter and full local Agent Manager path, but
// is fixture evidence, not a live archive or unseen-harness showcase.
func TestOpenCodeArchiveFlowsThroughIndexSearchAndResume(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	if err := os.MkdirAll(filepath.Dir(r.openCodeDB), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", r.openCodeDB)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE session_message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, seq INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE session_part (message_id TEXT NOT NULL, session_id TEXT NOT NULL, ordinal INTEGER NOT NULL, data TEXT NOT NULL)`,
		`INSERT INTO session_message VALUES ('msg-user', 'ses_archive_47', 1, '{"id":"msg-user","sessionID":"ses_archive_47","role":"user","time":{"created":1763798400000}}')`,
		`INSERT INTO session_part VALUES ('msg-user', 'ses_archive_47', 0, '{"id":"part-user","type":"text","text":"Which archive path should remain searchable?"}')`,
		`INSERT INTO session_message VALUES ('msg-assistant', 'ses_archive_47', 2, '{"id":"msg-assistant","sessionID":"ses_archive_47","role":"assistant","time":{"created":1763798401000}}')`,
		`INSERT INTO session_part VALUES ('msg-assistant', 'ses_archive_47', 0, '{"id":"part-assistant","type":"text","text":"Keep OpenCode archive identity through the shared parser."}')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if summary := r.tick(); summary.Discovered != 1 || summary.Read != 1 {
		t.Fatalf("OpenCode archive indexing summary = %+v", summary)
	}
	archiveCache := opencodearchive.CacheDirectory(r.openCodeDB, r.openCodeCache)
	entries, err := os.ReadDir(archiveCache)
	if err != nil || len(entries) != 1 {
		t.Fatalf("OpenCode archive cache entries = %v, err=%v", entries, err)
	}
	archivePath := filepath.Join(archiveCache, entries[0].Name())
	row := r.row(archivePath)
	if row.Harness != "opencode" || row.SessionID != "ses_archive_47" {
		t.Fatalf("indexed OpenCode identity = harness %q session %q", row.Harness, row.SessionID)
	}
	initialRunID := row.RunID

	// A later native database message refreshes the derived JSONL source and
	// remains on the same local indexed session lineage.
	db, err = sql.Open("sqlite", r.openCodeDB)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO session_message VALUES ('msg-followup', 'ses_archive_47', 3, '{"id":"msg-followup","sessionID":"ses_archive_47","role":"assistant","time":{"created":1763798402000}}')`,
		`INSERT INTO session_part VALUES ('msg-followup', 'ses_archive_47', 0, '{"id":"part-followup","type":"text","text":"The archive tail updated incrementally."}')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if summary := r.tick(); summary.Read != 1 {
		t.Fatalf("OpenCode archive tail summary = %+v", summary)
	}
	row = r.row(archivePath)
	if row.RunID != initialRunID || row.SessionID != "ses_archive_47" {
		t.Fatalf("OpenCode archive tail changed session lineage: initial run %q, current row %+v", initialRunID, row)
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
		t.Fatalf("OpenCode search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(searchRepo, searchRepo, searchRepo, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.SearchText(ctx, conversationsearch.TextSearchRequest{
		Query: "updated incrementally", PageSize: 10,
		Filters: conversationsearch.SearchFilters{Harnesses: []string{"opencode"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, hit := range response.Hits {
		if hit.Document.SourceRunID == row.RunID && hit.Document.SourceSessionID == row.SessionID && strings.Contains(hit.Document.Content, "updated incrementally") {
			found = true
		}
	}
	if !found {
		t.Fatalf("OpenCode archive tail query produced %d hits without session identity", len(response.Hits))
	}

	format, ok := agenttranscript.FormatFor("opencode")
	if !ok {
		t.Fatal("shared OpenCode format descriptor is unavailable")
	}
	codec := codecs.NewOpenCodeForTest()
	resume := codec.BuildContinueArgs(codec.NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: row.SessionID, Prompt: "continue this indexed session",
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode},
	})
	if !hasArgPair(resume, format.ResumeFlag, row.SessionID) {
		t.Fatalf("shared OpenCode resume flag %q missing from %#v", format.ResumeFlag, resume)
	}

	// Resume serialization is checked locally only; the adapter never launches
	// OpenCode or transmits the selected context.
	if strings.Contains(strings.Join(resume, " "), "Which archive path") {
		t.Fatal("resume args unexpectedly contain archived conversation content")
	}
}
