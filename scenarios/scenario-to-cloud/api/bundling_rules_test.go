package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	if !contains(entries, ".vrooli/cloud/bundle-metadata.json") {
		t.Fatalf("expected bundle metadata embedded in bundle")
	}
}

func TestMiniVrooliBundleSpec_OverridesServiceJSONEnabledResources(t *testing.T) {
	// [REQ:STC-P0-002] mini-Vrooli bundle should only enable required resources
	repoRoot := t.TempDir()
	writeFile(t, repoRoot, ".vrooli/service.json", `{
  "version": "2.0.0",
  "resources": {
    "postgres": { "enabled": true },
    "redis": { "enabled": true },
    "qdrant": { "enabled": true }
  }
}`)

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

	spec, err := MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		t.Fatalf("MiniVrooliBundleSpec: %v", err)
	}

	b := buildTarBytes(t, repoRoot, spec)
	serviceBytes := tarEntryBytes(t, b, ".vrooli/service.json")
	if len(serviceBytes) == 0 {
		t.Fatalf("expected .vrooli/service.json to be embedded/overridden")
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(serviceBytes, &doc); err != nil {
		t.Fatalf("service.json parse: %v", err)
	}

	resourcesAny, ok := doc["resources"]
	if !ok {
		t.Fatalf("service.json missing resources")
	}
	resources, ok := resourcesAny.(map[string]interface{})
	if !ok {
		t.Fatalf("service.json resources unexpected type")
	}

	getEnabled := func(id string) (bool, bool) {
		v, ok := resources[id]
		if !ok {
			return false, false
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			return false, false
		}
		b, ok := m["enabled"].(bool)
		return b, ok
	}

	if enabled, ok := getEnabled("postgres"); !ok || !enabled {
		t.Fatalf("expected postgres enabled, got: %v", resources["postgres"])
	}
	if enabled, ok := getEnabled("redis"); !ok || enabled {
		t.Fatalf("expected redis disabled, got: %v", resources["redis"])
	}
	if enabled, ok := getEnabled("qdrant"); !ok || enabled {
		t.Fatalf("expected qdrant disabled, got: %v", resources["qdrant"])
	}
}

func TestMiniVrooliBundleSpec_EmbedsTrimmedGoWork(t *testing.T) {
	// [REQ:STC-P0-002] mini-Vrooli bundle must not ship a go.work that references stripped modules.
	repoRoot := t.TempDir()

	writeFile(t, repoRoot, "go.work", `go 1.23.0

use (
	./packages/proto
	./scenarios/app-a/api
	./scenarios/app-b/api
	./scenarios/vrooli-autoheal/api
)
`)

	writeFile(t, repoRoot, "packages/proto/go.mod", "module example.com/proto\n\ngo 1.23\n")
	writeFile(t, repoRoot, "scenarios/app-a/api/go.mod", "module example.com/app-a\n\ngo 1.23\n")
	writeFile(t, repoRoot, "scenarios/app-b/api/go.mod", "module example.com/app-b\n\ngo 1.23\n")
	writeFile(t, repoRoot, "scenarios/vrooli-autoheal/api/go.mod", "module example.com/autoheal\n\ngo 1.23\n")

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

	b := buildTarBytes(t, repoRoot, spec)
	goWork := string(tarEntryBytes(t, b, "go.work"))
	if !strings.Contains(goWork, "go 1.23.0") {
		t.Fatalf("expected go.work to preserve version, got:\n%s", goWork)
	}
	if !strings.Contains(goWork, "\t./packages/proto") {
		t.Fatalf("expected packages/proto module, got:\n%s", goWork)
	}
	if !strings.Contains(goWork, "\t./scenarios/app-a/api") {
		t.Fatalf("expected app-a module, got:\n%s", goWork)
	}
	if !strings.Contains(goWork, "\t./scenarios/vrooli-autoheal/api") {
		t.Fatalf("expected autoheal module, got:\n%s", goWork)
	}
	if strings.Contains(goWork, "\t./scenarios/app-b/api") {
		t.Fatalf("did not expect stripped module in go.work, got:\n%s", goWork)
	}
}

func TestBuildMiniVrooliBundle_Smoke_ProducesSelfContainedMiniRepo(t *testing.T) {
	// [REQ:STC-P0-002] bundle build produces a deployable mini-Vrooli tarball
	// [REQ:STC-P0-008] tarball is self-contained (no excluded dirs, no absolute paths)
	repoRoot := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "out")

	// Minimal repo skeleton for bundling.
	writeFile(t, repoRoot, "go.work", "go 1.23.0\n")
	writeFile(t, repoRoot, "scripts/manage.sh", "#!/usr/bin/env bash\necho ok\n")
	writeFile(t, repoRoot, ".vrooli/service.json", `{"version":"2.0.0","resources":{"postgres":{"enabled":true},"redis":{"enabled":true}}}`)
	writeFile(t, repoRoot, "scenarios/app-a/README.md", "app-a\n")
	writeFile(t, repoRoot, "scenarios/vrooli-autoheal/README.md", "autoheal\n")
	writeFile(t, repoRoot, "resources/postgres/README.md", "pg\n")
	writeFile(t, repoRoot, "packages/proto/go.mod", "module example.com/proto\n\ngo 1.23\n")
	writeFile(t, repoRoot, "scenarios/app-a/api/go.mod", "module example.com/app-a\n\ngo 1.23\n")

	// Excluded directories should not be shipped.
	writeFile(t, repoRoot, "coverage/should-not-ship.txt", "nope\n")
	writeFile(t, repoRoot, "logs/should-not-ship.txt", "nope\n")

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
		Edge:  ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}
	manifest.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"

	artifact, err := BuildMiniVrooliBundle(repoRoot, outDir, manifest)
	if err != nil {
		t.Fatalf("BuildMiniVrooliBundle: %v", err)
	}
	if artifact.Path == "" || artifact.Sha256 == "" || artifact.SizeBytes <= 0 {
		t.Fatalf("unexpected artifact metadata: %+v", artifact)
	}

	b, err := os.ReadFile(artifact.Path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}

	entries := tarEntries(t, b)
	for _, name := range entries {
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			t.Fatalf("tar entry not self-contained: %q", name)
		}
		if strings.HasPrefix(name, "coverage/") || strings.HasPrefix(name, "logs/") {
			t.Fatalf("expected excluded dirs not to be present, found: %q", name)
		}
	}

	if !contains(entries, ".vrooli/cloud/manifest.json") {
		t.Fatalf("expected embedded manifest, entries=%v", entries)
	}
	if !contains(entries, ".vrooli/cloud/bundle-metadata.json") {
		t.Fatalf("expected embedded bundle metadata, entries=%v", entries)
	}

	goWork := string(tarEntryBytes(t, b, "go.work"))
	if !strings.Contains(goWork, "\t./packages/proto") || !strings.Contains(goWork, "\t./scenarios/app-a/api") {
		t.Fatalf("expected trimmed go.work to include present modules, got:\n%s", goWork)
	}

	serviceBytes := tarEntryBytes(t, b, ".vrooli/service.json")
	if len(serviceBytes) == 0 {
		t.Fatalf("expected .vrooli/service.json to be present")
	}
	if strings.Contains(string(serviceBytes), `"redis":{"enabled":true}`) {
		t.Fatalf("expected non-required resources to be disabled, got:\n%s", string(serviceBytes))
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

func tarEntryBytes(t *testing.T, b []byte, name string) []byte {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		if h.Name != name {
			continue
		}
		out, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		return out
	}
	return nil
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
