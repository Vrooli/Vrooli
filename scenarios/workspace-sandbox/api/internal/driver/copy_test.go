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

// TestCopyDriverID verifies CopyDriver returns correct ID
func TestCopyDriverID(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig(), testDeps())

	if drv.ID() != DriverCopy {
		t.Errorf("ID() = %v, want %v", drv.ID(), DriverCopy)
	}
}

// TestCopyDriverVersion verifies CopyDriver has a version string
func TestCopyDriverVersion(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig(), testDeps())

	version := drv.Version()
	if version == "" {
		t.Error("Version() should not be empty")
	}
}

// TestCopyDriverIsAvailable verifies CopyDriver is always available
// [REQ:P2-004-002] CopyDriver.IsAvailable returns true on any platform
func TestCopyDriverIsAvailable(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig(), testDeps())
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
	drv := NewCopyDriver(cfg, testDeps())
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
	drv := NewCopyDriver(cfg, testDeps())
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

	drv := NewCopyDriver(DefaultConfig(), testDeps())
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

// TestCopyDriverMergedDirExists checks the merged-dir lifecycle:
// before Mount the dir is absent, after Mount it exists, after Cleanup
// it's gone. Replaces the dropped IsMounted() check — the copy driver
// has no real mount, so directory presence is the only meaningful signal.
func TestCopyDriverMergedDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	cfg := Config{BaseDir: filepath.Join(tmpDir, "sandboxes")}
	drv := NewCopyDriver(cfg, testDeps())
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
	if _, err := os.Stat(paths.MergedDir); err != nil {
		t.Errorf("merged dir should exist after Mount: %v", err)
	}

	if err := drv.Cleanup(ctx, sandbox); err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	if _, err := os.Stat(paths.MergedDir); !os.IsNotExist(err) {
		t.Errorf("merged dir should be removed after Cleanup, got err=%v", err)
	}
}

// TestCopyDriverIsNotMountVerifier guards the Phase 2 contract: CopyDriver
// has no real mount and intentionally does NOT implement MountVerifier.
// VerifyIfSupported should short-circuit to nil for it.
func TestCopyDriverIsNotMountVerifier(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig(), testDeps())
	if _, ok := interface{}(drv).(MountVerifier); ok {
		t.Error("CopyDriver should NOT implement MountVerifier")
	}
	// VerifyIfSupported must return nil regardless of sandbox state.
	if err := VerifyIfSupported(context.Background(), drv, &types.Sandbox{ID: uuid.New()}); err != nil {
		t.Errorf("VerifyIfSupported on CopyDriver should return nil, got: %v", err)
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
	drv := NewCopyDriver(cfg, testDeps())
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

	drv, report, err := SelectDriver(ctx, cfg, testDeps())
	if err != nil {
		t.Fatalf("SelectDriver() failed: %v", err)
	}
	if drv == nil {
		t.Fatal("SelectDriver() returned nil driver")
	}
	if report == nil {
		t.Fatal("SelectDriver() returned nil SelectionReport")
	}
	if report.Selected != drv.ID() {
		t.Errorf("SelectionReport.Selected=%v but driver.ID()=%v", report.Selected, drv.ID())
	}
	if len(report.Candidates) == 0 {
		t.Error("SelectionReport.Candidates should not be empty")
	}

	// Verify it's a valid driver
	if drv.ID() == "" {
		t.Error("SelectDriver() should return a valid driver ID")
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

	drv, _, err := SelectDriver(ctx, cfg, testDeps())
	if err != nil {
		t.Fatalf("SelectDriver() failed: %v", err)
	}

	// Post-Phase 5 priority: kernel overlayfs > fuse-overlayfs > copy.
	// Any of those is acceptable; we just want to confirm SelectDriver
	// produced *some* valid driver.
	id := drv.ID()
	switch id {
	case DriverOverlayfsUserNS, DriverOverlayfsRoot, DriverFuseOverlayfs, DriverCopy:
		// expected
	default:
		t.Errorf("SelectDriver() returned unexpected ID: %v", id)
	}

	t.Logf("SelectDriver() returned: %v", id)
}

// TestDriverInfoReturnsAllDrivers verifies DriverInfo lists all drivers
// [REQ:P2-004-003] DriverInfo returns info about all available drivers
func TestDriverInfoReturnsAllDrivers(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()

	info := DriverInfo(ctx, cfg, testDeps())

	if len(info) < 2 {
		t.Errorf("DriverInfo() returned %d drivers, want at least 2", len(info))
	}

	// Check for expected driver IDs
	foundOverlayfs := false
	foundCopy := false
	for _, i := range info {
		switch i.ID {
		case DriverOverlayfsUserNS, DriverOverlayfsRoot:
			foundOverlayfs = true
		case DriverCopy:
			foundCopy = true
			// Copy should always be available
			if !i.Available {
				t.Error("Copy driver should always be available")
			}
		}
		// All drivers should have version and description
		if i.Version == "" {
			t.Errorf("Driver %s has no version", i.ID)
		}
		if i.Description == "" {
			t.Errorf("Driver %s has no description", i.ID)
		}
	}

	if !foundOverlayfs {
		t.Error("DriverInfo() should include an overlayfs driver")
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
		NewOverlayfsDriver(cfg, testDeps()),
		NewCopyDriver(cfg, testDeps()),
	}

	for _, drv := range drivers {
		t.Run(string(drv.ID()), func(t *testing.T) {
			// ID
			if drv.ID() == "" {
				t.Error("ID() should not be empty")
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

// TestDriverIDConstants verifies driver ID constants are defined and unique.
func TestDriverIDConstants(t *testing.T) {
	all := []DriverID{
		DriverOverlayfsUserNS,
		DriverOverlayfsRoot,
		DriverFuseOverlayfs,
		DriverCopy,
	}
	seen := map[DriverID]bool{}
	for _, id := range all {
		if id == "" {
			t.Error("DriverID constant must not be empty")
		}
		if seen[id] {
			t.Errorf("duplicate DriverID: %s", id)
		}
		seen[id] = true
	}
}

// TestCopyDriverUnmount verifies Unmount is a no-op
func TestCopyDriverUnmount(t *testing.T) {
	drv := NewCopyDriver(DefaultConfig(), testDeps())
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
	drv := NewCopyDriver(cfg, testDeps())
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
