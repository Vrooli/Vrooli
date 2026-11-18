package buildsnapshot

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newFixture returns a committed checkout with one tracked source, one ignored
// source, an uncommitted edit and a CRLF source that `.gitattributes` would
// normalize on a git checkout.
func newFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "--quiet", "--initial-branch=main")
	writeFixture(t, root, "go.mod", "module example.invalid/fixture\n\ngo 1.25\n")
	writeFixture(t, root, "main.go", "package main\n\nfunc main() {}\n")
	writeFixture(t, root, ".gitignore", "generated.go\n.vrooli/\n")
	writeFixture(t, root, ".gitattributes", "* text=auto\n")
	writeFixture(t, root, "crlf.go", "package main\r\n\r\nconst crlf = 1\r\n")
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "--quiet", "-m", "fixture")
	// An ignored Go file is a compiler input on a host that has generated it,
	// so the snapshot must carry it or the fingerprint would disagree.
	writeFixture(t, root, "generated.go", "package main\n\nconst generated = 1\n")
	writeFixture(t, root, "main.go", "package main\n\nfunc main() { println(generated) }\n")
	return root
}

func TestCreateCapturesTheCompilerInputsByteForByte(t *testing.T) {
	ctx := context.Background()
	root := newFixture(t)
	before := runGit(t, root, "status", "--porcelain")
	indexBefore := runGit(t, root, "ls-files", "-s")

	snapshot, err := Create(ctx, Options{Root: root, Targets: []string{"."}})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	t.Cleanup(func() { _ = snapshot.Remove(context.Background()) })

	if snapshot.Files < 4 {
		t.Fatalf("snapshot captured %d files, want the module, sources and the ignored source", snapshot.Files)
	}
	if _, err := os.Stat(filepath.Join(snapshot.Dir, "generated.go")); err != nil {
		t.Fatalf("snapshot lost the ignored compiler input: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(snapshot.Dir, "main.go")); err != nil || !strings.Contains(string(data), "println(generated)") {
		t.Fatalf("snapshot did not capture the uncommitted edit: %q %v", data, err)
	}
	// A git checkout would have normalized this file; the snapshot must not.
	if data, err := os.ReadFile(filepath.Join(snapshot.Dir, "crlf.go")); err != nil || !strings.Contains(string(data), "\r\n") {
		t.Fatalf("snapshot changed the bytes of a CRLF source: %q %v", data, err)
	}
	livePrint, err := buildinfo.ComputeSourceFingerprintForPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshotPrint, err := buildinfo.ComputeSourceFingerprintForPaths(snapshot.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if livePrint != snapshotPrint {
		t.Fatalf("snapshot fingerprint %s differs from the checkout %s; the snapshot is not the compiler input set", snapshotPrint, livePrint)
	}

	// The checkout's own state must be untouched: taking a snapshot is a read.
	if after := runGit(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("snapshot changed the checkout status: %q -> %q", before, after)
	}
	if after := runGit(t, root, "ls-files", "-s"); after != indexBefore {
		t.Fatal("snapshot staged files into the checkout index")
	}
	if worktrees := runGit(t, root, "worktree", "list", "--porcelain"); strings.Contains(worktrees, "vrooli-build-") {
		t.Fatalf("snapshot registered a worktree:\n%s", worktrees)
	}

	if err := snapshot.Remove(ctx); err != nil {
		t.Fatalf("remove snapshot: %v", err)
	}
	if _, err := os.Stat(snapshot.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot directory survived removal: %v", err)
	}
}

func TestCreateIncludesTheRepoContractAsACompilerInput(t *testing.T) {
	root := newFixture(t)
	writeFixture(t, root, ".vrooli/repo-contract.json", `{"fixture": true}`)
	paths, err := buildinfo.CompilerInputPaths(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, path := range paths {
		if path == ".vrooli/repo-contract.json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the repo contract is not a declared compiler input: %v", paths)
	}
	snapshot, err := Create(context.Background(), Options{Root: root, Targets: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = snapshot.Remove(context.Background()) })
	if _, err := os.Stat(filepath.Join(snapshot.Dir, ".vrooli", "repo-contract.json")); err != nil {
		t.Fatalf("snapshot does not carry the repo contract: %v", err)
	}
}

func TestCreateRefusesAnEmptyOrInvalidRequest(t *testing.T) {
	if _, err := Create(context.Background(), Options{Root: "", Targets: []string{"."}}); err == nil {
		t.Fatal("snapshot accepted an empty root")
	}
	root := newFixture(t)
	if _, err := Create(context.Background(), Options{Root: root, Targets: []string{"absent"}}); err == nil {
		t.Fatal("snapshot accepted targets that match nothing")
	}
}
