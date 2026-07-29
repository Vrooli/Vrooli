package scenariocli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type repoFixture struct {
	root string
	home string
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	created := map[string]any{}
	parent[key] = created
	return created
}

func newRepoFixture(t *testing.T) repoFixture {
	t.Helper()
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeRepoContractFixture(t, root)
	return repoFixture{root: root, home: home}
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()
	projectRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(projectRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal repo contract: %v", err)
	}
	layout := ensureObject(doc, "layout")
	layout["scenario_dir"] = "scenarios"
	sandbox := ensureObject(doc, "sandbox")
	sandbox["scenario_scope_prefix"] = "scenarios/"
	rootDoc := ensureObject(doc, "root")
	markers := ensureObject(rootDoc, "markers")
	markers["required_dirs"] = []any{"scenarios", "resources", "packages", "cmd", "internal"}
	for _, dir := range []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal repo contract: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), encoded, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
}
