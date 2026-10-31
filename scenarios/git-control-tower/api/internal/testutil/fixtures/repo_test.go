package fixtures

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestWriteRepoContract(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")

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

func TestProjectRootIsTrimpathSafe(t *testing.T) {
	root := repocontracttest.ProjectRoot(t)
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "repo-contract.json")); err != nil {
		t.Fatalf("project root %s does not contain the live repo contract: %v", root, err)
	}
}

func TestWriteFileCreatesParents(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "dir", "file.txt")

	repocontracttest.WriteFile(t, path, "contents")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != "contents" {
		t.Fatalf("expected contents, got %q", string(data))
	}
}

func TestWriteScenarioServiceJSON(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
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
