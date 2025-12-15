package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMiniVrooliBundleSpec_IncludesAutohealAndPackagesAndFiltersScenariosResources(t *testing.T) {
	// [REQ:STC-P0-002] deterministic mini-Vrooli tarball include/exclude rules
	repoRoot := t.TempDir()

	mkdir(t, repoRoot, ".vrooli")
	mkdir(t, repoRoot, "scripts")
	mkdir(t, repoRoot, "api")
	mkdir(t, repoRoot, "cli")
	mkdir(t, repoRoot, "src")
	writeFile(t, repoRoot, "go.work", "go 1.23\n")
	writeFile(t, repoRoot, "package.json", "{}\n")

	writeFile(t, repoRoot, "packages/pkg-a/README.md", "pkg-a\n")

	writeFile(t, repoRoot, "scenarios/app-a/README.md", "app-a\n")
	writeFile(t, repoRoot, "scenarios/app-b/README.md", "app-b\n")
	writeFile(t, repoRoot, "scenarios/vrooli-autoheal/README.md", "autoheal\n")

	writeFile(t, repoRoot, "resources/postgres/README.md", "pg\n")
	writeFile(t, repoRoot, "resources/redis/README.md", "redis\n")

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
		},
		Ports: ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:  ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true, Email: "ops@example.com"}},
	}
	manifest.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}

	spec, err := MiniVrooliBundleSpec(repoRoot, normalized)
	if err != nil {
		t.Fatalf("MiniVrooliBundleSpec: %v", err)
	}
	if !contains(spec.IncludeRoots, "packages") {
		t.Fatalf("expected packages to be included")
	}
	if !contains(spec.IncludeRoots, filepath.ToSlash("scenarios/app-a")) {
		t.Fatalf("expected scenarios/app-a to be included: %v", spec.IncludeRoots)
	}
	if contains(spec.IncludeRoots, filepath.ToSlash("scenarios/app-b")) {
		t.Fatalf("did not expect scenarios/app-b to be included: %v", spec.IncludeRoots)
	}
	if !contains(spec.IncludeRoots, filepath.ToSlash("scenarios/vrooli-autoheal")) {
		t.Fatalf("expected scenarios/vrooli-autoheal to be included")
	}
	if !contains(spec.IncludeRoots, filepath.ToSlash("resources/postgres")) {
		t.Fatalf("expected resources/postgres to be included")
	}
	if contains(spec.IncludeRoots, filepath.ToSlash("resources/redis")) {
		t.Fatalf("did not expect resources/redis to be included")
	}
}

func TestWriteDeterministicTarGz_IsReproducibleAndRelative(t *testing.T) {
	// [REQ:STC-P0-008] tarball is reproducible and self-contained (no absolute paths)
	repoRoot := t.TempDir()
	writeFile(t, repoRoot, "packages/pkg-a/README.md", "pkg-a\n")
	writeFile(t, repoRoot, "scenarios/app-a/README.md", "app-a\n")
	writeFile(t, repoRoot, "scenarios/vrooli-autoheal/README.md", "autoheal\n")

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
			Scenarios:       []string{"app-a", "vrooli-autoheal"},
		},
		Ports: ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:  ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true, Email: "ops@example.com"}},
	}

	spec, err := MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		t.Fatalf("MiniVrooliBundleSpec: %v", err)
	}

	b1 := buildTarBytes(t, repoRoot, spec)
	b2 := buildTarBytes(t, repoRoot, spec)
	if sha(b1) != sha(b2) {
		t.Fatalf("expected deterministic tarball; sha1=%s sha2=%s", sha(b1), sha(b2))
	}

	entries := tarEntries(t, b1)
	for _, name := range entries {
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			t.Fatalf("tar entry not self-contained: %q", name)
		}
	}
	if !contains(entries, "packages/pkg-a/README.md") {
		t.Fatalf("expected packages entry, got: %v", entries)
	}
	if !contains(entries, "scenarios/app-a/README.md") {
		t.Fatalf("expected scenario entry, got: %v", entries)
	}
	if !contains(entries, "scenarios/vrooli-autoheal/README.md") {
		t.Fatalf("expected autoheal entry, got: %v", entries)
	}
	if !contains(entries, ".vrooli/cloud/manifest.json") {
		t.Fatalf("expected manifest embedded in bundle")
	}
}

func buildTarBytes(t *testing.T, repoRoot string, spec MiniBundleSpec) []byte {
	t.Helper()
	var buf bytes.Buffer
	_, err := writeDeterministicTarGz(&buf, repoRoot, spec)
	if err != nil {
		t.Fatalf("writeDeterministicTarGz: %v", err)
	}
	return buf.Bytes()
}

func tarEntries(t *testing.T, b []byte) []string {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	var out []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		out = append(out, h.Name)
	}
	sort.Strings(out)
	return out
}

func sha(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func mkdir(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
}

func writeFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	mkdir(t, root, filepath.Dir(rel))
	if err := os.WriteFile(filepath.Join(root, rel), []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
