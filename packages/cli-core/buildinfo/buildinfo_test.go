package buildinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestComputeFingerprint_DetectsChanges(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	writeFile(t, root, "main.go", []byte("package main\n"))
	writeFile(t, root, "format.go", []byte("package main\n"))

	original, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	writeFile(t, root, "format.go", []byte("package main\n// updated\n"))
	updated, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint after update: %v", err)
	}

	if original == updated {
		t.Fatalf("expected fingerprint to change after modifying files")
	}

	// Ensure deterministic ordering by rerunning without modifications
	secondRun, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint second run: %v", err)
	}
	if updated != secondRun {
		t.Fatalf("expected fingerprint to be deterministic; got %s and %s", updated, secondRun)
	}
}

func TestComputeFingerprint_IgnoresModtimeAndSkippedDirs(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(filepath.Join(root, "dist"), 0o755); err != nil {
		t.Fatalf("setup dist: %v", err)
	}

	writeFile(t, root, "main.go", []byte("package main\n"))

	original, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	// Change modtime only
	if err := os.Chtimes(filepath.Join(root, "main.go"), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("touch file: %v", err)
	}

	// Add file in skipped directory
	writeFile(t, root, "dist/bundle.js", []byte("console.log('skip')"))

	updated, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint updated: %v", err)
	}

	if original != updated {
		t.Fatalf("expected fingerprint unchanged when only modtime or skipped dirs differ")
	}
}

func TestComputeFingerprint_ErrorsOnUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on Windows")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(target, []byte("data"), 0o200); err != nil { // write-only
		t.Fatalf("write: %v", err)
	}
	if _, err := ComputeFingerprint(root); err == nil {
		t.Fatalf("expected error for unreadable file")
	}
}

func TestSkipDirNormalizesSeparators(t *testing.T) {
	if !skipDir("node_modules\\pkg") {
		t.Fatalf("expected windows-style node_modules path to be skipped")
	}
	if !skipFile("custom\\cache\\index.json", []string{"custom/cache"}) {
		t.Fatalf("expected windows-style custom/cache path to be skipped via extra list")
	}
}

func writeFile(t *testing.T, root, name string, contents []byte) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
