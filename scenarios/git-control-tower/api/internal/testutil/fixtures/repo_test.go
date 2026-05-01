package fixtures

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteRepoContract(t *testing.T) {
	root := t.TempDir()
	WriteRepoContract(t, root)

	for _, path := range []string{
		"go.mod",
		filepath.Join(".vrooli", "repo-contract.json"),
		"scenarios",
		"resources",
		"packages",
		"cmd",
		"internal",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected fixture path %s: %v", path, err)
		}
	}
}

func TestWriteScenarioServiceJSON(t *testing.T) {
	root := t.TempDir()
	WriteRepoContract(t, root)
	WriteScenarioServiceJSON(t, root, "test-app", `{"service":{"name":"test-app"}}`)

	path := filepath.Join(root, "scenarios", "test-app", ".vrooli", "service.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read service fixture: %v", err)
	}
	if string(body) != `{"service":{"name":"test-app"}}` {
		t.Fatalf("service fixture body = %q", string(body))
	}
}
