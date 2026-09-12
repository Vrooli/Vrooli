package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceAcquisitionResolvesFromRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "resource.json"), []byte(`{"name":"fixture","acquisition":{"kind":"url","targets":[{"when":{"os":"darwin","arch":"amd64"},"unsupported":"vendor does not publish this platform"}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_CLI_SOURCE_ROOT", root)
	acquisition, ok := resourceAcquisition("fixture")
	if !ok || acquisition == nil {
		t.Fatal("resource acquisition was not loaded")
	}
	if _, err := acquisition.Resolve(map[string]string{"os": "darwin", "arch": "amd64"}); err == nil {
		t.Fatal("unsupported target resolved successfully")
	}
}

func TestAuthenticatedHTTPClientAddsOwnerHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "owner" || r.Header.Get("X-Bridge-Owner-Reauth") != "reauth" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := authenticatedHTTPClient("owner", "reauth").Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
}
