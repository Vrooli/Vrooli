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

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// Compile-time assertions: OverlayfsDriver implements both the composite
// Driver interface AND MountVerifier (kernel overlayfs has a real mount
// to verify).
var (
	_ Driver        = (*OverlayfsDriver)(nil)
	_ MountVerifier = (*OverlayfsDriver)(nil)
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
//
// Contract: this driver is available when the kernel overlayfs module is
// loaded AND the API can actually mount overlayfs — which on a stock
// distro means we're either inside a user namespace (kernel 5.11+ allows
// unprivileged mount via `unshare -U -m -r`) or running as root with
// CAP_SYS_ADMIN. We don't perform a transient test mount; the boot-time
// self-check in main.go verifies the deployment shape.
func (d *OverlayfsDriver) IsAvailable(ctx context.Context) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
	}

	// Module must be loaded.
	moduleLoaded := false
	if data, err := os.ReadFile("/proc/filesystems"); err == nil && strings.Contains(string(data), "overlay") {
		moduleLoaded = true
	} else {
		cmd := exec.CommandContext(ctx, "modprobe", "-n", "overlay")
		if err := cmd.Run(); err == nil {
			moduleLoaded = true
		}
	}
	if !moduleLoaded {
		return false, fmt.Errorf("overlayfs module not available")
	}

	// We must be able to mount. Inside a user namespace the kernel allows
	// unprivileged overlayfs mounts (5.11+). Otherwise CAP_SYS_ADMIN is
	// required. checkCapSysAdmin already returns false when we're inside
	// userns, so the two checks compose cleanly.
	if InUserNamespace() {
		return true, nil
	}
	if os.Geteuid() == 0 || checkCapSysAdmin() {
		return true, nil
	}
	return false, fmt.Errorf("overlayfs module loaded but mount requires either a user namespace (`unshare -U -m -r`) or CAP_SYS_ADMIN")
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

// ListSandboxDirs walks BaseDir and returns the IDs of every UUID-named
// subdirectory. See the Driver interface docstring for orphan-reconciliation
// rationale.
func (d *OverlayfsDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	return listSandboxDirsInBase(d.config.BaseDir)
}

// CleanupOrphan releases a sandbox by ID alone. Idempotent. Lazy umount
// (`umount -l`) is used because orphans by definition have no live
// owning process to coordinate with — the caller is the reconciler,
// not the agent that mounted it.
func (d *OverlayfsDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	sandboxDir := filepath.Join(d.config.BaseDir, id.String())
	if _, err := os.Stat(sandboxDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat orphan sandbox dir %q: %w", sandboxDir, err)
	}

	mergedDir := filepath.Join(sandboxDir, "merged")
	if isMountPoint(mergedDir) {
		// Best-effort: errors are surfaced via the rm -rf below if the
		// FS is genuinely wedged.
		_, _ = exec.CommandContext(ctx, "umount", "-l", mergedDir).CombinedOutput()
	}

	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("remove orphan sandbox dir %q: %w", sandboxDir, err)
	}
	return nil
}

// --- Temporal Safety Methods ---

// VerifyMountIntegrity performs comprehensive health checks on the mount.
// Delegates to shared helper.
func (d *OverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(s)
}
