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

// BaseDir returns the configured base directory for sandboxes.
func (d *OverlayfsDriver) BaseDir() string {
	return d.config.BaseDir
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
		if !isMountPoint(s.MergedDir) {
			return nil
		}
		return fmt.Errorf("unmount failed: %v (output: %s)", cmdErr, strings.TrimSpace(string(output)))
	}

	return nil
}

// GetChangedFiles returns the list of files changed in the upper layer.
// Delegates to shared helper for overlayfs-based change detection.
func (d *OverlayfsDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return getOverlayChangedFiles(s)
}

// --- Partial Approval Support (OT-P1-002) ---

// RemoveFromUpper removes a file from the upper (writable) layer.
// Delegates to shared helper with path traversal protection.
func (d *OverlayfsDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

// Cleanup removes all sandbox artifacts.
// Delegates to shared helper with driver-specific unmount.
func (d *OverlayfsDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return cleanupSandboxDir(d.config.BaseDir, s.ID, func() error {
		return d.Unmount(ctx, s)
	})
}

// --- Temporal Safety Methods ---

// IsMounted checks if the sandbox overlay is currently mounted.
// Delegates to shared helper.
func (d *OverlayfsDriver) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	if s.MergedDir == "" {
		return false, nil // No merge dir means nothing to check
	}
	return isMountPoint(s.MergedDir), nil
}

// VerifyMountIntegrity performs comprehensive health checks on the mount.
// Delegates to shared helper.
func (d *OverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(s)
}
