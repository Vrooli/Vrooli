package inference

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogRejectsStaleConcreteModelSelector(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCatalog(path); err != nil {
		t.Fatalf("load maintained catalog: %v", err)
	}

	legacy := strings.Replace(string(contents), `"provider": "ollama",`, `"provider": "ollama", "model": "stale:tag",`, 1)
	if legacy == string(contents) {
		t.Fatal("test fixture did not find the first Ollama candidate")
	}
	legacyPath := filepath.Join(t.TempDir(), "legacy-catalog.json")
	if err := os.WriteFile(legacyPath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCatalog(legacyPath); err == nil || !strings.Contains(err.Error(), `unknown field "model"`) {
		t.Fatalf("LoadCatalog() error = %v, want stale model selector rejection", err)
	}
}
