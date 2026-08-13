package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-003] File diff endpoint

// --- Unit Tests using FakeGitRunner (fast, safe, no real git) ---

func TestGetDiff_WithFakeGit(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddUnstagedFile("modified.txt")

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "modified.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true")
	}
	if !fakeGit.AssertCalled("Diff") {
		t.Fatalf("expected Diff to be called")
	}
}

func TestGetDiff_SourceMode_HasDiffForUnstaged(t *testing.T) {
	repoDir := t.TempDir()
	path := "modified.txt"
	if err := os.WriteFile(filepath.Join(repoDir, path), []byte("line 1\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fakeGit := NewFakeGitRunner().
		AddUnstagedFile(path)

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: repoDir,
	}, DiffRequest{
		Path: path,
		Mode: ViewModeSource,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true for source mode")
	}
	if !fakeGit.AssertCalled("Diff") {
		t.Fatalf("expected Diff to be called for source mode")
	}
	if diff.Stats.Files != 1 {
		t.Fatalf("expected Stats.Files=1, got %d", diff.Stats.Files)
	}
}

func TestGetDiff_SourceMode_CommitHasDiff(t *testing.T) {
	fakeGit := NewFakeGitRunner()

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "file.txt",
		Commit: "abc123",
		Mode:   ViewModeSource,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true for commit source mode")
	}
	if !fakeGit.AssertCalled("ShowFileAtCommit") {
		t.Fatalf("expected ShowFileAtCommit to be called")
	}
	if !fakeGit.AssertCalled("ShowCommitDiff") {
		t.Fatalf("expected ShowCommitDiff to be called for source mode commit")
	}
}

func TestGetDiff_WithFakeGit_Staged(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddStagedFile("staged.txt")

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "staged.txt",
		Staged: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true for staged file")
	}
	if diff.Staged != true {
		t.Fatalf("expected Staged=true in response")
	}
}

func TestGetDiff_WithFakeGit_NoDiff(t *testing.T) {
	fakeGit := NewFakeGitRunner() // No files with changes

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "clean.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.HasDiff {
		t.Fatalf("expected HasDiff=false for clean file")
	}
}

func TestGetDiff_GitError(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeGit.DiffError = fmt.Errorf("simulated git diff failure")

	_, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "file.txt",
		Staged: false,
	})
	if err == nil {
		t.Fatalf("expected error from git failure")
	}
	if !strings.Contains(err.Error(), "simulated git diff failure") {
		t.Fatalf("expected error to contain 'simulated git diff failure', got: %v", err)
	}
}

func TestGetDiff_SourceMode_RejectsLargeFile(t *testing.T) {
	repoDir := t.TempDir()
	path := "big.txt"
	payload := bytes.Repeat([]byte("a"), int(maxDiffFileBytes+1))
	if err := os.WriteFile(filepath.Join(repoDir, path), payload, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := GetDiff(context.Background(), DiffDeps{
		Git:     NewFakeGitRunner(),
		RepoDir: repoDir,
	}, DiffRequest{
		Path: path,
		Mode: ViewModeSource,
	})
	if err == nil {
		t.Fatalf("expected error for large file")
	}
	var tooLarge *FileTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected FileTooLargeError, got: %v", err)
	}
}

func TestGetDiff_Untracked_RejectsBinary(t *testing.T) {
	repoDir := t.TempDir()
	path := "binary.dat"
	if err := os.WriteFile(filepath.Join(repoDir, path), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := GetDiff(context.Background(), DiffDeps{
		Git:     NewFakeGitRunner(),
		RepoDir: repoDir,
	}, DiffRequest{
		Path:      path,
		Untracked: true,
		Mode:      ViewModeDiff,
	})
	if err == nil {
		t.Fatalf("expected error for binary file")
	}
	var unsupported *UnsupportedBinaryError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedBinaryError, got: %v", err)
	}
}

// --- Parser Tests (pure functions, no git needed) ---

func TestParseDiffOutput_EmptyDiff(t *testing.T) {
	result := ParseDiffOutput("")
	if result.HasDiff {
		t.Fatalf("expected HasDiff=false for empty input, got true")
	}
	if len(result.Hunks) != 0 {
		t.Fatalf("expected 0 hunks, got %d", len(result.Hunks))
	}
}

func TestParseDiffOutput_SingleHunk(t *testing.T) {
	input := `diff --git a/file.txt b/file.txt
index 1234567..abcdef0 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
+added line
 line2
 line3
`
	result := ParseDiffOutput(input)
	if !result.HasDiff {
		t.Fatalf("expected HasDiff=true, got false")
	}
	if len(result.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(result.Hunks))
	}
	if result.Hunks[0].OldStart != 1 {
		t.Fatalf("expected OldStart=1, got %d", result.Hunks[0].OldStart)
	}
	if result.Hunks[0].OldCount != 3 {
		t.Fatalf("expected OldCount=3, got %d", result.Hunks[0].OldCount)
	}
	if result.Hunks[0].NewStart != 1 {
		t.Fatalf("expected NewStart=1, got %d", result.Hunks[0].NewStart)
	}
	if result.Hunks[0].NewCount != 4 {
		t.Fatalf("expected NewCount=4, got %d", result.Hunks[0].NewCount)
	}
	if result.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", result.Stats.Additions)
	}
	if result.Stats.Deletions != 0 {
		t.Fatalf("expected 0 deletions, got %d", result.Stats.Deletions)
	}
	if result.Stats.Files != 1 {
		t.Fatalf("expected 1 file, got %d", result.Stats.Files)
	}
}

func TestParseDiffOutput_MultipleHunks(t *testing.T) {
	input := `diff --git a/file.txt b/file.txt
index 1234567..abcdef0 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,2 @@
 line1
-removed
 line2
@@ -10,3 +9,4 @@
 line10
+added
 line11
 line12
`
	result := ParseDiffOutput(input)
	if len(result.Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(result.Hunks))
	}
	if result.Hunks[0].OldStart != 1 {
		t.Fatalf("expected first hunk OldStart=1, got %d", result.Hunks[0].OldStart)
	}
	if result.Hunks[1].OldStart != 10 {
		t.Fatalf("expected second hunk OldStart=10, got %d", result.Hunks[1].OldStart)
	}
	if result.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", result.Stats.Additions)
	}
	if result.Stats.Deletions != 1 {
		t.Fatalf("expected 1 deletion, got %d", result.Stats.Deletions)
	}
}

func TestParseDiffOutput_MultipleFiles(t *testing.T) {
	input := `diff --git a/file1.txt b/file1.txt
index 1234567..abcdef0 100644
--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,2 @@
 line1
+added
diff --git a/file2.txt b/file2.txt
index 1234567..abcdef0 100644
--- a/file2.txt
+++ b/file2.txt
@@ -1,2 +1,1 @@
 line1
-removed
`
	result := ParseDiffOutput(input)
	if result.Stats.Files != 2 {
		t.Fatalf("expected 2 files, got %d", result.Stats.Files)
	}
	if result.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", result.Stats.Additions)
	}
	if result.Stats.Deletions != 1 {
		t.Fatalf("expected 1 deletion, got %d", result.Stats.Deletions)
	}
}

func TestGetDiff_RequiresGitRunner(t *testing.T) {
	ctx := context.Background()
	_, err := GetDiff(ctx, DiffDeps{
		Git:     nil,
		RepoDir: "/tmp",
	}, DiffRequest{})
	if err == nil || !strings.Contains(err.Error(), "git runner is required") {
		t.Fatalf("expected 'git runner is required' error, got %v", err)
	}
}

func TestGetDiff_RequiresRepoDir(t *testing.T) {
	ctx := context.Background()
	_, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: "",
	}, DiffRequest{})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

func TestGetDiff_WithRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "checkout", "-b", "main")

	// Create and commit initial file
	filePath := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("initial content\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitCmd(t, repoDir, "add", "test.txt")
	runGitCmd(t, repoDir, "commit", "-m", "initial")

	// Modify file
	if err := os.WriteFile(filePath, []byte("initial content\nadded line\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	diff, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   "test.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}

	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true")
	}
	if diff.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", diff.Stats.Additions)
	}
	if diff.RepoDir != repoDir {
		t.Fatalf("expected RepoDir=%q, got %q", repoDir, diff.RepoDir)
	}
}

func TestGetDiff_StagedChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "checkout", "-b", "main")

	// Create and commit initial file
	filePath := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("initial content\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitCmd(t, repoDir, "add", "test.txt")
	runGitCmd(t, repoDir, "commit", "-m", "initial")

	// Modify and stage file
	if err := os.WriteFile(filePath, []byte("initial content\nstaged line\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitCmd(t, repoDir, "add", "test.txt")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check unstaged diff (should be empty since we staged)
	unstaged, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   "test.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("GetDiff (unstaged) failed: %v", err)
	}
	if unstaged.HasDiff {
		t.Fatalf("expected no unstaged diff")
	}

	// Check staged diff
	staged, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   "test.txt",
		Staged: true,
	})
	if err != nil {
		t.Fatalf("GetDiff (staged) failed: %v", err)
	}
	if !staged.HasDiff {
		t.Fatalf("expected staged diff")
	}
	if staged.Stats.Additions != 1 {
		t.Fatalf("expected 1 staged addition, got %d", staged.Stats.Additions)
	}
}

// runGitCmd is an alias for RunGitCommand for backward compatibility.
// New tests should use RunGitCommand directly.
func runGitCmd(t *testing.T, dir string, args ...string) {
	RunGitCommand(t, dir, args...)
}

// --- Enhanced Metrics Tests ---

func TestCountHunkChangedLines(t *testing.T) {
	tests := []struct {
		name string
		hunk DiffHunk
		want int
	}{
		{
			name: "mixed additions and deletions",
			hunk: DiffHunk{Lines: []string{
				" context",
				"+added1",
				"+added2",
				"-deleted1",
				" context",
			}},
			want: 3,
		},
		{
			name: "empty hunk",
			hunk: DiffHunk{Lines: []string{}},
			want: 0,
		},
		{
			name: "only context lines",
			hunk: DiffHunk{Lines: []string{" a", " b", " c"}},
			want: 0,
		},
		{
			name: "skips +++ and --- headers",
			hunk: DiffHunk{Lines: []string{
				"--- a/file.txt",
				"+++ b/file.txt",
				"+real addition",
			}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countHunkChangedLines(tt.hunk)
			if got != tt.want {
				t.Errorf("countHunkChangedLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseDiffOutput_EnhancedStats(t *testing.T) {
	// Two hunks: first has 3 changes, second has 5 changes
	raw := `diff --git a/file.txt b/file.txt
index 1234..5678 100644
--- a/file.txt
+++ b/file.txt
@@ -1,5 +1,6 @@
 context
+added1
+added2
-deleted1
 context
@@ -10,4 +11,7 @@
 context
+added3
+added4
+added5
-deleted2
-deleted3
 context
`
	resp := ParseDiffOutput(raw)

	if resp.Stats.HunkCount != 2 {
		t.Errorf("HunkCount = %d, want 2", resp.Stats.HunkCount)
	}
	if resp.Stats.Additions != 5 {
		t.Errorf("Additions = %d, want 5", resp.Stats.Additions)
	}
	if resp.Stats.Deletions != 3 {
		t.Errorf("Deletions = %d, want 3", resp.Stats.Deletions)
	}
	if resp.Stats.NetLines != 2 {
		t.Errorf("NetLines = %d, want 2", resp.Stats.NetLines)
	}
	if resp.Stats.LargestHunk != 5 {
		t.Errorf("LargestHunk = %d, want 5", resp.Stats.LargestHunk)
	}
	// Density = 2 hunks / 8 total changed lines = 0.25
	expectedDensity := 0.25
	if resp.Stats.Density < expectedDensity-0.01 || resp.Stats.Density > expectedDensity+0.01 {
		t.Errorf("Density = %f, want ~%f", resp.Stats.Density, expectedDensity)
	}
}

func TestParseDiffOutput_RenameDetection(t *testing.T) {
	raw := `diff --git a/old.txt b/new.txt
similarity index 95%
rename from old.txt
rename to new.txt
index 1234..5678 100644
--- a/old.txt
+++ b/new.txt
@@ -1,3 +1,3 @@
 line1
-old line
+new line
 line3
`
	resp := ParseDiffOutput(raw)

	if !resp.Stats.IsRename {
		t.Error("expected IsRename=true")
	}
	if resp.Stats.OldPath != "old.txt" {
		t.Errorf("OldPath = %q, want %q", resp.Stats.OldPath, "old.txt")
	}
}

func TestParseDiffOutput_EmptyDiff_EnhancedStats(t *testing.T) {
	resp := ParseDiffOutput("")

	if resp.HasDiff {
		t.Error("expected HasDiff=false for empty diff")
	}
	if resp.Stats.HunkCount != 0 {
		t.Errorf("HunkCount = %d, want 0", resp.Stats.HunkCount)
	}
	if resp.Stats.LargestHunk != 0 {
		t.Errorf("LargestHunk = %d, want 0", resp.Stats.LargestHunk)
	}
	if resp.Stats.Density != 0 {
		t.Errorf("Density = %f, want 0", resp.Stats.Density)
	}
}

func TestParseDiffOutput_SingleHunk_EnhancedStats(t *testing.T) {
	raw := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
+added
 line2
 line3
`
	resp := ParseDiffOutput(raw)

	if resp.Stats.HunkCount != 1 {
		t.Errorf("HunkCount = %d, want 1", resp.Stats.HunkCount)
	}
	if resp.Stats.LargestHunk != 1 {
		t.Errorf("LargestHunk = %d, want 1", resp.Stats.LargestHunk)
	}
	// Density = 1 hunk / 1 changed line = 1.0
	if resp.Stats.Density != 1.0 {
		t.Errorf("Density = %f, want 1.0", resp.Stats.Density)
	}
	if resp.Stats.NetLines != 1 {
		t.Errorf("NetLines = %d, want 1", resp.Stats.NetLines)
	}
}

func TestIsCommentLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
		ext     string
		want    bool
	}{
		{"Go single-line comment", "// this is a comment", ".go", true},
		{"Go block comment start", "/* block start", ".go", true},
		{"Go block comment end", "*/ end", ".go", true},
		{"Go mid-block star", "* continuation", ".go", true},
		{"Go code line", "fmt.Println()", ".go", false},
		{"TS comment", "// comment", ".ts", true},
		{"TSX comment", "// comment", ".tsx", true},
		{"Python comment", "# comment", ".py", true},
		{"Python code", "print('hello')", ".py", false},
		{"Shell comment", "# comment", ".sh", true},
		{"YAML comment", "# comment", ".yaml", true},
		{"HTML comment", "<!-- comment -->", ".html", true},
		{"HTML tag", "<div>", ".html", false},
		{"Unknown ext returns false", "// comment", ".xyz", false},
		{"Empty line returns false", "", ".go", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommentLine(tt.content, tt.ext)
			if got != tt.want {
				t.Errorf("isCommentLine(%q, %q) = %v, want %v", tt.content, tt.ext, got, tt.want)
			}
		})
	}
}

func TestEnrichCommentStats(t *testing.T) {
	resp := &DiffResponse{
		Hunks: []DiffHunk{
			{
				Lines: []string{
					" context line",
					"+// added comment",
					"+fmt.Println()",
					"+/* block comment */",
					"-// deleted comment",
					"-oldCode()",
					"--- a/file.go",
					"+++ b/file.go",
				},
			},
		},
		Stats: DiffStats{},
	}
	enrichCommentStats(resp, "file.go")

	if resp.Stats.CommentAdditions != 2 {
		t.Errorf("CommentAdditions = %d, want 2", resp.Stats.CommentAdditions)
	}
	if resp.Stats.CommentDeletions != 1 {
		t.Errorf("CommentDeletions = %d, want 1", resp.Stats.CommentDeletions)
	}
}

func TestEnrichCommentStats_NoPath(t *testing.T) {
	resp := &DiffResponse{
		Hunks: []DiffHunk{
			{Lines: []string{"+// comment"}},
		},
		Stats: DiffStats{},
	}
	enrichCommentStats(resp, "")

	if resp.Stats.CommentAdditions != 0 {
		t.Errorf("CommentAdditions = %d, want 0 (no-op for empty path)", resp.Stats.CommentAdditions)
	}
}

func TestParseDiffOutput_NewFileDetection(t *testing.T) {
	raw := `diff --git a/newfile.go b/newfile.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {}
`
	resp := ParseDiffOutput(raw)
	if !resp.Stats.IsNewFile {
		t.Error("expected IsNewFile to be true")
	}
	if resp.Stats.IsDeletedFile {
		t.Error("expected IsDeletedFile to be false")
	}
	if resp.Stats.Additions != 3 {
		t.Errorf("expected 3 additions, got %d", resp.Stats.Additions)
	}
}

func TestParseDiffOutput_DeletedFileDetection(t *testing.T) {
	raw := `diff --git a/old.go b/old.go
deleted file mode 100644
index abc1234..0000000
--- a/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func goodbye() {}
`
	resp := ParseDiffOutput(raw)
	if resp.Stats.IsNewFile {
		t.Error("expected IsNewFile to be false")
	}
	if !resp.Stats.IsDeletedFile {
		t.Error("expected IsDeletedFile to be true")
	}
	if resp.Stats.Deletions != 3 {
		t.Errorf("expected 3 deletions, got %d", resp.Stats.Deletions)
	}
}

func TestParseNumstatOutput_EnhancedFields(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantNet    int
		wantBinary bool
		wantBinLen int
	}{
		{
			name:       "normal file",
			input:      "10\t3\tfile.go\n",
			wantNet:    7,
			wantBinary: false,
			wantBinLen: 0,
		},
		{
			name:       "binary file",
			input:      "-\t-\timage.png\n",
			wantNet:    0,
			wantBinary: true,
			wantBinLen: 1,
		},
		{
			name:       "mixed",
			input:      "5\t2\tcode.go\n-\t-\tphoto.jpg\n",
			wantNet:    3,
			wantBinary: true,
			wantBinLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, binaries := parseNumstatOutput([]byte(tt.input))
			if tt.wantBinLen != len(binaries) {
				t.Errorf("binaries len = %d, want %d", len(binaries), tt.wantBinLen)
			}
			if tt.wantBinary {
				foundBin := false
				for _, s := range stats {
					if s.IsBinary {
						foundBin = true
						break
					}
				}
				if !foundBin {
					t.Error("expected at least one IsBinary=true entry")
				}
			}
			if !tt.wantBinary {
				for path, s := range stats {
					if s.NetLines != tt.wantNet {
						t.Errorf("stats[%s].NetLines = %d, want %d", path, s.NetLines, tt.wantNet)
					}
				}
			}
		})
	}
}

// --- SVG and preview mode tests ---
//
// SVG is XML text, not an opaque binary image. It must diff, count lines, and
// expose its source like any other text file, while remaining previewable.

func TestDetectBinaryKind_SVGIsText(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)
	if got := detectBinaryKind("icon.svg", svg); got != binaryNone {
		t.Fatalf("detectBinaryKind(icon.svg) = %v, want binaryNone (SVG is text)", got)
	}
	if got := detectBinaryKind("ICON.SVG", svg); got != binaryNone {
		t.Fatalf("detectBinaryKind(ICON.SVG) = %v, want binaryNone", got)
	}
}

func TestDetectBinaryKind_RasterImagesStayBinary(t *testing.T) {
	for _, path := range []string{"a.png", "a.jpg", "a.jpeg", "a.gif", "a.webp", "a.ico", "a.bmp", "a.tiff"} {
		if got := detectBinaryKind(path, []byte("whatever")); got != binaryImage {
			t.Errorf("detectBinaryKind(%s) = %v, want binaryImage", path, got)
		}
	}
}

// A multi-byte rune straddling the sample cutoff must not make valid UTF-8 text
// look like binary.
func TestDetectBinaryKind_MultibyteRuneAtSampleBoundary(t *testing.T) {
	for offset := range 4 {
		prefix := bytes.Repeat([]byte("a"), binarySampleBytes-offset)
		content := append(append([]byte{}, prefix...), []byte("→ trailing text")...)
		if got := detectBinaryKind("diagram.svg", content); got != binaryNone {
			t.Fatalf("offset %d: detectBinaryKind = %v, want binaryNone", offset, got)
		}
	}
}

func TestGetDiff_Untracked_SVGReturnsSourceNotBase64(t *testing.T) {
	repoDir := t.TempDir()
	path := "icon.svg"
	svg := "<svg xmlns=\"http://www.w3.org/2000/svg\">\n  <rect width=\"1\" height=\"1\"/>\n</svg>\n"
	if err := os.WriteFile(filepath.Join(repoDir, path), []byte(svg), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     NewFakeGitRunner(),
		RepoDir: repoDir,
	}, DiffRequest{Path: path, Untracked: true, Mode: ViewModePreview})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.FullContent != svg {
		t.Fatalf("FullContent = %q, want the raw SVG source", diff.FullContent)
	}
	// An untracked text file reports every line as an addition; reporting 0
	// contradicts the file list, which counts the same lines.
	if diff.Stats.Additions != 3 {
		t.Fatalf("Stats.Additions = %d, want 3", diff.Stats.Additions)
	}
	if len(diff.AnnotatedLines) == 0 {
		t.Fatal("expected annotated lines for SVG so the source view can render it")
	}
	if diff.ContentHash == "" {
		t.Fatal("expected a content hash for SVG so edits can be saved safely")
	}
}

func TestGetDiff_Untracked_RasterImageStaysBase64(t *testing.T) {
	repoDir := t.TempDir()
	path := "pixel.png"
	raw := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(repoDir, path), raw, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     NewFakeGitRunner(),
		RepoDir: repoDir,
	}, DiffRequest{Path: path, Untracked: true, Mode: ViewModePreview})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := base64.StdEncoding.EncodeToString(raw); diff.FullContent != want {
		t.Fatalf("FullContent = %q, want base64 %q", diff.FullContent, want)
	}
}

// Preview mode needs the whole file. Without this the View tab renders nothing
// for a tracked file that has changes.
func TestGetDiff_PreviewMode_TrackedFileHasFullContent(t *testing.T) {
	repoDir := t.TempDir()
	path := "icon.svg"
	svg := "<svg xmlns=\"http://www.w3.org/2000/svg\"/>\n"
	if err := os.WriteFile(filepath.Join(repoDir, path), []byte(svg), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     NewFakeGitRunner().AddUnstagedFile(path),
		RepoDir: repoDir,
	}, DiffRequest{Path: path, Mode: ViewModePreview})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.FullContent != svg {
		t.Fatalf("FullContent = %q, want the file source", diff.FullContent)
	}
}

func TestGetDiff_PreviewMode_CommitFileHasFullContent(t *testing.T) {
	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     NewFakeGitRunner(),
		RepoDir: "/fake/repo",
	}, DiffRequest{Path: "icon.svg", Commit: "abc123", Mode: ViewModePreview})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.FullContent == "" {
		t.Fatal("expected full content in preview mode for a file at a commit")
	}
}
