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

var (
	_ Driver        = (*OverlayfsDriver)(nil)
	_ MountVerifier = (*OverlayfsDriver)(nil)
)

// OverlayfsDriver implements the Driver interface using kernel overlayfs.
// Available inside a user namespace (5.11+) or with CAP_SYS_ADMIN.
//
// The same struct backs two DriverIDs (DriverOverlayfsUserNS and
// DriverOverlayfsRoot): mount machinery is identical, only the privilege
// path differs. Construct via NewOverlayfsUserNSDriver /
// NewOverlayfsRootDriver, or NewOverlayfsDriver to auto-pick from
// InUserNamespace.
type OverlayfsDriver struct {
	config Config
	id     DriverID
}

// NewOverlayfsDriver returns an OverlayfsDriver labelled with the variant
// appropriate to the current process: UserNS when running inside a user
// namespace, Root otherwise. Convenience constructor used by the
// auto-select path; explicit operator selection goes through
// NewOverlayfsUserNSDriver / NewOverlayfsRootDriver via NewDriverFor.
func NewOverlayfsDriver(cfg Config) *OverlayfsDriver {
	if InUserNamespace() {
		return NewOverlayfsUserNSDriver(cfg)
	}
	return NewOverlayfsRootDriver(cfg)
}

func NewOverlayfsUserNSDriver(cfg Config) *OverlayfsDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &OverlayfsDriver{config: cfg, id: DriverOverlayfsUserNS}
}

func NewOverlayfsRootDriver(cfg Config) *OverlayfsDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &OverlayfsDriver{config: cfg, id: DriverOverlayfsRoot}
}

func (d *OverlayfsDriver) ID() DriverID                 { return d.id }
func (d *OverlayfsDriver) RequiresBwrap() IsolationMode { return ModeBwrapRequired }
func (d *OverlayfsDriver) BaseDir() string              { return d.config.BaseDir }
func (d *OverlayfsDriver) Version() string              { return "1.0" }

// IsAvailable checks the kernel overlayfs module is loaded AND we can
// actually mount (userns 5.11+, root, or CAP_SYS_ADMIN).
func (d *OverlayfsDriver) IsAvailable(ctx context.Context) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
	}
	moduleLoaded := false
	if data, err := os.ReadFile("/proc/filesystems"); err == nil && strings.Contains(string(data), "overlay") {
		moduleLoaded = true
	} else if err := exec.CommandContext(ctx, "modprobe", "-n", "overlay").Run(); err == nil {
		moduleLoaded = true
	}
	if !moduleLoaded {
		return false, fmt.Errorf("overlayfs module not available")
	}
	if InUserNamespace() || os.Geteuid() == 0 || checkCapSysAdmin() {
		return true, nil
	}
	return false, fmt.Errorf("overlayfs requires a user namespace (`unshare -U -m -r`) or CAP_SYS_ADMIN")
}

// mount performs a kernel overlayfs mount with a syscall fast path and a
// `mount` command fallback for clearer errors.
func (d *OverlayfsDriver) mount(ctx context.Context, target, opts string) error {
	if err := syscall.Mount("overlay", target, "overlay", 0, opts); err == nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "mount", "-t", "overlay", "overlay", "-o", opts, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// unmount lazy-unmounts target. Idempotent: returns nil when target was
// not a mount point.
func (d *OverlayfsDriver) unmount(ctx context.Context, target string) error {
	if err := syscall.Unmount(target, syscall.MNT_DETACH); err == nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "umount", "-l", target).CombinedOutput()
	if err != nil && isMountPoint(target) {
		return fmt.Errorf("umount: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *OverlayfsDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())
	return mountOverlayPair(ctx, sandboxDir, s.ScopePath, os.Getenv("HOME"), d.mount)
}

func (d *OverlayfsDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	if s.MergedDir == "" {
		return nil
	}
	sandboxDir := filepath.Dir(s.MergedDir)
	unmountHomeOverlay(ctx, sandboxDir, d.unmount)
	if !isMountPoint(s.MergedDir) {
		return nil
	}
	return d.unmount(ctx, s.MergedDir)
}

func (d *OverlayfsDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return getOverlayChangedFiles(s)
}

func (d *OverlayfsDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

func (d *OverlayfsDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return cleanupSandboxDirAll(ctx, d.config.BaseDir, s.ID, d.unmount)
}

func (d *OverlayfsDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	return listSandboxDirsInBase(d.config.BaseDir)
}

func (d *OverlayfsDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	return cleanupSandboxDirAll(ctx, d.config.BaseDir, id, d.unmount)
}

func (d *OverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(s)
}
