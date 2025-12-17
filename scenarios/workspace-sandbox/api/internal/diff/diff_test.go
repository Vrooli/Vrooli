package diff

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// TestIsBinary tests binary file detection
func TestIsBinary(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "plain text",
			content:  []byte("Hello, World!\nThis is plain text."),
			expected: false,
		},
		{
			name:     "text with special chars",
			content:  []byte("Special chars: é ñ ü ö"),
			expected: false,
		},
		{
			name:     "binary with null byte at start",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			expected: true,
		},
		{
			name:     "binary with null byte in middle",
			content:  []byte("text before\x00text after"),
			expected: true,
		},
		{
			name:     "PNG header simulation",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00},
			expected: true,
		},
		{
			name:     "large text file",
			content:  []byte(strings.Repeat("This is a line of text.\n", 1000)),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinary(tt.content)
			if result != tt.expected {
				t.Errorf("isBinary() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestIsBinaryLargeFile tests that binary detection only checks first 8KB
func TestIsBinaryLargeFile(t *testing.T) {
	// Create content with null byte after 8KB
	content := make([]byte, 10000)
	for i := 0; i < 10000; i++ {
		content[i] = 'a'
	}
	// Place null byte after the 8KB check window
	content[9000] = 0x00

	if isBinary(content) {
		t.Error("isBinary should not detect null byte after 8KB window")
	}

	// Place null byte within the 8KB check window
	content[7000] = 0x00
	if !isBinary(content) {
		t.Error("isBinary should detect null byte within 8KB window")
	}
}

// TestDiffNewFile tests generating diffs for new files
func TestDiffNewFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	t.Run("new text file", func(t *testing.T) {
		// Create a text file
		content := "line 1\nline 2\nline 3\n"
		filePath := filepath.Join(tmpDir, "newfile.txt")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		diff, err := gen.diffNewFile(ctx, tmpDir, "newfile.txt")
		if err != nil {
			t.Fatalf("diffNewFile failed: %v", err)
		}

		// Check diff header
		if !strings.Contains(diff, "diff --git a/newfile.txt b/newfile.txt") {
			t.Error("diff should contain git-style header")
		}
		if !strings.Contains(diff, "new file mode") {
			t.Error("diff should indicate new file")
		}
		if !strings.Contains(diff, "--- /dev/null") {
			t.Error("diff should show /dev/null as source")
		}
		if !strings.Contains(diff, "+++ b/newfile.txt") {
			t.Error("diff should show destination file")
		}
		// Check that lines are prefixed with +
		if !strings.Contains(diff, "+line 1") {
			t.Error("diff should show added lines with + prefix")
		}
	})

	t.Run("new empty file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "empty.txt")
		if err := os.WriteFile(filePath, []byte{}, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		diff, err := gen.diffNewFile(ctx, tmpDir, "empty.txt")
		if err != nil {
			t.Fatalf("diffNewFile failed: %v", err)
		}

		if !strings.Contains(diff, "new file mode") {
			t.Error("empty file diff should still indicate new file")
		}
	})

	t.Run("new binary file", func(t *testing.T) {
		binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00}
		filePath := filepath.Join(tmpDir, "image.png")
		if err := os.WriteFile(filePath, binaryContent, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		diff, err := gen.diffNewFile(ctx, tmpDir, "image.png")
		if err != nil {
			t.Fatalf("diffNewFile failed: %v", err)
		}

		if !strings.Contains(diff, "Binary file") {
			t.Error("binary file diff should indicate binary")
		}
	})

	t.Run("new directory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "newdir")
		if err := os.Mkdir(dirPath, 0o755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		diff, err := gen.diffNewFile(ctx, tmpDir, "newdir")
		if err != nil {
			t.Fatalf("diffNewFile failed: %v", err)
		}

		if !strings.Contains(diff, "new file mode 040755") {
			t.Error("directory diff should show directory mode")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := gen.diffNewFile(ctx, tmpDir, "nonexistent.txt")
		if err == nil {
			t.Error("diffNewFile should fail for nonexistent file")
		}
	})
}

// TestDiffDeletedFile tests generating diffs for deleted files
func TestDiffDeletedFile(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	t.Run("deleted text file", func(t *testing.T) {
		content := "line 1\nline 2\n"
		filePath := filepath.Join(tmpDir, "deleted.txt")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		diff, err := gen.diffDeletedFile(ctx, tmpDir, "deleted.txt")
		if err != nil {
			t.Fatalf("diffDeletedFile failed: %v", err)
		}

		if !strings.Contains(diff, "deleted file mode") {
			t.Error("diff should indicate deleted file")
		}
		if !strings.Contains(diff, "+++ /dev/null") {
			t.Error("diff should show /dev/null as destination")
		}
		if !strings.Contains(diff, "-line 1") {
			t.Error("diff should show deleted lines with - prefix")
		}
	})

	t.Run("deleted binary file", func(t *testing.T) {
		binaryContent := []byte{0x00, 0x01, 0x02}
		filePath := filepath.Join(tmpDir, "binary.bin")
		if err := os.WriteFile(filePath, binaryContent, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		diff, err := gen.diffDeletedFile(ctx, tmpDir, "binary.bin")
		if err != nil {
			t.Fatalf("diffDeletedFile failed: %v", err)
		}

		if !strings.Contains(diff, "Binary file") {
			t.Error("binary file diff should indicate binary")
		}
	})

	t.Run("deleted directory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "deleteddir")
		if err := os.Mkdir(dirPath, 0o755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		diff, err := gen.diffDeletedFile(ctx, tmpDir, "deleteddir")
		if err != nil {
			t.Fatalf("diffDeletedFile failed: %v", err)
		}

		if !strings.Contains(diff, "deleted file mode 040755") {
			t.Error("directory diff should show directory mode")
		}
	})
}

// TestDiffModifiedFile tests generating diffs for modified files
func TestDiffModifiedFile(t *testing.T) {
	lowerDir := t.TempDir()
	upperDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	t.Run("modified text file", func(t *testing.T) {
		// Create original file in lower
		originalContent := "line 1\nline 2\nline 3\n"
		if err := os.WriteFile(filepath.Join(lowerDir, "file.txt"), []byte(originalContent), 0o644); err != nil {
			t.Fatalf("Failed to create original file: %v", err)
		}

		// Create modified file in upper
		modifiedContent := "line 1\nmodified line 2\nline 3\nnew line 4\n"
		if err := os.WriteFile(filepath.Join(upperDir, "file.txt"), []byte(modifiedContent), 0o644); err != nil {
			t.Fatalf("Failed to create modified file: %v", err)
		}

		diff, err := gen.diffModifiedFile(ctx, lowerDir, upperDir, "file.txt")
		if err != nil {
			t.Fatalf("diffModifiedFile failed: %v", err)
		}

		// Should have actual diff content
		if diff == "" {
			t.Error("diff should not be empty for modified file")
		}
		if !strings.Contains(diff, "diff --git") {
			t.Error("diff should contain git-style header")
		}
	})

	t.Run("identical files", func(t *testing.T) {
		content := "same content\n"
		if err := os.WriteFile(filepath.Join(lowerDir, "same.txt"), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(upperDir, "same.txt"), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		diff, err := gen.diffModifiedFile(ctx, lowerDir, upperDir, "same.txt")
		if err != nil {
			t.Fatalf("diffModifiedFile failed: %v", err)
		}

		// Identical files should produce empty diff
		if diff != "" {
			t.Error("identical files should produce empty diff")
		}
	})
}

// TestGenerateDiff tests the main diff generation function
func TestGenerateDiff(t *testing.T) {
	lowerDir := t.TempDir()
	upperDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	sandboxID := uuid.New()
	sandbox := &types.Sandbox{
		ID:       sandboxID,
		LowerDir: lowerDir,
		UpperDir: upperDir,
	}

	t.Run("empty changes list", func(t *testing.T) {
		result, err := gen.GenerateDiff(ctx, sandbox, []*types.FileChange{})
		if err != nil {
			t.Fatalf("GenerateDiff failed: %v", err)
		}

		if result.SandboxID != sandboxID {
			t.Error("result should have correct sandbox ID")
		}
		if len(result.Files) != 0 {
			t.Error("result should have empty files list")
		}
		if result.UnifiedDiff != "" {
			t.Error("result should have empty diff for no changes")
		}
	})

	t.Run("missing upper dir", func(t *testing.T) {
		badSandbox := &types.Sandbox{
			ID:       sandboxID,
			LowerDir: lowerDir,
			UpperDir: "",
		}
		_, err := gen.GenerateDiff(ctx, badSandbox, []*types.FileChange{})
		if err == nil {
			t.Error("should fail with empty upper dir")
		}
		if !strings.Contains(err.Error(), "upper directory path is empty") {
			t.Errorf("error message should mention empty upper dir, got: %v", err)
		}
	})

	t.Run("missing lower dir", func(t *testing.T) {
		badSandbox := &types.Sandbox{
			ID:       sandboxID,
			LowerDir: "",
			UpperDir: upperDir,
		}
		_, err := gen.GenerateDiff(ctx, badSandbox, []*types.FileChange{})
		if err == nil {
			t.Error("should fail with empty lower dir")
		}
	})

	t.Run("nonexistent upper dir", func(t *testing.T) {
		badSandbox := &types.Sandbox{
			ID:       sandboxID,
			LowerDir: lowerDir,
			UpperDir: "/nonexistent/path",
		}
		_, err := gen.GenerateDiff(ctx, badSandbox, []*types.FileChange{})
		if err == nil {
			t.Error("should fail with nonexistent upper dir")
		}
	})

	t.Run("mixed changes", func(t *testing.T) {
		// Create files
		if err := os.WriteFile(filepath.Join(upperDir, "new.txt"), []byte("new content\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lowerDir, "deleted.txt"), []byte("deleted content\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		changes := []*types.FileChange{
			{ID: uuid.New(), FilePath: "new.txt", ChangeType: types.ChangeTypeAdded},
			{ID: uuid.New(), FilePath: "deleted.txt", ChangeType: types.ChangeTypeDeleted},
		}

		result, err := gen.GenerateDiff(ctx, sandbox, changes)
		if err != nil {
			t.Fatalf("GenerateDiff failed: %v", err)
		}

		if result.TotalAdded != 1 {
			t.Errorf("TotalAdded = %d, want 1", result.TotalAdded)
		}
		if result.TotalDeleted != 1 {
			t.Errorf("TotalDeleted = %d, want 1", result.TotalDeleted)
		}
		if result.Generated.IsZero() {
			t.Error("Generated timestamp should be set")
		}
	})

	t.Run("changes are sorted", func(t *testing.T) {
		// Create files
		if err := os.WriteFile(filepath.Join(upperDir, "z.txt"), []byte("z\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(upperDir, "a.txt"), []byte("a\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		changes := []*types.FileChange{
			{ID: uuid.New(), FilePath: "z.txt", ChangeType: types.ChangeTypeAdded},
			{ID: uuid.New(), FilePath: "a.txt", ChangeType: types.ChangeTypeAdded},
		}

		result, err := gen.GenerateDiff(ctx, sandbox, changes)
		if err != nil {
			t.Fatalf("GenerateDiff failed: %v", err)
		}

		// Files should be sorted alphabetically in the result
		if len(result.Files) != 2 {
			t.Fatal("expected 2 files")
		}
		if result.Files[0].FilePath != "a.txt" {
			t.Errorf("first file should be a.txt, got %s", result.Files[0].FilePath)
		}
		if result.Files[1].FilePath != "z.txt" {
			t.Errorf("second file should be z.txt, got %s", result.Files[1].FilePath)
		}
	})
}

// TestFilterChanges tests filtering file changes by ID
func TestFilterChanges(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	changes := []*types.FileChange{
		{ID: id1, FilePath: "file1.txt"},
		{ID: id2, FilePath: "file2.txt"},
		{ID: id3, FilePath: "file3.txt"},
	}

	t.Run("filter single id", func(t *testing.T) {
		filtered := FilterChanges(changes, []uuid.UUID{id2})
		if len(filtered) != 1 {
			t.Fatalf("expected 1 result, got %d", len(filtered))
		}
		if filtered[0].ID != id2 {
			t.Error("filtered result should have matching ID")
		}
	})

	t.Run("filter multiple ids", func(t *testing.T) {
		filtered := FilterChanges(changes, []uuid.UUID{id1, id3})
		if len(filtered) != 2 {
			t.Fatalf("expected 2 results, got %d", len(filtered))
		}
	})

	t.Run("filter with no matches", func(t *testing.T) {
		filtered := FilterChanges(changes, []uuid.UUID{uuid.New()})
		if len(filtered) != 0 {
			t.Errorf("expected 0 results, got %d", len(filtered))
		}
	})

	t.Run("filter with empty id list", func(t *testing.T) {
		filtered := FilterChanges(changes, []uuid.UUID{})
		if len(filtered) != 0 {
			t.Errorf("expected 0 results, got %d", len(filtered))
		}
	})

	t.Run("filter with nil changes", func(t *testing.T) {
		filtered := FilterChanges(nil, []uuid.UUID{id1})
		if filtered != nil {
			t.Error("filtering nil should return nil")
		}
	})
}

// TestCopyFile tests the file copy utility
func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("copy regular file", func(t *testing.T) {
		content := "test content\n"
		src := filepath.Join(tmpDir, "source.txt")
		dst := filepath.Join(tmpDir, "dest.txt")

		if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		// Verify content
		dstContent, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("Failed to read destination: %v", err)
		}
		if string(dstContent) != content {
			t.Error("copied content doesn't match")
		}

		// Verify permissions preserved
		srcInfo, _ := os.Stat(src)
		dstInfo, _ := os.Stat(dst)
		if srcInfo.Mode() != dstInfo.Mode() {
			t.Error("file mode not preserved")
		}
	})

	t.Run("copy nonexistent source", func(t *testing.T) {
		err := copyFile("/nonexistent/file", filepath.Join(tmpDir, "dest.txt"))
		if err == nil {
			t.Error("should fail for nonexistent source")
		}
	})

	t.Run("copy to invalid destination", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source2.txt")
		if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		err := copyFile(src, "/nonexistent/dir/file.txt")
		if err == nil {
			t.Error("should fail for invalid destination path")
		}
	})
}

// TestCopyChanges tests copying changes from sandbox to target
func TestCopyChanges(t *testing.T) {
	upperDir := t.TempDir()
	targetDir := t.TempDir()
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:       uuid.New(),
		UpperDir: upperDir,
	}

	t.Run("copy added file", func(t *testing.T) {
		// Create file in upper
		if err := os.WriteFile(filepath.Join(upperDir, "added.txt"), []byte("new file\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		changes := []*types.FileChange{
			{ID: uuid.New(), FilePath: "added.txt", ChangeType: types.ChangeTypeAdded},
		}

		if err := CopyChanges(ctx, sandbox, changes, targetDir); err != nil {
			t.Fatalf("CopyChanges failed: %v", err)
		}

		// Verify file copied
		if _, err := os.Stat(filepath.Join(targetDir, "added.txt")); os.IsNotExist(err) {
			t.Error("file should have been copied to target")
		}
	})

	t.Run("copy deleted file", func(t *testing.T) {
		// Create file in target that will be deleted
		targetFile := filepath.Join(targetDir, "todelete.txt")
		if err := os.WriteFile(targetFile, []byte("delete me"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		changes := []*types.FileChange{
			{ID: uuid.New(), FilePath: "todelete.txt", ChangeType: types.ChangeTypeDeleted},
		}

		if err := CopyChanges(ctx, sandbox, changes, targetDir); err != nil {
			t.Fatalf("CopyChanges failed: %v", err)
		}

		// Verify file deleted
		if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
			t.Error("file should have been deleted from target")
		}
	})

	t.Run("copy to nested directory", func(t *testing.T) {
		// Create file in nested dir in upper
		nestedDir := filepath.Join(upperDir, "nested", "dir")
		if err := os.MkdirAll(nestedDir, 0o755); err != nil {
			t.Fatalf("Failed to create nested dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(nestedDir, "file.txt"), []byte("nested\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		changes := []*types.FileChange{
			{ID: uuid.New(), FilePath: "nested/dir/file.txt", ChangeType: types.ChangeTypeAdded},
		}

		if err := CopyChanges(ctx, sandbox, changes, targetDir); err != nil {
			t.Fatalf("CopyChanges failed: %v", err)
		}

		// Verify file copied with directory structure
		if _, err := os.Stat(filepath.Join(targetDir, "nested", "dir", "file.txt")); os.IsNotExist(err) {
			t.Error("nested file should have been copied")
		}
	})
}

// TestGenerateFileDiff tests generating diff for a single file
func TestGenerateFileDiff(t *testing.T) {
	lowerDir := t.TempDir()
	upperDir := t.TempDir()
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:       uuid.New(),
		LowerDir: lowerDir,
		UpperDir: upperDir,
	}

	t.Run("added file", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(upperDir, "new.txt"), []byte("content\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		change := &types.FileChange{
			ID:         uuid.New(),
			FilePath:   "new.txt",
			ChangeType: types.ChangeTypeAdded,
		}

		diff, err := GenerateFileDiff(ctx, sandbox, change)
		if err != nil {
			t.Fatalf("GenerateFileDiff failed: %v", err)
		}
		if !strings.Contains(diff, "new file") {
			t.Error("diff should indicate new file")
		}
	})

	t.Run("deleted file", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(lowerDir, "old.txt"), []byte("content\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		change := &types.FileChange{
			ID:         uuid.New(),
			FilePath:   "old.txt",
			ChangeType: types.ChangeTypeDeleted,
		}

		diff, err := GenerateFileDiff(ctx, sandbox, change)
		if err != nil {
			t.Fatalf("GenerateFileDiff failed: %v", err)
		}
		if !strings.Contains(diff, "deleted file") {
			t.Error("diff should indicate deleted file")
		}
	})

	t.Run("unknown change type", func(t *testing.T) {
		change := &types.FileChange{
			ID:         uuid.New(),
			FilePath:   "file.txt",
			ChangeType: "unknown",
		}

		_, err := GenerateFileDiff(ctx, sandbox, change)
		if err == nil {
			t.Error("should fail for unknown change type")
		}
	})
}

// TestPatcher tests the patch application
func TestPatcher(t *testing.T) {
	patcher := NewPatcher()
	ctx := context.Background()

	t.Run("apply empty diff", func(t *testing.T) {
		result, err := patcher.ApplyDiff(ctx, t.TempDir(), "", ApplyOptions{})
		if err != nil {
			t.Fatalf("ApplyDiff failed: %v", err)
		}
		if !result.Success {
			t.Error("empty diff should succeed")
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		targetDir := t.TempDir()

		// Create a file that would be modified
		if err := os.WriteFile(filepath.Join(targetDir, "test.txt"), []byte("original\n"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Simple patch that would modify the file
		diff := `--- a/test.txt
+++ b/test.txt
@@ -1 +1 @@
-original
+modified
`

		result, err := patcher.ApplyDiff(ctx, targetDir, diff, ApplyOptions{DryRun: true})
		if err != nil {
			t.Fatalf("ApplyDiff failed: %v", err)
		}

		// In dry run, file should be unchanged
		content, _ := os.ReadFile(filepath.Join(targetDir, "test.txt"))
		if string(content) == "modified\n" {
			t.Error("dry run should not modify files")
		}

		// Should still report success or at least not error
		_ = result // Result depends on whether git is available
	})
}

// TestIsGitRepo tests git repository detection
func TestIsGitRepo(t *testing.T) {
	t.Run("non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		if isGitRepo(tmpDir) {
			t.Error("temp directory should not be a git repo")
		}
	})

	// Note: We don't test actual git repos here as that would require
	// initializing one, which is heavy for unit tests
}

// TestNewGenerator verifies generator creation
func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	if gen == nil {
		t.Error("NewGenerator should return non-nil generator")
	}
}

// TestNewPatcher verifies patcher creation
func TestNewPatcher(t *testing.T) {
	patcher := NewPatcher()
	if patcher == nil {
		t.Error("NewPatcher should return non-nil patcher")
	}
}

// Benchmark for isBinary detection
func BenchmarkIsBinary(b *testing.B) {
	content := []byte(strings.Repeat("This is a line of text.\n", 500))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBinary(content)
	}
}

// Benchmark for FilterChanges
func BenchmarkFilterChanges(b *testing.B) {
	changes := make([]*types.FileChange, 1000)
	filterIDs := make([]uuid.UUID, 100)

	for i := 0; i < 1000; i++ {
		changes[i] = &types.FileChange{ID: uuid.New(), FilePath: "file.txt"}
	}
	for i := 0; i < 100; i++ {
		filterIDs[i] = changes[i*10].ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterChanges(changes, filterIDs)
	}
}

// --- Hunk-Level Approval Tests [OT-P1-001] ---

// TestParseUnifiedDiff tests parsing unified diffs into structured format
func TestParseUnifiedDiff(t *testing.T) {
	t.Run("parse single file with multiple hunks", func(t *testing.T) {
		diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
+newline
 line2
 line3
@@ -10,2 +11,3 @@
 line10
+anotherline
 line11
`
		files := ParseUnifiedDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}

		file := files[0]
		if file.Path != "file.txt" {
			t.Errorf("expected path 'file.txt', got '%s'", file.Path)
		}
		if file.ChangeType != types.ChangeTypeModified {
			t.Errorf("expected modified change type, got %s", file.ChangeType)
		}
		if len(file.Hunks) != 2 {
			t.Fatalf("expected 2 hunks, got %d", len(file.Hunks))
		}

		// Check first hunk
		hunk1 := file.Hunks[0]
		if hunk1.OldStart != 1 || hunk1.OldCount != 3 {
			t.Errorf("hunk1 old range: expected 1,3, got %d,%d", hunk1.OldStart, hunk1.OldCount)
		}
		if hunk1.NewStart != 1 || hunk1.NewCount != 4 {
			t.Errorf("hunk1 new range: expected 1,4, got %d,%d", hunk1.NewStart, hunk1.NewCount)
		}

		// Check second hunk
		hunk2 := file.Hunks[1]
		if hunk2.OldStart != 10 || hunk2.OldCount != 2 {
			t.Errorf("hunk2 old range: expected 10,2, got %d,%d", hunk2.OldStart, hunk2.OldCount)
		}
		if hunk2.NewStart != 11 || hunk2.NewCount != 3 {
			t.Errorf("hunk2 new range: expected 11,3, got %d,%d", hunk2.NewStart, hunk2.NewCount)
		}
	})

	t.Run("parse new file", func(t *testing.T) {
		diff := `diff --git a/newfile.txt b/newfile.txt
new file mode 100644
--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,2 @@
+line1
+line2
`
		files := ParseUnifiedDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}

		file := files[0]
		if file.ChangeType != types.ChangeTypeAdded {
			t.Errorf("expected added change type, got %s", file.ChangeType)
		}
	})

	t.Run("parse deleted file", func(t *testing.T) {
		diff := `diff --git a/deleted.txt b/deleted.txt
deleted file mode 100644
--- a/deleted.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`
		files := ParseUnifiedDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}

		file := files[0]
		if file.ChangeType != types.ChangeTypeDeleted {
			t.Errorf("expected deleted change type, got %s", file.ChangeType)
		}
	})

	t.Run("parse multiple files", func(t *testing.T) {
		diff := `diff --git a/file1.txt b/file1.txt
--- a/file1.txt
+++ b/file1.txt
@@ -1 +1,2 @@
 line1
+line2
diff --git a/file2.txt b/file2.txt
--- a/file2.txt
+++ b/file2.txt
@@ -1 +1,2 @@
 line1
+line2
`
		files := ParseUnifiedDiff(diff)
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}

		if files[0].Path != "file1.txt" {
			t.Errorf("expected first file 'file1.txt', got '%s'", files[0].Path)
		}
		if files[1].Path != "file2.txt" {
			t.Errorf("expected second file 'file2.txt', got '%s'", files[1].Path)
		}
	})

	t.Run("parse empty diff", func(t *testing.T) {
		files := ParseUnifiedDiff("")
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})
}

// TestFilterHunks tests filtering diffs by selected hunks
func TestFilterHunks(t *testing.T) {
	fileID := uuid.New()
	fileChanges := []*types.FileChange{
		{ID: fileID, FilePath: "file.txt"},
	}

	t.Run("filter to single hunk", func(t *testing.T) {
		diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
+newline
 line2
 line3
@@ -10,2 +11,3 @@
 line10
+anotherline
 line11
`
		// Select only the second hunk (starting at line 11)
		hunkRanges := []types.HunkRange{
			{FileID: fileID, StartLine: 11, EndLine: 13},
		}

		filtered := FilterHunks(diff, hunkRanges, fileChanges)

		// Should only contain the second hunk
		if !strings.Contains(filtered, "@@ -10,2 +11,3 @@") {
			t.Error("filtered diff should contain second hunk header")
		}
		if strings.Contains(filtered, "@@ -1,3 +1,4 @@") {
			t.Error("filtered diff should not contain first hunk header")
		}
	})

	t.Run("filter with empty hunk ranges", func(t *testing.T) {
		diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1 +1,2 @@
 line1
+line2
`
		// Empty hunk ranges should return the original diff
		filtered := FilterHunks(diff, []types.HunkRange{}, fileChanges)
		if filtered != diff {
			t.Error("empty hunk ranges should return original diff")
		}
	})

	t.Run("filter with no matching file", func(t *testing.T) {
		diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1 +1,2 @@
 line1
+line2
`
		otherFileID := uuid.New()
		hunkRanges := []types.HunkRange{
			{FileID: otherFileID, StartLine: 1, EndLine: 2},
		}

		filtered := FilterHunks(diff, hunkRanges, fileChanges)
		// No matching file, should return empty
		if filtered != "" {
			t.Errorf("expected empty string when no files match, got: %s", filtered)
		}
	})
}

// TestGetHunksForFile tests extracting hunks for a specific file
func TestGetHunksForFile(t *testing.T) {
	diff := `diff --git a/file1.txt b/file1.txt
--- a/file1.txt
+++ b/file1.txt
@@ -1 +1,2 @@
 line1
+line2
diff --git a/file2.txt b/file2.txt
--- a/file2.txt
+++ b/file2.txt
@@ -1,3 +1,4 @@
 a
+b
 c
 d
@@ -10 +11,2 @@
 x
+y
`

	t.Run("get hunks for existing file", func(t *testing.T) {
		hunks := GetHunksForFile(diff, "file2.txt")
		if len(hunks) != 2 {
			t.Fatalf("expected 2 hunks for file2.txt, got %d", len(hunks))
		}
	})

	t.Run("get hunks for non-existing file", func(t *testing.T) {
		hunks := GetHunksForFile(diff, "nonexistent.txt")
		if len(hunks) != 0 {
			t.Errorf("expected 0 hunks for non-existing file, got %d", len(hunks))
		}
	})
}
