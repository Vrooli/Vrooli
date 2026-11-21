package livesearch_test

import (
	"context"
	"net/http"
	"testing"

	"web-search/internal/livesearch"
	"web-search/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// TestHTTPSearxngClientParsesUnresponsiveEngines pins the engine-degradation
// signal: unresponsive_engines rides every SearXNG JSON envelope and was
// previously dropped on the floor (users saw mysteriously bad results with no
// explanation).
func TestHTTPSearxngClientParsesUnresponsiveEngines(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusOK, []byte(`{
		"results": [{"url": "https://a.example", "title": "A", "content": "snippet", "engine": "google", "score": 1.5}],
		"unresponsive_engines": [["duckduckgo", "CAPTCHA"], ["qwant", "parsing error"]]
	}`))

	client := livesearch.NewHTTPSearxngClient("http://searxng.local", doer)
	page, err := client.Search(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	require.Equal(t, []livesearch.EngineIssue{
		{Engine: "duckduckgo", Reason: "CAPTCHA"},
		{Engine: "qwant", Reason: "parsing error"},
	}, page.UnresponsiveEngines)
}

func TestHTTPSearxngClientNoUnresponsiveEnginesIsNil(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusOK, []byte(`{"results": [], "unresponsive_engines": []}`))

	client := livesearch.NewHTTPSearxngClient("http://searxng.local", doer)
	page, err := client.Search(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Nil(t, page.UnresponsiveEngines)
}

func TestHTTPSearxngClientToleratesMalformedUnresponsiveEntries(t *testing.T) {
	doer := &mocks.FakeDoer{}
	// Mixed shapes: a bare pair, a triple (newer searxng adds a suspended
	// flag), an empty entry, and a non-string engine.
	doer.AddResponse(http.StatusOK, []byte(`{
		"results": [],
		"unresponsive_engines": [["google", "Suspended", true], [], [42, "weird"], ["brave", "rate limited"]]
	}`))

	client := livesearch.NewHTTPSearxngClient("http://searxng.local", doer)
	page, err := client.Search(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Equal(t, []livesearch.EngineIssue{
		{Engine: "google", Reason: "Suspended"},
		{Engine: "brave", Reason: "rate limited"},
	}, page.UnresponsiveEngines)
}
