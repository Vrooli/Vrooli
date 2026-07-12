package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSelectorManifestReloadsProjectChanges(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "ui", "src", "consts", "selectors.manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeManifest := func(selector string) {
		t.Helper()
		content, err := json.Marshal(map[string]any{
			"selectors": map[string]any{
				"dictationStudio.streamStatus": map[string]any{"selector": selector},
			},
		})
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	writeManifest("[data-testid=\"old-status\"]")
	first, err := loadSelectorManifest(root)
	if err != nil {
		t.Fatalf("first loadSelectorManifest() error = %v", err)
	}
	if got := first["selectors"].(map[string]interface{})["dictationStudio.streamStatus"].(map[string]interface{})["selector"]; got != "[data-testid=\"old-status\"]" {
		t.Fatalf("first selector = %v", got)
	}

	writeManifest("[data-testid=\"new-status\"]")
	second, err := loadSelectorManifest(root)
	if err != nil {
		t.Fatalf("second loadSelectorManifest() error = %v", err)
	}
	if got := second["selectors"].(map[string]interface{})["dictationStudio.streamStatus"].(map[string]interface{})["selector"]; got != "[data-testid=\"new-status\"]" {
		t.Fatalf("second selector = %v, want reloaded value", got)
	}
}
