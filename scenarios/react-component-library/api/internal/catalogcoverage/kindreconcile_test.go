package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReconcileKindsDetectsFixtureExploit(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "library", "components", "Card", "component.json")
	versionDir := filepath.Join(filepath.Dir(manifest), "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(versionDir, "Card.tsx")
	if err := os.WriteFile(source, []byte("export function Card() { return <div />; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	assets := []Asset{{ID: "primitives.card", Kind: "fixture"}}
	impls := []Implementation{{CatalogID: "primitives.card", Root: "components", Path: manifest, Latest: "1.0.0"}}
	mismatches, err := ReconcileKinds(root, assets, impls)
	if err != nil {
		t.Fatal(err)
	}
	if len(mismatches) != 1 || mismatches[0].DeclaredKind != "fixture" || mismatches[0].DerivedKind != "component" {
		t.Fatalf("kind exploit was not reported: %+v", mismatches)
	}
}
