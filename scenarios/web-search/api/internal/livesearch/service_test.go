package livesearch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"web-search/internal/livesearch"
	"web-search/internal/testutil/mocks"
)

// fakeClient is a SearxngClient seam that records calls and returns canned raw
// results. callCount lets a test assert the upstream was (not) hit.
type fakeClient struct {
	results   []livesearch.RawResult
	err       error
	callCount int
	lastQuery string
	lastLimit int
}

func (f *fakeClient) Search(_ context.Context, query string, limit int) ([]livesearch.RawResult, error) {
	f.callCount++
	f.lastQuery = query
	f.lastLimit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

// fakeSynthesizer is a Synthesizer seam returning a canned synthesis.
type fakeSynthesizer struct {
	out       *livesearch.Synthesis
	err       error
	callCount int
}

func (f *fakeSynthesizer) Synthesize(_ context.Context, _ string, _ []livesearch.Result) (*livesearch.Synthesis, error) {
	f.callCount++
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func sampleRaw() []livesearch.RawResult {
	return []livesearch.RawResult{
		{URL: "https://anthropic.com", Title: "Anthropic", Content: "Claude maker", Engine: "google", Score: 0.9, Category: "general"},
		{URL: "https://example.com", Title: "Example", Content: "Other", Engine: "bing", Score: 0.4, Category: "general"},
	}
}

func TestServiceNormalizesSearxngResults(t *testing.T) {
	client := &fakeClient{results: sampleRaw()}
	svc := livesearch.NewService(livesearch.Deps{Client: client})

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "anthropic claude", Limit: 5})
	require.NoError(t, err)
	require.False(t, out.Degraded)
	require.False(t, out.Cached)
	require.Len(t, out.Results, 2)

	// content -> snippet, score carried raw, all fields mapped.
	first := out.Results[0]
	require.Equal(t, "https://anthropic.com", first.URL)
	require.Equal(t, "Anthropic", first.Title)
	require.Equal(t, "Claude maker", first.Snippet)
	require.Equal(t, "google", first.Engine)
	require.InDelta(t, 0.9, first.Score, 1e-9)
	require.Equal(t, "general", first.Category)
	require.Equal(t, "anthropic claude", client.lastQuery)
	require.Equal(t, 5, client.lastLimit)
}

func TestServiceDefaultsLimit(t *testing.T) {
	client := &fakeClient{results: sampleRaw()}
	svc := livesearch.NewService(livesearch.Deps{Client: client})
	_, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q"})
	require.NoError(t, err)
	require.Equal(t, livesearch.DefaultLimit, client.lastLimit)
}

func TestServiceCacheHitSkipsClient(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	client := &fakeClient{results: sampleRaw()}
	cache := livesearch.NewCache(time.Minute, clk)
	svc := livesearch.NewService(livesearch.Deps{Client: client, Cache: cache})

	// First call populates the cache.
	_, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "Anthropic", Limit: 5})
	require.NoError(t, err)
	require.Equal(t, 1, client.callCount)

	// Within the TTL, a case/whitespace-equivalent query serves from cache.
	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "  anthropic  ", Limit: 5})
	require.NoError(t, err)
	require.True(t, out.Cached)
	require.Equal(t, 1, client.callCount, "cache hit must not call searxng")
}

func TestServiceCacheExpiryRefetches(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	client := &fakeClient{results: sampleRaw()}
	cache := livesearch.NewCache(time.Minute, clk)
	svc := livesearch.NewService(livesearch.Deps{Client: client, Cache: cache})

	_, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q", Limit: 5})
	require.NoError(t, err)
	require.Equal(t, 1, client.callCount)

	// Advance past the TTL: the entry expires and the client is hit again.
	clk.Advance(2 * time.Minute)
	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q", Limit: 5})
	require.NoError(t, err)
	require.False(t, out.Cached)
	require.Equal(t, 2, client.callCount, "expired cache must refetch")
}

func TestServiceGovernorExhaustionDegradesWithoutClient(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	client := &fakeClient{results: sampleRaw()}
	// Capacity 1 per window: the second call in the same window is declined.
	gov := livesearch.NewGovernor(1, time.Minute, clk)
	svc := livesearch.NewService(livesearch.Deps{Client: client, Governor: gov})

	_, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "a", Limit: 5})
	require.NoError(t, err)
	require.Equal(t, 1, client.callCount)

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "b", Limit: 5})
	require.NoError(t, err)
	require.True(t, out.Degraded)
	require.NotEmpty(t, out.DegradedReason)
	require.Empty(t, out.Results)
	require.Equal(t, 1, client.callCount, "budget exhaustion must NOT call searxng")
}

func TestServiceGovernorWindowRefill(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	client := &fakeClient{results: sampleRaw()}
	gov := livesearch.NewGovernor(1, time.Minute, clk)
	svc := livesearch.NewService(livesearch.Deps{Client: client, Governor: gov})

	_, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "a", Limit: 5})
	require.NoError(t, err)

	// New window refills the bucket.
	clk.Advance(time.Minute)
	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "c", Limit: 5})
	require.NoError(t, err)
	require.False(t, out.Degraded)
	require.Equal(t, 2, client.callCount)
}

func TestServiceClientErrorSurfaces(t *testing.T) {
	client := &fakeClient{err: errors.New("searxng down")}
	svc := livesearch.NewService(livesearch.Deps{Client: client})
	_, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q"})
	require.Error(t, err)
}

func TestServiceSynthesisOffByDefault(t *testing.T) {
	client := &fakeClient{results: sampleRaw()}
	syn := &fakeSynthesizer{out: &livesearch.Synthesis{Text: "x", Citations: []livesearch.Citation{{ResultIndex: 0}}}}
	svc := livesearch.NewService(livesearch.Deps{Client: client, Synthesizer: syn})

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q", Synthesize: false})
	require.NoError(t, err)
	require.Nil(t, out.Synthesis)
	require.Equal(t, 0, syn.callCount, "synthesis must not run unless requested")
}

func TestServiceSynthesisCitedWhenRequested(t *testing.T) {
	client := &fakeClient{results: sampleRaw()}
	syn := &fakeSynthesizer{out: &livesearch.Synthesis{
		Text:      "Anthropic makes Claude.",
		Citations: []livesearch.Citation{{ResultIndex: 0, URL: "https://anthropic.com", Title: "Anthropic"}},
	}}
	svc := livesearch.NewService(livesearch.Deps{Client: client, Synthesizer: syn})

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "who makes claude", Synthesize: true})
	require.NoError(t, err)
	require.NotNil(t, out.Synthesis)
	require.False(t, out.Synthesis.Abstained)
	require.Equal(t, "Anthropic makes Claude.", out.Synthesis.Text)
	require.Len(t, out.Synthesis.Citations, 1)
	require.Equal(t, 0, out.Synthesis.Citations[0].ResultIndex)
	// Raw results still present — synthesis is additive.
	require.Len(t, out.Results, 2)
}

func TestServiceSynthesisFailureDoesNotBlockResults(t *testing.T) {
	client := &fakeClient{results: sampleRaw()}
	syn := &fakeSynthesizer{err: errors.New("ollama down")}
	svc := livesearch.NewService(livesearch.Deps{Client: client, Synthesizer: syn})

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q", Synthesize: true})
	require.NoError(t, err, "synthesis failure must not fail the search")
	require.Nil(t, out.Synthesis)
	require.Len(t, out.Results, 2)
}
