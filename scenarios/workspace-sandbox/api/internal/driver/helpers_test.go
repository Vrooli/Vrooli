package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// Change-detection unit tests live in
// internal/driver/changedetect/walker_contract_test.go where they
// parameterise across overlay + copy strategies.

// =============================================================================
// removeFromUpperSecure Tests
// =============================================================================

func TestRemoveFromUpperSecure_RemovesFile(t *testing.T) {
	upperDir := t.TempDir()

	testFile := filepath.Join(upperDir, "toremove.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := removeFromUpperSecure(upperDir, "toremove.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestRemoveFromUpperSecure_IdempotentForNonexistent(t *testing.T) {
	upperDir := t.TempDir()

	err := removeFromUpperSecure(upperDir, "doesnotexist.txt")
	if err != nil {
		t.Errorf("expected no error for non-existent file, got: %v", err)
	}
}

func TestRemoveFromUpperSecure_RejectsPathTraversal(t *testing.T) {
	upperDir := t.TempDir()

	testCases := []string{
		"../escape.txt",
		"../../etc/passwd",
		"subdir/../../escape.txt",
	}

	for _, tc := range testCases {
		err := removeFromUpperSecure(upperDir, tc)
		if err == nil {
			t.Errorf("expected error for path traversal attempt: %s", tc)
		}
	}
}

func TestRemoveFromUpperSecure_HandlesAbsolutePaths(t *testing.T) {
	upperDir := t.TempDir()

	testFile := filepath.Join(upperDir, "absolutepath.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := removeFromUpperSecure(upperDir, "/absolutepath.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestRemoveFromUpperSecure_CleansEmptyParents(t *testing.T) {
	upperDir := t.TempDir()

	nestedDir := filepath.Join(upperDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(nestedDir, "deep.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := removeFromUpperSecure(upperDir, "a/b/c/deep.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(upperDir, "a")); !os.IsNotExist(err) {
		t.Error("empty parent directory 'a' should have been removed")
	}
}

func TestRemoveFromUpperSecure_PreservesNonEmptyParents(t *testing.T) {
	upperDir := t.TempDir()

	nestedDir := filepath.Join(upperDir, "a", "b")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	file1 := filepath.Join(nestedDir, "keep.txt")
	file2 := filepath.Join(nestedDir, "remove.txt")
	if err := os.WriteFile(file1, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("remove"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := removeFromUpperSecure(upperDir, "a/b/remove.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("parent directory should be preserved (not empty)")
	}

	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("sibling file should be preserved")
	}
}

// =============================================================================
// isMountPoint Tests
// =============================================================================

func TestIsMountPoint_ReturnsFalseForRegularDir(t *testing.T) {
	tmpDir := t.TempDir()

	if testMounter().IsMountPoint(tmpDir) {
		t.Error("expected false for regular directory")
	}
}

func TestIsMountPoint_ReturnsFalseForNonexistent(t *testing.T) {
	if testMounter().IsMountPoint("/nonexistent/path/that/does/not/exist") {
		t.Error("expected false for non-existent path")
	}
}

// =============================================================================
// verifyOverlayMountIntegrity Tests
// =============================================================================

func TestVerifyOverlayMountIntegrity_RejectsEmptyMergedDir(t *testing.T) {
	s := &types.Sandbox{MergedDir: ""}

	err := verifyOverlayMountIntegrity(testMounter(), s)
	if err == nil {
		t.Error("expected error for empty merged directory")
	}
}

func TestVerifyOverlayMountIntegrity_RejectsNonexistentDir(t *testing.T) {
	s := &types.Sandbox{MergedDir: "/nonexistent/path"}

	err := verifyOverlayMountIntegrity(testMounter(), s)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestVerifyOverlayMountIntegrity_RejectsNonDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{MergedDir: tmpFile}

	err := verifyOverlayMountIntegrity(testMounter(), s)
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}

func TestVerifyOverlayMountIntegrity_RejectsUnmounted(t *testing.T) {
	tmpDir := t.TempDir()

	s := &types.Sandbox{MergedDir: tmpDir}

	err := verifyOverlayMountIntegrity(testMounter(), s)
	if err == nil {
		t.Error("expected error for unmounted directory")
	}
}

// =============================================================================
// cleanupSandboxDirAll Tests
// =============================================================================

func TestCleanupSandboxDirAll_RemovesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	sandboxID := uuid.New()

	sandboxDir := filepath.Join(baseDir, sandboxID.String())
	if err := os.MkdirAll(filepath.Join(sandboxDir, "upper"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandboxDir, "upper", "test.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := cleanupSandboxDirAll(context.Background(), testMounter(), baseDir, t.TempDir(), sandboxID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(sandboxDir); !os.IsNotExist(err) {
		t.Error("sandbox directory should have been removed")
	}
}

func TestCleanupSandboxDirAll_MissingDirIsNoop(t *testing.T) {
	baseDir := t.TempDir()
	sandboxID := uuid.New()

	if err := cleanupSandboxDirAll(context.Background(), testMounter(), baseDir, t.TempDir(), sandboxID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
