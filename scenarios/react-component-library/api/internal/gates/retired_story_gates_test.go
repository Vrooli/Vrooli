package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoryGrammarSkipsRetiredArchiveButChecksLiveAsset(t *testing.T) {
	root := t.TempDir()
	writeStoryGateAsset(t, root)
	writeStoryGateVersion(t, root, filepath.Join("components", "Card"), `{"id":"anatomy","role":"anatomy","expect":[{"kind":"visible","selector":"[data-testid=card]"}]}`)
	writeStoryGateVersion(t, root, filepath.Join(".retired", "archive-card"), `{"id":"legacy","expect":[{"kind":"visible","selector":"body"}]}`)

	result, err := ValidateStoryGrammar(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 {
		t.Fatalf("inspected %d story files, want only the live asset", result.Inspected)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("retired story affected live grammar findings: %+v", result.Findings)
	}
}

func TestStoryDistinctnessSkipsRetiredArchiveAndVersions(t *testing.T) {
	root := t.TempDir()
	writeStoryGateVersion(t, root, filepath.Join("components", "Card"), `{"id":"anatomy","role":"anatomy","expect":[{"kind":"visible","selector":"[data-testid=card]"}]},{"id":"empty","role":"boundary","expect":[{"kind":"count","selector":"[data-testid=card]","value":0}]}`)
	writeStoryGateVersion(t, root, filepath.Join(".retired", "archive-card"), `{"id":"legacy-a","expect":[{"kind":"visible","selector":"body"}]},{"id":"legacy-b","expect":[{"kind":"visible","selector":"body"}]}`)

	result, err := ValidateStoryDistinctness(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 {
		t.Fatalf("inspected %d story files, want only the live asset", result.Inspected)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("retired story affected live distinctness findings: %+v", result.Findings)
	}
}

func writeStoryGateAsset(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "components", "card.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"asset":{"id":"components.card","kind":"component"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeStoryGateVersion(t *testing.T, root, relativeAssetDir, stories string) {
	t.Helper()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", relativeAssetDir)
	versionDir := filepath.Join(assetDir, "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"catalogId":"components.card","assetKind":"component","latest":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(assetDir, "component.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"schemaVersion":5,"kind":"component","args":{"fields":[]},"stories":[` + stories + `]}`)
	if err := os.WriteFile(filepath.Join(versionDir, "story.json"), contract, 0o600); err != nil {
		t.Fatal(err)
	}
}
