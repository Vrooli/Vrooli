package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func localManifestGenerator(url string) *DeploymentManagerGenerator {
	return NewDeploymentManagerGenerator(WithDeploymentManagerURLResolver(func(context.Context) (string, error) { return url, nil }))
}

func TestDeploymentManagerGeneratorExportsRequestAndWritesReturnedManifest(t *testing.T) {
	var request bundleExportRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/bundles/export" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request = %s %s content-type=%q", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(bundleExportResponse{Status: "ok", Schema: "v1", Manifest: map[string]any{"scenario": "hello-desktop", "services": []any{"api"}}})
	}))
	defer server.Close()
	outputDir := t.TempDir()
	path, err := localManifestGenerator(server.URL).GenerateManifest(context.Background(), "hello-desktop", outputDir)
	if err != nil {
		t.Fatalf("GenerateManifest(): %v", err)
	}
	if request.Scenario != "hello-desktop" || request.Tier != "tier-2-desktop" || request.IncludeSecrets == nil || *request.IncludeSecrets || request.StageBundle || request.OutputDir != outputDir {
		t.Fatalf("export request = %#v", request)
	}
	if path != filepath.Join(outputDir, "bundle.json") {
		t.Fatalf("manifest path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || !json.Valid(data) || string(data) == "" {
		t.Fatalf("written manifest = %q, %v", data, err)
	}
}

func TestDeploymentManagerGeneratorUsesProvidedPathAndReportsFailures(t *testing.T) {
	t.Run("provided manifest path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(bundleExportResponse{ManifestPath: "/remote/bundle.json"})
		}))
		defer server.Close()
		path, err := localManifestGenerator(server.URL).GenerateManifest(context.Background(), "hello", t.TempDir())
		if err != nil || path != "/remote/bundle.json" {
			t.Fatalf("GenerateManifest() = %q, %v", path, err)
		}
	})
	for _, check := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"http failure", http.StatusBadGateway, "upstream unavailable", "returned status 502"},
		{"invalid json", http.StatusOK, "not-json", "failed to parse response"},
		{"missing manifest", http.StatusOK, `{}`, "returned no manifest"},
	} {
		t.Run(check.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(check.status)
				_, _ = w.Write([]byte(check.body))
			}))
			defer server.Close()
			_, err := localManifestGenerator(server.URL).GenerateManifest(context.Background(), "hello", t.TempDir())
			if err == nil || !strings.Contains(err.Error(), check.want) {
				t.Fatalf("error = %v, want %q", err, check.want)
			}
		})
	}
	resolverErr := errors.New("discovery unavailable")
	generator := NewDeploymentManagerGenerator(WithDeploymentManagerURLResolver(func(context.Context) (string, error) { return "", resolverErr }))
	if _, err := generator.GenerateManifest(context.Background(), "hello", t.TempDir()); !errors.Is(err, resolverErr) {
		t.Fatalf("resolver error = %v", err)
	}
}
