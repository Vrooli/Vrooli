// Package driver provides filesystem drivers for sandbox isolation.
//
// copy.go implements a cross-platform fallback driver that works on any OS
// by using file copies instead of overlayfs.
//
// [OT-P2-004] Cross-Platform Driver Interface
package driver

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// Compile-time assertion that CopyDriver satisfies the composite Driver
// interface (MountDriver + ChangeTracker). It intentionally does NOT
// implement MountVerifier — callers use VerifyIfSupported.
var _ Driver = (*CopyDriver)(nil)

// CopyDriver implements the Driver interface using file copies.
// This is a cross-platform fallback driver that works on any OS.
//
// # How It Works
//
// Instead of overlayfs, CopyDriver:
//  1. Copies the scope directory to an "original" snapshot directory
//  2. Creates a "workspace" directory where actual work happens
//  3. On diff, compares workspace against original to detect changes
//  4. On approval, copies changed files back to the canonical repo
//
// # Trade-offs vs OverlayfsDriver
//
// Pros:
//   - Works on any OS (macOS, Windows, Linux without overlayfs)
//   - No kernel dependencies or special privileges needed
//   - Simple implementation, easy to debug
//
// Cons:
//   - Higher disk usage (full copy of scope directory)
//   - Slower sandbox creation for large directories
//   - No true copy-on-write semantics
//
// # Recommended Use Cases
//
//   - Development on macOS/Windows
//   - Small to medium scope directories
//   - When overlayfs is unavailable
type CopyDriver struct {
	config Config
}

// NewCopyDriver creates a new copy-based fallback driver.
func NewCopyDriver(cfg Config) *CopyDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &CopyDriver{config: cfg}
}

// ID returns the canonical driver ID.
func (d *CopyDriver) ID() DriverID {
	return DriverCopy
}

// RequiresBwrap returns ModeNone: the copy driver has no real mount, so
// commands run directly in the workspace dir without namespace isolation.
func (d *CopyDriver) RequiresBwrap() IsolationMode {
	return ModeNone
}

// Capabilities reports the copy driver's contract: no home overlay
// support (it can't mount), no CoW (it copies), no namespace isolation.
//
// DOC: home-overlay seam — copy driver explicitly opts out.
func (d *CopyDriver) Capabilities() DriverCapabilities {
	return DriverCapabilities{
		HomeOverlay:        false,
		CoW:                false,
		NamespaceIsolation: ModeNone,
	}
}

// BaseDir returns the configured base directory for sandboxes.
func (d *CopyDriver) BaseDir() string {
	return d.config.BaseDir
}

// Version returns the driver version.
func (d *CopyDriver) Version() string {
	return "1.0"
}

// IsAvailable checks if the copy driver can operate on this system.
// The copy driver is always available as it uses only standard file operations.
func (d *CopyDriver) IsAvailable(ctx context.Context) (bool, error) {
	// Copy driver is always available
	return true, nil
}

// Mount creates the sandbox workspace by copying the scope directory.
//
// Directory structure:
//
//	{baseDir}/{sandboxID}/
//	  original/   - snapshot of scope at creation time (read-only reference)
//	  workspace/  - working directory where changes are made
//
// Note: We don't have a separate merged dir like overlayfs.
// The workspace IS the merged view.
func (d *CopyDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())

	paths := &MountPaths{
		LowerDir:  filepath.Join(sandboxDir, "original"),  // Snapshot of original
		UpperDir:  filepath.Join(sandboxDir, "workspace"), // Working directory
		WorkDir:   filepath.Join(sandboxDir, "meta"),      // Metadata/temp storage
		MergedDir: filepath.Join(sandboxDir, "workspace"), // Same as UpperDir for copy driver
	}

	// Create directories
	for _, dir := range []string{paths.LowerDir, paths.UpperDir, paths.WorkDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Copy scope directory to both original and workspace
	// Original serves as the baseline for diff generation
	// Workspace is where actual work happens
	if err := copyDirectory(s.ScopePath, paths.LowerDir); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("failed to copy scope to original: %w", err)
	}

	if err := copyDirectory(s.ScopePath, paths.UpperDir); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("failed to copy scope to workspace: %w", err)
	}

	// Copy driver does not provide a home overlay — record it explicitly.
	s.HomeOverlayState = types.HomeOverlayUnsupported
	return paths, nil
}

// Unmount is a no-op for the copy driver since there's no actual mount.
func (d *CopyDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	// No mount to unmount
	return nil
}

// GetChangedFiles returns the list of files that differ between workspace and original.
func (d *CopyDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	if s.UpperDir == "" {
		return nil, fmt.Errorf("sandbox workspace directory not set")
	}

	originalDir := s.LowerDir
	workspaceDir := s.UpperDir

	var changes []*types.FileChange

	// Track files we've seen in workspace
	workspaceFiles := make(map[string]bool)

	// Walk workspace to find added and modified files
	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == workspaceDir {
			return nil
		}

		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}

		if isOverlayMarker(relPath) {
			return nil
		}

		// Skip hidden files and directories that might be metadata
		if strings.HasPrefix(relPath, ".") {
			return nil
		}

		workspaceFiles[relPath] = true

		// Check if file exists in original
		originalPath := filepath.Join(originalDir, relPath)
		originalInfo, originalErr := os.Stat(originalPath)

		var changeType types.ChangeType

		if os.IsNotExist(originalErr) {
			// File doesn't exist in original - it's added
			changeType = types.ChangeTypeAdded
		} else if originalErr != nil {
			// Error accessing original - treat as modified
			changeType = types.ChangeTypeModified
		} else if info.IsDir() && originalInfo.IsDir() {
			// Both are directories - skip
			return nil
		} else if filesAreDifferent(originalPath, path, originalInfo, info) {
			// Files are different - modified
			changeType = types.ChangeTypeModified
		} else {
			// Files are the same - no change
			return nil
		}

		change := &types.FileChange{
			ID:             StableFileID(s.ID, relPath),
			SandboxID:      s.ID,
			FilePath:       relPath,
			ChangeType:     changeType,
			FileSize:       info.Size(),
			FileMode:       int(info.Mode()),
			DetectedAt:     time.Now(),
			ApprovalStatus: types.ApprovalPending,
		}

		changes = append(changes, change)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk workspace directory: %w", err)
	}

	// Walk original to find deleted files
	err = filepath.Walk(originalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == originalDir {
			return nil
		}

		relPath, err := filepath.Rel(originalDir, path)
		if err != nil {
			return err
		}

		if isOverlayMarker(relPath) {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(relPath, ".") {
			return nil
		}

		// If we already saw this in workspace, it's not deleted
		if workspaceFiles[relPath] {
			return nil
		}

		// File exists in original but not in workspace - deleted
		if !info.IsDir() {
			change := &types.FileChange{
				ID:             StableFileID(s.ID, relPath),
				SandboxID:      s.ID,
				FilePath:       relPath,
				ChangeType:     types.ChangeTypeDeleted,
				FileSize:       info.Size(),
				FileMode:       int(info.Mode()),
				DetectedAt:     time.Now(),
				ApprovalStatus: types.ApprovalPending,
			}
			changes = append(changes, change)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk original directory: %w", err)
	}

	return changes, nil
}

// RemoveFromUpper removes a file from the workspace directory.
// Delegates to shared helper with path traversal protection.
func (d *CopyDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no workspace directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

// Cleanup removes all sandbox artifacts.
func (d *CopyDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())
	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("failed to remove sandbox directory: %w", err)
	}
	return nil
}

// ListSandboxDirs walks BaseDir and returns the IDs of every UUID-named
// subdirectory. See the Driver interface docstring for orphan-reconciliation
// rationale.
func (d *CopyDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	return listSandboxDirsInBase(d.config.BaseDir)
}

// CleanupOrphan releases a sandbox by ID alone. CopyDriver has no mounts
// to release, so this is purely directory removal. Idempotent: missing
// dirs are a no-op.
func (d *CopyDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	sandboxDir := filepath.Join(d.config.BaseDir, id.String())
	if _, err := os.Stat(sandboxDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat orphan sandbox dir %q: %w", sandboxDir, err)
	}
	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("remove orphan sandbox dir %q: %w", sandboxDir, err)
	}
	return nil
}

// CopyDriver intentionally does not implement MountVerifier: there is no
// real mount to verify. Callers should use VerifyIfSupported, which
// short-circuits to nil for drivers without a mount.

// --- Helper Functions ---

// copyDirectory recursively copies a directory tree.
func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file.
func copyFile(src, dst string, mode fs.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// filesAreDifferent checks if two files have different content.
func filesAreDifferent(path1, path2 string, info1, info2 fs.FileInfo) bool {
	// Different sizes means different content
	if info1.Size() != info2.Size() {
		return true
	}

	// Different modes means different
	if info1.Mode() != info2.Mode() {
		return true
	}

	// For small files, compare content directly
	if info1.Size() < 64*1024 { // 64KB threshold
		content1, err1 := os.ReadFile(path1)
		content2, err2 := os.ReadFile(path2)
		if err1 != nil || err2 != nil {
			return true
		}
		return string(content1) != string(content2)
	}

	// For larger files, modification time is a reasonable heuristic
	// (though not perfect - content could differ even with same mtime)
	return info1.ModTime() != info2.ModTime()
}

func isOverlayMarker(relPath string) bool {
	baseName := filepath.Base(relPath)
	if baseName == ".wh..opq" {
		return true
	}
	return strings.HasPrefix(baseName, ".wh.")
}
