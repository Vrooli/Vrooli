package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const elfHeader = "\x7fELF\x02\x01\x01\x00binary payload"

// trackedRepo wires a fake index (via Staged) plus fake file contents.
func trackedRepo(files map[string]string) (*FakeGitRunner, *FakeFileIO) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO()
	for path, content := range files {
		git.Staged[path] = content
		fs.WithFile("/repo/"+path, content)
	}
	return git, fs
}

func analyze(t *testing.T, git *FakeGitRunner, fs *FakeFileIO) *TrackedBinariesResponse {
	t.Helper()
	resp, err := AnalyzeTrackedBinaries(context.Background(), HealthDeps{FS: fs, RepoDir: "/repo"}, git)
	if err != nil {
		t.Fatalf("AnalyzeTrackedBinaries: %v", err)
	}
	return resp
}

func TestAnalyzeTrackedBinaries_FlagsExecutablesOnly(t *testing.T) {
	git, fs := trackedRepo(map[string]string{
		"scenarios/demo/cli/cli":     elfHeader,
		"scenarios/demo/cli/main.go": "package main\n",
		"README.md":                  "# demo\n",
	})

	resp := analyze(t, git, fs)

	if len(resp.Binaries) != 1 {
		t.Fatalf("expected exactly 1 binary, got %+v", resp.Binaries)
	}
	got := resp.Binaries[0]
	if got.Path != "scenarios/demo/cli/cli" {
		t.Fatalf("path = %q", got.Path)
	}
	if got.Format != "elf" {
		t.Fatalf("format = %q, want elf", got.Format)
	}
	if got.OwnerDir != "scenarios/demo" || got.IgnorePattern != "/cli/cli" {
		t.Fatalf("ignore target = %q %q, want scenarios/demo /cli/cli", got.OwnerDir, got.IgnorePattern)
	}
}

// Untracking never shrinks the repository; the response must say so or the UI
// will imply a reclaim that did not happen.
func TestAnalyzeTrackedBinaries_AlwaysWarnsAboutHistory(t *testing.T) {
	git, fs := trackedRepo(map[string]string{"scenarios/demo/api/main": elfHeader})

	resp := analyze(t, git, fs)

	if !strings.Contains(resp.HistoryWarning, "history") {
		t.Fatalf("history warning missing: %q", resp.HistoryWarning)
	}
	if resp.TotalBytes != int64(len(elfHeader)) {
		t.Fatalf("total bytes = %d, want %d", resp.TotalBytes, len(elfHeader))
	}
}

func TestAnalyzeTrackedBinaries_CleanRepoReportsNothing(t *testing.T) {
	git, fs := trackedRepo(map[string]string{
		"scenarios/demo/cli/main.go": "package main\n",
	})

	resp := analyze(t, git, fs)

	if len(resp.Binaries) != 0 {
		t.Fatalf("expected no binaries, got %+v", resp.Binaries)
	}
	if resp.HistoryWarning != "" {
		t.Fatalf("clean repo must not carry a warning, got %q", resp.HistoryWarning)
	}
}

// Largest first: the whole point of the list is deciding what to remove, and
// size is the cost being paid.
func TestAnalyzeTrackedBinaries_SortsLargestFirst(t *testing.T) {
	git, fs := trackedRepo(map[string]string{
		"scenarios/a/api/main": elfHeader,
		"scenarios/b/api/main": elfHeader + strings.Repeat("x", 100),
	})

	resp := analyze(t, git, fs)

	if len(resp.Binaries) != 2 || resp.Binaries[0].Path != "scenarios/b/api/main" {
		t.Fatalf("expected largest first, got %+v", resp.Binaries)
	}
}

func TestAnalyzeTrackedBinaries_MarksAlreadyIgnored(t *testing.T) {
	git, fs := trackedRepo(map[string]string{"scenarios/demo/cli/cli": elfHeader})
	fs.WithFile("/repo/scenarios/demo/.gitignore", "/cli/cli\n")

	resp := analyze(t, git, fs)

	if len(resp.Binaries) != 1 || !resp.Binaries[0].AlreadyIgnored {
		t.Fatalf("expected AlreadyIgnored, got %+v", resp.Binaries)
	}
}

// A repo-root binary has no owning scenario directory, so the root .gitignore
// is the only correct target.
func TestIgnoreTargetForBinary_RootBinaryHasNoOwnerDir(t *testing.T) {
	owner, pattern := ignoreTargetForBinary("vrooli-ports-migrate")
	if owner != "" || pattern != "/vrooli-ports-migrate" {
		t.Fatalf("got %q %q", owner, pattern)
	}
}

func TestUntrackBinary_WritesIgnoreThenRemovesFromIndex(t *testing.T) {
	git, fs := trackedRepo(map[string]string{"scenarios/demo/cli/cli": elfHeader})

	resp, err := UntrackBinary(context.Background(),
		HealthDeps{FS: fs, RepoDir: "/repo"}, git,
		UntrackBinaryRequest{Path: "scenarios/demo/cli/cli", OwnerDir: "scenarios/demo", IgnorePattern: "/cli/cli"},
	)
	if err != nil {
		t.Fatalf("UntrackBinary: %v", err)
	}
	if !resp.Success || !resp.RemovedFromIndex {
		t.Fatalf("expected success, got %+v", resp)
	}
	if !strings.Contains(fs.Files["/repo/scenarios/demo/.gitignore"], "/cli/cli") {
		t.Fatalf("gitignore not updated: %q", fs.Files["/repo/scenarios/demo/.gitignore"])
	}
	if _, stillTracked := git.Staged["scenarios/demo/cli/cli"]; stillTracked {
		t.Fatal("binary is still in the index")
	}
}

// The ignore is written before the index removal on purpose. If the removal
// fails, an ignored-but-tracked file is harmless; the reverse order would leave
// the file untracked AND unignored, so the next `git add -A` re-stages it.
func TestUntrackBinary_KeepsIgnoreWhenIndexRemovalFails(t *testing.T) {
	git, fs := trackedRepo(map[string]string{"scenarios/demo/cli/cli": elfHeader})
	git.RemoveFromIndexError = errors.New("index.lock held")

	resp, err := UntrackBinary(context.Background(),
		HealthDeps{FS: fs, RepoDir: "/repo"}, git,
		UntrackBinaryRequest{Path: "scenarios/demo/cli/cli", OwnerDir: "scenarios/demo", IgnorePattern: "/cli/cli"},
	)
	if err != nil {
		t.Fatalf("UntrackBinary returned a transport error: %v", err)
	}
	if resp.Success || resp.RemovedFromIndex {
		t.Fatalf("expected failure, got %+v", resp)
	}
	if !strings.Contains(fs.Files["/repo/scenarios/demo/.gitignore"], "/cli/cli") {
		t.Fatal("ignore should survive a failed index removal")
	}
}

func TestUntrackBinary_RejectsEscapingPaths(t *testing.T) {
	git, fs := trackedRepo(nil)

	for _, bad := range []string{"../outside/binary", "/etc/passwd", ""} {
		resp, err := UntrackBinary(context.Background(),
			HealthDeps{FS: fs, RepoDir: "/repo"}, git,
			UntrackBinaryRequest{Path: bad},
		)
		if err != nil {
			t.Fatalf("UntrackBinary(%q): %v", bad, err)
		}
		if resp.Success || resp.Error == "" {
			t.Fatalf("path %q must be rejected, got %+v", bad, resp)
		}
	}
}

// Re-running on an already-ignored binary must not duplicate the pattern.
func TestUntrackBinary_DoesNotDuplicateExistingPattern(t *testing.T) {
	git, fs := trackedRepo(map[string]string{"scenarios/demo/cli/cli": elfHeader})
	fs.WithFile("/repo/scenarios/demo/.gitignore", "/cli/cli\n")

	resp, err := UntrackBinary(context.Background(),
		HealthDeps{FS: fs, RepoDir: "/repo"}, git,
		UntrackBinaryRequest{Path: "scenarios/demo/cli/cli", OwnerDir: "scenarios/demo", IgnorePattern: "/cli/cli"},
	)
	if err != nil {
		t.Fatalf("UntrackBinary: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %+v", resp)
	}
	if resp.IgnoreAddedTo != "" {
		t.Fatalf("should not report an ignore write, got %q", resp.IgnoreAddedTo)
	}
	if strings.Count(fs.Files["/repo/scenarios/demo/.gitignore"], "/cli/cli") != 1 {
		t.Fatalf("pattern duplicated: %q", fs.Files["/repo/scenarios/demo/.gitignore"])
	}
}
