package conversationsearch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
)

func TestLexicalCandidatesEscapeSyntaxAndApplyDefaultContentPolicy(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	insertSearchDocument(t, repository, "doc-old", "git-control-tower phase capacity", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "doc-new", "phase resource monitor", ContentClassQuotedProse, fixtureTimeValue(2))
	insertSearchDocument(t, repository, "doc-tool", "phase resource diagnostic", ContentClassToolResult, fixtureTimeValue(3))

	candidates, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: `"git-control-tower" phase`, Limit: 10})
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	for _, candidate := range candidates {
		require.NotEqual(t, "doc-tool", candidate.Document.DocumentID)
		require.Positive(t, candidate.Rank)
	}

	toolCandidates, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: "diagnostic", Limit: 10, ContentClasses: []ContentClass{ContentClassToolResult}})
	require.NoError(t, err)
	require.Len(t, toolCandidates, 1)
	require.Equal(t, "doc-tool", toolCandidates[0].Document.DocumentID)
}

func TestLexicalCandidatesApplyFacetsAndStableSortCursor(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	insertSearchDocument(t, repository, "doc-a", "shared capacity", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "doc-b", "shared capacity", ContentClassProse, fixtureTimeValue(2))

	first, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: "capacity", Limit: 1, Sort: SearchSortNewest, Harnesses: []string{"claude-code"}, Tags: []string{"testing"}})
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, "doc-b", first[0].Document.DocumentID)

	// A newer concurrent insert sorts ahead of the already-returned cursor and
	// therefore cannot duplicate or displace the older continuation.
	insertSearchDocument(t, repository, "doc-c", "shared capacity", ContentClassProse, fixtureTimeValue(3))
	next, err := repository.LexicalCandidates(context.Background(), CandidateQuery{
		Query: "capacity", Limit: 10, Sort: SearchSortNewest,
		After: &CandidateCursor{OccurredAt: first[0].Document.OccurredAt, DocumentID: first[0].Document.DocumentID},
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	require.Equal(t, "doc-a", next[0].Document.DocumentID)
}

func TestLexicalCandidatesBrowseByStructuredFilterWithoutFTSQuery(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	insertSearchDocument(t, repository, "doc-a", "unrelated content", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "doc-b", "other content", ContentClassProse, fixtureTimeValue(2))

	candidates, err := repository.LexicalCandidates(context.Background(), CandidateQuery{
		Limit: 10, Sort: SearchSortOldest, Harnesses: []string{"claude-code"},
	})
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	require.Equal(t, "doc-a", candidates[0].Document.DocumentID)
	require.Zero(t, candidates[0].Score)
}

func TestLexicalCandidatesRejectInjectionAsData(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	insertSearchDocument(t, repository, "doc-safe", "ordinary searchable content", ContentClassProse, fixtureTimeValue(1))

	_, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: `' OR 1=1 --`, Limit: 10})
	require.NoError(t, err)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM conversation_search_documents`))
	require.Equal(t, 1, count)
}

func TestBuildFTSExpressionDropsGenericRecallWordsButKeepsAllStopwordQueriesUsable(t *testing.T) {
	t.Parallel()
	expression, err := buildFTSExpressionFor("find the previous conversation about Test Genie adaptive concurrency design", true)
	require.NoError(t, err)
	require.Equal(t, `"Test" AND "Genie" AND "adaptive" AND "concurrency" AND "design"`, expression)

	gibberish, err := buildFTSExpressionFor("zxqv jklp 9981 nonsensical recall", true)
	require.NoError(t, err)
	require.Equal(t, `"zxqv" AND "jklp" AND "9981" AND "nonsensical"`, gibberish)

	fallback, err := buildFTSExpression("what was the conversation")
	require.NoError(t, err)
	require.NotEmpty(t, fallback)
}

func TestContextDocumentsBoundsAndOmitsHiddenRows(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	for sequence := 1; sequence <= 6; sequence++ {
		document := searchDocument("doc-"+string(rune('a'+sequence-1)), "context", ContentClassProse, fixtureTimeValue(sequence))
		document.EventSequence = int64(sequence)
		document.SourceEventID = document.DocumentID + "-event"
		document.SourceMessageID = document.DocumentID + "-message"
		document.Visible = sequence != 2
		require.NoError(t, repository.UpsertDocument(context.Background(), document))
	}

	documents, err := repository.ContextDocuments(context.Background(), "run-1", 4, 2, 1)
	require.NoError(t, err)
	require.Len(t, documents, 4)
	require.Equal(t, []int64{1, 3, 4, 5}, []int64{documents[0].EventSequence, documents[1].EventSequence, documents[2].EventSequence, documents[3].EventSequence})
	_, err = repository.ContextDocuments(context.Background(), "run-1", 4, 21, 0)
	require.ErrorContains(t, err, "bounds")
}

func TestLexicalSearchPersistsAcrossDatabaseRestartWithoutSemanticResources(t *testing.T) {
	t.Parallel()
	dsn := "file:" + filepath.Join(t.TempDir(), "conversation-search.db")
	db, err := sqlx.Connect("sqlite", dsn)
	require.NoError(t, err)
	require.NoError(t, coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)))
	repository := NewSQLiteRepository(db)
	insertSearchDocument(t, repository, "doc-persistent", "durable lexical floor", ContentClassProse, fixtureTimeValue(1))
	require.NoError(t, db.Close())

	db, err = sqlx.Connect("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	repository = NewSQLiteRepository(db)
	candidates, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: "durable floor", Limit: 10})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "doc-persistent", candidates[0].Document.DocumentID)
}

func BenchmarkLexicalCandidatesRepresentativeScale(b *testing.B) {
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("file:conversation-search-benchmark-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, db.Close()) })
	require.NoError(b, coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)))
	repository := NewSQLiteRepository(db)
	for index := 0; index < 30_000; index++ {
		content := fmt.Sprintf("routine conversation record %d", index)
		if index%100 == 0 {
			content += " test-genie system monitor capacity correction"
		}
		if index == 0 {
			content += strings.Repeat(" maximum-size-message", ((256<<10)-len(content))/len(" maximum-size-message"))
		}
		document := searchDocument(fmt.Sprintf("benchmark-%05d", index), content, ContentClassProse, fixtureTimeValue(index%59))
		document.SourceEventID = fmt.Sprintf("event-%05d", index)
		document.SourceMessageID = fmt.Sprintf("message-%05d", index)
		document.Harness = []string{"codex", "claude-code"}[index%2]
		document.ProjectScope = []string{"vrooli", "fixture-project"}[index%2]
		document.Model = []string{"gpt-fixture", "claude-fixture"}[index%2]
		require.NoError(b, repository.UpsertDocument(context.Background(), document))
	}

	b.Run("common_identifier_query", func(b *testing.B) {
		for b.Loop() {
			candidates, queryErr := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: "test-genie capacity", Limit: 26})
			require.NoError(b, queryErr)
			require.Len(b, candidates, 26)
		}
	})
	b.Run("punctuation_heavy_miss", func(b *testing.B) {
		for b.Loop() {
			candidates, queryErr := repository.LexicalCandidates(context.Background(), CandidateQuery{Query: `"missing-scenario/v2" OR (nothing)`, Limit: 26})
			require.NoError(b, queryErr)
			require.Empty(b, candidates)
		}
	})
}

func insertSearchDocument(t testing.TB, repository *SQLiteRepository, id, content string, class ContentClass, occurredAt time.Time) {
	t.Helper()
	require.NoError(t, repository.UpsertDocument(context.Background(), searchDocument(id, content, class, occurredAt)))
}

func searchDocument(id, content string, class ContentClass, occurredAt time.Time) Document {
	document := testDocument()
	document.DocumentID = id
	document.SourceEventID = id + "-event"
	document.SourceMessageID = id + "-message"
	document.Content = content
	document.ContentClass = class
	document.ContentHash = digest(content)
	document.OccurredAt = occurredAt
	document.IndexedAt = occurredAt
	return document
}

func fixtureTimeValue(minute int) time.Time {
	return time.Date(2026, 9, 4, 12, minute, 0, 0, time.UTC)
}
