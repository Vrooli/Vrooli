package manifest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func manifestTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func writeManifestFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest file: %v", err)
	}
	return path
}

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	err := Run(nil, []string{"unknown"})
	if err == nil {
		t.Fatal("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFixRejectsWriteAndOutTogether(t *testing.T) {
	err := runFix(nil, []string{"manifest.json", "--write", "--out", "fixed.json"})
	if err == nil {
		t.Fatal("expected mutually-exclusive flag error")
	}
	if !strings.Contains(err.Error(), "--write and --out cannot be combined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientTemplateIncludesVariantQuery(t *testing.T) {
	var gotVariant string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/manifest/template" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotVariant = r.URL.Query().Get("variant")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"variant":"full","manifest":{"version":"1.0.0"},"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := manifestTestClient(server.URL)
	_, resp, err := client.Template("full")
	if err != nil {
		t.Fatalf("Template returned error: %v", err)
	}
	if gotVariant != "full" {
		t.Fatalf("expected variant query to be full, got %q", gotVariant)
	}
	if resp.Variant != "full" {
		t.Fatalf("expected variant full in response, got %q", resp.Variant)
	}
}

func TestClientFixWrapsManifestUnderManifestField(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/manifest/fix" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"issues":[],"manifest":{"version":"1.0.0"},"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := manifestTestClient(server.URL)
	manifest := map[string]interface{}{"version": "1.0.0"}
	if _, _, err := client.Fix(manifest); err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}

	rawManifest, ok := gotBody["manifest"]
	if !ok {
		t.Fatal("expected manifest wrapper field in request body")
	}
	m, ok := rawManifest.(map[string]any)
	if !ok {
		t.Fatalf("expected manifest object, got %T", rawManifest)
	}
	if m["version"] != "1.0.0" {
		t.Fatalf("expected wrapped manifest version 1.0.0, got %#v", m["version"])
	}
}
