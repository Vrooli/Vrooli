package fetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web-search/internal/research/fetch"

	"github.com/stretchr/testify/require"
)

// httpTestFetcher builds an HTTPFetcher pointed at a live httptest server so
// the tests exercise the real net/http path (headers, status, content-type,
// body capping) rather than a canned Doer.
func httpTestFetcher(handler http.HandlerFunc) (*fetch.HTTPFetcher, *httptest.Server, func()) {
	server := httptest.NewServer(handler)
	f := fetch.NewHTTPFetcher(0, 0)
	return f, server, server.Close
}

func TestHTTPFetcherExtractsReadableText(t *testing.T) {
	var gotUA, gotAccept string
	f, server, done := httpTestFetcher(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><nav>menu</nav><p>Real content here.</p></body></html>`))
	})
	defer done()

	text, err := f.Fetch(context.Background(), server.URL+"/article")
	require.NoError(t, err)
	require.Contains(t, text, "Real content here.")
	require.NotContains(t, text, "menu")
	require.Contains(t, gotUA, "vrooli-web-search")
	require.Contains(t, gotAccept, "text/html")
}

func TestHTTPFetcherErrorsOnNon2xx(t *testing.T) {
	f, server, done := httpTestFetcher(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer done()

	_, err := f.Fetch(context.Background(), server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 502")
}

func TestHTTPFetcherRejectsNonTextContentTypes(t *testing.T) {
	f, server, done := httpTestFetcher(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.7 ..."))
	})
	defer done()

	_, err := f.Fetch(context.Background(), server.URL+"/doc.pdf")
	require.Error(t, err)
	require.Contains(t, err.Error(), "content-type")
}

func TestHTTPFetcherCapsBodySize(t *testing.T) {
	huge := "<p>" + strings.Repeat("a", 4<<20) + "</p>"
	f, server, done := httpTestFetcher(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(huge))
	})
	defer done()

	text, err := f.Fetch(context.Background(), server.URL)
	require.NoError(t, err)
	// Default cap is 2 MiB; the 4 MiB body must have been truncated.
	require.LessOrEqual(t, len(text), 2<<20)
	require.NotEmpty(t, text)
}

func TestHTTPFetcherRejectsEmptyURL(t *testing.T) {
	f := fetch.NewHTTPFetcher(0, 0)
	_, err := f.Fetch(context.Background(), "  ")
	require.Error(t, err)
}

func TestHTTPFetcherFollowsRedirects(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<p>landed</p>`))
	})

	f := fetch.NewHTTPFetcher(0, 0)
	text, err := f.Fetch(context.Background(), server.URL+"/start")
	require.NoError(t, err)
	require.Contains(t, text, "landed")
}
