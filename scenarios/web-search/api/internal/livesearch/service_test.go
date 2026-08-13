package livesearch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"web-search/internal/livesearch"

	"github.com/vrooli/api-core/scheduletest"
)

// fakeClient is a SearxngClient seam that records calls and returns canned raw
// results. callCount lets a test assert the upstream was (not) hit; delay
// simulates a normal upstream response time for latency-budget tests.
type fakeClient struct {
	results      []livesearch.RawResult
	engineIssues []livesearch.EngineIssue
	err          error
	delay        time.Duration
	callCount    int
	lastQuery    string
	lastLimit    int
}

func (f *fakeClient) Search(_ context.Context, query string, limit int) (livesearch.SearchPage, error) {
	f.callCount++
	f.lastQuery = query
	f.lastLimit = limit
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return livesearch.SearchPage{}, f.err
	}
	return livesearch.SearchPage{Results: f.results, UnresponsiveEngines: f.engineIssues}, nil
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
	clk := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
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
	clk := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
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
	clk := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
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
	clk := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
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

// TestServiceL0PathMakesNoLLMCalls pins the L0 contract end-to-end through the
// service: a plain (non-synthesize) query returns fully normalized results AND
// the synthesizer seam records zero invocations — the fast path involves no LLM
// even when one is wired.
func TestServiceL0PathMakesNoLLMCalls(t *testing.T) {
	client := &fakeClient{results: sampleRaw()}
	syn := &fakeSynthesizer{out: &livesearch.Synthesis{Text: "never"}}
	svc := livesearch.NewService(livesearch.Deps{Client: client, Synthesizer: syn})

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "anthropic claude", Limit: 5})
	require.NoError(t, err)
	require.Equal(t, 0, syn.callCount, "L0 path must make zero LLM calls")
	require.Nil(t, out.Synthesis)

	// The response is the normalized SearchHit shape, not raw SearXNG JSON.
	require.Len(t, out.Results, 2)
	require.Equal(t, "https://anthropic.com", out.Results[0].URL)
	require.Equal(t, "Anthropic", out.Results[0].Title)
	require.Equal(t, "Claude maker", out.Results[0].Snippet)
	require.Equal(t, "google", out.Results[0].Engine)
}

// TestServiceL0LatencyBudgetWithNormalUpstream is the bounded-budget guard for
// the L0 round-trip: given a normal upstream response time (simulated 10ms),
// the whole query+normalization path completes well inside the 2000ms budget —
// no hidden sleeps, retries, or accidental LLM work on the fast path.
func TestServiceL0LatencyBudgetWithNormalUpstream(t *testing.T) {
	client := &fakeClient{results: sampleRaw(), delay: 10 * time.Millisecond}
	svc := livesearch.NewService(livesearch.Deps{Client: client})

	start := time.Now()
	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "anthropic", Limit: 5})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, out.Results, 2)
	require.Less(t, elapsed, 2*time.Second, "L0 path must stay within the 2000ms latency budget given a normal upstream")
}

// TestServiceSynthesisLeavesRawHitsIdentical pins structural additivity at the
// strictest level: the raw hits returned WITH synthesis are deep-equal to the
// hits returned WITHOUT it — synthesis appends, it never mutates, reorders, or
// removes a raw result.
func TestServiceSynthesisLeavesRawHitsIdentical(t *testing.T) {
	syn := &fakeSynthesizer{out: &livesearch.Synthesis{
		Text:      "Anthropic makes Claude.",
		Citations: []livesearch.Citation{{ResultIndex: 0, URL: "https://anthropic.com", Title: "Anthropic"}},
	}}

	plain, err := livesearch.NewService(livesearch.Deps{Client: &fakeClient{results: sampleRaw()}}).
		Search(context.Background(), livesearch.SearchInput{Query: "q", Limit: 5})
	require.NoError(t, err)
	require.Nil(t, plain.Synthesis)

	synth, err := livesearch.NewService(livesearch.Deps{Client: &fakeClient{results: sampleRaw()}, Synthesizer: syn}).
		Search(context.Background(), livesearch.SearchInput{Query: "q", Limit: 5, Synthesize: true})
	require.NoError(t, err)
	require.NotNil(t, synth.Synthesis)

	require.Equal(t, plain.Results, synth.Results, "raw hits must be identical with and without synthesis")
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

// TestSearchThreadsDegradedEngines pins the degradation signal end-to-end at
// the service layer: a fresh search surfaces the client's unresponsive
// engines, and a cache hit replays the snapshot recorded at fetch time.
func TestSearchThreadsDegradedEngines(t *testing.T) {
	issues := []livesearch.EngineIssue{{Engine: "duckduckgo", Reason: "CAPTCHA"}}
	client := &fakeClient{
		results:      []livesearch.RawResult{{URL: "https://a.example", Title: "A"}},
		engineIssues: issues,
	}
	clk := scheduletest.New(time.Time{})
	svc := livesearch.NewService(livesearch.Deps{
		Client: client,
		Cache:  livesearch.NewCache(time.Minute, clk),
	})

	out, err := svc.Search(context.Background(), livesearch.SearchInput{Query: "q", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(out.DegradedEngines) != 1 || out.DegradedEngines[0].Engine != "duckduckgo" {
		t.Fatalf("expected degraded engines on fresh search, got %+v", out.DegradedEngines)
	}

	// Second call hits the cache; the engine snapshot rides along.
	out, err = svc.Search(context.Background(), livesearch.SearchInput{Query: "q", Limit: 5})
	if err != nil {
		t.Fatalf("cached Search: %v", err)
	}
	if !out.Cached {
		t.Fatal("expected cache hit on second search")
	}
	if len(out.DegradedEngines) != 1 || out.DegradedEngines[0].Reason != "CAPTCHA" {
		t.Fatalf("expected cached degraded engines, got %+v", out.DegradedEngines)
	}
	if client.callCount != 1 {
		t.Fatalf("expected single upstream call, got %d", client.callCount)
	}
}
