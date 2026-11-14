package pipeline

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestOCIRegistryPushConfirmsArtifactAndReferrers(t *testing.T) {
	var mu sync.Mutex
	manifests := map[string][]byte{}
	referrers := []any{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/blobs/uploads") {
			w.Header().Set("Location", server.URL+"/uploads/1")
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/uploads/") && r.Method == http.MethodPut {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusCreated)
			return
		}
		if strings.Contains(r.URL.Path, "/manifests/") {
			key := strings.TrimPrefix(r.URL.Path, "/v2/vrooli/plugin-test/manifests/")
			mu.Lock()
			defer mu.Unlock()
			if r.Method == http.MethodPut {
				body, _ := io.ReadAll(r.Body)
				manifests[key] = body
				var parsed struct {
					Subject map[string]any `json:"subject"`
				}
				if json.Unmarshal(body, &parsed) == nil && parsed.Subject != nil {
					referrers = append(referrers, map[string]any{"digest": key})
				}
				w.WriteHeader(http.StatusCreated)
				return
			}
			body, ok := manifests[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			_, _ = w.Write(body)
			return
		}
		if strings.Contains(r.URL.Path, "/referrers/") && r.Method == http.MethodGet {
			mu.Lock()
			defer mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"schemaVersion": 2, "manifests": referrers})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	registry := &ociRegistry{base: server.URL, repository: "vrooli/plugin-test", client: server.Client()}
	coordinate, err := registry.pushPackage(t.Context(), "0.1.0", []byte("plugin"), []byte("signature"), []byte("provenance"), []byte("sbom"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(coordinate, "/manifests/0.1.0") {
		t.Fatalf("coordinate = %q", coordinate)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(manifests) != 4 {
		t.Fatalf("manifest count = %d, want package plus three referrers", len(manifests))
	}
}
