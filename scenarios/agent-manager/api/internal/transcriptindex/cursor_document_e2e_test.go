package transcriptindex_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/domain"
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
