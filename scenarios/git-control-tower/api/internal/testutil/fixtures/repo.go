// Package fixtures contains filesystem fixtures for API tests.
package fixtures

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// WriteRepoContract creates the minimal Vrooli repository layout needed by
// scenario discovery and envelope tests.
//
// The contract is the live repo's .vrooli/repo-contract.json copied verbatim —
// never a hand-typed copy. That keeps the single source of truth authoritative
// and prevents fixtures from silently drifting when the contract schema gains a
// required field (e.g. runtime_home). Required directories come from the live
// contract's root.markers so root detection succeeds.
func WriteRepoContract(t *testing.T, root string) {
	t.Helper()

	contract := liveRepoContract(t)

	var parsed struct {
		Root struct {
			Markers struct {
				RequiredDirs []string `json:"required_dirs"`
			} `json:"markers"`
		} `json:"root"`
	}
	if err := json.Unmarshal(contract, &parsed); err != nil {
		t.Fatalf("parse live repo contract: %v", err)
	}
	dirs := parsed.Root.Markers.RequiredDirs
	if len(dirs) == 0 {
		dirs = []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"}
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("create repo fixture dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatalf("write repo fixture go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo-contract fixture: %v", err)
	}
}

// liveRepoContract reads the repository's authoritative
// .vrooli/repo-contract.json by walking up from this source file until the
// contract is found, returning the raw bytes for verbatim copy into a fixture.
func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed; cannot locate live repo contract")
	}
	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, ".vrooli", "repo-contract.json")
		if data, err := os.ReadFile(candidate); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate .vrooli/repo-contract.json above fixtures package")
		}
		dir = parent
	}
}

// RunGitCommand executes a git command with deterministic author/committer
// environment for integration tests that intentionally touch real git.
func RunGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
	}
}

// SetupGitRepo creates a temporary git repository for integration tests.
func SetupGitRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	RunGitCommand(t, repoDir, "init")
	RunGitCommand(t, repoDir, "checkout", "-b", "main")
	return repoDir
}

// WriteFile creates a file with parent directories under test control.
func WriteFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

// WriteScenarioServiceJSON writes a service.json fixture for a scenario slug.
func WriteScenarioServiceJSON(t *testing.T, repoRoot, slug, content string) {
	t.Helper()

	dir := filepath.Join(repoRoot, "scenarios", slug, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create service fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write service fixture: %v", err)
	}
}
