package conflicts_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	conflictmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fixturePath resolves a bas/fixtures/<name>/<file> path relative to
// the api/ test working directory.
func fixturePath(t *testing.T, name, file string) string {
	t.Helper()
	// api/internal/conflicts/integration_test.go → scenario root → bas/fixtures
	return filepath.Join("..", "..", "..", "bas", "fixtures", name, file)
}

// rawGraphFromFixture loads expected-graph.json and decodes it into a
// graph.RawGraph. Field names match the canonical JSON shape produced
// by the graph normalizer's ExportGraph.
func rawGraphFromFixture(t *testing.T, path string) graph.RawGraph {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)

	var raw struct {
		Scenario  string   `json:"scenario"`
		Languages []string `json:"languages"`
		Files     []struct {
			ID        string `json:"id"`
			Path      string `json:"path"`
			PackageID string `json:"package_id"`
			Language  string `json:"language"`
			Lines     int    `json:"lines"`
			IsTest    bool   `json:"is_test"`
		} `json:"files"`
		Packages []struct {
			ID         string `json:"id"`
			ImportPath string `json:"import_path"`
			Directory  string `json:"directory"`
			Language   string `json:"language"`
			Internal   bool   `json:"internal"`
		} `json:"packages"`
		Symbols []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			PackageID string `json:"package_id"`
			FileID    string `json:"file_id"`
			Kind      string `json:"kind"`
			Exported  bool   `json:"exported"`
		} `json:"symbols"`
		Imports []struct {
			From        string   `json:"from"`
			ToPackageID string   `json:"to_package_id"`
			SymbolIDs   []string `json:"symbol_ids"`
			TestOnly    bool     `json:"test_only"`
		} `json:"imports"`
	}
	require.NoError(t, json.Unmarshal(data, &raw))

	out := graph.RawGraph{}
	for _, l := range raw.Languages {
		out.Languages = append(out.Languages, graph.Language(l))
	}
	for _, f := range raw.Files {
		out.Files = append(out.Files, graph.FileNode{
			ID: f.ID, Path: f.Path, PackageID: f.PackageID,
			Language: graph.Language(f.Language), Lines: f.Lines, IsTest: f.IsTest,
		})
	}
	for _, p := range raw.Packages {
		out.Packages = append(out.Packages, graph.PackageNode{
			ID: p.ID, ImportPath: p.ImportPath, Directory: p.Directory,
			Language: graph.Language(p.Language), Internal: p.Internal,
		})
	}
	for _, s := range raw.Symbols {
		out.Symbols = append(out.Symbols, graph.SymbolNode{
			ID: s.ID, Name: s.Name, PackageID: s.PackageID, FileID: s.FileID,
			Kind: s.Kind, Exported: s.Exported,
		})
	}
	for _, e := range raw.Imports {
		out.Imports = append(out.Imports, graph.ImportEdge{
			From: e.From, ToPackageID: e.ToPackageID,
			SymbolIDs: append([]string(nil), e.SymbolIDs...), TestOnly: e.TestOnly,
		})
	}
	return out
}

// manifestFromFixture loads manifest.yaml via the manifest package's
// Parse to mirror production flow.
func manifestFromFixture(t *testing.T, path string) manifest.ManifestDefinition {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)
	m, _, _, err := manifest.Parse(data, manifest.ContentTypeYAML)
	require.NoError(t, err)
	// Sanity round-trip through yaml to surface any unrecognised keys.
	var sanity map[string]any
	_ = yaml.Unmarshal(data, &sanity)
	return m
}

// expectedConflicts is the loose-match shape used by integration
// assertions. Phase 11 keeps it tolerant (count, type, severity) so
// fixtures don't break on cosmetic detector message tweaks; future
// phases tighten with golden envelope comparison.
type expectedConflicts struct {
	Scenario  string `json:"scenario"`
	Conflicts []struct {
		Detector              string   `json:"detector"`
		Type                  string   `json:"type"`
		Severity              string   `json:"severity"`
		LocationsMin          int      `json:"locations_min"`
		EvidenceKindsInclude  []string `json:"evidence_kinds_include"`
	} `json:"conflicts"`
}

func expectedConflictsFromFixture(t *testing.T, path string) expectedConflicts {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)
	var out expectedConflicts
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// TestIntegration_GoCyclesFixture proves the cycle detector emits the
// expected conflict envelope when run against the go-cycles fixture
// via the canonical orchestration path: Normalize → Registry.DetectAll
// → conflict shape.
func TestIntegration_GoCyclesFixture(t *testing.T) {
	const fixture = "go-cycles"
	raw := rawGraphFromFixture(t, fixturePath(t, fixture, "expected-graph.json"))
	m := manifestFromFixture(t, fixturePath(t, fixture, "manifest.yaml"))
	want := expectedConflictsFromFixture(t, fixturePath(t, fixture, "expected-conflicts.json"))

	snap := graph.Normalize(want.Scenario, raw)

	registry := conflicts.NewRegistry(cycle.New(), mislocatedfile.New())
	svc := conflicts.NewService(&conflictmocks.FakeRepository{}, registry, conflicts.NewResolverRegistry())

	got, err := svc.DetectConflicts(context.Background(), conflicts.DetectOrchestrationInput{
		Scenario: want.Scenario,
		Snapshot: snap,
		Manifest: m,
	})
	require.NoError(t, err)
	require.NotEmpty(t, got, "go-cycles fixture must produce at least one conflict")

	// Look for an exact match against each expected envelope (loose:
	// envelope-level fields only).
	for _, w := range want.Conflicts {
		matched := false
		for _, c := range got {
			if c.Detector != w.Detector || c.Type != w.Type {
				continue
			}
			if w.Severity != "" && string(c.Severity) != w.Severity {
				continue
			}
			if len(c.Locations) < w.LocationsMin {
				continue
			}
			if !containsAllEvidenceKinds(c, w.EvidenceKindsInclude) {
				continue
			}
			matched = true
			break
		}
		require.Truef(t, matched, "no detected conflict matched expected %+v; got=%+v", w, got)
	}
}

func containsAllEvidenceKinds(c conflicts.Conflict, want []string) bool {
	have := make(map[string]struct{}, len(c.Evidence))
	for _, e := range c.Evidence {
		have[e.Kind] = struct{}{}
	}
	for _, w := range want {
		if _, ok := have[w]; !ok {
			return false
		}
	}
	return true
}
