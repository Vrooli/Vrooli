package domains

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func mkdirs(t *testing.T, root string, rel ...string) {
	t.Helper()
	for _, r := range rel {
		if err := os.MkdirAll(filepath.Join(root, r), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", r, err)
		}
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestFolderExtractor(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, dir,
		"api/internal/graph",
		"api/internal/conflicts",
		"api/internal/server",   // non-domain
		"api/internal/database", // non-domain
		"api/handlers/graph",    // makes graph also own handlers path
	)

	ext, err := NewFolderExtractorWithSurfaceProvider(nil, fakeSurfaceProvider{
		inv: SurfaceInventory{Surfaces: []Surface{{ID: "api", Kind: "api", Path: filepath.Join(dir, "api"), Status: "known"}}},
	}).Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	gotNames := make([]string, 0, len(ext.Domains))
	for _, d := range ext.Domains {
		gotNames = append(gotNames, d.Name)
	}
	if !reflect.DeepEqual(gotNames, []string{"conflicts", "graph"}) {
		t.Fatalf("folder domains = %v, want [conflicts graph]", gotNames)
	}
	// graph owns both internal and handlers paths.
	for _, d := range ext.Domains {
		if d.Name == "graph" {
			if !reflect.DeepEqual(d.Paths, []string{"api/internal/graph/", "api/handlers/graph/"}) {
				t.Fatalf("graph paths = %v", d.Paths)
			}
		}
	}
}

func TestFolderExtractor_Exemptions(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, dir, "api/internal/graph", "api/internal/recipes")
	ext, err := NewFolderExtractorWithSurfaceProvider([]string{"recipes"}, fakeSurfaceProvider{
		inv: SurfaceInventory{Surfaces: []Surface{{ID: "api", Kind: "api", Path: filepath.Join(dir, "api"), Status: "known"}}},
	}).Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(ext.Domains) != 1 || ext.Domains[0].Name != "graph" {
		t.Fatalf("expected only graph, got %+v", ext.Domains)
	}
}

func TestFolderExtractor_MissingDir(t *testing.T) {
	ext, err := NewFolderExtractor().Extract(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("missing api/internal should not error: %v", err)
	}
	if len(ext.Domains) != 0 {
		t.Fatalf("expected no domains, got %d", len(ext.Domains))
	}
}

func TestFolderExtractor_SurfaceProviderWarningPropagates(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, dir, "api/internal/graph")
	ext, err := NewFolderExtractorWithSurfaceProvider(nil, fakeSurfaceProvider{
		inv: SurfaceInventory{
			Surfaces: []Surface{{ID: "api", Kind: "api", Path: filepath.Join(dir, "api"), Status: "known"}},
			Warnings: []ExtractionWarning{{
				Kind:    "code_facts.unavailable",
				Summary: "fallback",
			}},
		},
	}).Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(ext.Warnings) != 1 || ext.Warnings[0].Kind != "code_facts.unavailable" {
		t.Fatalf("warnings = %+v", ext.Warnings)
	}
}

func TestFolderExtractor_SurfaceProviderError(t *testing.T) {
	_, err := NewFolderExtractorWithSurfaceProvider(nil, fakeSurfaceProvider{err: errors.New("boom")}).
		Extract(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected provider error")
	}
}

func TestCLIGroupExtractor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "cli/manifest.json", `{
      "name": "x",
      "groups": [
        {"name": "graph", "commands": []},
        {"name": "conflicts", "commands": []}
      ]
    }`)
	ext, err := NewCLIGroupExtractorWithSurfaceProvider(fakeSurfaceProvider{
		inv: SurfaceInventory{Surfaces: []Surface{{ID: "cli", Kind: "cli", Path: filepath.Join(dir, "cli"), Status: "known"}}},
	}).Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	gotNames := make([]string, 0, len(ext.Domains))
	for _, d := range ext.Domains {
		gotNames = append(gotNames, d.Name)
	}
	if !reflect.DeepEqual(gotNames, []string{"conflicts", "graph"}) {
		t.Fatalf("cli domains = %v", gotNames)
	}
	if !reflect.DeepEqual(ext.Domains[1].Paths, []string{"cli/domains/graph/"}) {
		t.Fatalf("graph cli path = %v", ext.Domains[1].Paths)
	}
}

func TestCLIGroupExtractor_MissingManifest(t *testing.T) {
	ext, err := NewCLIGroupExtractor().Extract(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("missing cli/manifest.json should not error: %v", err)
	}
	if len(ext.Domains) != 0 {
		t.Fatalf("expected no domains, got %d", len(ext.Domains))
	}
}

func TestCLIGroupExtractor_Malformed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "cli/manifest.json", `{not json`)
	if _, err := NewCLIGroupExtractor().Extract(context.Background(), dir); err == nil {
		t.Fatal("expected parse error for malformed manifest")
	}
}

type fakeSurfaceProvider struct {
	inv SurfaceInventory
	err error
}

func (f fakeSurfaceProvider) Inspect(context.Context, string) (SurfaceInventory, error) {
	return f.inv, f.err
}

type countingSurfaceProvider struct {
	inv   SurfaceInventory
	calls *int
}

func (p countingSurfaceProvider) Inspect(context.Context, string) (SurfaceInventory, error) {
	(*p.calls)++
	return p.inv, nil
}

func TestRunLadderMemoizesSurfaceInspectionPerRun(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, dir,
		"api/internal/graph",
		"cli",
		"ui/src/features/analytics",
	)
	writeFile(t, dir, "cli/manifest.json", `{"groups": []}`)

	calls := 0
	provider := countingSurfaceProvider{
		calls: &calls,
		inv: SurfaceInventory{Surfaces: []Surface{
			{ID: "api", Kind: "api", Path: filepath.Join(dir, "api"), Status: "known"},
			{ID: "cli", Kind: "cli", Path: filepath.Join(dir, "cli"), Status: "known"},
			{ID: "ui", Kind: "ui", Path: filepath.Join(dir, "ui"), Status: "known"},
		}},
	}

	if _, err := RunLadder(context.Background(), dir, ExtractorsForWithSurfaceProvider(nil, nil, provider)); err != nil {
		t.Fatalf("run ladder: %v", err)
	}
	if calls != 1 {
		t.Fatalf("surface inspections = %d, want one per reconcile run", calls)
	}
}
