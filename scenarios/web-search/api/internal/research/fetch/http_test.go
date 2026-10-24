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
	f.AllowPrivateAddresses = true
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

func TestHTTPFetcherDistinguishesExactCapFromOverCapBody(t *testing.T) {
	exact := strings.Repeat("a", fetch.DefaultMaxBodyBytes)
	f, exactServer, exactDone := httpTestFetcher(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(exact))
	})
	defer exactDone()

	text, err := f.Fetch(context.Background(), exactServer.URL)
	require.NoError(t, err)
	require.Len(t, text, fetch.DefaultMaxBodyBytes)

	over, overServer, overDone := httpTestFetcher(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(exact + "b"))
	})
	defer overDone()

	_, err = over.Fetch(context.Background(), overServer.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds")
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
	f.AllowPrivateAddresses = true
	text, err := f.Fetch(context.Background(), server.URL+"/start")
	require.NoError(t, err)
	require.Contains(t, text, "landed")
}

func TestHTTPFetcherObservationRetainsRedirectChain(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/middle", http.StatusFound)
	})
	mux.HandleFunc("/middle", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("redirected source"))
	})

	f := fetch.NewHTTPFetcher(0, 0)
	f.AllowPrivateAddresses = true
	observation, err := f.FetchObservation(context.Background(), server.URL+"/start")
	require.NoError(t, err)
	require.Equal(t, server.URL+"/final", observation.FinalURL)
	require.Equal(t, []string{server.URL + "/middle", server.URL + "/final"}, observation.RedirectURLs)
	require.Equal(t, "redirected source", string(observation.Content))
}

func TestHTTPFetcherPreservesVersionSelectingURLParameters(t *testing.T) {
	f, server, done := httpTestFetcher(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("version"); got != "2026-09" {
			t.Errorf("version query = %q, want 2026-09", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("versioned source"))
	})
	defer done()

	text, err := f.Fetch(context.Background(), server.URL+"/source?version=2026-09")
	require.NoError(t, err)
	require.Equal(t, "versioned source", text)
}

func TestHTTPFetcherCreatesNewValidationObservationForNotModified(t *testing.T) {
	var gotMatch, gotSince string
	f, server, done := httpTestFetcher(func(w http.ResponseWriter, r *http.Request) {
		gotMatch = r.Header.Get("If-None-Match")
		gotSince = r.Header.Get("If-Modified-Since")
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Last-Modified", "Tue, 01 Sep 2026 00:00:00 GMT")
		if gotMatch == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("unchanged source"))
	})
	defer done()
	first, err := f.FetchObservationWithValidators(context.Background(), server.URL, fetch.HTTPValidators{})
	require.NoError(t, err)
	require.Equal(t, "unchanged source", string(first.Content))
	second, err := f.FetchObservationWithValidators(context.Background(), server.URL, fetch.HTTPValidators{ETag: first.ETag, LastModified: first.LastModified})
	require.NoError(t, err)
	require.Equal(t, "not_modified", second.FailureCode)
	require.Empty(t, second.Content)
	require.Equal(t, `"v1"`, gotMatch)
	require.Equal(t, "Tue, 01 Sep 2026 00:00:00 GMT", gotSince)
	require.NotEqual(t, first.RetrievedAt, second.RetrievedAt)
}

func TestHTTPFetcherRejectsLoopbackByDefault(t *testing.T) {
	f := fetch.NewHTTPFetcher(0, 0)
	_, err := f.Fetch(context.Background(), "http://127.0.0.1:9/private")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private address")
}

func TestHTTPFetcherRejectsPrivateHostnameResolutionByDefault(t *testing.T) {
	f := fetch.NewHTTPFetcher(0, 0)
	_, err := f.Fetch(context.Background(), "http://localhost:9/private")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private address")
}
