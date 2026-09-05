package livesearch

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// searxngFixture is a trimmed real-shape SearXNG JSON response: the documented
// {"results":[{url,title,content,engine,score,category}]} envelope.
const searxngFixture = `{
  "results": [
    {"url":"https://www.anthropic.com/claude","title":"Claude","content":"Anthropic's AI assistant.","engine":"google","score":1.5,"category":"general"},
    {"url":"https://docs.anthropic.com","title":"Docs","content":"API reference.","engine":"duckduckgo","score":0.8,"category":"general"}
  ]
}`

func TestNormalizationMapsSearxngFixture(t *testing.T) {
	var env searxngEnvelope
	require.NoError(t, json.Unmarshal([]byte(searxngFixture), &env))
	require.Len(t, env.Results, 2)

	got := normalizeAll(env.Results)
	require.Len(t, got, 2)

	require.Equal(t, "https://www.anthropic.com/claude", got[0].URL)
	require.Equal(t, "Claude", got[0].Title)
	// SearXNG "content" maps to the domain snippet.
	require.Equal(t, "Anthropic's AI assistant.", got[0].Snippet)
	require.Equal(t, "google", got[0].Engine)
	// Score carried RAW (no re-normalization), so >1.0 survives.
	require.InDelta(t, 1.5, got[0].Score, 1e-9)
	require.Equal(t, "general", got[0].Category)

	require.Equal(t, "duckduckgo", got[1].Engine)
	require.InDelta(t, 0.8, got[1].Score, 1e-9)
}

func TestNormalizeTrimsWhitespace(t *testing.T) {
	got := normalize(RawResult{URL: "  https://x.com  ", Title: " T ", Content: "  body  ", Engine: " e "})
	require.Equal(t, "https://x.com", got.URL)
	require.Equal(t, "T", got.Title)
	require.Equal(t, "body", got.Snippet)
	require.Equal(t, "e", got.Engine)
}

// TestNormalizeMissingOptionalFieldsDefaults pins the missing-optional-field
// contract: a SearXNG result that omits content/score/category (engines vary in
// what they populate) still decodes and normalizes into a valid Result with
// empty/zero defaults — never a panic, never a dropped hit.
func TestNormalizeMissingOptionalFieldsDefaults(t *testing.T) {
	const sparseFixture = `{
	  "results": [
	    {"url":"https://no-snippet.example","title":"Bare"},
	    {"url":"https://no-title.example","content":"only content","engine":"bing"}
	  ]
	}`
	var env searxngEnvelope
	require.NoError(t, json.Unmarshal([]byte(sparseFixture), &env))

	got := normalizeAll(env.Results)
	require.Len(t, got, 2)

	// No snippet/engine/score/category → empty/zero defaults, fields present.
	require.Equal(t, "https://no-snippet.example", got[0].URL)
	require.Equal(t, "Bare", got[0].Title)
	require.Empty(t, got[0].Snippet)
	require.Empty(t, got[0].Engine)
	require.Zero(t, got[0].Score)
	require.Empty(t, got[0].Category)

	// No title → still a valid hit with the populated fields carried.
	require.Equal(t, "https://no-title.example", got[1].URL)
	require.Empty(t, got[1].Title)
	require.Equal(t, "only content", got[1].Snippet)
	require.Equal(t, "bing", got[1].Engine)
}

// TestNormalizeAllDeduplicatesByURL pins the REQ-P0-001 dedup contract:
// SearXNG fans a query across engines, so the same page returns repeatedly;
// normalization keeps ONE hit per URL — the highest-scored occurrence (with
// its engine attribution) — at the position of the URL's first appearance.
func TestNormalizeAllDeduplicatesByURL(t *testing.T) {
	got := normalizeAll([]RawResult{
		{URL: "https://dup.example", Title: "Dup (low)", Engine: "bing", Score: 0.4},
		{URL: "https://unique.example", Title: "Unique", Engine: "google", Score: 0.9},
		{URL: "https://dup.example", Title: "Dup (high)", Engine: "google", Score: 1.2},
		{URL: " https://dup.example ", Title: "Dup (trimmed, mid)", Engine: "ddg", Score: 0.7},
	})
	require.Len(t, got, 2)

	// The duplicate keeps its first-appearance position but the
	// highest-scored occurrence's fields win (whitespace-trimmed URLs
	// count as the same page).
	require.Equal(t, "https://dup.example", got[0].URL)
	require.Equal(t, "Dup (high)", got[0].Title)
	require.Equal(t, "google", got[0].Engine)
	require.InDelta(t, 1.2, got[0].Score, 1e-9)

	require.Equal(t, "https://unique.example", got[1].URL)
}

// BenchmarkNormalizeAll20Results measures the pure-CPU normalization cost of a
// 20-result SearXNG response (the REQ-P0-001 performance budget is <10ms; the
// actual cost is a few microseconds of string trimming).
func BenchmarkNormalizeAll20Results(b *testing.B) {
	raw := make([]RawResult, 20)
	for i := range raw {
		raw[i] = RawResult{
			URL:      "  https://example.com/page/" + strconv.Itoa(i) + "  ",
			Title:    " Example Page Title ",
			Content:  "  A snippet of body text returned by the engine.  ",
			Engine:   " google ",
			Score:    1.25,
			Category: " general ",
		}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		out := normalizeAll(raw)
		if len(out) != 20 {
			b.Fatalf("expected 20 results, got %d", len(out))
		}
	}
}
