package domains

import (
	"context"
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

	ext, err := NewFolderExtractor().Extract(context.Background(), dir)
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
	ext, err := NewFolderExtractorWithExemptions([]string{"recipes"}).Extract(context.Background(), dir)
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

func TestCLIGroupExtractor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, cliManifestRel(dir), `{
      "name": "x",
      "groups": [
        {"name": "graph", "commands": []},
        {"name": "conflicts", "commands": []}
      ]
    }`)
	ext, err := NewCLIGroupExtractor().Extract(context.Background(), dir)
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
	writeFile(t, dir, cliManifestRel(dir), `{not json`)
	if _, err := NewCLIGroupExtractor().Extract(context.Background(), dir); err == nil {
		t.Fatal("expected parse error for malformed manifest")
	}
}
