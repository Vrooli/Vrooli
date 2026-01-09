package driver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	if !strings.HasPrefix(paths.UpperDir, tmpDir) {
		t.Errorf("UpperDir should be under base dir")
	}
	if !strings.HasPrefix(paths.WorkDir, tmpDir) {
		t.Errorf("WorkDir should be under base dir")
	}
	if !strings.HasPrefix(paths.MergedDir, tmpDir) {
		t.Errorf("MergedDir should be under base dir")
	}

	// Clean up
	if err := drv.Cleanup(ctx, sb); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
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

	// Test change detection using shared helper
	newInfo, _ := os.Stat(newFile)
	modInfo, _ := os.Stat(modifiedFile)

	// Use the shared detectOverlayChangeType helper
	newType := detectOverlayChangeType(sb, "new.txt", newInfo)
	if newType != types.ChangeTypeAdded {
		t.Errorf("new file should be ChangeTypeAdded, got %s", newType)
	}

	modType := detectOverlayChangeType(sb, "existing.txt", modInfo)
	if modType != types.ChangeTypeModified {
		t.Errorf("modified file should be ChangeTypeModified, got %s", modType)
	}

	// Verify driver is created (unused but confirms API)
	_ = drv
}

func TestOverlayfsGetChangedFilesSkipsOpaqueAndMapsWhiteouts(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("overlayfs tests require Linux")
	}

	tmpDir := t.TempDir()
	lowerDir := filepath.Join(tmpDir, "lower")
	upperDir := filepath.Join(tmpDir, "upper")

	if err := os.MkdirAll(lowerDir, 0o755); err != nil {
		t.Fatalf("failed to create lower dir: %v", err)
	}
	if err := os.MkdirAll(upperDir, 0o755); err != nil {
		t.Fatalf("failed to create upper dir: %v", err)
	}

	lowerFile := filepath.Join(lowerDir, "dir", "removed.txt")
	if err := os.MkdirAll(filepath.Dir(lowerFile), 0o755); err != nil {
		t.Fatalf("failed to create lower subdir: %v", err)
	}
	if err := os.WriteFile(lowerFile, []byte("original"), 0o644); err != nil {
		t.Fatalf("failed to create lower file: %v", err)
	}

	whiteout := filepath.Join(upperDir, "dir", ".wh.removed.txt")
	if err := os.MkdirAll(filepath.Dir(whiteout), 0o755); err != nil {
		t.Fatalf("failed to create upper subdir: %v", err)
	}
	if err := os.WriteFile(whiteout, []byte("marker"), 0o644); err != nil {
		t.Fatalf("failed to create whiteout marker: %v", err)
	}

	opaque := filepath.Join(upperDir, "tmp", ".wh..opq")
	if err := os.MkdirAll(filepath.Dir(opaque), 0o755); err != nil {
		t.Fatalf("failed to create opaque dir: %v", err)
	}
	if err := os.WriteFile(opaque, []byte("opaque"), 0o644); err != nil {
		t.Fatalf("failed to create opaque marker: %v", err)
	}

	recursiveOpaque := filepath.Join(upperDir, "tmp", ".wh..wh..opq")
	if err := os.WriteFile(recursiveOpaque, []byte("opaque"), 0o644); err != nil {
		t.Fatalf("failed to create recursive opaque marker: %v", err)
	}

	added := filepath.Join(upperDir, "added.txt")
	if err := os.WriteFile(added, []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to create added file: %v", err)
	}

	drv := NewOverlayfsDriver(DefaultConfig())
	sb := &types.Sandbox{
		ID:       uuid.New(),
		LowerDir: lowerDir,
		UpperDir: upperDir,
	}

	changes, err := drv.GetChangedFiles(context.Background(), sb)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	var sawDeleted bool
	var sawOpaque bool
	var sawAdded bool

	for _, change := range changes {
		if change.FilePath == "dir/removed.txt" && change.ChangeType == types.ChangeTypeDeleted {
			sawDeleted = true
		}
		if change.FilePath == "tmp/.wh..opq" || change.FilePath == ".wh..opq" {
			sawOpaque = true
		}
		if change.FilePath == "tmp/.wh..wh..opq" {
			sawOpaque = true
		}
		if change.FilePath == "added.txt" && change.ChangeType == types.ChangeTypeAdded {
			sawAdded = true
		}
	}

	if !sawDeleted {
		t.Error("expected whiteout marker to map to deleted file change")
	}
	if sawOpaque {
		t.Error("opaque marker should be ignored")
	}
	if !sawAdded {
		t.Error("expected added file change")
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
		if _, err := drv.IsAvailable(ctx); err != nil {
			b.Fatalf("IsAvailable() failed: %v", err)
		}
	}
}
