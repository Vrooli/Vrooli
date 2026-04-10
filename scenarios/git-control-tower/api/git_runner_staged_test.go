package main

import (
	"context"
	"sort"
	"testing"
)

func TestFakeGitRunner_ListStagedFiles(t *testing.T) {
	fake := NewFakeGitRunner().
		AddStagedFile("src/api.go").
		AddStagedFile("src/handler.go")

	files, err := fake.ListStagedFiles(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(files)
	if len(files) != 2 {
		t.Fatalf("expected 2 staged files, got %d", len(files))
	}
	if files[0] != "src/api.go" || files[1] != "src/handler.go" {
		t.Errorf("unexpected staged files: %v", files)
	}
}

func TestFakeGitRunner_ListStagedFiles_Empty(t *testing.T) {
	fake := NewFakeGitRunner()

	files, err := fake.ListStagedFiles(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 staged files, got %d", len(files))
	}
}
