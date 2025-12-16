package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-002] Repository status API

func TestParsePorcelainV2Status_BranchAndFiles(t *testing.T) {
	out := []byte(strings.Join([]string{
		"# branch.oid 0123456789abcdef",
		"# branch.head main",
		"# branch.upstream origin/main",
		"# branch.ab +2 -1",
		"1 M. N... 100644 100644 100644 abcdef1 abcdef2 file1.txt",
		"1 .M N... 100644 100644 100644 abcdef1 abcdef2 file2.txt",
		"? untracked.txt",
		"! ignored.log",
		"u UU N... 100644 100644 100644 100644 abcdef1 abcdef2 abcdef3 conflict.txt",
		"",
	}, "\x00"))

	parsed, err := ParsePorcelainV2Status(out)
	if err != nil {
		t.Fatalf("ParsePorcelainV2Status failed: %v", err)
	}

	if parsed.Branch.Head != "main" {
		t.Fatalf("expected branch head=main, got %q", parsed.Branch.Head)
	}
	if parsed.Branch.Upstream != "origin/main" {
		t.Fatalf("expected upstream origin/main, got %q", parsed.Branch.Upstream)
	}
	if parsed.Branch.Ahead != 2 || parsed.Branch.Behind != 1 {
		t.Fatalf("expected ahead=2 behind=1, got ahead=%d behind=%d", parsed.Branch.Ahead, parsed.Branch.Behind)
	}

	assertContains(t, parsed.Files.Staged, "file1.txt")
	assertContains(t, parsed.Files.Unstaged, "file2.txt")
	assertContains(t, parsed.Files.Untracked, "untracked.txt")
	assertContains(t, parsed.Files.Ignored, "ignored.log")
	assertContains(t, parsed.Files.Conflicts, "conflict.txt")
}

func TestGetRepoStatus_UsesGitAndDetectsScopes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "checkout", "-b", "main")

	writeFile(t, filepath.Join(repoDir, "scenarios", "alpha", "README.md"), "alpha")
	writeFile(t, filepath.Join(repoDir, "resources", "beta", "README.md"), "beta")
	writeFile(t, filepath.Join(repoDir, "notes.txt"), "notes")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	})
	if err != nil {
		t.Fatalf("GetRepoStatus failed: %v", err)
	}

	if status.Branch.Head == "" {
		t.Fatalf("expected branch head to be set")
	}
	if len(status.Files.Untracked) < 3 {
		t.Fatalf("expected at least 3 untracked files, got %d (%v)", len(status.Files.Untracked), status.Files.Untracked)
	}
	if len(status.Scopes["scenario:alpha"]) == 0 {
		t.Fatalf("expected scenario:alpha scope to be detected, got scopes=%v", status.Scopes)
	}
	if len(status.Scopes["resource:beta"]) == 0 {
		t.Fatalf("expected resource:beta scope to be detected, got scopes=%v", status.Scopes)
	}
	if len(status.Scopes["other"]) == 0 {
		t.Fatalf("expected other scope to be detected, got scopes=%v", status.Scopes)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, v := range values {
		if v == expected {
			return
		}
	}
	t.Fatalf("expected %q in %v", expected, values)
}

