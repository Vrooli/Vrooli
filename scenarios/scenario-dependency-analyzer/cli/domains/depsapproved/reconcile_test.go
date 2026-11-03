package depsapproved

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverBuildableModulesSkipsDocsEvidence(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("module example.com/test\n\ngo 1.26.0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod")
	write("resources/go.mod")
	write("docs/go.mod")
	write("docs/evidence/preimage/go.mod")

	modules, err := discoverBuildableModulesAt(root)
	if err != nil {
		t.Fatalf("discoverBuildableModulesAt() error = %v", err)
	}
	for _, active := range []string{"go.mod", "resources/go.mod"} {
		if _, ok := modules[active]; !ok {
			t.Errorf("active module %q was not discovered", active)
		}
	}
	for _, excluded := range []string{"docs/go.mod", "docs/evidence/preimage/go.mod"} {
		if _, ok := modules[excluded]; ok {
			t.Errorf("docs evidence module %q was discovered", excluded)
		}
	}
}
