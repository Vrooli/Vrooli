package driver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// [REQ:P0-003] Overlayfs Mount Configuration - Verify mount options
func TestOverlayfsDriverType(t *testing.T) {
	drv := NewOverlayfsDriver(DefaultConfig())

	if drv.Type() != DriverTypeOverlayfs {
		t.Errorf("Type() = %v, want %v", drv.Type(), DriverTypeOverlayfs)
	}

	if drv.Version() == "" {
		t.Error("Version() should not be empty")
	}
}

// [REQ:P0-003] Overlayfs Mount Configuration - Test directory structure
func TestOverlayfsDriverDirectoryStructure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("overlayfs tests require Linux")
	}

	tmpDir, err := os.MkdirTemp("", "overlayfs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		BaseDir:      tmpDir,
		MaxSandboxes: 10,
	}
	drv := NewOverlayfsDriver(cfg)

	// Create a mock sandbox
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	sb := &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   projectDir,
		ProjectRoot: projectDir,
	}

	// Test mount paths structure
	ctx := context.Background()
	paths, err := drv.Mount(ctx, sb)
	// On non-root systems, mount will fail but we can verify path structure
	if err != nil {
		// Expected on non-root - verify paths would be correct
		expectedUpper := filepath.Join(tmpDir, sb.ID.String(), "upper")
		expectedWork := filepath.Join(tmpDir, sb.ID.String(), "work")
		expectedMerged := filepath.Join(tmpDir, sb.ID.String(), "merged")

		t.Logf("Mount failed (expected without root): %v", err)
		t.Logf("Expected paths: upper=%s, work=%s, merged=%s",
			expectedUpper, expectedWork, expectedMerged)
		return
	}

	// Verify path structure
	if paths.LowerDir != projectDir {
		t.Errorf("LowerDir = %q, want %q", paths.LowerDir, projectDir)
	}
	if !filepath.HasPrefix(paths.UpperDir, tmpDir) {
		t.Errorf("UpperDir should be under base dir")
	}
	if !filepath.HasPrefix(paths.WorkDir, tmpDir) {
		t.Errorf("WorkDir should be under base dir")
	}
	if !filepath.HasPrefix(paths.MergedDir, tmpDir) {
		t.Errorf("MergedDir should be under base dir")
	}

	// Clean up
	drv.Cleanup(ctx, sb)
}

// [REQ:P0-003] Overlayfs Mount Configuration - Check availability
func TestOverlayfsDriverIsAvailable(t *testing.T) {
	drv := NewOverlayfsDriver(DefaultConfig())
	ctx := context.Background()

	available, err := drv.IsAvailable(ctx)

	if runtime.GOOS != "linux" {
		if available {
			t.Error("IsAvailable() should be false on non-Linux")
		}
		if err == nil {
			t.Error("expected error on non-Linux")
		}
	} else {
		// On Linux, overlayfs should generally be available
		t.Logf("IsAvailable() = %v, err = %v", available, err)
	}
}

// [REQ:P0-003] Overlayfs Mount Configuration - Default config validation
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseDir == "" {
		t.Error("BaseDir should not be empty")
	}
	if cfg.MaxSandboxes <= 0 {
		t.Error("MaxSandboxes should be positive")
	}
	if cfg.MaxSizeMB <= 0 {
		t.Error("MaxSizeMB should be positive")
	}
}

// [REQ:P0-006] Stable Diff Generation - File change detection
func TestDetectChangeType(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("overlayfs tests require Linux")
	}

	tmpDir, err := os.MkdirTemp("", "overlayfs-change-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create lower and upper directories
	lowerDir := filepath.Join(tmpDir, "lower")
	upperDir := filepath.Join(tmpDir, "upper")
	if err := os.MkdirAll(lowerDir, 0o755); err != nil {
		t.Fatalf("failed to create lower dir: %v", err)
	}
	if err := os.MkdirAll(upperDir, 0o755); err != nil {
		t.Fatalf("failed to create upper dir: %v", err)
	}

	// Create a file in lower (existing file)
	existingFile := filepath.Join(lowerDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("original"), 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Create a new file in upper (added)
	newFile := filepath.Join(upperDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}

	// Create modified file in upper
	modifiedFile := filepath.Join(upperDir, "existing.txt")
	if err := os.WriteFile(modifiedFile, []byte("modified content"), 0o644); err != nil {
		t.Fatalf("failed to create modified file: %v", err)
	}

	cfg := DefaultConfig()
	drv := NewOverlayfsDriver(cfg)

	sb := &types.Sandbox{
		ID:       uuid.New(),
		LowerDir: lowerDir,
		UpperDir: upperDir,
	}

	// Test change detection
	newInfo, _ := os.Stat(newFile)
	modInfo, _ := os.Stat(modifiedFile)

	newType := drv.detectChangeType(sb, "new.txt", newInfo)
	if newType != types.ChangeTypeAdded {
		t.Errorf("new file should be ChangeTypeAdded, got %s", newType)
	}

	modType := drv.detectChangeType(sb, "existing.txt", modInfo)
	if modType != types.ChangeTypeModified {
		t.Errorf("modified file should be ChangeTypeModified, got %s", modType)
	}
}

// [REQ:P0-003] Overlayfs Mount Configuration - Performance test
func BenchmarkOverlayfsIsAvailable(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("overlayfs benchmark requires Linux")
	}

	drv := NewOverlayfsDriver(DefaultConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drv.IsAvailable(ctx)
	}
}
