package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBundleBuildEndpoint_BuildsTarballArtifact(t *testing.T) {
	// [REQ:STC-P0-002] API can build a mini-Vrooli tarball from a manifest
	repoRoot := t.TempDir()

	mkdirAll(t, repoRoot, ".vrooli")
	mkdirAll(t, repoRoot, "scripts")
	mkdirAll(t, repoRoot, "api")
	mkdirAll(t, repoRoot, "cli")
	mkdirAll(t, repoRoot, "src")
	mkdirAll(t, repoRoot, "packages/pkg-a")
	mkdirAll(t, repoRoot, "scenarios/app-a")
	mkdirAll(t, repoRoot, "scenarios/vrooli-autoheal")
	mkdirAll(t, repoRoot, "scenarios/scenario-to-cloud/coverage")
	writeFileBytes(t, repoRoot, "go.work", []byte("go 1.23\n"))
	writeFileBytes(t, repoRoot, "package.json", []byte("{}\n"))
	writeFileBytes(t, repoRoot, "packages/pkg-a/README.md", []byte("pkg-a\n"))
	writeFileBytes(t, repoRoot, "scenarios/app-a/README.md", []byte("app-a\n"))
	writeFileBytes(t, repoRoot, "scenarios/vrooli-autoheal/README.md", []byte("autoheal\n"))

	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", repoRoot)
	t.Setenv("API_PORT", "0")

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	manifest := CloudManifest{
		Version: "1.0.0",
		Target:  ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "203.0.113.10"}},
		Scenario: ManifestScenario{
			ID: "app-a",
		},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"app-a"},
			Resources: []string{},
		},
		Bundle: ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
		},
		Ports: ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:  ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true, Email: "ops@example.com"}},
	}
	manifest.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"

	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/v1/bundle/build", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var out struct {
		Artifact BundleArtifact `json:"artifact"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Artifact.Path == "" {
		t.Fatalf("expected artifact path")
	}
	if _, err := os.Stat(out.Artifact.Path); err != nil {
		t.Fatalf("artifact missing: %v", err)
	}

	names := readTarNames(t, out.Artifact.Path)
	if !contains(names, "packages/pkg-a/README.md") {
		t.Fatalf("expected packages file in tar, got: %v", names)
	}
	if !contains(names, "scenarios/app-a/README.md") {
		t.Fatalf("expected scenario file in tar, got: %v", names)
	}
	if !contains(names, "scenarios/vrooli-autoheal/README.md") {
		t.Fatalf("expected autoheal file in tar, got: %v", names)
	}
	if !contains(names, ".vrooli/cloud/manifest.json") {
		t.Fatalf("expected embedded manifest in tar, got: %v", names)
	}
	if !contains(names, ".vrooli/cloud/bundle-metadata.json") {
		t.Fatalf("expected embedded bundle metadata in tar, got: %v", names)
	}
}

func TestBuildMiniVrooliBundle_DeterministicNameAndBytes(t *testing.T) {
	// [REQ:STC-P0-002] bundle output name and bytes should be deterministic for the same inputs
	repoRoot := t.TempDir()
	outDir := filepath.Join(repoRoot, "out")

	mkdirAll(t, repoRoot, ".vrooli")
	writeFileBytes(t, repoRoot, ".vrooli/service.json", []byte(`{"version":"2.0.0","resources":{"postgres":{"enabled":true},"redis":{"enabled":true}}}`))
	writeFileBytes(t, repoRoot, "packages/pkg-a/README.md", []byte("pkg-a\n"))
	writeFileBytes(t, repoRoot, "scenarios/app-a/README.md", []byte("app-a\n"))
	writeFileBytes(t, repoRoot, "scenarios/vrooli-autoheal/README.md", []byte("autoheal\n"))
	writeFileBytes(t, repoRoot, "resources/postgres/README.md", []byte("pg\n"))

	manifest := CloudManifest{
		Version: "1.0.0",
		Target:  ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "203.0.113.10"}},
		Scenario: ManifestScenario{
			ID: "app-a",
		},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"app-a"},
			Resources: []string{"postgres"},
		},
		Bundle: ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
			Scenarios:       []string{"app-a", "vrooli-autoheal"},
			Resources:       []string{"postgres"},
		},
		Ports: ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:  ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true, Email: "ops@example.com"}},
	}

	a1, err := BuildMiniVrooliBundle(repoRoot, outDir, manifest)
	if err != nil {
		t.Fatalf("BuildMiniVrooliBundle(1): %v", err)
	}
	a2, err := BuildMiniVrooliBundle(repoRoot, outDir, manifest)
	if err != nil {
		t.Fatalf("BuildMiniVrooliBundle(2): %v", err)
	}
	if a1.Path != a2.Path {
		t.Fatalf("expected deterministic output path; got %q vs %q", a1.Path, a2.Path)
	}
	if a1.Sha256 != a2.Sha256 {
		t.Fatalf("expected deterministic sha; got %q vs %q", a1.Sha256, a2.Sha256)
	}

	b1, err := os.ReadFile(a1.Path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	b2, err := os.ReadFile(a2.Path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("expected deterministic bundle bytes")
	}
}

func readTarNames(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	var names []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar: %v", err)
		}
		names = append(names, h.Name)
	}
	sort.Strings(names)
	return names
}

func mkdirAll(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
}

func writeFileBytes(t *testing.T, root, rel string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), contents, 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
