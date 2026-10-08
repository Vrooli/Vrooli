package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-control-tower/internal/pushsafety"
)

// Adapter contract tests use only t.TempDir repositories and local bundle URLs.
// Network-dependent orchestration tests use fake Commands/FakeGitRunner instead.
func TestRecoveryAdapterPreservesLiveWorkspace(t *testing.T) {
	ctx := context.Background()
	c := safetyCommands{path: "git"}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	run := func(dir string, input []byte, args ...string) string {
		t.Helper()
		b, e := c.Run(ctx, dir, input, args...)
		if e != nil {
			t.Fatalf("%v: %v", args, e)
		}
		return strings.TrimSpace(string(b))
	}
	run(root, nil, "init", source)
	run(source, nil, "config", "user.name", "Recovery Test")
	run(source, nil, "config", "user.email", "recovery@example.test")
	write := func(p, body string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(source, p), []byte(body), 0o755); e != nil {
			t.Fatal(e)
		}
	}
	write("keep", "base\n")
	write("deleted-locally", "committed file\n")
	if e := os.Symlink("keep", filepath.Join(source, "kept-link")); e != nil {
		t.Fatal(e)
	}
	run(source, nil, "add", "keep", "deleted-locally", "kept-link")
	run(source, nil, "commit", "-m", "base")
	base := run(source, nil, "rev-parse", "HEAD")
	path := "artifact [literal]\t.bin"
	write(path, "generated binary\n")
	run(source, nil, "add", "--", path)
	run(source, nil, "commit", "-m", "introduce generated artifact")
	first := run(source, nil, "rev-parse", "HEAD")
	write("keep", "committed source\n")
	write(path, "second generated version\n")
	run(source, nil, "add", "keep", path)
	run(source, nil, "commit", "-m", "source improvement")
	second := run(source, nil, "rev-parse", "HEAD")
	// A later deletion still requires removal from the introducing commit.
	run(source, nil, "rm", "--", path)
	run(source, nil, "commit", "-m", "remove artifact")
	third := run(source, nil, "rev-parse", "HEAD")
	run(source, nil, "commit", "--allow-empty", "-m", "retain logical checkpoint")
	head := run(source, nil, "rev-parse", "HEAD")
	write("keep", "staged source\n")
	run(source, nil, "add", "keep")
	write("keep", "unstaged source\n")
	write("untracked", "untracked work\n")
	if e := os.Remove(filepath.Join(source, "deleted-locally")); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(source, ".git", "info", "exclude"), []byte("ignored-local\n"), 0o600); e != nil {
		t.Fatal(e)
	}
	write("ignored-local", "ignored work\n")
	write(path, "local generated copy\n")
	indexPath := filepath.Join(source, ".git", "index")
	beforeIndex, e := os.ReadFile(indexPath)
	if e != nil {
		t.Fatal(e)
	}
	beforeStatus := run(source, nil, "status", "--porcelain=v2", "-z")
	beforeRefs := run(source, nil, "show-ref")
	r := pushsafety.Report{Complete: true, State: "blocked", CanPrepare: true, Head: head, Base: base, Commits: []string{first, second, third, head}, Fingerprint: strings.Repeat("a", 64), Limit: pushsafety.GitHubLimitBytes, Files: []pushsafety.File{{Blocked: true, Bytes: 440820652, OID: first, Paths: []string{path}, Commits: []string{first, second}}}}
	store := pushsafety.DiskStore{Root: filepath.Join(root, "artifacts")}
	a, e := pushsafety.Prepare(ctx, c, store, source, r)
	if e != nil {
		t.Fatal(e)
	}
	if a.State != "prepared" || len(a.Mappings) != 4 {
		t.Fatalf("%+v", a)
	}
	exerciseRecoveryHTTP(t, source, r)
	afterIndex, e := os.ReadFile(indexPath)
	if e != nil {
		t.Fatal(e)
	}
	if string(beforeIndex) != string(afterIndex) {
		t.Fatal("source index changed")
	}
	if run(source, nil, "status", "--porcelain=v2", "-z") != beforeStatus || run(source, nil, "show-ref") != beforeRefs {
		t.Fatal("source refs or working state changed")
	}
	for p, want := range map[string]string{"keep": "unstaged source\n", "untracked": "untracked work\n", "ignored-local": "ignored work\n", path: "local generated copy\n"} {
		b, e := os.ReadFile(filepath.Join(source, p))
		if e != nil || string(b) != want {
			t.Fatalf("live file changed %s", p)
		}
	}
	if _, e := os.Lstat(filepath.Join(source, "deleted-locally")); !os.IsNotExist(e) {
		t.Fatal("local deletion was undone")
	}
	link, e := os.Readlink(filepath.Join(source, "kept-link"))
	if e != nil || link != "keep" {
		t.Fatal("source symlink changed")
	}
	restored := filepath.Join(root, "restored-repaired")
	run(root, nil, "clone", "--bare", "--", a.RepairedBundle, restored)
	if run(restored, nil, "rev-parse", "refs/heads/recovery-candidate") != a.Candidate {
		t.Fatal("candidate restore mismatch")
	}
	for _, m := range a.Mappings {
		b, e := c.Run(ctx, restored, nil, "ls-tree", "-r", "-z", m.Replacement)
		if e != nil || strings.Contains(string(b), path) {
			t.Fatal("historical artifact retained")
		}
	}
	// Idempotency retains the same candidate and does not repeat cloning.
	a2, e := pushsafety.Prepare(ctx, c, store, source, r)
	if e != nil || a2.Candidate != a.Candidate {
		t.Fatalf("retry lost artifact: %+v %v", a2, e)
	}
}

func TestSafetyRunnerIgnoresInheritedRepositoryAndIndex(t *testing.T) {
	root := t.TempDir()
	c := safetyCommands{path: "git"}
	ctx := context.Background()
	if _, e := c.Run(ctx, root, nil, "init", "target"); e != nil {
		t.Fatal(e)
	}
	t.Setenv("GIT_DIR", filepath.Join(root, "unrelated.git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(root, "do-not-touch"))
	t.Setenv("GIT_WORK_TREE", root)
	b, e := c.Run(ctx, filepath.Join(root, "target"), nil, "rev-parse", "--show-toplevel")
	if e != nil || strings.TrimSpace(string(b)) != filepath.Join(root, "target") {
		t.Fatalf("repository redirected: %s %v", b, e)
	}
}

func TestPushTransfersOnlyInspectedCommitToExplicitDestination(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	c := safetyCommands{path: "git"}
	run := func(dir string, args ...string) string {
		t.Helper()
		b, e := c.Run(ctx, dir, nil, args...)
		if e != nil {
			t.Fatalf("%v: %v", args, e)
		}
		return strings.TrimSpace(string(b))
	}
	run(root, "init", "source")
	run(root, "init", "--bare", "remote.git")
	source := filepath.Join(root, "source")
	remote := filepath.Join(root, "remote.git")
	run(source, "config", "user.name", "Test")
	run(source, "config", "user.email", "test@example.test")
	run(source, "commit", "--allow-empty", "-m", "inspected")
	inspected := run(source, "rev-parse", "HEAD")
	run(source, "commit", "--allow-empty", "-m", "concurrent commit")
	run(source, "remote", "add", "origin", remote)
	r := &ExecGitRunner{}
	if e := r.Push(ctx, source, "origin", "different-destination", inspected, false, nil); e != nil {
		t.Fatal(e)
	}
	if run(remote, "rev-parse", "refs/heads/different-destination") != inspected {
		t.Fatal("transferred an uninspected commit")
	}
	if run(source, "rev-parse", "HEAD") == inspected {
		t.Fatal("source branch moved backwards")
	}
}

// A real index with a tiny LFS-style pointer and a huge working copy must use
// the staged object size. This uses an unborn temporary repository, no hooks,
// no network, and compares the index bytes before and after inspection.
func TestStagedSafetyReadsIndexAndPreservesWorkspace(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	c := safetyCommands{path: "git"}
	run := func(args ...string) {
		t.Helper()
		if _, err := c.Run(ctx, dir, nil, args...); err != nil {
			t.Fatal(err)
		}
	}
	run("init")
	path := filepath.Join(dir, "asset.bin")
	pointer := []byte("version https://git-lfs.github.com/spec/v1\noid sha256:abc\nsize 440820652\n")
	if err := os.WriteFile(path, pointer, 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "asset.bin")
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(pushsafety.GitHubLimitBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	index := filepath.Join(dir, ".git", "index")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	r := pushsafety.Report{Limit: pushsafety.GitHubLimitBytes}
	pushsafety.InspectStaged(ctx, c, dir, &r)
	if !r.StagedComplete || len(r.StagedFiles) != 0 {
		t.Fatalf("working copy was mistaken for staged bytes: %+v", r)
	}
	after, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("inspection changed the index")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != pushsafety.GitHubLimitBytes+1 {
		t.Fatal("inspection changed working file")
	}
}
