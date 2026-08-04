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

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/driver/changedetect"
	"workspace-sandbox/internal/process"
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
// # Trade-offs vs OverlayDriver
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
	config  Config
	clock   clock.Clock
	starter process.Starter
}

// NewCopyDriver creates a new copy-based fallback driver. deps.Clock is
// the time source for FileChange.DetectedAt timestamps. CopyDriver does
// not actually mount or spawn helper binaries, but the constructor still
// takes Deps for symmetry with the overlay drivers — every driver
// factory now has the same shape so SelectDriver / NewDriverFor can
// pass through a single Deps value.
func NewCopyDriver(cfg Config, deps Deps) *CopyDriver {
	deps.Validate("driver.NewCopyDriver")
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &CopyDriver{config: cfg, clock: deps.Clock, starter: deps.Starter}
}

// ID returns the canonical driver ID.
func (d *CopyDriver) ID() DriverID {
	return DriverCopy
}

// RequiredContainment returns ContainmentNone: the copy driver has no real
// mount, so commands run directly in the workspace dir without a
// containment backend.
func (d *CopyDriver) RequiredContainment() ContainmentLevel {
	return ContainmentNone
}

// Capabilities reports the copy driver's contract: no home overlay
// support (it can't mount), no CoW (it copies), no namespace isolation.
//
// DOC: home-overlay seam — copy driver explicitly opts out.
func (d *CopyDriver) Capabilities() DriverCapabilities {
	return DriverCapabilities{
		HomeOverlay:        false,
		CoW:                false,
		NamespaceIsolation: ContainmentNone,
		Tracking:           true,
		Protected:          false,
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
	return changedetect.Walk(ctx,
		changedetect.WalkOpts{Lower: s.LowerDir, Upper: s.UpperDir, SandboxID: s.ID, IgnoreMatcher: diff.NewGitIgnoreMatcher(s.ProjectRoot, diff.NewExecCommandRunner(d.starter))},
		&changedetect.CopyStrategy{FileIDFn: StableFileID},
		d.clock.Now(),
	)
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
