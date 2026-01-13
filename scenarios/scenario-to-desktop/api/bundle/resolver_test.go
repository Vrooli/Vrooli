package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultRuntimeResolverResolve(t *testing.T) {
	t.Run("finds runtime directory relative to cwd", func(t *testing.T) {
		// Save current directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalDir)

		// Create a temp directory structure: tmpDir/subdir and tmpDir/runtime
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		runtimeDir := filepath.Join(tmpDir, "runtime")

		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
			t.Fatalf("failed to create runtime dir: %v", err)
		}

		// Change to subdir
		if err := os.Chdir(subDir); err != nil {
			t.Fatalf("failed to change to subdir: %v", err)
		}

		resolver := &defaultRuntimeResolver{}
		result, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		absRuntimeDir, _ := filepath.Abs(runtimeDir)
		if result != absRuntimeDir {
			t.Errorf("Resolve() = %q, want %q", result, absRuntimeDir)
		}
	})

	t.Run("returns error when runtime not found", func(t *testing.T) {
		// Save current directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(originalDir)

		// Create a temp directory with no runtime subdirectory
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "deep", "nested", "path")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		// Change to the isolated directory
		if err := os.Chdir(subDir); err != nil {
			t.Fatalf("failed to change to subdir: %v", err)
		}

		resolver := &defaultRuntimeResolver{}
		_, err = resolver.Resolve()
		if err == nil {
			t.Error("expected error when runtime not found")
		}
	})
}
