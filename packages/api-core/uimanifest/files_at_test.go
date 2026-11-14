package uimanifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAtRespectsDeclaredRegistryAndOverlay(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "ui"), 0o755)
	os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755)
	os.WriteFile(filepath.Join(root, "ui/manifest.json"), []byte(`{"slots":{"widget":{"dir":"ui/src/widgets"}},"files":{"selectorRegistry":{"path":"ui/src/app/selectors.ts"},"librarySelectors":{"path":"ui/src/generated/library.ts"}}}`), 0o644)
	os.WriteFile(filepath.Join(root, ".vrooli/ui-manifest.json"), []byte(`{"slots":{"widget":{"dir":"ui/src/moved/widgets"}},"files":{"selectorRegistry":{"path":"ui/src/moved/selectors.ts"}}}`), 0o644)
	m, err := LoadAt(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Slots["widget"].Dir != "ui/src/moved/widgets" {
		t.Fatal("slot overlay was dropped")
	}
	if got := m.SelectorManifestPath(); got != "ui/src/moved/selectors.manifest.json" {
		t.Fatal(got)
	}
	if got := m.ResolveFile("librarySelectors", ""); got != "ui/src/generated/library.ts" {
		t.Fatal(got)
	}
	os.WriteFile(filepath.Join(root, ".vrooli/ui-manifest.json"), []byte(`{"files":{"selectorRegistry":{"path":"../other/selectors.ts"}}}`), 0o644)
	if _, err := LoadAt(root); err == nil {
		t.Fatal("escaping path accepted")
	}
}
