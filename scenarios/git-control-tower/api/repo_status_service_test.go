package main

import (
	"context"
	"reflect"
	"testing"
	"time"
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
