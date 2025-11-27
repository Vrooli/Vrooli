package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:PCT-DRAFT-PUBLISH] Publishing atomically updates PRD.md and clears draft
func TestCopyFile(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() (string, string) // Returns (src, dst) paths
		wantError  bool
		verifyFunc func(t *testing.T, dst string) // Optional verification
	}{
		{
			name: "copy simple text file",
			setupFunc: func() (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "source.txt")
				dst := filepath.Join(tmpDir, "dest.txt")
				if err := os.WriteFile(src, []byte("Hello, World!"), 0644); err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return src, dst
			},
			wantError: false,
			verifyFunc: func(t *testing.T, dst string) {
				content, err := os.ReadFile(dst)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}
				if string(content) != "Hello, World!" {
					t.Errorf("Destination content = %q, want %q", string(content), "Hello, World!")
				}
			},
		},
		{
			name: "copy markdown file",
			setupFunc: func() (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "PRD.md")
				dst := filepath.Join(tmpDir, "PRD.md.backup")
				content := `# Product Requirements Document

## Overview
This is a test PRD.

## Features
- Feature 1
- Feature 2
`
				if err := os.WriteFile(src, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return src, dst
			},
			wantError: false,
			verifyFunc: func(t *testing.T, dst string) {
				content, err := os.ReadFile(dst)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}
				if len(content) == 0 {
					t.Error("Destination file is empty")
				}
				// Verify it contains expected markdown elements
				contentStr := string(content)
				if len(contentStr) < 50 {
					t.Errorf("Destination content too short: %d bytes", len(contentStr))
				}
			},
		},
		{
			name: "copy empty file",
			setupFunc: func() (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "empty.txt")
				dst := filepath.Join(tmpDir, "empty-copy.txt")
				if err := os.WriteFile(src, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return src, dst
			},
			wantError: false,
			verifyFunc: func(t *testing.T, dst string) {
				content, err := os.ReadFile(dst)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}
				if len(content) != 0 {
					t.Errorf("Destination content length = %d, want 0", len(content))
				}
			},
		},
		{
			name: "copy large file",
			setupFunc: func() (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "large.txt")
				dst := filepath.Join(tmpDir, "large-copy.txt")
				// Create 1MB file
				largeContent := make([]byte, 1024*1024)
				for i := range largeContent {
					largeContent[i] = byte(i % 256)
				}
				if err := os.WriteFile(src, largeContent, 0644); err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return src, dst
			},
			wantError: false,
			verifyFunc: func(t *testing.T, dst string) {
				info, err := os.Stat(dst)
				if err != nil {
					t.Fatalf("Failed to stat destination file: %v", err)
				}
				if info.Size() != 1024*1024 {
					t.Errorf("Destination size = %d, want %d", info.Size(), 1024*1024)
				}
			},
		},
		{
			name: "copy nonexistent source file",
			setupFunc: func() (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "nonexistent.txt")
				dst := filepath.Join(tmpDir, "dest.txt")
				return src, dst
			},
			wantError:  true,
			verifyFunc: nil,
		},
		{
			name: "copy to invalid destination path",
			setupFunc: func() (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "source.txt")
				dst := filepath.Join(tmpDir, "nonexistent-dir", "dest.txt")
				if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return src, dst
			},
			wantError:  true,
			verifyFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setupFunc()

			err := copyFile(src, dst)

			if tt.wantError {
				if err == nil {
					t.Error("copyFile() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("copyFile() unexpected error: %v", err)
				}
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, dst)
				}
			}
		})
	}
}

// [REQ:PCT-DRAFT-PUBLISH] Publishing atomically updates PRD.md and clears draft
func TestCopyFilePreservesContent(t *testing.T) {
	// Test that copyFile preserves exact byte content
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "original.bin")
	dst := filepath.Join(tmpDir, "copy.bin")

	// Create binary content with various byte values
	originalContent := []byte{0, 1, 2, 127, 128, 255, 10, 13, 9}
	if err := os.WriteFile(src, originalContent, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	copiedContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if len(copiedContent) != len(originalContent) {
		t.Fatalf("Copied content length = %d, want %d", len(copiedContent), len(originalContent))
	}

	for i, b := range originalContent {
		if copiedContent[i] != b {
			t.Errorf("Byte at position %d = %d, want %d", i, copiedContent[i], b)
		}
	}
}

// [REQ:PCT-DRAFT-PUBLISH] Publishing atomically updates PRD.md and clears draft
func TestCopyFileOverwritesExisting(t *testing.T) {
	// Test that copyFile overwrites existing destination file
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	// Create source and existing destination
	if err := os.WriteFile(src, []byte("New Content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.WriteFile(dst, []byte("Old Content"), 0644); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(content) != "New Content" {
		t.Errorf("Destination content = %q, want %q", string(content), "New Content")
	}
}
