package staleness

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckNewerFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a reference time
	refTime := time.Now().Add(-1 * time.Hour)

	// Create an old file (before refTime)
	oldFile := filepath.Join(tmpDir, "old.go")
	if err := os.WriteFile(oldFile, []byte("package old"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set old modification time
	oldTime := refTime.Add(-1 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a new file (after refTime)
	newFile := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(newFile, []byte("package new"), 0644); err != nil {
		t.Fatal(err)
	}
	// File will have current time which is after refTime

	// Test: should find newer file
	found, file := CheckNewerFiles(tmpDir, "*.go", refTime)
	if !found {
		t.Error("expected to find newer file")
	}
	if file != newFile {
		t.Errorf("expected file %s, got %s", newFile, file)
	}

	// Test: should not find file newer than now
	found, _ = CheckNewerFiles(tmpDir, "*.go", time.Now().Add(1*time.Hour))
	if found {
		t.Error("should not find file newer than future time")
	}

	// Test: pattern mismatch
	found, _ = CheckNewerFiles(tmpDir, "*.txt", refTime)
	if found {
		t.Error("should not match .txt files")
	}
}

func TestCheckNewerFiles_SkipDirs(t *testing.T) {
	tmpDir := t.TempDir()
	refTime := time.Now().Add(-1 * time.Hour)

	// Create a new file in a skipped directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	vendorFile := filepath.Join(vendorDir, "new.go")
	if err := os.WriteFile(vendorFile, []byte("package vendor"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not find file in vendor directory
	found, _ := CheckNewerFiles(tmpDir, "*.go", refTime)
	if found {
		t.Error("should skip vendor directory")
	}

	// Create a new file in a non-skipped directory
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(srcFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find file in src directory
	found, file := CheckNewerFiles(tmpDir, "*.go", refTime)
	if !found {
		t.Error("expected to find file in src directory")
	}
	if file != srcFile {
		t.Errorf("expected file %s, got %s", srcFile, file)
	}
}

func TestIsFileNewer(t *testing.T) {
	tmpDir := t.TempDir()
	refTime := time.Now().Add(-1 * time.Hour)

	// Create a new file
	newFile := filepath.Join(tmpDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test: file is newer
	if !IsFileNewer(newFile, refTime) {
		t.Error("expected file to be newer than refTime")
	}

	// Test: file is not newer than future
	if IsFileNewer(newFile, time.Now().Add(1*time.Hour)) {
		t.Error("file should not be newer than future time")
	}

	// Test: nonexistent file
	if IsFileNewer("/nonexistent", refTime) {
		t.Error("nonexistent file should return false")
	}
}

func TestGetModTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test: existing file
	modTime := GetModTime(testFile)
	if modTime.IsZero() {
		t.Error("expected non-zero mod time for existing file")
	}

	// Test: nonexistent file
	modTime = GetModTime("/nonexistent")
	if !modTime.IsZero() {
		t.Error("expected zero mod time for nonexistent file")
	}
}
