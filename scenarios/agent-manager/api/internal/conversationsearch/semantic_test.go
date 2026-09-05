package conversationsearch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
)

type semanticRetrieverStub struct {
	candidates []SemanticCandidate
	err        error
}

type blockingSemanticRetriever struct{}

func (blockingSemanticRetriever) SearchSemantic(ctx context.Context, _ SemanticSearchRequest) ([]SemanticCandidate, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type cancellationIgnoringSemanticRetriever struct{ release <-chan struct{} }

func (r cancellationIgnoringSemanticRetriever) SearchSemantic(context.Context, SemanticSearchRequest) ([]SemanticCandidate, error) {
	<-r.release
	return nil, nil
}

func (s semanticRetrieverStub) SearchSemantic(context.Context, SemanticSearchRequest) ([]SemanticCandidate, error) {
	return s.candidates, s.err
}

func TestHybridFusionDeduplicatesStableIDsAndPreservesLegEvidence(t *testing.T) {
	t.Parallel()
	base, repository := searchFixtureService(t)
	docA := searchDocument("doc-a", "adaptive phase scheduling", ContentClassProse, fixtureTimeValue(1))
	docB := searchDocument("doc-b", "measured history controls admission", ContentClassProse, fixtureTimeValue(2))
	require.NoError(t, repository.UpsertDocument(context.Background(), docA))
	require.NoError(t, repository.UpsertDocument(context.Background(), docB))

	service, err := NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"), WithSemanticRetriever(semanticRetrieverStub{candidates: []SemanticCandidate{
		{Document: docA, Score: 0.91, Evidence: []RankEvidence{{Leg: SearchLegDense, Rank: 1, Score: 0.91}}},
		{Document: docB, Score: 0.84, Evidence: []RankEvidence{{Leg: SearchLegDense, Rank: 2, Score: 0.84}}},
	}}))
	require.NoError(t, err)
	_ = base

	response, err := service.SearchHybrid(context.Background(), TextSearchRequest{Query: "adaptive phase", PageSize: 10})
	require.NoError(t, err)
	require.Len(t, response.Hits, 2)
	byID := map[string]SearchHit{}
	for _, hit := range response.Hits {
		byID[hit.Document.DocumentID] = hit
	}
	require.Len(t, byID["doc-a"].Evidence, 2, "duplicate lexical/semantic identity must retain both contributions")
	require.Equal(t, SearchLegLexical, byID["doc-a"].Evidence[0].Leg)
	require.Equal(t, SearchLegDense, byID["doc-a"].Evidence[1].Leg)
	require.Len(t, byID["doc-b"].Evidence, 1)
	require.Empty(t, response.Degradations)
}

func TestHybridKeepsLexicalResultsForEveryOptionalFailureClass(t *testing.T) {
	t.Parallel()
	failures := []struct {
		name   string
		err    error
		reason DegradationReason
	}{
		{"collection mismatch", ErrIndexLayoutMismatch, DegradationIndexLayoutMismatch},
		{"qdrant outage", ErrVectorStoreUnavailable, DegradationVectorStoreUnavailable},
		{"embedder outage", ErrEmbeddingUnavailable, DegradationEmbeddingUnavailable},
		{"reranker outage", ErrRerankUnavailable, DegradationRerankUnavailable},
		{"partial timeout", context.DeadlineExceeded, DegradationDeadline},
	}
	for _, test := range failures {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, repository := searchFixtureService(t)
			insertSearchDocument(t, repository, "doc-lexical", "phase scheduling correction", ContentClassProse, fixtureTimeValue(1))
			service, err := NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"), WithSemanticRetriever(semanticRetrieverStub{err: test.err}))
			require.NoError(t, err)
			response, err := service.SearchHybrid(context.Background(), TextSearchRequest{Query: "phase correction", PageSize: 10})
			require.NoError(t, err)
			require.Len(t, response.Hits, 1)
			require.Len(t, response.Degradations, 1)
			require.Equal(t, test.reason, response.Degradations[0].Reason)
		})
	}
}

func TestHybridBoundsSemanticLegIndependently(t *testing.T) {
	t.Parallel()
	_, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "doc-lexical", "phase scheduling correction", ContentClassProse, fixtureTimeValue(1))
	service, err := NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"), WithSemanticRetriever(blockingSemanticRetriever{}))
	require.NoError(t, err)
	started := time.Now()
	response, err := service.SearchHybrid(context.Background(), TextSearchRequest{Query: "phase correction", PageSize: 10})
	require.NoError(t, err)
	require.Less(t, time.Since(started), 2*time.Second)
	require.Len(t, response.Hits, 1)
	require.Equal(t, DegradationDeadline, response.Degradations[0].Reason)
}

func TestHybridDeadlineDoesNotTrustOptionalRetrieverCancellation(t *testing.T) {
	t.Parallel()
	_, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "doc-lexical", "phase scheduling correction", ContentClassProse, fixtureTimeValue(1))
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	service, err := NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"), WithSemanticRetriever(cancellationIgnoringSemanticRetriever{release: release}))
	require.NoError(t, err)
	started := time.Now()
	response, err := service.SearchHybrid(context.Background(), TextSearchRequest{Query: "phase correction", PageSize: 10})
	require.NoError(t, err)
	require.Less(t, time.Since(started), 2*time.Second)
	require.Len(t, response.Hits, 1)
	require.Equal(t, DegradationDeadline, response.Degradations[0].Reason)
}

type sharedEngineStub struct {
	response aisearch.SearchResponse
	err      error
	query    aisearch.SearchQuery
}

func (s *sharedEngineStub) Search(_ context.Context, query aisearch.SearchQuery, _ ...aisearch.SearchOption) (aisearch.SearchResponse, error) {
	s.query = query
	return s.response, s.err
}

func TestSharedSemanticRetrieverPushesFiltersAndRechecksProjection(t *testing.T) {
	t.Parallel()
	_, repository := searchFixtureService(t)
	document := searchDocument("doc-filtered", "semantic passage", ContentClassProse, fixtureTimeValue(2))
	document.ProjectScope = "/workspace/alpha"
	require.NoError(t, repository.UpsertDocument(context.Background(), document))
	engine := &sharedEngineStub{response: aisearch.SearchResponse{Method: "hybrid", Reranker: "none", Results: []aisearch.SearchResult{{Score: 0.8, SourceID: document.DocumentID}}}}
	retriever, err := NewSharedSemanticRetriever(engine, repository, aisearch.NewWeightedAdmission(2))
	require.NoError(t, err)
	after := document.OccurredAt.Add(-time.Second)
	results, err := retriever.SearchSemantic(context.Background(), SemanticSearchRequest{Query: "meaning", Limit: 5, Filters: SearchFilters{ProjectScopes: []string{"/workspace/alpha"}, OccurredAfter: &after}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, aisearch.ModeDense, engine.query.Mode, "the outer conversation service owns hybrid fusion")
	require.NotNil(t, engine.query.Filter)
	require.NotEmpty(t, engine.query.Filter.Must)
	require.NotEmpty(t, engine.query.Filter.Ranges)
	require.Equal(t, []SearchLeg{SearchLegDense, SearchLegSparse}, []SearchLeg{results[0].Evidence[0].Leg, results[0].Evidence[1].Leg})

	engine.response.Results = []aisearch.SearchResult{{Score: 0.8, SourceID: document.DocumentID}}
	results, err = retriever.SearchSemantic(context.Background(), SemanticSearchRequest{Query: "meaning", Limit: 5, Filters: SearchFilters{ProjectScopes: []string{"/workspace/beta"}}})
	require.NoError(t, err)
	require.Empty(t, results, "the authoritative SQLite projection must reject an out-of-scope vector hit")
}

func TestSharedSemanticRetrieverCalibratesWeakDenseCandidates(t *testing.T) {
	t.Parallel()
	_, repository := searchFixtureService(t)
	weakDocument := searchDocument("doc-weak", "background nearest neighbor", ContentClassProse, fixtureTimeValue(1))
	strongDocument := searchDocument("doc-strong", "meaningful semantic match", ContentClassProse, fixtureTimeValue(2))
	require.NoError(t, repository.UpsertDocument(context.Background(), weakDocument))
	require.NoError(t, repository.UpsertDocument(context.Background(), strongDocument))
	engine := &sharedEngineStub{response: aisearch.SearchResponse{Method: "dense", Results: []aisearch.SearchResult{
		{Score: conversationDenseStrongThreshold - 0.01, SourceID: weakDocument.DocumentID},
		{Score: conversationDenseStrongThreshold, SourceID: strongDocument.DocumentID},
	}}}
	retriever, err := NewSharedSemanticRetriever(engine, repository, aisearch.NewWeightedAdmission(2))
	require.NoError(t, err)

	results, err := retriever.SearchSemantic(context.Background(), SemanticSearchRequest{Query: "semantic intent", Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.True(t, results[0].Weak)
	require.False(t, results[1].Weak)

	engine.response.Results[1].Weak = true
	results, err = retriever.SearchSemantic(context.Background(), SemanticSearchRequest{Query: "semantic intent", Limit: 5})
	require.NoError(t, err)
	require.True(t, results[1].Weak, "an upstream weak signal must remain weak above the local threshold")
}

func TestSharedSearchFakesResolveSemanticParaphrase(t *testing.T) {
	t.Parallel()
	_, repository := searchFixtureService(t)
	document := searchDocument("doc-correction", "phase admission is adaptive and uses measured resource history", ContentClassProse, fixtureTimeValue(1))
	require.NoError(t, repository.UpsertDocument(context.Background(), document))
	store := searchtest.NewVectorStore()
	store.QueryResults = []aisearch.SearchResult{{ID: document.DocumentID, Score: 0.88, Payload: map[string]any{"source_id": document.DocumentID, "body": document.Content}}}
	engine := aisearch.NewService(aisearch.ServiceOptions{
		Embedder:      &searchtest.Embedder{Vector: []float64{0.1, 0.2, 0.3}, AvailableValue: true},
		SparseEncoder: aisearch.NewBM25SparseEncoder(), VectorStore: store, ApplyFloor: false, MaxLimit: 100,
		Project: func(result aisearch.SearchResult) aisearch.SearchResult {
			result.SourceID, _ = result.Payload["source_id"].(string)
			return result
		},
	})
	retriever, err := NewSharedSemanticRetriever(engine, repository, aisearch.NewWeightedAdmission(2))
	require.NoError(t, err)
	results, err := retriever.SearchSemantic(context.Background(), SemanticSearchRequest{Query: "the correction where scheduling stopped being a fixed ceiling", Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, document.DocumentID, results[0].Document.DocumentID)
	require.Equal(t, []SearchLeg{SearchLegDense}, []SearchLeg{results[0].Evidence[0].Leg})
}

func TestSemanticSourceIsPagedAndEmbeddingTextUsesRestrainedContext(t *testing.T) {
	t.Parallel()
	document := searchDocument("doc-source", "raw conversation content", ContentClassProse, fixtureTimeValue(1))
	document.Role = "assistant"
	document.RunLabel = "Admission review"
	document.ProjectScope = "/workspace/alpha"
	doc := semanticSourceDoc(document)
	chunk := aisearch.Chunk{Body: doc.Body, Meta: doc.Meta}
	text := (conversationEmbeddingComposer{}).Compose(chunk)
	require.Contains(t, text, "assistant · Admission review · /workspace/alpha")
	require.Contains(t, text, document.Content)
	require.NotContains(t, text, document.SourceSessionID)
	require.NotContains(t, text, document.SourceEventID)
}

func TestEmbeddingComposerStaysBelowModelContextByteBudget(t *testing.T) {
	t.Parallel()
	text := (conversationEmbeddingComposer{}).Compose(aisearch.Chunk{
		Body: strings.Repeat("x", DefaultMaxChunkBytes),
		Meta: map[string]any{
			"role":          strings.Repeat("r", 1024),
			"run_label":     strings.Repeat("l", 1024),
			"project_scope": strings.Repeat("p", 1024),
		},
	})
	require.LessOrEqual(t, len(text), maxEmbeddingInputBytes)
	require.Contains(t, text, strings.Repeat("r", maxEmbeddingContextFieldBytes))
	require.Contains(t, text, strings.Repeat("x", 1024), "conversation content must remain in the bounded embedding input")
}

func TestSemanticSourceBoundsNormalizedDocumentsAndResumesWithinSourcePage(t *testing.T) {
	t.Parallel()
	documents := []Document{
		searchDocument("doc-1", "first", ContentClassProse, fixtureTimeValue(1)),
		searchDocument("tool-noise", "large tool result", ContentClassToolResult, fixtureTimeValue(1)),
		searchDocument("doc-2", "second", ContentClassProse, fixtureTimeValue(1)),
		searchDocument("doc-3", "third", ContentClassProse, fixtureTimeValue(1)),
	}
	source := NewSemanticSource(overflowingSourceRepository{documents: documents})

	first, err := source.LoadPage(context.Background(), aisearch.PageRequest{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Documents, 2)
	require.False(t, first.Done)
	require.NotEmpty(t, first.NextCursor)
	require.Equal(t, []string{"doc-1", "doc-2"}, []string{first.Documents[0].ID, first.Documents[1].ID})

	second, err := source.LoadPage(context.Background(), aisearch.PageRequest{Cursor: first.NextCursor, Limit: 2})
	require.NoError(t, err)
	require.Len(t, second.Documents, 1)
	require.True(t, second.Done)
	require.Empty(t, second.NextCursor)
	require.Equal(t, "doc-3", second.Documents[0].ID)
}

func TestStagedSemanticSourceReadsOnlyImmutableEligibleGeneration(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	generation := "semantic-snapshot"
	require.NoError(t, repository.BeginStagedGeneration(context.Background(), generation))
	first := testDocument()
	first.DocumentID = "a-prose"
	first.ContentClass = ContentClassProse
	second := testDocument()
	second.DocumentID = "b-tool"
	second.ContentClass = ContentClassToolResult
	third := testDocument()
	third.DocumentID = "c-quoted"
	third.ContentClass = ContentClassQuotedProse
	for _, document := range []Document{first, second, third} {
		require.NoError(t, repository.StageDocument(context.Background(), generation, document))
	}

	source := NewStagedSemanticSource(repository, generation)
	page, err := source.LoadPage(context.Background(), aisearch.PageRequest{Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Documents, 1)
	require.Equal(t, first.DocumentID, page.Documents[0].ID)
	require.False(t, page.Done)
	next, err := source.LoadPage(context.Background(), aisearch.PageRequest{Cursor: page.NextCursor, Limit: 1})
	require.NoError(t, err)
	require.Len(t, next.Documents, 1)
	require.Equal(t, third.DocumentID, next.Documents[0].ID)
	require.True(t, next.Done)
}

func TestAgentManagerSearchJSONIsStrictAndHybrid(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", ".vrooli", "search.json")
	file, err := aisearch.LoadSearchFile(path)
	require.NoError(t, err)
	provider, ok := file.Provider(ConversationSearchProviderID)
	require.True(t, ok)
	require.Equal(t, aisearch.EngineHybrid, provider.ResolvedTuning().Engine)
	require.Equal(t, aisearch.ClassLocalIndex, provider.Class)
	require.GreaterOrEqual(t, len(provider.Tests.Cases), 8)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	raw = append(raw[:len(raw)-2], []byte(",\"unknown_contract_field\":true}]}")...)
	_, err = aisearch.ParseSearchFile(raw)
	require.Error(t, err)
}

func TestClassifySemanticError(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, classifySemanticError(errors.New("qdrant connection refused")), ErrVectorStoreUnavailable)
	require.ErrorIs(t, classifySemanticError(errors.New("embedding policy unavailable")), ErrEmbeddingUnavailable)
}

func TestLiveSemanticResourcesWhenExplicitlyEnabled(t *testing.T) {
	if os.Getenv("AGENT_MANAGER_LIVE_SEARCH_TEST") != "1" {
		t.Skip("set AGENT_MANAGER_LIVE_SEARCH_TEST=1 under the scenario lifecycle to exercise Ollama and Qdrant")
	}
	_, repository := searchFixtureService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runtime, err := BuildSemanticRuntime(ctx, SemanticRuntimeOptions{
		SearchFilePath: filepath.Join("..", "..", "..", ".vrooli", "search.json"),
		Source:         sourceRepositoryStub{}, Projection: repository,
	})
	require.NoError(t, err)
	require.NoError(t, runtime.InitializationError)
	require.NotEmpty(t, runtime.Collection)
	require.NotEmpty(t, runtime.EmbeddingModel)
}

type sourceRepositoryStub struct{}

func (sourceRepositoryStub) LoadSourcePage(context.Context, *SourceCursor, int) (SourcePage, error) {
	return SourcePage{}, nil
}

type overflowingSourceRepository struct{ documents []Document }

func (s overflowingSourceRepository) LoadSourcePage(context.Context, *SourceCursor, int) (SourcePage, error) {
	return SourcePage{Documents: s.documents}, nil
}
