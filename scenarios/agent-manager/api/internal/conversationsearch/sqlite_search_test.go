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

func TestNativeSequenceSearchPersistsAndPagesBySourceStep(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	for _, item := range []struct {
		id            string
		native, event int64
	}{{"native-7", 7, 90}, {"native-2b", 2, 20}, {"native-2a", 2, 10}} {
		document := testDocument()
		document.DocumentID, document.SourceEventID = item.id, item.id+"-event"
		document.NativeRecordSequence, document.EventSequence = item.native, item.event
		require.NoError(t, repository.UpsertDocument(context.Background(), document))
	}
	first, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Limit: 2, Sort: SearchSortNativeSequence})
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.Equal(t, "native-2a", first[0].Document.DocumentID)
	require.EqualValues(t, 2, first[0].Document.NativeRecordSequence)
	require.Equal(t, "native-2b", first[1].Document.DocumentID)
	next, err := repository.LexicalCandidates(context.Background(), CandidateQuery{Limit: 2, Sort: SearchSortNativeSequence, After: &CandidateCursor{NativeRecordSequence: first[1].Document.NativeRecordSequence, EventSequence: first[1].Document.EventSequence, DocumentID: first[1].Document.DocumentID}})
	require.NoError(t, err)
	require.Len(t, next, 1)
	require.Equal(t, "native-7", next[0].Document.DocumentID)
	require.Equal(t, int64(7), next[0].Document.NativeRecordSequence)
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
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM conversation_search_catalog`))
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

	hit := searchDocument("doc-d", "context", ContentClassProse, fixtureTimeValue(4))
	hit.EventSequence = 4
	documents, err := repository.ContextDocuments(context.Background(), hit, 2, 1)
	require.NoError(t, err)
	require.Len(t, documents, 4)
	require.Equal(t, []int64{1, 3, 4, 5}, []int64{documents[0].EventSequence, documents[1].EventSequence, documents[2].EventSequence, documents[3].EventSequence})
	_, err = repository.ContextDocuments(context.Background(), hit, 21, 0)
	require.ErrorContains(t, err, "bounds")
}

func TestNativeContextDocumentsUsesSourceStepWithEventAndIdentityPreserved(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	items := []struct {
		id, session   string
		native, event int64
	}{
		{"step-9", "thread-native", 9, 2},
		{"step-4b", "thread-native", 4, 8},
		{"step-4a", "thread-native", 4, 3},
	}
	for _, item := range items {
		document := searchDocument(item.id, item.id+" sanitized body", ContentClassProse, fixtureTimeValue(int(item.native)))
		document.Harness = "antigravity"
		document.NativeRecordSequence, document.EventSequence = item.native, item.event
		document.SourceSessionID = item.session
		require.NoError(t, repository.UpsertDocument(context.Background(), document))
	}
	hit := searchDocument("step-4a", "step-4a sanitized body", ContentClassProse, fixtureTimeValue(4))
	hit.Harness, hit.EventSequence, hit.NativeRecordSequence = "antigravity", 3, 4
	hit.SourceSessionID = "thread-native"
	window, err := repository.ContextDocuments(context.Background(), hit, 0, 2)
	require.NoError(t, err)
	require.Equal(t, []string{"step-4a", "step-4b", "step-9"}, []string{window[0].DocumentID, window[1].DocumentID, window[2].DocumentID})
	require.Equal(t, []int64{4, 4, 9}, []int64{window[0].NativeRecordSequence, window[1].NativeRecordSequence, window[2].NativeRecordSequence})
	require.Equal(t, []int64{3, 8, 2}, []int64{window[0].EventSequence, window[1].EventSequence, window[2].EventSequence})
	require.Equal(t, []string{"thread-native", "thread-native", "thread-native"}, []string{window[0].SourceSessionID, window[1].SourceSessionID, window[2].SourceSessionID})
	require.Equal(t, []time.Time{fixtureTimeValue(4), fixtureTimeValue(4), fixtureTimeValue(9)}, []time.Time{window[0].OccurredAt, window[1].OccurredAt, window[2].OccurredAt})
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
