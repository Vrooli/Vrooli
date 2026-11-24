package transcriptindex_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/transcriptindex"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
	_ "modernc.org/sqlite"
)

func TestCursorComposersFlowThroughAgentManagerImportIndexAndSearch(t *testing.T) {
	r := newRig(t, 1<<20)
	dbPath := filepath.Join(r.home, ".cursor", "User", "workspaceStorage", "workspace-a", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB);
	INSERT INTO cursorDiskKV(key,value) VALUES
	('composerData:composer-a', '{"composerId":"composer-a","fullConversationHeadersOnly":[{"bubbleId":"u1","type":1},{"bubbleId":"a1","type":2}]}'),
	('bubbleId:composer-a:u1', '{"text":"Question about a meadow"}'),
	('bubbleId:composer-a:a1', '{"text":"Cursor-search-marker amber-meadow-482."}'),
	('composerData:composer-b', '{"composerId":"composer-b","fullConversationHeadersOnly":[{"bubbleId":"u2","type":1},{"bubbleId":"a2","type":2}]}'),
	('composerData:mismatched-key', '{"composerId":"different-composer","fullConversationHeadersOnly":[]}'),
	('bubbleId:composer-b:u2', '{"text":"Question about a forest"}'),
	('bubbleId:composer-b:a2', '{"text":"Cursor-search-marker green-forest-715."}');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	summary := r.tick()
	if summary.Read != 2 || summary.Discovered != 2 {
		t.Fatalf("Cursor discovery/import summary = %+v", summary)
	}
	rows, err := r.repos.TranscriptIndex.ListTranscriptFiles(context.Background())
	if err != nil || len(rows) == 0 {
		t.Fatalf("Cursor transcript rows = %d, err=%v", len(rows), err)
	}
	seen := map[string]string{}
	for _, row := range rows {
		if row.Harness == "cursor" {
			seen[row.SessionID] = row.RunID
		}
	}
	if len(seen) != 2 || seen["composer-a"] == "" || seen["composer-b"] == "" || seen["composer-a"] == seen["composer-b"] {
		t.Fatalf("Cursor composer run attribution = %#v", seen)
	}
	for _, runID := range seen {
		run := r.run(runID)
		if run.ExecutionMode != domain.ExecutionModeImported || run.ResolvedConfig.TranscriptCodec != "cursor" {
			t.Fatalf("Cursor source run lost its read-only native provenance: %+v", run)
		}
	}

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
	job, err := searchIndexer.RunOnce(context.Background(), 0)
	if err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Cursor search indexing job = %#v, err=%v", job, err)
	}
	service, err := conversationsearch.NewService(searchRepo, searchRepo, searchRepo, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.SearchText(context.Background(), conversationsearch.TextSearchRequest{Query: "amber-meadow-482", PageSize: 10, Filters: conversationsearch.SearchFilters{Harnesses: []string{"cursor"}}})
	if err != nil || len(response.Hits) != 1 || response.Hits[0].Document.SourceSessionID != "composer-a" {
		t.Fatalf("Cursor search response = %+v, err=%v", response, err)
	}

	if _, ok := agenttranscript.FormatFor("cursor"); !ok {
		t.Fatal("Cursor shared transcript descriptor missing")
	}
	if f, _ := agenttranscript.FormatFor("cursor"); f.ResumeFlag != "" {
		t.Fatalf("Cursor resume was advertised without a supported selector: %+v", f)
	}

	// Replace composer-a in place with same-length text. Per-composer content
	// hashes must cause only its snapshot to be reimported and old text to leave search.
	db, err = sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE cursorDiskKV SET value='{"text":"Cursor-search-marker cider-meadow-593."}' WHERE key='bubbleId:composer-a:a1'`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	r.restart()
	updated := r.tick()
	if updated.Read != 1 || len(updated.Failures) != 0 {
		t.Fatalf("Cursor composer rewrite summary = %+v", updated)
	}
	job, err = searchIndexer.RunOnce(context.Background(), 0)
	if err != nil || job.State != conversationsearch.ReindexComplete {
		t.Fatalf("Cursor rewritten search indexing job = %#v, err=%v", job, err)
	}
	old, err := service.SearchText(context.Background(), conversationsearch.TextSearchRequest{Query: "amber-meadow-482", PageSize: 10, Filters: conversationsearch.SearchFilters{Harnesses: []string{"cursor"}}})
	if err != nil || len(old.Hits) != 0 {
		t.Fatalf("stale Cursor content remains searchable: hits=%d err=%v", len(old.Hits), err)
	}
	newHit, err := service.SearchText(context.Background(), conversationsearch.TextSearchRequest{Query: "cider-meadow-593", PageSize: 10, Filters: conversationsearch.SearchFilters{Harnesses: []string{"cursor"}}})
	if err != nil || len(newHit.Hits) != 1 || strings.TrimSpace(newHit.Hits[0].Document.SourceSessionID) != "composer-a" {
		t.Fatalf("rewritten Cursor search result = %+v, err=%v", newHit, err)
	}

	// Removing composer-a from the shared state DB must reconcile only that
	// composer. The database path still exists because composer-b remains.
	db, err = sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM cursorDiskKV WHERE key='composerData:composer-a' OR key LIKE 'bubbleId:composer-a:%'`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(48 * time.Hour)
	r.restart()
	removed, err := r.index.RunOnce(context.Background())
	if err != nil || len(removed.Failures) != 0 {
		t.Fatalf("Cursor composer removal reconciliation = %+v err=%v", removed, err)
	}
	if removed.Discovered != 1 || removed.Changed != 0 || removed.Read != 0 || removed.Unknown != 1 {
		rows, _ := r.repos.TranscriptIndex.ListTranscriptFiles(context.Background())
		t.Fatalf("Cursor removal affected a sibling or missed the removed composer: summary=%+v rows=%+v", removed, rows)
	}
	rows, err = r.repos.TranscriptIndex.ListTranscriptFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var removedRow, siblingRow *transcriptindex.FileRow
	for index := range rows {
		row := &rows[index]
		if row.Harness != "cursor" {
			continue
		}
		switch row.SessionID {
		case "composer-a":
			removedRow = row
		case "composer-b":
			siblingRow = row
		}
	}
	if removedRow == nil || removedRow.Active || siblingRow == nil || r.run(removedRow.RunID).Status != domain.RunStatusUnknown || r.run(siblingRow.RunID).Status == domain.RunStatusUnknown {
		t.Fatalf("per-composer removal state: removed=%+v sibling=%+v", removedRow, siblingRow)
	}
}

type cursorInterruptedCommitStore struct {
	transcriptindex.Store
	fail bool
}

func (s *cursorInterruptedCommitStore) BeginTranscriptCapture(ctx context.Context, expected *transcriptindex.FileRow, intent transcriptindex.FileRow) error {
	return s.Store.(interface {
		BeginTranscriptCapture(context.Context, *transcriptindex.FileRow, transcriptindex.FileRow) error
	}).BeginTranscriptCapture(ctx, expected, intent)
}

func (s *cursorInterruptedCommitStore) FailTranscriptCapture(ctx context.Context, key, intent, status, message string) error {
	return s.Store.(interface {
		FailTranscriptCapture(context.Context, string, string, string, string) error
	}).FailTranscriptCapture(ctx, key, intent, status, message)
}

func (s *cursorInterruptedCommitStore) CommitTranscriptAndRun(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, expected *transcriptindex.FileRow, intent string, update func(*domain.Run) error) error {
	if s.fail {
		s.fail = false
		return errors.New("simulated interrupted transcript commit")
	}
	return s.Store.(interface {
		CommitTranscriptAndRun(context.Context, transcriptindex.FileRow, []transcriptindex.UsageRow, *transcriptindex.FileRow, string, func(*domain.Run) error) error
	}).CommitTranscriptAndRun(ctx, file, usage, expected, intent, update)
}

func TestCursorInterruptedComposerImportResumesAfterIndexerRestart(t *testing.T) {
	r := newRig(t, 1<<20)
	dbPath := filepath.Join(r.home, ".cursor", "User", "workspaceStorage", "workspace-interrupted", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB);
	INSERT INTO cursorDiskKV(key,value) VALUES
	('composerData:interrupted-composer', '{"composerId":"interrupted-composer","fullConversationHeadersOnly":[{"bubbleId":"u1","type":1},{"bubbleId":"a1","type":2}]}'),
	('bubbleId:interrupted-composer:u1', '{"text":"synthetic interrupted import question"}'),
	('bubbleId:interrupted-composer:a1', '{"text":"Cursor-restart-marker copper-hill-728."}');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store := &cursorInterruptedCommitStore{Store: r.repos.TranscriptIndex, fail: true}
	interrupted := orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: r.roots(), RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	first, err := interrupted.RunOnce(context.Background())
	if err != nil || len(first.Failures) == 0 {
		t.Fatalf("interrupted Cursor capture was not surfaced: %+v err=%v", first, err)
	}
	row := r.row(dbPath)
	if row.SessionID != "interrupted-composer" || row.Offset != 0 || row.Status != transcriptindex.StatusPending || !strings.Contains(row.Error, "capture interrupted") {
		t.Fatalf("interrupted Cursor checkpoint advanced or lost identity: %+v", row)
	}
	firstRunID := row.RunID
	r.restart()
	resumed := r.tick()
	row = r.row(dbPath)
	if resumed.Read != 1 || row.Offset <= 0 || row.Status != transcriptindex.StatusIndexed || row.Error != "" || row.SessionID != "interrupted-composer" || row.RunID != firstRunID {
		t.Fatalf("Cursor import did not reconcile after restart: summary=%+v row=%+v", resumed, row)
	}
	count, err := r.events.Count(context.Background(), uuid.MustParse(firstRunID))
	if err != nil || count == 0 {
		t.Fatalf("restarted Cursor import did not record events: count=%d err=%v", count, err)
	}
}

func TestCursorImportGateRejectsDatabaseWithoutNativeKeyValueTable(t *testing.T) {
	r := newRig(t, 1<<20)
	dbPath := filepath.Join(r.home, ".cursor", "User", "workspaceStorage", "workspace-bad", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE notCursorKV (key TEXT, value BLOB)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	summary, err := r.index.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	matched := false
	for _, failure := range summary.Failures {
		if strings.Contains(failure, filepath.Dir(filepath.Dir(dbPath))) && strings.Contains(failure, "cursorDiskKV") {
			matched = true
		}
	}
	if !matched || summary.Discovered != 0 || summary.Read != 0 {
		t.Fatalf("invalid Cursor import did not fail closed: summary=%+v", summary)
	}
	rows, err := r.repos.TranscriptIndex.ListTranscriptFiles(context.Background())
	if err != nil || len(rows) != 0 {
		t.Fatalf("invalid Cursor DB created transcript rows: count=%d err=%v", len(rows), err)
	}
}
