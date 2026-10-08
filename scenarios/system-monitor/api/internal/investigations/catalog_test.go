package investigations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltinCatalogWithoutRepoRoot(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", "")
	t.Chdir(t.TempDir())
	catalog, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.Entries()); got != 6 {
		t.Fatalf("catalog entries = %d, want 6", got)
	}
}

func TestEmptyBuiltinCatalogIsAnError(t *testing.T) {
	fsys := os.DirFS(t.TempDir())
	if err := os.Mkdir(filepath.Join(t.TempDir(), "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Use a fresh directory because os.DirFS is intentionally rooted.
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "catalog", "entries.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBuiltin(os.DirFS(root))
	if err == nil {
		t.Fatal("expected empty built-in catalog to fail")
	}
	_ = fsys
}

func TestOperatorOverlayAddsEntries(t *testing.T) {
	state := t.TempDir()
	overlay := filepath.Join(state, "investigations")
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := []Entry{{ID: "operator-check", Name: "Operator Check", Mode: ModeNative, Query: "cpu", Platforms: []string{"linux"}, Enabled: true}}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(filepath.Join(overlay, "operator-check.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(state)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := catalog.Get("operator-check")
	if !ok || got.Source != "operator" {
		t.Fatalf("operator entry = %#v, present=%v", got, ok)
	}
}

func TestOperatorOverlayReplacesBuiltinByID(t *testing.T) {
	state := t.TempDir()
	overlay := filepath.Join(state, "investigations")
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := []Entry{{ID: "cpu-analyzer", Name: "Operator CPU", Mode: ModeNative, Query: "memory", Platforms: []string{"linux"}, Enabled: true}}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(filepath.Join(overlay, "cpu.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(state)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := catalog.Get("cpu-analyzer")
	if !ok || got.Name != "Operator CPU" || got.Source != "operator" {
		t.Fatalf("replaced entry = %#v, present=%v", got, ok)
	}
}
