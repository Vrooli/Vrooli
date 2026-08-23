package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git-control-tower/internal/testutil/fixtures"
)

type fakeCommitCheckReader struct {
	runs map[string][]CommitCheckRun
}

func (r fakeCommitCheckReader) ListForCommits(ctx context.Context, repoPath string, hashes []string) (map[string][]CommitCheckRun, error) {
	out := map[string][]CommitCheckRun{}
	for _, hash := range hashes {
		if runs := r.runs[hash]; len(runs) > 0 {
			out[hash] = runs
		}
	}
	return out, nil
}

func TestGetRepoHistoryFiltersGraphOnlyLines(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.HistoryLines = []string{
		"* 1234567 first commit",
		"|\\",
		"| * 89abcde second commit",
		"| |\\",
		"| | * fedcba9 third commit",
	}

	history, err := GetRepoHistory(context.Background(), RepoHistoryDeps{
		Git:     fake,
		RepoDir: "/fake/repo",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("GetRepoHistory returned error: %v", err)
	}

	expected := []string{
		"* 1234567 first commit",
		"| * 89abcde second commit",
		"| | * fedcba9 third commit",
	}
	if !reflect.DeepEqual(history.Lines, expected) {
		t.Fatalf("unexpected history lines: got %v want %v", history.Lines, expected)
	}
}

func TestGetRepoHistoryAttachesCommitChecks(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.HistoryLines = []string{"* abc1234 first commit"}
	fake.HistoryDetails = []RepoHistoryEntry{
		{Hash: "abc1234", Author: "Dev", Date: "2026-05-09", Subject: "first commit", Files: []string{"file.go"}},
	}
	run := CommitCheckRun{
		Kind:      CommitCheckKindPrecommit,
		Status:    CommitCheckStatusPassed,
		Command:   "custom check",
		Summary:   "checks passed",
		Timestamp: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
	}

	history, err := GetRepoHistory(context.Background(), RepoHistoryDeps{
		Git:           fake,
		RepoDir:       "/fake/repo",
		Limit:         10,
		IncludeChecks: true,
		CommitChecks:  fakeCommitCheckReader{runs: map[string][]CommitCheckRun{"abc1234": {run}}},
	})
	if err != nil {
		t.Fatalf("GetRepoHistory returned error: %v", err)
	}
	if len(history.Entries) != 1 {
		t.Fatalf("entries = %#v", history.Entries)
	}
	if len(history.Entries[0].Checks) != 1 || history.Entries[0].Checks[0].Command != "custom check" {
		t.Fatalf("checks = %#v", history.Entries[0].Checks)
	}
	if !fake.AssertCalled("LogDetails") {
		t.Fatalf("expected LogDetails when checks are requested")
	}
}

func TestScopeKeyForPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"scenarios/my-app/api/main.go", "scenario:my-app"},
		{"resources/postgres/config.yaml", "resource:postgres"},
		{"packages/api-core/storage/types.go", "package:api-core"},
		{"apps/dashboard/src/App.tsx", "app:dashboard"},
		{"services/auth/handler.go", "service:auth"},
		{"README.md", "other"},
		{".gitignore", "other"},
		{"docs/guide.md", "other"},
	}

	for _, tt := range tests {
		got := scopeKeyForPath(tt.path)
		if got != tt.want {
			t.Errorf("scopeKeyForPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDetectScopesUsesContractTargetKindsAndIDs(t *testing.T) {
	repoDir := t.TempDir()
	fixtures.WriteRepoContract(t, repoDir)
	fixtures.WriteFile(t, filepath.Join(repoDir, "internal", "tools", "compiler", "tool.json"), `{}`)

	got := detectScopes(repoDir, RepoFilesStatus{Untracked: []string{
		"internal/tools/compiler/main.go",
		"internal/other.go",
	}})
	if got["tool:compiler"][0] != "internal/tools/compiler/main.go" {
		t.Fatalf("tool scope = %#v, want compiler path", got["tool:compiler"])
	}
	if got["control-plane:internal"][0] != "internal/other.go" {
		t.Fatalf("control-plane scope = %#v, want internal path", got["control-plane:internal"])
	}
	if _, ok := got["scenario:compiler"]; ok {
		t.Fatal("contract target was incorrectly attributed by the fallback scenario vocabulary")
	}
}

func TestDetectScopesNoContractMatchesFallbackMap(t *testing.T) {
	files := RepoFilesStatus{
		Staged:    []string{"scenarios/app/api/main.go", "apps/dashboard/src/App.tsx"},
		Unstaged:  []string{"resources/postgres/config.yaml", "services/auth/handler.go"},
		Untracked: []string{"packages/api-core/types.go", "README.md"},
		Ignored:   []string{"docs/guide.md"},
	}
	got := detectScopes(t.TempDir(), files)
	want := map[string][]string{
		"scenario:app":      {"scenarios/app/api/main.go"},
		"app:dashboard":     {"apps/dashboard/src/App.tsx"},
		"resource:postgres": {"resources/postgres/config.yaml"},
		"service:auth":      {"services/auth/handler.go"},
		"package:api-core":  {"packages/api-core/types.go"},
		"other":             {"README.md", "docs/guide.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("no-contract scopes = %#v, want %#v", got, want)
	}
}

func TestDetectScopesCorruptContractMatchesAbsentContract(t *testing.T) {
	files := RepoFilesStatus{Untracked: []string{"scenarios/app/main.go", "internal/tools/compiler/main.go"}}
	absent := detectScopes(t.TempDir(), files)
	corruptDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(corruptDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, ".vrooli", "repo-contract.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	corrupt := detectScopes(corruptDir, files)
	if !reflect.DeepEqual(corrupt, absent) {
		t.Fatalf("corrupt-contract scopes = %#v, want absent-contract scopes %#v", corrupt, absent)
	}
}

// TestGetRepoStatus_StagedRenameDoesNotInflateAdditions guards the reporting
// defect where a staged rename was counted as a whole-file addition with no
// matching deletion. `git status --porcelain=v2` reports a rename as a single
// record naming only the destination, so a diff pathspec built from the status
// file lists hid the origin from git and left it unable to pair the halves.
// Across a large rename-heavy change set that inflated reported additions by
// the total length of every moved file.
func TestGetRepoStatus_StagedRenameDoesNotInflateAdditions(t *testing.T) {
	repoDir := t.TempDir()
	RunGitCommand(t, repoDir, "init")
	RunGitCommand(t, repoDir, "checkout", "-b", "main")

	body := strings.Repeat("a line of content\n", 500)
	WriteTestFile(t, filepath.Join(repoDir, "before.txt"), body)
	RunGitCommand(t, repoDir, "add", "-A")
	RunGitCommand(t, repoDir, "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-m", "seed")

	RunGitCommand(t, repoDir, "mv", "before.txt", "after.txt")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	})
	if err != nil {
		t.Fatalf("GetRepoStatus failed: %v", err)
	}

	if got := status.Files.Renames["after.txt"]; got != "before.txt" {
		t.Fatalf("expected rename origin before.txt, got %q (renames=%v)", got, status.Files.Renames)
	}

	stats, ok := status.FileStats.Staged["after.txt"]
	if !ok {
		t.Fatalf("expected staged stats for after.txt, got %v", status.FileStats.Staged)
	}
	if stats.Additions != 0 || stats.Deletions != 0 {
		t.Errorf("moving a file changes no lines: got +%d/-%d, want +0/-0", stats.Additions, stats.Deletions)
	}
	if !stats.IsRename || stats.OldPath != "before.txt" {
		t.Errorf("expected rename metadata pointing at before.txt, got is_rename=%v old_path=%q", stats.IsRename, stats.OldPath)
	}
	if _, ok := status.FileStats.Staged["before.txt"]; ok {
		t.Errorf("origin path must not appear as its own entry: %v", status.FileStats.Staged)
	}
}

// TestGetRepoStatus_StagedRenameWithEditsCountsOnlyTheEdit checks the other
// half of the contract: a rename that also changes content reports the edit,
// not the whole file.
func TestGetRepoStatus_StagedRenameWithEditsCountsOnlyTheEdit(t *testing.T) {
	repoDir := t.TempDir()
	RunGitCommand(t, repoDir, "init")
	RunGitCommand(t, repoDir, "checkout", "-b", "main")

	body := strings.Repeat("a line of content\n", 500)
	WriteTestFile(t, filepath.Join(repoDir, "before.txt"), body)
	RunGitCommand(t, repoDir, "add", "-A")
	RunGitCommand(t, repoDir, "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-m", "seed")

	RunGitCommand(t, repoDir, "mv", "before.txt", "after.txt")
	WriteTestFile(t, filepath.Join(repoDir, "after.txt"), body+"one appended line\n")
	RunGitCommand(t, repoDir, "add", "-A")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	})
	if err != nil {
		t.Fatalf("GetRepoStatus failed: %v", err)
	}

	stats := status.FileStats.Staged["after.txt"]
	if stats.Additions != 1 || stats.Deletions != 0 {
		t.Errorf("expected the single appended line only: got +%d/-%d, want +1/-0", stats.Additions, stats.Deletions)
	}
	if !stats.IsRename || stats.OldPath != "before.txt" {
		t.Errorf("expected rename metadata pointing at before.txt, got is_rename=%v old_path=%q", stats.IsRename, stats.OldPath)
	}
}

func TestWithRenameOrigins(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		renames map[string]string
		want    []string
	}{
		{
			name:  "no renames leaves paths untouched",
			paths: []string{"a.go", "b.go"},
			want:  []string{"a.go", "b.go"},
		},
		{
			name:    "origin is appended for each rename",
			paths:   []string{"new.go", "plain.go"},
			renames: map[string]string{"new.go": "old.go"},
			want:    []string{"new.go", "plain.go", "old.go"},
		},
		{
			name:    "renames outside paths are ignored",
			paths:   []string{"plain.go"},
			renames: map[string]string{"new.go": "old.go"},
			want:    []string{"plain.go"},
		},
		{
			name:    "an origin already present is not duplicated",
			paths:   []string{"new.go", "old.go"},
			renames: map[string]string{"new.go": "old.go"},
			want:    []string{"new.go", "old.go"},
		},
		{
			name:    "two renames sharing one origin append it once",
			paths:   []string{"copy-a.go", "copy-b.go"},
			renames: map[string]string{"copy-a.go": "source.go", "copy-b.go": "source.go"},
			want:    []string{"copy-a.go", "copy-b.go", "source.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withRenameOrigins(tt.paths, tt.renames)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("withRenameOrigins() = %v, want %v", got, tt.want)
			}
		})
	}
}
