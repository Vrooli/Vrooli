package existence

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDirs_AllExist(t *testing.T) {
	root := t.TempDir()

	// Create directories
	dirs := []string{"api", "cli", "docs"}
	for _, d := range dirs {
		mustMkdir(t, filepath.Join(root, d))
	}

	result := ValidateDirs(root, dirs, io.Discard)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ItemsChecked != len(dirs) {
		t.Errorf("expected %d items checked, got %d", len(dirs), result.ItemsChecked)
	}
}

func TestValidateDirs_MissingDir(t *testing.T) {
	root := t.TempDir()

	// Create only some directories
	mustMkdir(t, filepath.Join(root, "api"))
	// "cli" is missing

	result := ValidateDirs(root, []string{"api", "cli"}, io.Discard)
	if result.Success {
		t.Fatal("expected failure for missing directory")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidateDirs_FileInsteadOfDir(t *testing.T) {
	root := t.TempDir()

	// Create a file where a directory is expected
	writeFile(t, filepath.Join(root, "api"), "not a directory")

	result := ValidateDirs(root, []string{"api"}, io.Discard)
	if result.Success {
		t.Fatal("expected failure when file exists instead of directory")
	}
}

func TestValidateFiles_AllExist(t *testing.T) {
	root := t.TempDir()

	// Create files
	files := []string{"README.md", "PRD.md"}
	for _, f := range files {
		writeFile(t, filepath.Join(root, f), "content")
	}

	result := ValidateFiles(root, files, io.Discard)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ItemsChecked != len(files) {
		t.Errorf("expected %d items checked, got %d", len(files), result.ItemsChecked)
	}
}

func TestValidateFiles_MissingFile(t *testing.T) {
	root := t.TempDir()

	// Create only one file
	writeFile(t, filepath.Join(root, "README.md"), "content")
	// "PRD.md" is missing

	result := ValidateFiles(root, []string{"README.md", "PRD.md"}, io.Discard)
	if result.Success {
		t.Fatal("expected failure for missing file")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidateFiles_DirInsteadOfFile(t *testing.T) {
	root := t.TempDir()

	// Create a directory where a file is expected
	mustMkdir(t, filepath.Join(root, "README.md"))

	result := ValidateFiles(root, []string{"README.md"}, io.Discard)
	if result.Success {
		t.Fatal("expected failure when directory exists instead of file")
	}
}

func TestValidateFiles_NestedPath(t *testing.T) {
	root := t.TempDir()

	// Create nested file
	mustMkdir(t, filepath.Join(root, "api"))
	writeFile(t, filepath.Join(root, "api", "main.go"), "package main")

	result := ValidateFiles(root, []string{"api/main.go"}, io.Discard)
	if !result.Success {
		t.Fatalf("expected success for nested file, got error: %v", result.Error)
	}
}

func TestFileExists_ExistingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	writeFile(t, path, "content")

	if !FileExists(path) {
		t.Error("expected FileExists to return true for existing file")
	}
}

func TestFileExists_MissingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nonexistent.txt")

	if FileExists(path) {
		t.Error("expected FileExists to return false for missing file")
	}
}

func TestFileExists_Directory(t *testing.T) {
	root := t.TempDir()
	dirPath := filepath.Join(root, "somedir")
	mustMkdir(t, dirPath)

	if FileExists(dirPath) {
		t.Error("expected FileExists to return false for directory")
	}
}

func TestValidator_Interface(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "api"))
	writeFile(t, filepath.Join(root, "README.md"), "# Test")

	v := New(root, io.Discard)

	// Test ValidateDirs through interface
	dirResult := v.ValidateDirs([]string{"api"})
	if !dirResult.Success {
		t.Errorf("ValidateDirs failed: %v", dirResult.Error)
	}

	// Test ValidateFiles through interface
	fileResult := v.ValidateFiles([]string{"README.md"})
	if !fileResult.Success {
		t.Errorf("ValidateFiles failed: %v", fileResult.Error)
	}
}

func TestResolveRequirementsWithOverrides_AdditionalPaths(t *testing.T) {
	dirs, files := ResolveRequirementsWithOverrides(
		[]string{"extensions"},        // additional dirs
		[]string{"config/extra.json"}, // additional files
		nil,                           // excluded dirs
		nil,                           // excluded files
	)

	// Check additional dir is included
	if !contains(dirs, "extensions") {
		t.Error("expected additional dir 'extensions' to be included")
	}

	// Check additional file is included
	if !contains(files, "config/extra.json") {
		t.Error("expected additional file 'config/extra.json' to be included")
	}
}

func TestResolveRequirementsWithOverrides_Exclusions(t *testing.T) {
	dirs, files := ResolveRequirementsWithOverrides(
		nil,                // additional dirs
		nil,                // additional files
		[]string{"cli"},    // excluded dirs
		[]string{"PRD.md"}, // excluded files
	)

	// Check excluded dir is not included
	if contains(dirs, "cli") {
		t.Error("expected excluded dir 'cli' to be removed")
	}

	// Check excluded file is not included
	if contains(files, "PRD.md") {
		t.Error("expected excluded file 'PRD.md' to be removed")
	}

	// Check other standard dirs are still present
	if !contains(dirs, "api") {
		t.Error("expected standard dir 'api' to remain")
	}
}

func TestResolveRequirementsWithOverrides_Deduplication(t *testing.T) {
	dirs, _ := ResolveRequirementsWithOverrides(
		[]string{"api", "api", "./api"}, // duplicate dirs
		nil,
		nil,
		nil,
	)

	// Count occurrences of "api"
	count := 0
	for _, d := range dirs {
		if d == "api" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'api' to appear once after deduplication, got %d", count)
	}
}

func TestCanonicalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"./api", "api"},
		{"api/", "api"},
		{"./api/", "api"},
		{"api/../api", "api"},
		{".", ""},
		{"", ""},
		{"api/sub/dir", "api/sub/dir"},
	}

	for _, tc := range tests {
		got := canonicalizePath(tc.input)
		if got != tc.expected {
			t.Errorf("canonicalizePath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// Test helpers

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		mustMkdir(t, dir)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
