package treedigest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// walkOnlyRunner forces the filesystem-walk fallback (no git available).
func walkOnlyRunner(dir, name string, args ...string) ([]byte, error) {
	return nil, errors.New("git unavailable")
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestComputeDeterministic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/main.go", "package main\n")
	writeFile(t, dir, "PRD.md", "# X\n")

	a, err := ComputeWithRunner(dir, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ComputeWithRunner(dir, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("digest not deterministic: %s vs %s", a, b)
	}
	if !strings.HasPrefix(a, "td:") || len(a) != len("td:")+64 {
		t.Fatalf("unexpected digest shape: %q", a)
	}
}

func TestComputeChangesWhenFileChanges(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/main.go", "package main\n")

	before, _ := ComputeWithRunner(dir, walkOnlyRunner)
	writeFile(t, dir, "api/main.go", "package main // edited\n")
	after, _ := ComputeWithRunner(dir, walkOnlyRunner)

	if before == after {
		t.Fatal("digest must change when a file's bytes change")
	}
}

func TestComputeChangesWhenFileAddedOrDeleted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/main.go", "package main\n")
	base, _ := ComputeWithRunner(dir, walkOnlyRunner)

	writeFile(t, dir, "api/new.go", "package main\n")
	added, _ := ComputeWithRunner(dir, walkOnlyRunner)
	if added == base {
		t.Fatal("digest must change when a file is added")
	}

	if err := os.Remove(filepath.Join(dir, "api", "new.go")); err != nil {
		t.Fatal(err)
	}
	removed, _ := ComputeWithRunner(dir, walkOnlyRunner)
	if removed != base {
		t.Fatal("digest must return to baseline when the added file is removed")
	}
}

func TestComputeIgnoresExcludedDirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/main.go", "package main\n")
	base, _ := ComputeWithRunner(dir, walkOnlyRunner)

	// Churn in generated/state dirs must not affect the digest — otherwise
	// every test run would stale-ify itself by writing coverage artifacts.
	writeFile(t, dir, "coverage/runs/x/phase-results/unit.json", "{}")
	writeFile(t, dir, "data/state.db", "binary")
	writeFile(t, dir, "node_modules/dep/index.js", "x")
	writeFile(t, dir, "dist/bundle.js", "x")

	after, _ := ComputeWithRunner(dir, walkOnlyRunner)
	if after != base {
		t.Fatal("digest must ignore coverage/, data/, node_modules/, dist/")
	}
}

func TestComputeUsesGitEnumerationWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tracked.go", "package x\n")
	writeFile(t, dir, "ignored.log", "noise\n")

	// Fake git: only tracked.go is enumerated (ignored.log is gitignored).
	gitRunner := func(d, name string, args ...string) ([]byte, error) {
		if name == "git" && len(args) > 0 && args[0] == "ls-files" {
			return []byte("tracked.go\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	withIgnored, err := ComputeWithRunner(dir, gitRunner)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "ignored.log")); err != nil {
		t.Fatal(err)
	}
	withoutIgnored, err := ComputeWithRunner(dir, gitRunner)
	if err != nil {
		t.Fatal(err)
	}
	if withIgnored != withoutIgnored {
		t.Fatal("git-ignored files must not influence the digest")
	}
}

func TestComputeSkipsTrackedButDeletedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "package x\n")
	gitRunner := func(d, name string, args ...string) ([]byte, error) {
		if name == "git" && len(args) > 0 && args[0] == "ls-files" {
			return []byte("a.go\nb.go\n"), nil // b.go tracked but deleted in WT
		}
		return nil, errors.New("unexpected command")
	}
	if _, err := ComputeWithRunner(dir, gitRunner); err != nil {
		t.Fatalf("deleted tracked file must not fail digest: %v", err)
	}
}

func TestComputeRejectsMissingDir(t *testing.T) {
	if _, err := ComputeWithRunner(filepath.Join(t.TempDir(), "nope"), walkOnlyRunner); err == nil {
		t.Fatal("expected error for missing directory")
	}
	if _, err := ComputeWithRunner("", walkOnlyRunner); err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestCollectGitContext(t *testing.T) {
	calls := map[string][]byte{
		"rev-parse HEAD":              []byte("abc123\n"),
		"rev-parse --abbrev-ref HEAD": []byte("agi\n"),
		"status --porcelain -- .":     []byte(" M api/main.go\n?? new.txt\n"),
	}
	runner := func(d, name string, args ...string) ([]byte, error) {
		if out, ok := calls[strings.Join(args, " ")]; ok {
			return out, nil
		}
		return nil, errors.New("unexpected")
	}
	ctx := CollectGitContextWithRunner(t.TempDir(), runner)
	if ctx.Sha != "abc123" || ctx.Branch != "agi" || !ctx.Dirty {
		t.Fatalf("unexpected context: %+v", ctx)
	}
	if !strings.Contains(ctx.DirtySummary, "2 path(s)") {
		t.Fatalf("unexpected dirty summary: %q", ctx.DirtySummary)
	}

	// Outside a git work tree: zero value, no error.
	zero := CollectGitContextWithRunner(t.TempDir(), walkOnlyRunner)
	if zero.Sha != "" || zero.Dirty {
		t.Fatalf("expected zero context outside git, got %+v", zero)
	}
}
