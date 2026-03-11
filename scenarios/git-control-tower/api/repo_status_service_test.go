package main

import (
	"context"
	"reflect"
	"testing"
)

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
