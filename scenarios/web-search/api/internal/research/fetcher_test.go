package research_test

import (
	"context"
	"net/http"
	"testing"

	"web-search/internal/research"
	"web-search/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// TestBrowserlessFetcherExtractsReadableText asserts the production fetcher
// POSTs to /chrome/content with the url and returns extracted readable text.
func TestBrowserlessFetcherExtractsReadableText(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusOK, []byte(`<html><body><nav>menu</nav><p>Real content here.</p></body></html>`))

	f := research.NewBrowserlessFetcher("http://browserless.local", doer)
	text, err := f.Fetch(context.Background(), "https://example.com/article")
	require.NoError(t, err)
	require.Contains(t, text, "Real content here.")
	require.NotContains(t, text, "menu")

	require.Len(t, doer.Requests, 1)
	req := doer.Requests[0]
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "http://browserless.local/chrome/content", req.URL.String())
}

// TestBrowserlessFetcherErrorsOnNon2xx asserts a non-2xx browserless response is
// an error (the L2 pipeline then skips that page).
func TestBrowserlessFetcherErrorsOnNon2xx(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusBadGateway, []byte("upstream error"))

	f := research.NewBrowserlessFetcher("", doer)
	_, err := f.Fetch(context.Background(), "https://example.com")
	require.Error(t, err)
}

func TestBrowserlessFetcherRejectsEmptyURL(t *testing.T) {
	f := research.NewBrowserlessFetcher("", &mocks.FakeDoer{})
	_, err := f.Fetch(context.Background(), "  ")
	require.Error(t, err)
}
