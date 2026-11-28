package conversationsearch

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/agenttranscript"
)

type coldCoverageStatus struct {
	StatusRepository
	calls atomic.Int32
}

func (s *coldCoverageStatus) CountCoverage(ctx context.Context) (uint64, uint64, uint64, error) {
	if s.calls.Add(1) == 1 {
		<-ctx.Done()
		return 0, 0, 0, ctx.Err()
	}
	return 1, 1, 1, nil
}

func TestSearchDoesNotBlockOnColdCoverageCount(t *testing.T) {
	t.Parallel()
	_, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "doc-fast", "bounded provider probe", ContentClassProse, fixtureTimeValue(1))
	status := &coldCoverageStatus{StatusRepository: repository}
	service, err := NewService(repository, repository, status, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	started := time.Now()
	response, err := service.SearchText(context.Background(), TextSearchRequest{Query: "bounded provider", PageSize: 5})
	require.NoError(t, err)
	require.Len(t, response.Hits, 1)
	require.Less(t, time.Since(started), time.Second)
	require.Eventually(t, func() bool { return status.calls.Load() >= 2 }, time.Second, 10*time.Millisecond)
}

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

func TestNativeSequenceSearchCarriesSourceOrderAcrossSignedCursor(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	for _, item := range []struct {
		id            string
		native, event int64
	}{{"source-8", 8, 80}, {"source-3b", 3, 30}, {"source-3a", 3, 10}} {
		document := searchDocument(item.id, "phase native record", ContentClassProse, fixtureTimeValue(1))
		document.NativeRecordSequence, document.EventSequence = item.native, item.event
		require.NoError(t, repository.UpsertDocument(context.Background(), document))
	}
	first, err := service.SearchText(context.Background(), TextSearchRequest{Query: "phase", Sort: SearchSortNativeSequence, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"source-3a", "source-3b"}, []string{first.Hits[0].Document.DocumentID, first.Hits[1].Document.DocumentID})
	require.EqualValues(t, 3, first.Hits[0].Document.NativeRecordSequence)
	require.NotEmpty(t, first.NextCursor)
	next, err := service.SearchText(context.Background(), TextSearchRequest{Query: "phase", Sort: SearchSortNativeSequence, PageSize: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Equal(t, []string{"source-8"}, []string{next.Hits[0].Document.DocumentID})
}

func TestAntigravitySearchContextAndResumeKeepNativeIdentityAndOrder(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	items := []struct {
		id, session   string
		native, event int64
	}{
		{"ag-step-9", "ag-thread-7", 9, 2},
		{"ag-step-4b", "ag-thread-7", 4, 8},
		{"ag-step-4a", "ag-thread-7", 4, 3},
	}
	for _, item := range items {
		document := searchDocument(item.id, "antigravity-native-context token", ContentClassProse, fixtureTimeValue(int(item.native)))
		document.Harness = "antigravity"
		document.NativeRecordSequence, document.EventSequence = item.native, item.event
		document.SourceSessionID = item.session
		require.NoError(t, repository.UpsertDocument(context.Background(), document))
	}
	search, err := service.SearchText(context.Background(), TextSearchRequest{Query: "antigravity-native-context", Sort: SearchSortNativeSequence, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"ag-step-4a", "ag-step-4b"}, []string{search.Hits[0].Document.DocumentID, search.Hits[1].Document.DocumentID})
	selected := search.Hits[0].Document
	_, window, err := service.Context(context.Background(), selected.DocumentID, 1, 2)
	require.NoError(t, err)
	require.Equal(t, []int64{4, 4, 9}, []int64{window[0].NativeRecordSequence, window[1].NativeRecordSequence, window[2].NativeRecordSequence})
	require.Equal(t, []int64{3, 8, 2}, []int64{window[0].EventSequence, window[1].EventSequence, window[2].EventSequence})
	require.Equal(t, selected.SourceSessionID, window[0].SourceSessionID)
	resume, err := agenttranscript.BuildNativeResumeInvocation(selected.Harness, selected.SourceSessionID, "continue selected native context")
	require.NoError(t, err)
	require.Equal(t, "agy", resume.Command)
	require.Equal(t, []string{"--conversation", selected.SourceSessionID, "--print", "continue selected native context"}, resume.Args)
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
