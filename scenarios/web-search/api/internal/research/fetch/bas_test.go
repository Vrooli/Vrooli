package fetch_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"web-search/internal/research/fetch"
)

func bundleServer(t *testing.T, html string) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	deleted := make(chan struct{})
	const id = "81dbd4b4-53ba-4e54-98a4-f98790da56da"
	artifact := []byte(html)
	sum := sha256.Sum256(artifact)
	manifest := map[string]any{
		"schema_version": "ux-capture.v1", "id": id,
		"source":    map[string]any{"kind": "url", "url": "https://example.com/spa"},
		"session":   map[string]any{"id": "session-123"},
		"lenses":    map[string]any{"L1": map[string]any{"status": "present"}},
		"privacy":   map[string]any{"redaction_status": "complete", "external": true, "ephemeral": true},
		"artifacts": []any{map[string]any{"kind": "frozen-page", "path": "page/frozen.html", "size_bytes": len(artifact), "sha256": hex.EncodeToString(sum[:])}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/ux-capture/bundles/"+id+"/artifacts/page/frozen.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(artifact)
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/ux-capture/bundles/"+id {
			close(deleted)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		require.Equal(t, "/api/v1/ux-capture/public", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var request map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "https://example.com/spa", request["url"])
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"manifest": manifest})
	}))
	return server, deleted
}

func TestBASFetcherExtractsReadableTextFromCanonicalBundle(t *testing.T) {
	server, deleted := bundleServer(t, `<html><body><nav>menu</nav><article><p>Rendered article text.</p></article></body></html>`)
	defer server.Close()
	f := &fetch.BASFetcher{Resolve: func(context.Context) (string, error) { return server.URL, nil }}

	text, err := f.Fetch(context.Background(), "https://example.com/spa")
	require.NoError(t, err)
	require.Contains(t, text, "Rendered article text.")
	require.NotContains(t, text, "menu")
	select {
	case <-deleted:
	default:
		t.Fatal("external ephemeral capture was not deleted through BAS owner")
	}
}

func TestBASFetcherCarriesBundleSessionIdentityIntoObservation(t *testing.T) {
	server, deleted := bundleServer(t, `<html><body><p>Rendered article text.</p></body></html>`)
	defer server.Close()
	f := &fetch.BASFetcher{Resolve: func(context.Context) (string, error) { return server.URL, nil }}

	observation, err := f.FetchObservation(context.Background(), "https://example.com/spa")
	require.NoError(t, err)
	require.Equal(t, "session-123", observation.ProducerExecutionID)
	require.Equal(t, "bas-readable-text-v1", observation.ExtractionRevision)
	require.Equal(t, "metadata-and-extracted-content", observation.Retention)
	select {
	case <-deleted:
	default:
		t.Fatal("external ephemeral capture was not deleted through BAS owner")
	}
}

func TestBASFetcherRejectsMissingReadableArtifact(t *testing.T) {
	server, deleted := bundleServer(t, "<html><body></body></html>")
	defer server.Close()
	f := &fetch.BASFetcher{Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	_, err := f.Fetch(context.Background(), "https://example.com/spa")
	require.ErrorContains(t, err, "no readable page text")
	select {
	case <-deleted:
	default:
		t.Fatal("unreadable content did not clean up external ephemeral bundle")
	}
}

func TestBASFetcherRejectsEmptyURL(t *testing.T) {
	f := &fetch.BASFetcher{}
	_, err := f.Fetch(context.Background(), " ")
	require.Error(t, err)
}
