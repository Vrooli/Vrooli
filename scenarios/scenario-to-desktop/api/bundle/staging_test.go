package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCLIStagerStage(t *testing.T) {
	t.Run("creates bin directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		stager := &defaultCLIStager{fileOps: &defaultFileOperations{}}

		err := stager.Stage(tmpDir, "linux")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		binDir := filepath.Join(tmpDir, "bin")
		if _, err := os.Stat(binDir); os.IsNotExist(err) {
			t.Error("expected bin directory to be created")
		}
	})

	t.Run("creates vrooli shim for linux", func(t *testing.T) {
		tmpDir := t.TempDir()
		stager := &defaultCLIStager{fileOps: &defaultFileOperations{}}

		err := stager.Stage(tmpDir, "linux")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		shimPath := filepath.Join(tmpDir, "bin", "vrooli")
		info, err := os.Stat(shimPath)
		if os.IsNotExist(err) {
			t.Error("expected vrooli shim to be created")
		}
		// Check if executable
		if info.Mode()&0o111 == 0 {
			t.Error("expected vrooli shim to be executable")
		}
	})

	t.Run("skips shim for windows", func(t *testing.T) {
		tmpDir := t.TempDir()
		stager := &defaultCLIStager{fileOps: &defaultFileOperations{}}

		err := stager.Stage(tmpDir, "windows")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		shimPath := filepath.Join(tmpDir, "bin", "vrooli")
		if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
			t.Error("expected no vrooli shim for windows")
		}
	})

	t.Run("copies cli binaries", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a cli directory with a mock binary
		cliDir := filepath.Join(tmpDir, "cli")
		if err := os.MkdirAll(cliDir, 0o755); err != nil {
			t.Fatalf("failed to create cli dir: %v", err)
		}
		mockBin := filepath.Join(cliDir, "test-cli")
		if err := os.WriteFile(mockBin, []byte("test content"), 0o755); err != nil {
			t.Fatalf("failed to write mock binary: %v", err)
		}

		stager := &defaultCLIStager{fileOps: &defaultFileOperations{}}

		err := stager.Stage(tmpDir, "linux")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check if the binary was copied
		copiedBin := filepath.Join(tmpDir, "bin", "test-cli")
		info, err := os.Stat(copiedBin)
		if os.IsNotExist(err) {
			t.Error("expected cli binary to be copied")
		} else if info.Mode()&0o111 == 0 {
			t.Error("expected copied binary to be executable")
		}
	})
}
