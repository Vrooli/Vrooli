package conversationsearch

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestTextAndRegexSearchContractTextPagingAndCursorIntegrity(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "doc-a", "phase capacity correction", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "doc-b", "phase capacity correction", ContentClassProse, fixtureTimeValue(2))
	insertSearchDocument(t, repository, "doc-c", "phase capacity correction", ContentClassProse, fixtureTimeValue(3))

	first, err := service.SearchText(context.Background(), TextSearchRequest{Query: "phase capacity", Sort: SearchSortNewest, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, first.Hits, 2)
	require.Equal(t, "doc-c", first.Hits[0].Document.DocumentID)
	require.NotEmpty(t, first.NextCursor)
	require.Equal(t, uint64(3), first.CanonicalVisibleMessages)
	require.Equal(t, first.CatalogDocuments, first.LexicalDocuments)

	second, err := service.SearchText(context.Background(), TextSearchRequest{Query: "phase capacity", Sort: SearchSortNewest, PageSize: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Hits, 1)
	require.Equal(t, "doc-a", second.Hits[0].Document.DocumentID)
	require.Empty(t, second.NextCursor)

	_, err = service.SearchText(context.Background(), TextSearchRequest{Query: "different", Sort: SearchSortNewest, PageSize: 2, Cursor: first.NextCursor})
	require.ErrorContains(t, err, "fingerprint")
	tampered := first.NextCursor[:len(first.NextCursor)-1] + "x"
	_, err = service.SearchText(context.Background(), TextSearchRequest{Query: "phase capacity", Sort: SearchSortNewest, PageSize: 2, Cursor: tampered})
	require.ErrorContains(t, err, "page_cursor")
}

func TestSearchHitProvenanceAndContextBounds(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	for sequence := 1; sequence <= 5; sequence++ {
		document := searchDocument("doc-"+string(rune('a'+sequence-1)), "context phase capacity", ContentClassProse, fixtureTimeValue(sequence))
		document.EventSequence = int64(sequence)
		document.SourceEventID = document.DocumentID + "-event"
		document.SourceMessageID = document.DocumentID + "-message"
		require.NoError(t, repository.UpsertDocument(context.Background(), document))
	}

	response, err := service.SearchText(context.Background(), TextSearchRequest{Query: "capacity", Sort: SearchSortNewest, PageSize: 1})
	require.NoError(t, err)
	hit := response.Hits[0]
	require.NotEmpty(t, hit.Document.EvidenceRef)
	require.NotEmpty(t, hit.Highlights)
	require.NotEmpty(t, hit.Document.Harness)
	require.NotEmpty(t, hit.Document.SourceSessionID)
	require.NotEmpty(t, hit.Document.ProjectScope)
	require.Equal(t, ContentClassProse, hit.Document.ContentClass)
	require.Equal(t, 1, hit.Rank)
	require.NotZero(t, hit.Score)
	require.Equal(t, "/runs/run-1?event=doc-e-event", hit.DeepLink)

	matched, contextDocuments, err := service.Context(context.Background(), "doc-c", 2, 1)
	require.NoError(t, err)
	require.Equal(t, "doc-c", matched.DocumentID)
	require.Equal(t, []int64{1, 2, 3, 4}, []int64{contextDocuments[0].EventSequence, contextDocuments[1].EventSequence, contextDocuments[2].EventSequence, contextDocuments[3].EventSequence})
}

func TestTextSearchBoundsUnicodeSnippetAndHighlights(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	content := strings.Repeat("préface 🚀 ", 300) + "phase capacity correction" + strings.Repeat(" suffix", 300)
	insertSearchDocument(t, repository, "doc-long", content, ContentClassProse, fixtureTimeValue(1))

	response, err := service.SearchText(context.Background(), TextSearchRequest{Query: "phase", PageSize: 1})
	require.NoError(t, err)
	require.Len(t, response.Hits, 1)
	require.LessOrEqual(t, len(response.Hits[0].Snippet), maximumSnippetBytes)
	require.True(t, utf8.ValidString(response.Hits[0].Snippet))
	require.NotEmpty(t, response.Hits[0].Highlights)

	_, err = service.SearchText(context.Background(), TextSearchRequest{Query: "phase", PageSize: maximumSearchPageSize + 1})
	require.ErrorContains(t, err, "page_size")
}

func TestTextSearchLabelsSingleGenericTermFromLongQueryWeak(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "doc-generic", "running the required recall pass", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "doc-specific", "nonsensical recall zxqv evidence", ContentClassProse, fixtureTimeValue(2))

	response, err := service.SearchText(context.Background(), TextSearchRequest{Query: "zxqv jklp 9981 nonsensical recall", PageSize: 10})
	require.NoError(t, err)
	require.Len(t, response.Hits, 1)
	require.Equal(t, "doc-specific", response.Hits[0].Document.DocumentID)
	require.False(t, response.Hits[0].Weak)
	require.True(t, weakLexicalCoverage("running the required recall pass", "zxqv jklp 9981 nonsensical recall"))
}

func TestTextSearchSupportsFilteredBrowsingWithoutQuery(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "doc-old", strings.Repeat("bounded browse content ", 150), ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "doc-new", "new browse content", ContentClassProse, fixtureTimeValue(2))

	response, err := service.SearchText(context.Background(), TextSearchRequest{
		Sort: SearchSortNewest, PageSize: 1,
		Filters: SearchFilters{Harnesses: []string{"claude-code"}},
	})
	require.NoError(t, err)
	require.Len(t, response.Hits, 1)
	require.Equal(t, "doc-new", response.Hits[0].Document.DocumentID)
	require.NotEmpty(t, response.NextCursor)
	require.Empty(t, response.Hits[0].Highlights)

	_, err = service.SearchText(context.Background(), TextSearchRequest{Sort: SearchSortNewest})
	require.ErrorIs(t, err, ErrInvalidRequest)
	_, err = service.SearchText(context.Background(), TextSearchRequest{Filters: SearchFilters{Harnesses: []string{"claude-code"}}})
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func searchFixtureService(t testing.TB) (*Service, *SQLiteRepository) {
	t.Helper()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	service, err := NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	return service, repository
}
