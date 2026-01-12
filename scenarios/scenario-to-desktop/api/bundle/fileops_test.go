package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPathPreservesExecutablesAndDirectories(t *testing.T) {
	fileOps := &defaultFileOperations{}
	root := t.TempDir()

	srcDir := filepath.Join(root, "src", "resources", "playwright")
	nested := filepath.Join(srcDir, "chromium")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	// Executable asset inside the directory tree.
	chromeSrc := filepath.Join(nested, "chrome")
	if err := os.WriteFile(chromeSrc, []byte("fake-chrome"), 0o755); err != nil {
		t.Fatalf("write source chrome: %v", err)
	}

	dstDir := filepath.Join(root, "bundle", "resources", "playwright")
	if err := fileOps.CopyPath(srcDir, dstDir); err != nil {
		t.Fatalf("CopyPath: %v", err)
	}

	chromeDst := filepath.Join(dstDir, "chromium", "chrome")
	info, err := os.Stat(chromeDst)
	if err != nil {
		t.Fatalf("stat copied chrome: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved, got mode %v", info.Mode())
	}

	// Direct file copy still works and preserves mode.
	fileDst := filepath.Join(root, "bundle", "bin", "shim")
	if err := fileOps.CopyPath(chromeSrc, fileDst); err != nil {
		t.Fatalf("CopyPath file: %v", err)
	}
	fileInfo, err := os.Stat(fileDst)
	if err != nil {
		t.Fatalf("stat copied file: %v", err)
	}
	if fileInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved on file copy, got mode %v", fileInfo.Mode())
	}
}

func TestCopyFileSameSourceAndDest(t *testing.T) {
	fileOps := &defaultFileOperations{}
	root := t.TempDir()

	// Create a file
	filePath := filepath.Join(root, "test.txt")
	if err := os.WriteFile(filePath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Copy to same location should not error
	if err := fileOps.CopyFile(filePath, filePath); err != nil {
		t.Fatalf("CopyFile same src/dst should succeed: %v", err)
	}

	// Verify content is preserved
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(content) != "test content" {
		t.Fatalf("expected content preserved, got %q", string(content))
	}
}

func TestNormalizeBundlePath(t *testing.T) {
	fileOps := &defaultFileOperations{}

	tests := []struct {
		input    string
		expected string
	}{
		{"../foo/bar", "foo/bar"},
		{"../../foo", "foo"},
		{"..", ""},
		{"foo/bar", "foo/bar"},
		{"../../../deeply/nested", "deeply/nested"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := fileOps.NormalizeBundlePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeBundlePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithinBase(t *testing.T) {
	fileOps := &defaultFileOperations{}

	tests := []struct {
		base     string
		target   string
		expected bool
	}{
		{"/home/user", "/home/user/file.txt", true},
		{"/home/user", "/home/user/sub/file.txt", true},
		{"/home/user", "/home/other/file.txt", false},
		{"/home/user", "/home/../etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := fileOps.WithinBase(tt.base, tt.target)
			if result != tt.expected {
				t.Errorf("WithinBase(%q, %q) = %v, want %v", tt.base, tt.target, result, tt.expected)
			}
		})
	}
}
