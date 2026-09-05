package fixtures

import (
	"os"
	"os/exec"
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

func TestWriteFileCreatesParents(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "dir", "file.txt")

	WriteFile(t, path, "contents")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != "contents" {
		t.Fatalf("expected contents, got %q", string(data))
	}
}

func TestSetupGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := SetupGitRepo(t)

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		t.Fatalf("expected initialized git repo: %v", err)
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
