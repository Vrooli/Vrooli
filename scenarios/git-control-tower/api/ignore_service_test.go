package main

import (
	"context"
	"strings"
	"testing"
)

func TestIgnorePath_ProjectLevel(t *testing.T) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO()
	repoDir := "/fake/repo"

	deps := IgnoreDeps{Git: git, FS: fs, RepoDir: repoDir}
	req := IgnoreRequest{Path: "build/output.log"}

	result, err := IgnorePath(context.Background(), deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}
	if result.GitignorePath != repoDir+"/.gitignore" {
		t.Fatalf("expected root gitignore, got %s", result.GitignorePath)
	}

	// Verify the entry was written to the root .gitignore.
	content, ok := fs.Files[repoDir+"/.gitignore"]
	if !ok {
		t.Fatal("expected .gitignore to be written")
	}
	if !strings.Contains(content, "build/output.log") {
		t.Fatalf("expected entry in .gitignore, got: %s", content)
	}
}

func TestIgnorePath_GroupLevel(t *testing.T) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO()
	repoDir := "/fake/repo"

	deps := IgnoreDeps{Git: git, FS: fs, RepoDir: repoDir}
	req := IgnoreRequest{
		Path:     "scenarios/foo/build/out.log",
		Level:    "group",
		GroupDir: "scenarios/foo/",
	}

	result, err := IgnorePath(context.Background(), deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}
	if result.GitignorePath != repoDir+"/scenarios/foo/.gitignore" {
		t.Fatalf("expected group gitignore, got %s", result.GitignorePath)
	}

	content, ok := fs.Files[repoDir+"/scenarios/foo/.gitignore"]
	if !ok {
		t.Fatal("expected group .gitignore to be written")
	}
	if !strings.Contains(content, "build/out.log") {
		t.Fatalf("expected stripped entry, got: %s", content)
	}
}

func TestIgnorePath_GroupLevel_CreatesGitignore(t *testing.T) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO()
	repoDir := "/fake/repo"

	deps := IgnoreDeps{Git: git, FS: fs, RepoDir: repoDir}
	req := IgnoreRequest{
		Path:     "resources/postgres/data",
		Level:    "group",
		GroupDir: "resources/postgres",
	}

	result, err := IgnorePath(context.Background(), deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}

	// Verify a new .gitignore was created in the group dir.
	content, ok := fs.Files[repoDir+"/resources/postgres/.gitignore"]
	if !ok {
		t.Fatal("expected group .gitignore to be created")
	}
	if !strings.Contains(content, "data") {
		t.Fatalf("expected entry 'data' in group .gitignore, got: %s", content)
	}
}

func TestIgnorePath_GroupLevel_NoGroupDir(t *testing.T) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO()
	repoDir := "/fake/repo"

	deps := IgnoreDeps{Git: git, FS: fs, RepoDir: repoDir}
	req := IgnoreRequest{
		Path:  "foo/bar.txt",
		Level: "group",
		// GroupDir intentionally omitted.
	}

	result, err := IgnorePath(context.Background(), deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure when GroupDir is empty")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected error message")
	}
}

func TestIgnorePath_InvalidPath(t *testing.T) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO()
	repoDir := "/fake/repo"

	deps := IgnoreDeps{Git: git, FS: fs, RepoDir: repoDir}
	req := IgnoreRequest{Path: "../../../etc/passwd"}

	result, err := IgnorePath(context.Background(), deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure for path traversal")
	}
}

func TestIgnorePath_Deduplication(t *testing.T) {
	git := NewFakeGitRunner()
	fs := NewFakeFileIO().WithFile("/fake/repo/.gitignore", "build/output.log\n")
	repoDir := "/fake/repo"

	deps := IgnoreDeps{Git: git, FS: fs, RepoDir: repoDir}
	req := IgnoreRequest{Path: "build/output.log"}

	result, err := IgnorePath(context.Background(), deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}

	// File content should not have the entry duplicated.
	content := fs.Files[repoDir+"/.gitignore"]
	count := strings.Count(content, "build/output.log")
	if count != 1 {
		t.Fatalf("expected 1 occurrence, got %d in: %s", count, content)
	}
}
