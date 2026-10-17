package experimentation

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfigStore(t *testing.T) *ConfigStore {
	t.Helper()
	variantsDir := findTestConfigPath(t, "variants", true)
	brandingPath := findTestConfigPath(t, "branding.json", false)
	store := NewConfigStore(variantsDir, brandingPath, nil)
	if err := store.LoadAll(); err != nil {
		t.Fatalf("load tracked configuration: %v", err)
	}
	return store
}

func findTestConfigPath(t *testing.T, name string, directory bool) string {
	t.Helper()
	for _, candidate := range []string{
		filepath.Join("..", "config", name),
		filepath.Join("..", "..", "config", name),
		filepath.Join("..", "..", "..", "config", name),
	} {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() == directory {
			return candidate
		}
	}
	t.Fatalf("tracked config %q not found", name)
	return ""
}
