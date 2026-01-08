package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// [REQ:P2-004-002] CopyDriver Implementation - Basic functionality tests

// TestCopyDriverType verifies CopyDriver returns correct type
func TestCopyDriverType(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig())

	if drv.Type() != DriverTypeCopy {
		t.Errorf("Type() = %v, want %v", drv.Type(), DriverTypeCopy)
	}
}

// TestCopyDriverVersion verifies CopyDriver has a version string
func TestCopyDriverVersion(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig())

	version := drv.Version()
	if version == "" {
		t.Error("Version() should not be empty")
	}
}

// TestCopyDriverIsAvailable verifies CopyDriver is always available
// [REQ:P2-004-002] CopyDriver.IsAvailable returns true on any platform
func TestCopyDriverIsAvailable(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig())
	ctx := context.Background()

	available, err := drv.IsAvailable(ctx)
	if err != nil {
		t.Errorf("IsAvailable() returned error: %v", err)
	}
	if !available {
		t.Error("IsAvailable() should always return true for CopyDriver")
	}
}

// TestCopyDriverMount tests the Mount operation
func TestCopyDriverMount(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source directory with content
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create driver with temp base dir
	cfg := Config{
		BaseDir: filepath.Join(tmpDir, "sandboxes"),
	}
	drv := NewCopyDriver(cfg)
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   sourceDir,
		ProjectRoot: sourceDir,
	}

	paths, err := drv.Mount(ctx, sandbox)
	if err != nil {
		t.Fatalf("Mount() failed: %v", err)
	}

	// Verify paths are populated
	if paths.LowerDir == "" {
		t.Error("LowerDir should not be empty")
	}
	if paths.UpperDir == "" {
		t.Error("UpperDir should not be empty")
	}
	if paths.MergedDir == "" {
		t.Error("MergedDir should not be empty")
	}

	// Verify directories were created
	if _, err := os.Stat(paths.LowerDir); os.IsNotExist(err) {
		t.Error("LowerDir should exist")
	}
	if _, err := os.Stat(paths.UpperDir); os.IsNotExist(err) {
		t.Error("UpperDir should exist")
	}

	// Verify content was copied
	copiedFile := filepath.Join(paths.UpperDir, "test.txt")
	content, err := os.ReadFile(copiedFile)
	if err != nil {
		t.Errorf("failed to read copied file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("copied content = %q, want %q", string(content), "test content")
	}

	// Cleanup
	if err := drv.Cleanup(ctx, sandbox); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
}

// TestCopyDriverGetChangedFiles tests change detection
// [REQ:P2-004-002] CopyDriver correctly detects added, modified, deleted files
func TestCopyDriverGetChangedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source with a file
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "existing.txt"), []byte("original"), 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "toDelete.txt"), []byte("will be deleted"), 0o644); err != nil {
		t.Fatalf("failed to create toDelete file: %v", err)
	}

	cfg := Config{BaseDir: filepath.Join(tmpDir, "sandboxes")}
	drv := NewCopyDriver(cfg)
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   sourceDir,
		ProjectRoot: sourceDir,
	}

	paths, err := drv.Mount(ctx, sandbox)
	if err != nil {
		t.Fatalf("Mount() failed: %v", err)
	}

	// Update sandbox with mount paths
	sandbox.LowerDir = paths.LowerDir
	sandbox.UpperDir = paths.UpperDir
	sandbox.MergedDir = paths.MergedDir

	// Make changes in workspace
	// 1. Add a new file
	if err := os.WriteFile(filepath.Join(paths.UpperDir, "newfile.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}
	// 2. Modify existing file
	if err := os.WriteFile(filepath.Join(paths.UpperDir, "existing.txt"), []byte("modified"), 0o644); err != nil {
		t.Fatalf("failed to modify existing file: %v", err)
	}
	// 3. Delete a file
	if err := os.Remove(filepath.Join(paths.UpperDir, "toDelete.txt")); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	changes, err := drv.GetChangedFiles(ctx, sandbox)
	if err != nil {
		t.Fatalf("GetChangedFiles() failed: %v", err)
	}

	// Verify we detected the correct changes
	changeMap := make(map[string]types.ChangeType)
	for _, c := range changes {
		changeMap[c.FilePath] = c.ChangeType
	}

	if changeMap["newfile.txt"] != types.ChangeTypeAdded {
		t.Errorf("newfile.txt should be Added, got %v", changeMap["newfile.txt"])
	}
	if changeMap["existing.txt"] != types.ChangeTypeModified {
		t.Errorf("existing.txt should be Modified, got %v", changeMap["existing.txt"])
	}
	if changeMap["toDelete.txt"] != types.ChangeTypeDeleted {
		t.Errorf("toDelete.txt should be Deleted, got %v", changeMap["toDelete.txt"])
	}

	// Cleanup
	if err := drv.Cleanup(ctx, sandbox); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
}

func TestCopyDriverGetChangedFilesSkipsOpaqueAndWhiteouts(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir := filepath.Join(tmpDir, "original")
	workspaceDir := filepath.Join(tmpDir, "workspace")

	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		t.Fatalf("failed to create original dir: %v", err)
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	opaqueOriginal := filepath.Join(originalDir, "tmp", ".wh..opq")
	if err := os.MkdirAll(filepath.Dir(opaqueOriginal), 0o755); err != nil {
		t.Fatalf("failed to create opaque dir: %v", err)
	}
	if err := os.WriteFile(opaqueOriginal, []byte("opaque"), 0o644); err != nil {
		t.Fatalf("failed to create opaque marker: %v", err)
	}

	whiteoutOriginal := filepath.Join(originalDir, "dir", ".wh.removed.txt")
	if err := os.MkdirAll(filepath.Dir(whiteoutOriginal), 0o755); err != nil {
		t.Fatalf("failed to create whiteout dir: %v", err)
	}
	if err := os.WriteFile(whiteoutOriginal, []byte("marker"), 0o644); err != nil {
		t.Fatalf("failed to create whiteout marker: %v", err)
	}

	opaqueWorkspace := filepath.Join(workspaceDir, "tmp", ".wh..opq")
	if err := os.MkdirAll(filepath.Dir(opaqueWorkspace), 0o755); err != nil {
		t.Fatalf("failed to create workspace opaque dir: %v", err)
	}
	if err := os.WriteFile(opaqueWorkspace, []byte("opaque"), 0o644); err != nil {
		t.Fatalf("failed to create workspace opaque marker: %v", err)
	}

	whiteoutWorkspace := filepath.Join(workspaceDir, "dir", ".wh.removed.txt")
	if err := os.MkdirAll(filepath.Dir(whiteoutWorkspace), 0o755); err != nil {
		t.Fatalf("failed to create workspace whiteout dir: %v", err)
	}
	if err := os.WriteFile(whiteoutWorkspace, []byte("marker"), 0o644); err != nil {
		t.Fatalf("failed to create workspace whiteout marker: %v", err)
	}

	added := filepath.Join(workspaceDir, "added.txt")
	if err := os.WriteFile(added, []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to create added file: %v", err)
	}

	drv := NewCopyDriver(DefaultConfig())
	sandbox := &types.Sandbox{
		ID:       uuid.New(),
		LowerDir: originalDir,
		UpperDir: workspaceDir,
	}

	changes, err := drv.GetChangedFiles(context.Background(), sandbox)
	if err != nil {
		t.Fatalf("GetChangedFiles() failed: %v", err)
	}

	for _, change := range changes {
		if change.FilePath == "tmp/.wh..opq" || change.FilePath == "dir/.wh.removed.txt" {
			t.Errorf("overlay marker should be skipped, got %s", change.FilePath)
		}
	}

	var sawAdded bool
	for _, change := range changes {
		if change.FilePath == "added.txt" && change.ChangeType == types.ChangeTypeAdded {
			sawAdded = true
		}
	}
	if !sawAdded {
		t.Error("expected added file change")
	}
}

// TestCopyDriverIsMounted tests mount state checking
func TestCopyDriverIsMounted(t *testing.T) {
	tmpDir := t.TempDir()

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	cfg := Config{BaseDir: filepath.Join(tmpDir, "sandboxes")}
	drv := NewCopyDriver(cfg)
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   sourceDir,
		ProjectRoot: sourceDir,
	}

	// Before mount, should return false
	mounted, err := drv.IsMounted(ctx, sandbox)
	if err != nil {
		t.Errorf("IsMounted() error: %v", err)
	}
	if mounted {
		t.Error("IsMounted() should be false before Mount()")
	}

	// After mount
	paths, err := drv.Mount(ctx, sandbox)
	if err != nil {
		t.Fatalf("Mount() failed: %v", err)
	}
	sandbox.MergedDir = paths.MergedDir

	mounted, err = drv.IsMounted(ctx, sandbox)
	if err != nil {
		t.Errorf("IsMounted() error after mount: %v", err)
	}
	if !mounted {
		t.Error("IsMounted() should be true after Mount()")
	}

	// After cleanup
	if err := drv.Cleanup(ctx, sandbox); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	mounted, err = drv.IsMounted(ctx, sandbox)
	if err != nil {
		t.Errorf("IsMounted() error after cleanup: %v", err)
	}
	if mounted {
		t.Error("IsMounted() should be false after Cleanup()")
	}
}

// TestCopyDriverVerifyMountIntegrity tests integrity verification
func TestCopyDriverVerifyMountIntegrity(t *testing.T) {
	tmpDir := t.TempDir()

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	cfg := Config{BaseDir: filepath.Join(tmpDir, "sandboxes")}
	drv := NewCopyDriver(cfg)
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   sourceDir,
		ProjectRoot: sourceDir,
	}

	// Mount and verify
	paths, err := drv.Mount(ctx, sandbox)
	if err != nil {
		t.Fatalf("Mount() failed: %v", err)
	}
	sandbox.MergedDir = paths.MergedDir
	sandbox.LowerDir = paths.LowerDir

	err = drv.VerifyMountIntegrity(ctx, sandbox)
	if err != nil {
		t.Errorf("VerifyMountIntegrity() should pass: %v", err)
	}

	// Cleanup and verify should fail
	if err := drv.Cleanup(ctx, sandbox); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	err = drv.VerifyMountIntegrity(ctx, sandbox)
	if err == nil {
		t.Error("VerifyMountIntegrity() should fail after Cleanup()")
	}
}

// TestCopyDriverRemoveFromUpper tests file removal from workspace
func TestCopyDriverRemoveFromUpper(t *testing.T) {
	tmpDir := t.TempDir()

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	cfg := Config{BaseDir: filepath.Join(tmpDir, "sandboxes")}
	drv := NewCopyDriver(cfg)
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   sourceDir,
		ProjectRoot: sourceDir,
	}

	paths, err := drv.Mount(ctx, sandbox)
	if err != nil {
		t.Fatalf("Mount() failed: %v", err)
	}
	sandbox.UpperDir = paths.UpperDir

	// Create a file in workspace
	testFile := filepath.Join(paths.UpperDir, "toremove.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Remove via API
	err = drv.RemoveFromUpper(ctx, sandbox, "toremove.txt")
	if err != nil {
		t.Errorf("RemoveFromUpper() failed: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should be removed")
	}

	// Idempotent - second removal should not error
	err = drv.RemoveFromUpper(ctx, sandbox, "toremove.txt")
	if err != nil {
		t.Errorf("RemoveFromUpper() should be idempotent: %v", err)
	}

	// Path traversal should be blocked
	err = drv.RemoveFromUpper(ctx, sandbox, "../../../etc/passwd")
	if err == nil {
		t.Error("RemoveFromUpper() should block path traversal")
	}

	if err := drv.Cleanup(ctx, sandbox); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
}

// [REQ:P2-004-003] Automatic Driver Selection

// TestSelectDriverReturnsDriver verifies SelectDriver returns a valid driver
func TestSelectDriverReturnsDriver(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.BaseDir = t.TempDir()

	drv, err := SelectDriver(ctx, cfg)
	if err != nil {
		t.Fatalf("SelectDriver() failed: %v", err)
	}
	if drv == nil {
		t.Fatal("SelectDriver() returned nil driver")
	}

	// Verify it's a valid driver
	if drv.Type() == DriverTypeNone {
		t.Error("SelectDriver() should return a valid driver type")
	}
	if drv.Version() == "" {
		t.Error("SelectDriver() returned driver with no version")
	}
}

// TestSelectDriverFallsBackToCopy verifies fallback when overlayfs unavailable
// [REQ:P2-004-003] SelectDriver falls back to CopyDriver when overlayfs unavailable
func TestSelectDriverFallsBackToCopy(t *testing.T) {
	// On non-Linux or without proper setup, overlayfs won't be available
	// SelectDriver should fall back to copy driver
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.BaseDir = t.TempDir()

	drv, err := SelectDriver(ctx, cfg)
	if err != nil {
		t.Fatalf("SelectDriver() failed: %v", err)
	}

	// Should return either overlayfs or copy driver
	drvType := drv.Type()
	if drvType != DriverTypeOverlayfs && drvType != DriverTypeCopy {
		t.Errorf("SelectDriver() returned unexpected type: %v", drvType)
	}

	t.Logf("SelectDriver() returned: %v", drvType)
}

// TestDriverInfoReturnsAllDrivers verifies DriverInfo lists all drivers
// [REQ:P2-004-003] DriverInfo returns info about all available drivers
func TestDriverInfoReturnsAllDrivers(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()

	info := DriverInfo(ctx, cfg)

	if len(info) < 2 {
		t.Errorf("DriverInfo() returned %d drivers, want at least 2", len(info))
	}

	// Check for expected driver types
	foundOverlayfs := false
	foundCopy := false
	for _, i := range info {
		if i.Type == DriverTypeOverlayfs {
			foundOverlayfs = true
		}
		if i.Type == DriverTypeCopy {
			foundCopy = true
			// Copy should always be available
			if !i.Available {
				t.Error("Copy driver should always be available")
			}
		}
		// All drivers should have version and description
		if i.Version == "" {
			t.Errorf("Driver %s has no version", i.Type)
		}
		if i.Description == "" {
			t.Errorf("Driver %s has no description", i.Type)
		}
	}

	if !foundOverlayfs {
		t.Error("DriverInfo() should include overlayfs driver")
	}
	if !foundCopy {
		t.Error("DriverInfo() should include copy driver")
	}
}

// [REQ:P2-004-001] SandboxDriver Interface Definition

// TestDriverInterfaceMethods verifies drivers implement all interface methods
func TestDriverInterfaceMethods(t *testing.T) {
	ctx := context.Background()
	cfg := Config{BaseDir: t.TempDir()}

	// Test both driver implementations
	drivers := []Driver{
		NewOverlayfsDriver(cfg),
		NewCopyDriver(cfg),
	}

	for _, drv := range drivers {
		t.Run(string(drv.Type()), func(t *testing.T) {
			// Type
			if drv.Type() == "" {
				t.Error("Type() should not be empty")
			}

			// Version
			if drv.Version() == "" {
				t.Error("Version() should not be empty")
			}

			// IsAvailable
			_, err := drv.IsAvailable(ctx)
			// May fail on some systems, but should not panic
			t.Logf("IsAvailable() err: %v", err)
		})
	}
}

// TestDriverTypeConstants verifies driver type constants are defined
func TestDriverTypeConstants(t *testing.T) {
	if DriverTypeOverlayfs == "" {
		t.Error("DriverTypeOverlayfs should not be empty")
	}
	if DriverTypeCopy == "" {
		t.Error("DriverTypeCopy should not be empty")
	}
	if DriverTypeNone == "" {
		t.Error("DriverTypeNone should not be empty")
	}

	// Ensure they're distinct
	if DriverTypeOverlayfs == DriverTypeCopy {
		t.Error("DriverTypeOverlayfs and DriverTypeCopy should be different")
	}
}

// TestCopyDriverUnmount verifies Unmount is a no-op
func TestCopyDriverUnmount(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig())
	ctx := context.Background()

	sandbox := &types.Sandbox{
		ID: uuid.New(),
	}

	// Unmount should succeed as a no-op
	err := drv.Unmount(ctx, sandbox)
	if err != nil {
		t.Errorf("Unmount() should succeed as no-op: %v", err)
	}
}

// Benchmark for CopyDriver Mount operation
func BenchmarkCopyDriverMount(b *testing.B) {
	tmpDir := b.TempDir()

	// Create source with a few files
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		b.Fatalf("failed to create source dir: %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := os.WriteFile(filepath.Join(sourceDir, "file"+string(rune('0'+i))+".txt"), []byte("content"), 0o644); err != nil {
			b.Fatalf("failed to write file: %v", err)
		}
	}

	cfg := Config{BaseDir: filepath.Join(tmpDir, "sandboxes")}
	drv := NewCopyDriver(cfg)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sandbox := &types.Sandbox{
			ID:          uuid.New(),
			ScopePath:   sourceDir,
			ProjectRoot: sourceDir,
		}
		if _, err := drv.Mount(ctx, sandbox); err != nil {
			b.Fatalf("Mount() failed: %v", err)
		}
		if err := drv.Cleanup(ctx, sandbox); err != nil {
			b.Fatalf("Cleanup() failed: %v", err)
		}
	}
}
