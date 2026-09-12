package graph_test

import (
	"reflect"
	"testing"

	"architecture-cartographer/internal/graph"
)

// TestSnapshot_CloneIsDeep proves Clone returns a value the caller can
// mutate without aliasing the original — every slice is freshly
// allocated.
func TestSnapshot_CloneIsDeep(t *testing.T) {
	orig := graph.GraphSnapshot{
		Scenario:    "demo",
		ContentHash: "h",
		Languages:   []graph.Language{graph.LanguageGo},
		Files: []graph.FileNode{
			{ID: "f1", Path: "a/x.go", PackageID: "p1", Language: graph.LanguageGo},
		},
		Packages: []graph.PackageNode{{ID: "p1", ImportPath: "pkg/a"}},
		Symbols:  []graph.SymbolNode{{ID: "s1", Name: "Foo", PackageID: "p1", FileID: "f1"}},
		Imports: []graph.ImportEdge{
			{From: "p1", ToPackageID: "p2", SymbolIDs: []string{"s1"}},
		},
	}
	clone := orig.Clone()

	// Mutate the clone's slices and verify the original is untouched.
	clone.Languages[0] = graph.LanguageTypeScript
	clone.Files[0].Path = "MUTATED"
	clone.Packages[0].ImportPath = "MUTATED"
	clone.Symbols[0].Name = "MUTATED"
	clone.Imports[0].SymbolIDs[0] = "MUTATED"

	if orig.Languages[0] != graph.LanguageGo {
		t.Fatalf("Languages aliased: %v", orig.Languages)
	}
	if orig.Files[0].Path != "a/x.go" {
		t.Fatalf("Files aliased: %v", orig.Files)
	}
	if orig.Packages[0].ImportPath != "pkg/a" {
		t.Fatalf("Packages aliased: %v", orig.Packages)
	}
	if orig.Symbols[0].Name != "Foo" {
		t.Fatalf("Symbols aliased: %v", orig.Symbols)
	}
	if orig.Imports[0].SymbolIDs[0] != "s1" {
		t.Fatalf("Imports.SymbolIDs aliased: %v", orig.Imports[0].SymbolIDs)
	}
}

// TestSnapshot_NormalizeIsContentStable proves Normalize is
// deterministic: identical input → byte-identical snapshot ID + hash.
func TestSnapshot_NormalizeIsContentStable(t *testing.T) {
	raw := graph.RawGraph{
		Languages: []graph.Language{graph.LanguageGo},
		Files: []graph.FileNode{
			{ID: "f2", Path: "y.go"},
			{ID: "f1", Path: "x.go"},
		},
		Imports: []graph.ImportEdge{
			{From: "p1", ToPackageID: "p2"},
			{From: "p1", ToPackageID: "p2"}, // duplicate
		},
	}
	a := graph.Normalize("demo", raw)
	b := graph.Normalize("demo", raw)

	if a.ContentHash != b.ContentHash {
		t.Fatalf("ContentHash unstable: %q vs %q", a.ContentHash, b.ContentHash)
	}
	if a.ID != b.ID {
		t.Fatalf("ID unstable: %q vs %q", a.ID, b.ID)
	}
	if !reflect.DeepEqual(a.Files, b.Files) {
		t.Fatalf("Files order unstable: %v vs %v", a.Files, b.Files)
	}
	if len(a.Imports) != 1 {
		t.Fatalf("Imports not deduped: %v", a.Imports)
	}
}
