package livesearch

import (
	"encoding/json"
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
