package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveFileContentSuccess(t *testing.T) {
	repoDir := SetupTestRepo(t)
	filePath := filepath.Join(repoDir, "src/main.ts")
	WriteTestFile(t, filePath, "const a = 1\n")

	beforeHash := hashContentBytes([]byte("const a = 1\n"))
	result, err := SaveFileContent(context.Background(), FileContentDeps{
		FS:      OSFileIO{},
		RepoDir: repoDir,
	}, SaveFileContentRequest{
		Path:         "src/main.ts",
		Content:      "const a = 2\n",
		ExpectedHash: beforeHash,
	})
	if err != nil {
		t.Fatalf("SaveFileContent returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true")
	}
	if result.Path != "src/main.ts" {
		t.Fatalf("expected path src/main.ts, got %s", result.Path)
	}

	written, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(written) != "const a = 2\n" {
		t.Fatalf("unexpected file content: %q", string(written))
	}
	if result.ContentHash != hashContentBytes(written) {
		t.Fatalf("content hash mismatch: got %s", result.ContentHash)
	}
}

func TestSaveFileContentConflict(t *testing.T) {
	repoDir := SetupTestRepo(t)
	WriteTestFile(t, filepath.Join(repoDir, "README.md"), "hello\n")

	_, err := SaveFileContent(context.Background(), FileContentDeps{
		FS:      OSFileIO{},
		RepoDir: repoDir,
	}, SaveFileContentRequest{
		Path:         "README.md",
		Content:      "updated\n",
		ExpectedHash: "stale-hash",
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	var conflictErr *FileContentConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected FileContentConflictError, got %T (%v)", err, err)
	}
	if conflictErr.CurrentHash == "" {
		t.Fatal("expected current hash in conflict error")
	}
}

func TestSaveFileContentRejectsTraversal(t *testing.T) {
	repoDir := SetupTestRepo(t)

	_, err := SaveFileContent(context.Background(), FileContentDeps{
		FS:      OSFileIO{},
		RepoDir: repoDir,
	}, SaveFileContentRequest{
		Path:    "../outside.txt",
		Content: "nope",
	})
	if err == nil {
		t.Fatal("expected invalid path error")
	}
}

func TestSaveFileContentRejectsBinary(t *testing.T) {
	repoDir := SetupTestRepo(t)
	binPath := filepath.Join(repoDir, "assets/logo.png")
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(binPath, []byte{0x89, 0x50, 0x4e, 0x47, 0x0, 0x1}, 0o644); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	_, err := SaveFileContent(context.Background(), FileContentDeps{
		FS:      OSFileIO{},
		RepoDir: repoDir,
	}, SaveFileContentRequest{
		Path:    "assets/logo.png",
		Content: "text",
	})
	if err == nil {
		t.Fatal("expected unsupported binary error")
	}
	var unsupportedErr *UnsupportedBinaryError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected UnsupportedBinaryError, got %T (%v)", err, err)
	}
}

func TestSaveFileContentRejectsTooLarge(t *testing.T) {
	repoDir := SetupTestRepo(t)
	largePath := filepath.Join(repoDir, "large.txt")
	tooLarge := bytes.Repeat([]byte("a"), int(maxDiffFileBytes)+1)
	if err := os.WriteFile(largePath, tooLarge, 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}

	_, err := SaveFileContent(context.Background(), FileContentDeps{
		FS:      OSFileIO{},
		RepoDir: repoDir,
	}, SaveFileContentRequest{
		Path:    "large.txt",
		Content: "x",
	})
	if err == nil {
		t.Fatal("expected file too large error")
	}
	var tooLargeErr *FileTooLargeError
	if !errors.As(err, &tooLargeErr) {
		t.Fatalf("expected FileTooLargeError, got %T (%v)", err, err)
	}
}
