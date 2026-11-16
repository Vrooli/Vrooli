package collections

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
)

type fakeDoer struct {
	responses map[string]string
	methods   []string
	paths     []string
	hosts     []string
	apiKeys   []string
}

func TestListCommandUsesConfiguredOwnerEndpointAndWritesJSON(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	manifest := filepath.Join(root, "resources", "qdrant", "resource.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile("../../../resource.json")
	if err != nil {
		t.Fatalf("read source manifest: %v", err)
	}
	manifestBytes = []byte(strings.Replace(string(manifestBytes), `"host": 6333`, `"host": 6421`, 1))
	if err := os.WriteFile(manifest, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	doer := &fakeDoer{responses: map[string]string{
		"/collections":         `{"result":{"collections":[]}}`,
		"/aliases":             `{"result":{"aliases":[]}}`,
	}}
	var stdout bytes.Buffer
	h := &Handlers{
		Client:   Client{HTTP: doer},
		RepoRoot: root,
		UserHome: home,
		GetEnv: func(key string) string {
			if key == "QDRANT_URL" {
				return "http://stale-ambient.test:6333"
			}
			return ""
		},
		Stdout: &stdout,
	}
	if err := h.List([]string{"--json"}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := stdout.String(), "{\n  \"collections\": []\n}\n"; got != want {
		t.Fatalf("JSON output = %q, want %q", got, want)
	}
	if got, want := strings.Join(doer.paths, ","), "/collections,/aliases"; got != want {
		t.Fatalf("request paths = %q, want %q", got, want)
	}
	for i, host := range doer.hosts {
		if host != "localhost:6421" {
			t.Fatalf("request %d host = %q, want manifest-resolved endpoint localhost:6421", i, host)
		}
	}
	resolved, err := resourceenv.ResolveResource(root, home, "qdrant", resourceenv.ResolveOptions{})
	if err != nil {
		t.Fatalf("ResolveResource() error = %v", err)
	}
	if got, want := resolved.Values["QDRANT_URL"], "http://localhost:6421"; got != want {
		t.Fatalf("registered QDRANT_URL = %q, want %q", got, want)
	}
}

func TestListDoesNotFallBackToAmbientEndpointWhenOwnerResolutionIsEmpty(t *testing.T) {
	doer := &fakeDoer{responses: map[string]string{}}
	h := &Handlers{
		Client:   Client{HTTP: doer},
		RepoRoot: t.TempDir(),
		UserHome: t.TempDir(),
		GetEnv: func(key string) string {
			if key == "QDRANT_URL" || key == "QDRANT_BASE_URL" {
				return "http://stale-ambient.test:6333"
			}
			return ""
		},
		ResolveEnvironment: func(string, string, string) (map[string]string, error) {
			return map[string]string{}, nil
		},
	}
	if err := h.List(nil); err == nil || !strings.Contains(err.Error(), "no API endpoint") {
		t.Fatalf("List() error = %v, want missing registered endpoint", err)
	}
	if len(doer.methods) != 0 {
		t.Fatalf("made %d requests without a registered endpoint, want none", len(doer.methods))
	}
}

func (f *fakeDoer) Do(request *http.Request) (*http.Response, error) {
	f.methods = append(f.methods, request.Method)
	f.paths = append(f.paths, request.URL.Path)
	f.hosts = append(f.hosts, request.URL.Host)
	f.apiKeys = append(f.apiKeys, request.Header.Get("api-key"))
	body, ok := f.responses[request.URL.Path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func TestListReturnsSortedCollectionAliasesVectorsAndCountsUsingGETOnly(t *testing.T) {
	doer := &fakeDoer{responses: map[string]string{
		"/collections":         `{"result":{"collections":[{"name":"zeta"},{"name":"alpha"}]}}`,
		"/aliases":              `{"result":{"aliases":[{"alias_name":"notes-live","collection_name":"alpha"},{"alias_name":"old-notes","collection_name":"alpha"}]}}`,
		"/collections/zeta":    `{"result":{"status":"green","points_count":8,"indexed_vectors_count":8,"segments_count":1,"config":{"params":{"vectors":{"size":3,"distance":"Cosine"}}}}}`,
		"/collections/alpha":   `{"result":{"status":"green","points_count":4,"indexed_vectors_count":4,"segments_count":2,"config":{"params":{"vectors":{"body":{"size":4,"distance":"Dot"},"title":{"size":8,"distance":"Cosine"}}}}}}`,
	}}
	inventory, err := (Client{BaseURL: "http://qdrant.test/", APIKey: "test-key", HTTP: doer}).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(inventory.Collections), 2; got != want {
		t.Fatalf("collection count = %d, want %d", got, want)
	}
	alpha := inventory.Collections[0]
	if alpha.Name != "alpha" || strings.Join(alpha.Aliases, ",") != "notes-live,old-notes" {
		t.Fatalf("first collection = %+v", alpha)
	}
	if got, want := len(alpha.Vectors), 2; got != want || alpha.Vectors[0].Name != "body" || alpha.Vectors[0].Size != 4 || alpha.Vectors[1].Name != "title" || alpha.Vectors[1].Size != 8 {
		t.Fatalf("named vector inventory = %+v", alpha.Vectors)
	}
	if alpha.PointsCount == nil || *alpha.PointsCount != 4 {
		t.Fatalf("alpha points_count = %v, want 4", alpha.PointsCount)
	}
	zeta := inventory.Collections[1]
	if len(zeta.Vectors) != 1 || zeta.Vectors[0].Name != "default" || zeta.Vectors[0].Size != 3 {
		t.Fatalf("unnamed vector inventory = %+v", zeta.Vectors)
	}
	if len(doer.methods) != 4 {
		t.Fatalf("request methods = %v, want 4 GET requests", doer.methods)
	}
	for i, method := range doer.methods {
		if method != http.MethodGet {
			t.Fatalf("request %d method = %q, want GET", i, method)
		}
		if doer.apiKeys[i] != "test-key" {
			t.Fatalf("request %d api-key = %q", i, doer.apiKeys[i])
		}
	}
}

func TestListRejectsMissingVectorConfiguration(t *testing.T) {
	doer := &fakeDoer{responses: map[string]string{
		"/collections":         `{"result":{"collections":[{"name":"broken"}]}}`,
		"/aliases":             `{"result":{"aliases":[]}}`,
		"/collections/broken":  `{"result":{"config":{"params":{"vectors":null}}}}`,
	}}
	if _, err := (Client{BaseURL: "http://qdrant.test", HTTP: doer}).List(context.Background()); err == nil || !strings.Contains(err.Error(), "vector configuration is unavailable") {
		t.Fatalf("List() error = %v, want unavailable vector configuration", err)
	}
}

func TestListRejectsNonSuccessResponse(t *testing.T) {
	doer := statusDoer{status: http.StatusForbidden}
	if _, err := (Client{BaseURL: "http://qdrant.test", HTTP: doer}).List(context.Background()); err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("List() error = %v, want HTTP 403", err)
	}
}

type statusDoer struct{ status int }

func (d statusDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: d.status, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
}
