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
