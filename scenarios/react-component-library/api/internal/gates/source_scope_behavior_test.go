package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImplementationSourceScopeSeparatesPublishedAndDraftVersions(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Card")
	for _, version := range []string{"1.0.0", "1.1.0"} {
		versionDir := filepath.Join(assetDir, "versions", version)
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(versionDir, "Card.tsx"), []byte("export const Card = () => null;"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"catalogId":"components.card","libraryId":"react-component-library:Card","latest":"1.0.0","draft":"1.1.0"}`
	if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(root, "packages", "react-component-library", "package.json")
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packagePath, []byte(`{"exports":{"./Card/1.0.0":"./dist/Card.js","./Card/latest":"./dist/Card.js"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	published, err := implementationSources(root, "components.card")
	if err != nil {
		t.Fatal(err)
	}
	if len(published) != 1 || published[0].Version != "1.0.0" {
		t.Fatalf("published sources = %+v, want only the exported release", published)
	}
	draft, err := implementationSourcesForScope(root, "react-component-library:Card", "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(draft) != 1 || draft[0].Version != "1.1.0" || draft[0].Latest != "1.0.0" {
		t.Fatalf("explicit draft scope = %+v, want the unpublished version", draft)
	}
	_, got, found, err := implementationSource(root, "components.card")
	if err != nil || !found || got == "" {
		t.Fatalf("implementationSource = %q, found=%t, %v", got, found, err)
	}
	if got, err := implementationSourcesForScope(root, "components.missing", "1.1.0"); err != nil || len(got) != 0 {
		t.Fatalf("unknown explicit asset scope = %+v, %v", got, err)
	}
}

func TestActiveLibrarySourcesUsesLatestAndDraftAndHonorsAssetScope(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Card")
	for _, version := range []string{"1.0.0", "1.1.0", "0.9.0"} {
		versionDir := filepath.Join(assetDir, "versions", version)
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(versionDir, "Card.tsx"), []byte("export const Card = () => null;"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"catalogId":"components.card","latest":"1.0.0","draft":"1.1.0"}`
	if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	all, err := activeLibrarySources(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("active sources = %v, want latest and draft but not history", all)
	}
	draft, err := activeLibrarySources(Scope{Root: root, Assets: []string{"components.card"}, Version: "1.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft) != 1 || filepath.Base(draft[0]) != "Card.tsx" {
		t.Fatalf("draft asset-scoped sources = %v", draft)
	}
	wrongAsset, err := activeLibrarySources(Scope{Root: root, Assets: []string{"components.other"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(wrongAsset) != 0 {
		t.Fatalf("wrong asset scope returned sources: %v", wrongAsset)
	}
}

func TestActiveLibrarySourcesFallsBackToVersionFilesWithoutManifest(t *testing.T) {
	root := t.TempDir()
	versionDir := filepath.Join(root, "scenarios", "react-component-library", "library", "primitives", "Standalone", "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"Standalone.ts": "export const standalone = true;", "Story.tsx": "export const story = true;", "README.md": "docs"} {
		if err := os.WriteFile(filepath.Join(versionDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	set := Scope{Root: root}
	files, err := activeLibrarySources(set)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("manifest-free fallback should include only TS sources, got %v", files)
	}
	if got := countCatalogSources(set); got != 2 {
		t.Fatalf("fallback source count = %d, want 2", got)
	}
}

func TestExportedVersionsPrefersResolutionProjection(t *testing.T) {
	root := t.TempDir()
	resolution := filepath.Join(root, "packages", "react-component-library", "dist", "exports", "resolution.json")
	if err := os.MkdirAll(filepath.Dir(resolution), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resolution, []byte(`{"./Card/2.0.0":{"source":"./Card/versions/2.0.0/Card.tsx"},"./Card/latest":{"source":"latest"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	versions := exportedVersions(root, "Card")
	if !versions["2.0.0"] || len(versions) != 1 {
		t.Fatalf("exported resolution versions = %v, want only the concrete pinned version", versions)
	}
	if got := exportedVersions(t.TempDir(), "Card"); got != nil {
		t.Fatalf("missing package exports should remain unknown, got %v", got)
	}
}
