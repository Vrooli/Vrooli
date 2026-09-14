//go:build rehearsal

package conversationsearch

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
)

// Temporary rehearsal harness (deleted after the 2026-09-14 rehearsal).
type timingDB struct {
	*sqlx.DB
	mu      sync.Mutex
	maxExec time.Duration
	maxSQL  string
	execs   int
}

func (d *timingDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := d.DB.ExecContext(ctx, query, args...)
	elapsed := time.Since(start)
	d.mu.Lock()
	d.execs++
	if elapsed > d.maxExec {
		d.maxExec, d.maxSQL = elapsed, strings.Join(strings.Fields(query), " ")
		if len(d.maxSQL) > 90 {
			d.maxSQL = d.maxSQL[:90]
		}
	}
	d.mu.Unlock()
	return result, err
}

func (d *timingDB) reset() (time.Duration, string, int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	max, query, n := d.maxExec, d.maxSQL, d.execs
	d.maxExec, d.maxSQL, d.execs = 0, "", 0
	return max, query, n
}

func openRehearsal(t *testing.T, path string, readOnly bool) *sqlx.DB {
	dsn := "file:" + path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	if readOnly {
		dsn = "file:" + path + "?mode=ro&_pragma=busy_timeout(10000)"
	}
	db, err := sqlx.Connect("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRehearsalProjectionUpgrade(t *testing.T) {
	workPath, beforePath := os.Getenv("REHEARSAL_WORK_DB"), os.Getenv("REHEARSAL_BEFORE_DB")
	if workPath == "" || beforePath == "" {
		t.Skip("rehearsal paths not set")
	}
	ctx := context.Background()
	raw := openRehearsal(t, workPath, false)
	start := time.Now()
	require.NoError(t, coredb.EnsureSchemas(ctx, raw, coredb.SchemaProviderFunc(Schema)), "startup schema application must accept a legacy projection")
	t.Logf("schema apply on legacy DB: %s", time.Since(start))
	timed := &timingDB{DB: raw}
	repository := NewSQLiteRepository(timed)

	var legacyPresent bool
	require.NoError(t, raw.Get(&legacyPresent, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name = 'conversation_search_documents')`))
	var maxExec time.Duration
	var maxSQL string
	var execs int
	if legacyPresent {
		start = time.Now()
		require.NoError(t, repository.copyLegacyCatalog(ctx))
		maxExec, maxSQL, execs = timed.reset()
		t.Logf("copy legacy catalog: %s over %d statements; longest writer hold %s (%s)", time.Since(start), execs, maxExec, maxSQL)
		start = time.Now()
		require.NoError(t, repository.dropLegacyProjection(ctx))
		maxExec, maxSQL, execs = timed.reset()
		t.Logf("drop legacy objects: %s over %d statements; longest %s (%s)", time.Since(start), execs, maxExec, maxSQL)
	}
	upgraded, err := repository.UpgradeLegacyProjection(ctx)
	require.NoError(t, err)
	require.False(t, upgraded)

	start = time.Now()
	purged, err := repository.PruneSettledGenerations(ctx)
	require.NoError(t, err)
	maxExec, maxSQL, execs = timed.reset()
	t.Logf("prune settled staging: %s, generations=%d, statements=%d, longest %s (%s)", time.Since(start), len(purged), execs, maxExec, maxSQL)
	requireLexicalIntegrity(t, repository)

	var catalog, lexical int
	require.NoError(t, raw.Get(&catalog, `SELECT COUNT(*) FROM conversation_search_catalog`))
	require.NoError(t, raw.Get(&lexical, `SELECT COUNT(*) FROM conversation_search_catalog_fts_docsize`))
	t.Logf("catalog=%d lexical=%d", catalog, lexical)

	before := openRehearsal(t, beforePath, true)
	for _, term := range []string{"retention", "vacuum", "swarm", "playwright", "\"drift check\"", "codex OR claude"} {
		var legacyIDs, currentIDs []string
		require.NoError(t, before.Select(&legacyIDs, `SELECT d.document_id FROM conversation_search_fts
JOIN conversation_search_documents d ON d.rowid = conversation_search_fts.rowid
WHERE conversation_search_fts MATCH ? AND d.visible = 1
ORDER BY bm25(conversation_search_fts), d.document_id LIMIT 25`, term))
		require.NoError(t, raw.Select(&currentIDs, `SELECT d.document_id FROM conversation_search_catalog_fts
JOIN conversation_search_catalog d ON d.id = conversation_search_catalog_fts.rowid
WHERE conversation_search_catalog_fts MATCH ? AND d.visible = 1
ORDER BY bm25(conversation_search_catalog_fts), d.document_id LIMIT 25`, term))
		var legacyCount, currentCount int
		require.NoError(t, before.Get(&legacyCount, `SELECT COUNT(*) FROM conversation_search_fts WHERE conversation_search_fts MATCH ?`, term))
		require.NoError(t, raw.Get(&currentCount, `SELECT COUNT(*) FROM conversation_search_catalog_fts WHERE conversation_search_catalog_fts MATCH ?`, term))
		sameOrder := strings.Join(legacyIDs, ",") == strings.Join(currentIDs, ",")
		a, b := append([]string(nil), legacyIDs...), append([]string(nil), currentIDs...)
		sort.Strings(a)
		sort.Strings(b)
		t.Logf("parity %-18q matches legacy=%d current=%d top25 sameOrder=%v sameSet=%v", term, legacyCount, currentCount, sameOrder, strings.Join(a, ",") == strings.Join(b, ","))
	}

	var commonProject, commonHarness, rareModel string
	require.NoError(t, raw.Get(&commonProject, `SELECT project_scope FROM conversation_search_catalog GROUP BY 1 ORDER BY COUNT(*) DESC LIMIT 1`))
	require.NoError(t, raw.Get(&commonHarness, `SELECT harness FROM conversation_search_catalog GROUP BY 1 ORDER BY COUNT(*) DESC LIMIT 1`))
	require.NoError(t, raw.Get(&rareModel, `SELECT model FROM conversation_search_catalog WHERE model <> '' GROUP BY 1 ORDER BY COUNT(*) ASC LIMIT 1`))
	cases := []struct {
		name  string
		query CandidateQuery
	}{
		{"relevance + project", CandidateQuery{Query: "retention policy", Limit: 101, ProjectScopes: []string{commonProject}}},
		{"relevance + harness", CandidateQuery{Query: "vacuum", Limit: 101, Harnesses: []string{commonHarness}}},
		{"newest, no query", CandidateQuery{Limit: 101, Sort: SearchSortNewest}},
		{"newest + harness", CandidateQuery{Limit: 101, Sort: SearchSortNewest, Harnesses: []string{commonHarness}}},
		{"newest + rare model", CandidateQuery{Limit: 101, Sort: SearchSortNewest, Models: []string{rareModel}}},
		{"newest + project + time", CandidateQuery{Limit: 101, Sort: SearchSortNewest, ProjectScopes: []string{commonProject}, OccurredAfter: ptrTime(time.Now().Add(-72 * time.Hour))}},
	}
	for _, test := range cases {
		start := time.Now()
		candidates, err := repository.LexicalCandidates(ctx, test.query)
		require.NoError(t, err)
		t.Logf("latency %-24s %8s results=%d", test.name, time.Since(start).Round(time.Millisecond), len(candidates))
	}

	if os.Getenv("REHEARSAL_FULL_REBUILD") == "1" {
		source, err := NewSQLiteSource(raw, MustNormalizer(NormalizerConfig{}))
		require.NoError(t, err)
		indexer, err := NewIndexer(IndexerOptions{Source: source, Repository: repository})
		require.NoError(t, err)
		start = time.Now()
		job, err := indexer.RunOnce(ctx, 0)
		require.NoError(t, err)
		maxExec, maxSQL, execs = timed.reset()
		t.Logf("full rebuild: %s state=%s docs=%d statements=%d longest %s (%s)", time.Since(start), job.State, job.PlannedDocuments, execs, maxExec, maxSQL)
		var staged int
		require.NoError(t, raw.Get(&staged, `SELECT COUNT(*) FROM conversation_search_generation_documents`))
		t.Logf("staged rows after rebuild activation: %d", staged)
		requireLexicalIntegrity(t, repository)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
