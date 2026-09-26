package bundle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type fixedRuntimeResolver struct{ dir string }

func (r fixedRuntimeResolver) Resolve() (string, error) { return r.dir, nil }

func testCLIStager(t *testing.T) *defaultCLIStager {
	t.Helper()
	runtimeDir, err := filepath.Abs(filepath.Join("..", "..", "runtime"))
	if err != nil {
		t.Fatal(err)
	}
	return &defaultCLIStager{fileOps: &defaultFileOperations{}, runtimeResolver: fixedRuntimeResolver{dir: runtimeDir}}
}

func TestDefaultCLIStagerStage(t *testing.T) {
	t.Run("creates bin directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		stager := testCLIStager(t)

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
		stager := testCLIStager(t)

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

	t.Run("creates windows shim", func(t *testing.T) {
		tmpDir := t.TempDir()
		stager := testCLIStager(t)

		err := stager.Stage(tmpDir, "windows")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		shimPath := filepath.Join(tmpDir, "bin", "vrooli.exe")
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			t.Error("expected compiled vrooli.exe shim for windows")
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

		stager := testCLIStager(t)

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

func TestDefaultCLIStagerStagesCurrentPlatformShim(t *testing.T) {
	tmpDir := t.TempDir()
	stager := testCLIStager(t)

	if err := stager.Stage(tmpDir, runtime.GOOS); err != nil {
		t.Fatalf("stage current platform %q: %v", runtime.GOOS, err)
	}

	shimName := "vrooli"
	if runtime.GOOS == "windows" {
		shimName += ".exe"
	}
	shimPath := filepath.Join(tmpDir, "bin", shimName)
	info, err := os.Stat(shimPath)
	if err != nil {
		t.Fatalf("stat current platform shim %q: %v", shimPath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("current platform shim %q is empty", shimPath)
	}
}
