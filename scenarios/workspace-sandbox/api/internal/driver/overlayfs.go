package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// OverlayfsDriver implements the Driver interface using Linux overlayfs.
type OverlayfsDriver struct {
	config Config
}

// NewOverlayfsDriver creates a new overlayfs driver.
func NewOverlayfsDriver(cfg Config) *OverlayfsDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &OverlayfsDriver{config: cfg}
}

// Type returns the driver type.
func (d *OverlayfsDriver) Type() DriverType {
	return DriverTypeOverlayfs
}

// Version returns the driver version.
func (d *OverlayfsDriver) Version() string {
	return "1.0"
}

// IsAvailable checks if overlayfs is available on this system.
func (d *OverlayfsDriver) IsAvailable(ctx context.Context) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
	}

	// Check if overlayfs module is available
	if _, err := os.Stat("/proc/filesystems"); err == nil {
		data, err := os.ReadFile("/proc/filesystems")
		if err == nil && strings.Contains(string(data), "overlay") {
			return true, nil
		}
	}

	// Try to check via modprobe (may need privileges)
	cmd := exec.CommandContext(ctx, "modprobe", "-n", "overlay")
	if err := cmd.Run(); err == nil {
		return true, nil
	}

	return false, fmt.Errorf("overlayfs module not available")
}

// Mount creates the overlay mount for a sandbox.
func (d *OverlayfsDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	// Create sandbox directory structure
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())

	paths := &MountPaths{
		LowerDir:  s.ScopePath,
		UpperDir:  filepath.Join(sandboxDir, "upper"),
		WorkDir:   filepath.Join(sandboxDir, "work"),
		MergedDir: filepath.Join(sandboxDir, "merged"),
	}

	// Create directories
	for _, dir := range []string{paths.UpperDir, paths.WorkDir, paths.MergedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Build mount options
	mountOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		paths.LowerDir, paths.UpperDir, paths.WorkDir)

	// Attempt mount
	if err := d.mountOverlay(ctx, paths.MergedDir, mountOpts); err != nil {
		// Cleanup on failure
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("failed to mount overlay: %w", err)
	}

	return paths, nil
}

// mountOverlay performs the actual overlay mount.
func (d *OverlayfsDriver) mountOverlay(ctx context.Context, target, opts string) error {
	if d.config.UseFuseOverlayfs {
		return d.mountFuseOverlay(ctx, target, opts)
	}
	return d.mountKernelOverlay(ctx, target, opts)
}

// mountKernelOverlay uses the kernel overlay filesystem.
func (d *OverlayfsDriver) mountKernelOverlay(ctx context.Context, target, opts string) error {
	// Try syscall mount first
	err := syscall.Mount("overlay", target, "overlay", 0, opts)
	if err == nil {
		return nil
	}

	// Fall back to mount command for better error handling
	cmd := exec.CommandContext(ctx, "mount", "-t", "overlay", "overlay", "-o", opts, target)
	output, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return fmt.Errorf("mount failed: %v (output: %s)", cmdErr, strings.TrimSpace(string(output)))
	}

	return nil
}

// mountFuseOverlay uses fuse-overlayfs for unprivileged operation.
func (d *OverlayfsDriver) mountFuseOverlay(ctx context.Context, target, opts string) error {
	// fuse-overlayfs has slightly different option format
	cmd := exec.CommandContext(ctx, "fuse-overlayfs", "-o", opts, target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fuse-overlayfs mount failed: %v (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Unmount removes the overlay mount.
func (d *OverlayfsDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	if s.MergedDir == "" {
		return nil // Nothing to unmount
	}

	// Try lazy unmount first (handles busy mounts)
	err := syscall.Unmount(s.MergedDir, syscall.MNT_DETACH)
	if err == nil {
		return nil
	}

	// Fall back to umount command
	cmd := exec.CommandContext(ctx, "umount", "-l", s.MergedDir)
	output, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		// Check if already unmounted
		if !d.isMounted(s.MergedDir) {
			return nil
		}
		return fmt.Errorf("unmount failed: %v (output: %s)", cmdErr, strings.TrimSpace(string(output)))
	}

	return nil
}

// isMounted checks if a path is a mount point.
func (d *OverlayfsDriver) isMounted(path string) bool {
	cmd := exec.Command("mountpoint", "-q", path)
	return cmd.Run() == nil
}

// GetChangedFiles returns the list of files changed in the upper layer.
func (d *OverlayfsDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	if s.UpperDir == "" {
		return nil, fmt.Errorf("sandbox upper directory not set")
	}

	var changes []*types.FileChange

	err := filepath.Walk(s.UpperDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == s.UpperDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(s.UpperDir, path)
		if err != nil {
			return err
		}

		// Skip overlayfs internal files
		if strings.HasPrefix(relPath, ".overlay") {
			return nil
		}

		// Determine change type
		changeType := d.detectChangeType(s, relPath, info)

		change := &types.FileChange{
			ID:             uuid.New(),
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
		return nil, fmt.Errorf("failed to walk upper directory: %w", err)
	}

	return changes, nil
}

// detectChangeType determines if a file was added, modified, or deleted.
func (d *OverlayfsDriver) detectChangeType(s *types.Sandbox, relPath string, upperInfo os.FileInfo) types.ChangeType {
	// Check if file exists in lower layer
	lowerPath := filepath.Join(s.LowerDir, relPath)
	lowerInfo, err := os.Stat(lowerPath)

	if os.IsNotExist(err) {
		return types.ChangeTypeAdded
	}

	if err != nil {
		// Can't determine, assume modified
		return types.ChangeTypeModified
	}

	// Check for overlayfs whiteout (deletion marker)
	if d.isWhiteout(filepath.Join(s.UpperDir, relPath)) {
		return types.ChangeTypeDeleted
	}

	// Compare sizes and modes
	if lowerInfo.Size() != upperInfo.Size() || lowerInfo.Mode() != upperInfo.Mode() {
		return types.ChangeTypeModified
	}

	// TODO: Could add content hash comparison for more accuracy
	return types.ChangeTypeModified
}

// isWhiteout checks if a file is an overlayfs whiteout marker.
func (d *OverlayfsDriver) isWhiteout(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	// Overlayfs whiteouts are character devices with major 0, minor 0
	if info.Mode()&os.ModeCharDevice != 0 {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return stat.Rdev == 0
		}
	}

	return false
}

// --- Partial Approval Support (OT-P1-002) ---

// RemoveFromUpper removes a file from the upper (writable) layer.
// This is used after partial approval to clean up applied files
// while preserving unapproved changes for follow-up approvals.
// Returns nil if file doesn't exist (idempotent).
func (d *OverlayfsDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}

	// Security: ensure filePath is relative and doesn't escape the sandbox
	cleanPath := filepath.Clean(filePath)
	if filepath.IsAbs(cleanPath) {
		// Strip leading slash for relative path construction
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", filePath)
	}

	fullPath := filepath.Join(s.UpperDir, cleanPath)

	// Verify fullPath is actually under UpperDir (defense in depth)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	absUpperDir, err := filepath.Abs(s.UpperDir)
	if err != nil {
		return fmt.Errorf("failed to resolve upper dir: %w", err)
	}
	if !strings.HasPrefix(absFullPath, absUpperDir) {
		return fmt.Errorf("path escapes upper directory: %s", filePath)
	}

	// Remove the file - ignore not-exist errors (idempotent)
	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file from upper layer: %w", err)
	}

	// Also remove empty parent directories up to UpperDir
	// This prevents leftover empty directories after file removal
	dir := filepath.Dir(fullPath)
	for dir != absUpperDir && dir != "." && dir != "/" {
		// Try to remove - will fail if not empty, which is fine
		if rmErr := os.Remove(dir); rmErr != nil {
			break // Directory not empty or error, stop
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// Cleanup removes all sandbox artifacts.
func (d *OverlayfsDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	// Unmount first
	if err := d.Unmount(ctx, s); err != nil {
		// Log but continue with cleanup
		fmt.Printf("warning: unmount failed during cleanup: %v\n", err)
	}

	// Remove sandbox directory
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())
	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("failed to remove sandbox directory: %w", err)
	}

	return nil
}

// --- Temporal Safety Methods ---

// IsMounted checks if the sandbox overlay is currently mounted.
// This is critical for temporal safety - we must verify state before operations.
func (d *OverlayfsDriver) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	if s.MergedDir == "" {
		return false, nil // No merge dir means nothing to check
	}
	return d.isMounted(s.MergedDir), nil
}

// VerifyMountIntegrity performs comprehensive health checks on the mount.
// Returns nil if healthy, error describing the problem otherwise.
//
// Checks performed:
//   - Mount point exists and is a directory
//   - Mount is actually mounted (not just a directory)
//   - Mount is accessible (can list contents)
//   - Upper dir exists and is writable
func (d *OverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	// Check merged dir exists
	if s.MergedDir == "" {
		return fmt.Errorf("merged directory path is empty")
	}

	info, err := os.Stat(s.MergedDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("merged directory does not exist: %s", s.MergedDir)
	}
	if err != nil {
		return fmt.Errorf("cannot stat merged directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("merged path is not a directory: %s", s.MergedDir)
	}

	// Verify it's actually mounted
	if !d.isMounted(s.MergedDir) {
		return fmt.Errorf("merged directory is not mounted (may be stale): %s", s.MergedDir)
	}

	// Check accessibility by attempting to list the directory
	entries, err := os.ReadDir(s.MergedDir)
	if err != nil {
		return fmt.Errorf("merged directory is not accessible: %w", err)
	}
	_ = entries // We just want to verify access, don't need the entries

	// Check upper dir for write capability
	if s.UpperDir != "" {
		upperInfo, err := os.Stat(s.UpperDir)
		if err != nil {
			return fmt.Errorf("upper directory check failed: %w", err)
		}
		if !upperInfo.IsDir() {
			return fmt.Errorf("upper path is not a directory: %s", s.UpperDir)
		}

		// Try to verify write access by checking directory is writable
		// Note: We don't actually write a test file to avoid side effects
		if upperInfo.Mode().Perm()&0200 == 0 {
			return fmt.Errorf("upper directory appears not writable: %s", s.UpperDir)
		}
	}

	return nil
}
