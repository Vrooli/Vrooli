package conflicts_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/conflicts/detectors/glossarydrift"
	"architecture-cartographer/internal/conflicts/detectors/layering"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	conflictmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"

	"github.com/stretchr/testify/require"
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
			RepoPath   string `json:"repo_path"`
			Language   string `json:"language"`
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
			ID: p.ID, ImportPath: p.ImportPath, RepoPath: p.RepoPath,
			Language: graph.Language(p.Language),
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

// domainMapFromFixture derives the fixture scenario's domain map from its
// on-disk sources (its docs/concepts/DOMAINS.md) via the production
// extraction ladder — mirroring exactly what production runs.
func domainMapFromFixture(t *testing.T, name string) domains.DerivedDomainMap {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "bas", "fixtures", name))
	require.NoError(t, err)
	extractions, err := domains.RunLadder(context.Background(), dir, domains.DefaultExtractors())
	require.NoError(t, err, "run ladder over fixture %s", name)
	m, err := domains.Resolve(name, extractions, time.Time{})
	require.NoError(t, err, "resolve domain map for fixture %s", name)
	return m
}

// expectedConflicts is the loose-match shape used by integration
// assertions. Phase 11 keeps it tolerant (count, type, severity) so
// fixtures don't break on cosmetic detector message tweaks; future
// phases tighten with golden envelope comparison.
type expectedConflicts struct {
	Scenario  string `json:"scenario"`
	Conflicts []struct {
		Detector             string   `json:"detector"`
		Type                 string   `json:"type"`
		Severity             string   `json:"severity"`
		LocationsMin         int      `json:"locations_min"`
		EvidenceKindsInclude []string `json:"evidence_kinds_include"`
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
func TestIntegration_Fixtures(t *testing.T) {
	for _, fixture := range []string{"go-cycles", "go-mislocated", "ts-junk-drawer"} {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			raw := rawGraphFromFixture(t, fixturePath(t, fixture, "expected-graph.json"))
			dmap := domainMapFromFixture(t, fixture)
			want := expectedConflictsFromFixture(t, fixturePath(t, fixture, "expected-conflicts.json"))

			snap := graph.Normalize(want.Scenario, raw)

			registry := conflicts.NewRegistry(
				cycle.New(),
				glossarydrift.New(),
				layering.New(),
				mislocatedfile.New(),
			)
			svc := conflicts.NewService(&conflictmocks.FakeRepository{}, registry, conflicts.NewResolverRegistry())

			got, err := svc.DetectConflicts(context.Background(), conflicts.DetectOrchestrationInput{
				Scenario:  want.Scenario,
				Snapshot:  snap,
				DomainMap: dmap,
			})
			require.NoError(t, err)
			require.NotEmpty(t, got, "%s fixture must produce at least one conflict", fixture)

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
		})
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
