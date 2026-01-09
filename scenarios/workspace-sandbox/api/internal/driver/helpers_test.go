package driver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// =============================================================================
// getOverlayChangedFiles Tests
// =============================================================================

func TestGetOverlayChangedFiles_EmptyUpperDir(t *testing.T) {
	s := &types.Sandbox{UpperDir: ""}

	_, err := getOverlayChangedFiles(s)
	if err == nil {
		t.Error("expected error for empty upper directory")
	}
}

func TestGetOverlayChangedFiles_SkipsDirectories(t *testing.T) {
	// Create temp directories
	upperDir := t.TempDir()
	lowerDir := t.TempDir()

	// Create a nested directory structure with a file
	nestedDir := filepath.Join(upperDir, "path1", "path2")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file in the nested directory
	testFile := filepath.Join(nestedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
		LowerDir: lowerDir,
	}

	changes, err := getOverlayChangedFiles(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have one file, not the directories
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
		for _, c := range changes {
			t.Logf("  - %s (%s)", c.FilePath, c.ChangeType)
		}
	}

	if len(changes) == 1 && changes[0].FilePath != "path1/path2/test.txt" {
		t.Errorf("expected path1/path2/test.txt, got %s", changes[0].FilePath)
	}
}

func TestGetOverlayChangedFiles_DetectsAddedFiles(t *testing.T) {
	upperDir := t.TempDir()
	lowerDir := t.TempDir()

	// Create a new file in upper (not in lower)
	testFile := filepath.Join(upperDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
		LowerDir: lowerDir,
	}

	changes, err := getOverlayChangedFiles(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].ChangeType != types.ChangeTypeAdded {
		t.Errorf("expected ChangeTypeAdded, got %s", changes[0].ChangeType)
	}
}

func TestGetOverlayChangedFiles_DetectsModifiedFiles(t *testing.T) {
	upperDir := t.TempDir()
	lowerDir := t.TempDir()

	// Create same file in both, but with different content (different size)
	lowerFile := filepath.Join(lowerDir, "existing.txt")
	if err := os.WriteFile(lowerFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	upperFile := filepath.Join(upperDir, "existing.txt")
	if err := os.WriteFile(upperFile, []byte("modified content"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
		LowerDir: lowerDir,
	}

	changes, err := getOverlayChangedFiles(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].ChangeType != types.ChangeTypeModified {
		t.Errorf("expected ChangeTypeModified, got %s", changes[0].ChangeType)
	}
}

func TestGetOverlayChangedFiles_SkipsOverlayInternal(t *testing.T) {
	upperDir := t.TempDir()
	lowerDir := t.TempDir()

	// Create .overlay directory (internal overlayfs)
	overlayDir := filepath.Join(upperDir, ".overlay")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlayDir, "internal.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a normal file
	if err := os.WriteFile(filepath.Join(upperDir, "normal.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
		LowerDir: lowerDir,
	}

	changes, err := getOverlayChangedFiles(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have normal.txt, not .overlay/*
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}

	for _, c := range changes {
		if c.FilePath == ".overlay/internal.txt" {
			t.Error("should not include .overlay internal files")
		}
	}
}

func TestGetOverlayChangedFiles_HandlesWhiteoutMarkers(t *testing.T) {
	upperDir := t.TempDir()
	lowerDir := t.TempDir()

	// Create file in lower that we'll "delete"
	lowerFile := filepath.Join(lowerDir, "deleted.txt")
	if err := os.WriteFile(lowerFile, []byte("to be deleted"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create whiteout marker in upper (simulating deletion)
	// Note: We can't create actual character devices without root, so we create a regular file
	// The real whiteout detection happens via isWhiteout() which checks for char device
	whiteoutFile := filepath.Join(upperDir, ".wh.deleted.txt")
	if err := os.WriteFile(whiteoutFile, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
		LowerDir: lowerDir,
	}

	changes, err := getOverlayChangedFiles(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	// The whiteout marker should be converted to a delete for the target file
	if changes[0].FilePath != "deleted.txt" {
		t.Errorf("expected deleted.txt, got %s", changes[0].FilePath)
	}
	if changes[0].ChangeType != types.ChangeTypeDeleted {
		t.Errorf("expected ChangeTypeDeleted, got %s", changes[0].ChangeType)
	}
}

func TestGetOverlayChangedFiles_SkipsOpaqueMarker(t *testing.T) {
	upperDir := t.TempDir()
	lowerDir := t.TempDir()

	// Create opaque whiteout marker (directory replacement)
	if err := os.WriteFile(filepath.Join(upperDir, ".wh..opq"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a normal file
	if err := os.WriteFile(filepath.Join(upperDir, "normal.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
		LowerDir: lowerDir,
	}

	changes, err := getOverlayChangedFiles(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have normal.txt
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}

	for _, c := range changes {
		if c.FilePath == ".wh..opq" {
			t.Error("should not include opaque markers")
		}
	}
}

// =============================================================================
// removeFromUpperSecure Tests
// =============================================================================

func TestRemoveFromUpperSecure_RemovesFile(t *testing.T) {
	upperDir := t.TempDir()

	// Create a file to remove
	testFile := filepath.Join(upperDir, "toremove.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := removeFromUpperSecure(upperDir, "toremove.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestRemoveFromUpperSecure_IdempotentForNonexistent(t *testing.T) {
	upperDir := t.TempDir()

	// Should not error for non-existent file
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

	// Create the file
	testFile := filepath.Join(upperDir, "absolutepath.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Using absolute-looking path should work (strips leading /)
	err := removeFromUpperSecure(upperDir, "/absolutepath.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestRemoveFromUpperSecure_CleansEmptyParents(t *testing.T) {
	upperDir := t.TempDir()

	// Create nested file
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

	// Empty parent directories should be cleaned up
	if _, err := os.Stat(filepath.Join(upperDir, "a")); !os.IsNotExist(err) {
		t.Error("empty parent directory 'a' should have been removed")
	}
}

func TestRemoveFromUpperSecure_PreservesNonEmptyParents(t *testing.T) {
	upperDir := t.TempDir()

	// Create nested file
	nestedDir := filepath.Join(upperDir, "a", "b")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create two files - we'll remove one
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

	// Parent directory should still exist (has other file)
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("parent directory should be preserved (not empty)")
	}

	// The other file should still exist
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("sibling file should be preserved")
	}
}

// =============================================================================
// isMountPoint Tests
// =============================================================================

func TestIsMountPoint_ReturnsFalseForRegularDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Regular directory is not a mount point
	if isMountPoint(tmpDir) {
		t.Error("expected false for regular directory")
	}
}

func TestIsMountPoint_ReturnsFalseForNonexistent(t *testing.T) {
	if isMountPoint("/nonexistent/path/that/does/not/exist") {
		t.Error("expected false for non-existent path")
	}
}

// =============================================================================
// verifyOverlayMountIntegrity Tests
// =============================================================================

func TestVerifyOverlayMountIntegrity_RejectsEmptyMergedDir(t *testing.T) {
	s := &types.Sandbox{MergedDir: ""}

	err := verifyOverlayMountIntegrity(s)
	if err == nil {
		t.Error("expected error for empty merged directory")
	}
}

func TestVerifyOverlayMountIntegrity_RejectsNonexistentDir(t *testing.T) {
	s := &types.Sandbox{MergedDir: "/nonexistent/path"}

	err := verifyOverlayMountIntegrity(s)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestVerifyOverlayMountIntegrity_RejectsNonDirectory(t *testing.T) {
	// Create a regular file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &types.Sandbox{MergedDir: tmpFile}

	err := verifyOverlayMountIntegrity(s)
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}

func TestVerifyOverlayMountIntegrity_RejectsUnmounted(t *testing.T) {
	// A regular directory is not a mount point
	tmpDir := t.TempDir()

	s := &types.Sandbox{MergedDir: tmpDir}

	err := verifyOverlayMountIntegrity(s)
	if err == nil {
		t.Error("expected error for unmounted directory")
	}
}

// =============================================================================
// cleanupSandboxDir Tests
// =============================================================================

func TestCleanupSandboxDir_RemovesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	sandboxID := uuid.New()

	// Create sandbox directory structure
	sandboxDir := filepath.Join(baseDir, sandboxID.String())
	if err := os.MkdirAll(filepath.Join(sandboxDir, "upper"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandboxDir, "upper", "test.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Unmount function that does nothing (no actual mount)
	unmountCalled := false
	unmountFn := func() error {
		unmountCalled = true
		return nil
	}

	err := cleanupSandboxDir(baseDir, sandboxID, unmountFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify unmount was called
	if !unmountCalled {
		t.Error("unmount function should have been called")
	}

	// Verify directory is gone
	if _, err := os.Stat(sandboxDir); !os.IsNotExist(err) {
		t.Error("sandbox directory should have been removed")
	}
}

func TestCleanupSandboxDir_ContinuesOnUnmountError(t *testing.T) {
	baseDir := t.TempDir()
	sandboxID := uuid.New()

	// Create sandbox directory
	sandboxDir := filepath.Join(baseDir, sandboxID.String())
	if err := os.MkdirAll(sandboxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Unmount function that returns error
	unmountFn := func() error {
		return os.ErrNotExist // Simulate unmount failure
	}

	err := cleanupSandboxDir(baseDir, sandboxID, unmountFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should still be removed despite unmount error
	if _, err := os.Stat(sandboxDir); !os.IsNotExist(err) {
		t.Error("sandbox directory should have been removed even after unmount error")
	}
}

// =============================================================================
// isWhiteout Tests
// =============================================================================

func TestIsWhiteout_ReturnsFalseForRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(tmpFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if isWhiteout(tmpFile) {
		t.Error("regular file should not be detected as whiteout")
	}
}

func TestIsWhiteout_ReturnsFalseForNonexistent(t *testing.T) {
	if isWhiteout("/nonexistent/path") {
		t.Error("non-existent path should not be detected as whiteout")
	}
}

func TestIsWhiteout_ReturnsFalseForDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	if isWhiteout(tmpDir) {
		t.Error("directory should not be detected as whiteout")
	}
}

// Note: We can't test actual whiteout detection without root privileges
// to create character devices. The isWhiteout function is correct when
// tested against real overlayfs mounts.
