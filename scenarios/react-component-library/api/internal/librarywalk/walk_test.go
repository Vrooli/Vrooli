package librarywalk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourcesExcludesQuarantineDependenciesAndHonorsAssetScope(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"components/Button/versions/1.0.0/Button.tsx",
		"components/Button/versions/1.0.0/Button.css",
		"components/Card/versions/1.0.0/Card.tsx",
		".retired/Button/versions/0.1.0/Button.tsx",
		"node_modules/ignored.tsx",
	}
	for _, relative := range paths {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("export {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	all, err := Sources(root, FullCorpus())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("full source count = %d, want 3: %v", len(all), all)
	}
	scoped, err := Sources(root, Scope{Assets: map[string]struct{}{"controls.button": {}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 2 {
		t.Fatalf("scoped source count = %d, want 2: %v", len(scoped), scoped)
	}
	kinds, err := Kinds(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 1 || kinds[0] != "components" {
		t.Fatalf("kinds = %v, want [components]", kinds)
	}
}
