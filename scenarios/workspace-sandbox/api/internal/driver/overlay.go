// Package driver. overlay.go: the unified overlay driver that backs all
// three overlay-flavored DriverIDs (overlayfs-userns, overlayfs-root,
// fuse-overlayfs). The bodies of every method are identical across
// flavors — only the mount/unmount syscalls, the availability probe,
// the version string, and the required isolation mode differ. Those
// vary points are captured as closures + fields on a single struct.
package driver

import (
	"context"
	"fmt"
	"log/slog"
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
	_ Driver        = (*OverlayDriver)(nil)
	_ MountVerifier = (*OverlayDriver)(nil)
)

// availabilityFunc reports whether this overlay flavor can be used on
// the current host. The error explains *why* when available is false so
// /driver/options can render an actionable diagnostic.
type availabilityFunc func(ctx context.Context) (bool, error)

// OverlayDriver implements the Driver interface for every overlay-flavored
// backend (kernel overlayfs in a user namespace, kernel overlayfs as
// root/CAP_SYS_ADMIN, and fuse-overlayfs userspace). Construct via the
// flavor-specific factories — never directly.
type OverlayDriver struct {
	id           DriverID
	config       Config
	mount        mountFunc
	unmount      unmountFunc
	availability availabilityFunc
	version      func() string
	isolation    IsolationMode
}

// NewOverlayfsUserNSDriver builds the kernel-overlayfs flavor that runs
// inside an unprivileged user namespace (Linux 5.11+).
func NewOverlayfsUserNSDriver(cfg Config) *OverlayDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	d := &OverlayDriver{
		id:        DriverOverlayfsUserNS,
		config:    cfg,
		isolation: ModeBwrapRequired,
		version:   func() string { return "1.0" },
	}
	d.mount = kernelOverlayMount
	d.unmount = kernelOverlayUnmount
	d.availability = func(ctx context.Context) (bool, error) {
		if runtime.GOOS != "linux" {
			return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
		}
		if !overlayKernelModuleLoaded(ctx) {
			return false, fmt.Errorf("overlayfs module not available")
		}
		if !InUserNamespace() {
			return false, fmt.Errorf("overlayfs-userns requires the API to be wrapped in a user namespace (`unshare -U -m -r`)")
		}
		return true, nil
	}
	return d
}

// NewOverlayfsRootDriver builds the kernel-overlayfs flavor that runs as
// root or with CAP_SYS_ADMIN on the host (NOT inside a user namespace).
func NewOverlayfsRootDriver(cfg Config) *OverlayDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	d := &OverlayDriver{
		id:        DriverOverlayfsRoot,
		config:    cfg,
		isolation: ModeBwrapRequired,
		version:   func() string { return "1.0" },
	}
	d.mount = kernelOverlayMount
	d.unmount = kernelOverlayUnmount
	d.availability = func(ctx context.Context) (bool, error) {
		if runtime.GOOS != "linux" {
			return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
		}
		if !overlayKernelModuleLoaded(ctx) {
			return false, fmt.Errorf("overlayfs module not available")
		}
		if InUserNamespace() {
			return false, fmt.Errorf("overlayfs-root requires running on the host (not inside a user namespace)")
		}
		if os.Geteuid() == 0 || checkCapSysAdmin() {
			return true, nil
		}
		return false, fmt.Errorf("overlayfs-root requires root or CAP_SYS_ADMIN")
	}
	return d
}

// NewFuseOverlayfsDriver builds the userspace fuse-overlayfs flavor.
// Available without a user namespace as long as the fuse-overlayfs
// binary, fusermount, and /dev/fuse exist.
func NewFuseOverlayfsDriver(cfg Config) *OverlayDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	d := &OverlayDriver{
		id:        DriverFuseOverlayfs,
		config:    cfg,
		isolation: ModeBwrapPreferred,
		version:   fuseOverlayfsVersion,
	}
	d.mount = fuseOverlayMount
	d.unmount = fuseOverlayUnmount
	d.availability = func(ctx context.Context) (bool, error) {
		if _, err := exec.LookPath("fuse-overlayfs"); err != nil {
			return false, fmt.Errorf("fuse-overlayfs not found in PATH: %w", err)
		}
		if _, err := exec.LookPath("fusermount"); err != nil {
			if _, err := exec.LookPath("fusermount3"); err != nil {
				return false, fmt.Errorf("fusermount/fusermount3 not found: %w", err)
			}
		}
		if _, err := os.Stat("/dev/fuse"); err != nil {
			return false, fmt.Errorf("/dev/fuse not available: %w", err)
		}
		return true, nil
	}
	return d
}

// NewOverlayfsDriver auto-picks the kernel-overlay flavor that matches
// the current process: UserNS variant when wrapped in a user namespace,
// Root variant otherwise. Convenience wrapper for tests and the
// /driver/info diagnostic; production selection goes through SelectDriver.
func NewOverlayfsDriver(cfg Config) *OverlayDriver {
	if InUserNamespace() {
		return NewOverlayfsUserNSDriver(cfg)
	}
	return NewOverlayfsRootDriver(cfg)
}

// --- Driver interface ---

func (d *OverlayDriver) ID() DriverID                 { return d.id }
func (d *OverlayDriver) Version() string              { return d.version() }
func (d *OverlayDriver) RequiresBwrap() IsolationMode { return d.isolation }
func (d *OverlayDriver) BaseDir() string              { return d.config.BaseDir }
func (d *OverlayDriver) HomeOverlayBaseDir() string   { return d.config.HomeOverlayBaseDir }

// Capabilities reports the overlay driver's static contract. All overlay
// flavors support a per-sandbox $HOME overlay and copy-on-write. The
// isolation mode mirrors RequiresBwrap.
//
// DOC: home-overlay seam — driver-side capability declaration.
func (d *OverlayDriver) Capabilities() DriverCapabilities {
	return DriverCapabilities{
		HomeOverlay:        true,
		CoW:                true,
		NamespaceIsolation: d.isolation,
	}
}

func (d *OverlayDriver) IsAvailable(ctx context.Context) (bool, error) {
	return d.availability(ctx)
}

func (d *OverlayDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())
	paths, err := mountProjectOverlay(ctx, sandboxDir, s.ScopePath, d.mount)
	if err != nil {
		return nil, err
	}
	hostHome := os.Getenv("HOME")
	if hostHome == "" {
		s.HomeOverlayState = types.HomeOverlayNotRequested
		return paths, nil
	}
	lower, upper, work, merged, err := mountHomeOverlay(ctx, d.config.HomeOverlayBaseDir, s.ID, hostHome, d.mount, d.unmount)
	if err != nil {
		// Boundary-of-responsibility: the driver mounts what it can; the
		// caller (handler) decides whether absence is fatal based on the
		// active profile's RequiresHomeOverlay flag.
		slog.Warn("home overlay mount failed",
			"sandboxId", s.ID.String(),
			"driver", d.id,
			"homeOverlayBaseDir", d.config.HomeOverlayBaseDir,
			"hostHome", hostHome,
			"error", err.Error(),
		)
		s.HomeOverlayState = types.HomeOverlayAbsent
		return paths, nil
	}
	paths.HomeLowerDir = lower
	paths.HomeUpperDir = upper
	paths.HomeWorkDir = work
	paths.HomeMergedDir = merged
	s.HomeOverlayState = types.HomeOverlayPresent
	return paths, nil
}

func (d *OverlayDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	if s.MergedDir == "" {
		return nil
	}
	unmountHomeOverlay(ctx, d.config.HomeOverlayBaseDir, s.ID, d.unmount)
	if err := removeHomeOverlayDir(d.config.HomeOverlayBaseDir, s.ID); err != nil {
		fmt.Fprintf(os.Stderr, "home overlay dir cleanup: %v\n", err)
	}
	if !isMountPoint(s.MergedDir) {
		return nil
	}
	return d.unmount(ctx, s.MergedDir)
}

func (d *OverlayDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return cleanupSandboxDirAll(ctx, d.config.BaseDir, d.config.HomeOverlayBaseDir, s.ID, d.unmount)
}

func (d *OverlayDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	a, errA := listSandboxDirsInBase(d.config.BaseDir)
	b, errB := listHomeOverlayDirs(d.config.HomeOverlayBaseDir)
	return mergeUUIDLists(a, errA, b, errB)
}

func (d *OverlayDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	return cleanupSandboxDirAll(ctx, d.config.BaseDir, d.config.HomeOverlayBaseDir, id, d.unmount)
}

func (d *OverlayDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return getOverlayChangedFiles(s)
}

func (d *OverlayDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

func (d *OverlayDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(s)
}

// --- Per-flavor mount/unmount syscalls ---

// kernelOverlayMount uses the kernel mount syscall, with a `mount`-command
// fallback that surfaces a clearer error message on failure.
func kernelOverlayMount(ctx context.Context, target, opts string) error {
	if err := syscall.Mount("overlay", target, "overlay", 0, opts); err == nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "mount", "-t", "overlay", "overlay", "-o", opts, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// kernelOverlayUnmount lazy-unmounts target. Returns nil when target is
// not a mount point.
func kernelOverlayUnmount(ctx context.Context, target string) error {
	if err := syscall.Unmount(target, syscall.MNT_DETACH); err == nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "umount", "-l", target).CombinedOutput()
	if err != nil && isMountPoint(target) {
		return fmt.Errorf("umount: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// fuseOverlayMount invokes the fuse-overlayfs binary.
func fuseOverlayMount(ctx context.Context, target, opts string) error {
	out, err := exec.CommandContext(ctx, "fuse-overlayfs", "-o", opts, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fuse-overlayfs: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// fuseOverlayUnmount tries fusermount -u, falls back to lazy -u -z.
// Idempotent: returns nil when target is not a mount point.
func fuseOverlayUnmount(ctx context.Context, target string) error {
	bin := fusermountBin()
	if _, err := exec.CommandContext(ctx, bin, "-u", target).CombinedOutput(); err == nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, bin, "-u", "-z", target).CombinedOutput()
	if err != nil && isMountPoint(target) {
		return fmt.Errorf("fusermount: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// fusermountBin returns "fusermount" when available, "fusermount3"
// otherwise. Resolved per call so a host that gains the binary doesn't
// need a process restart.
func fusermountBin() string {
	if _, err := exec.LookPath("fusermount"); err == nil {
		return "fusermount"
	}
	return "fusermount3"
}

// fuseOverlayfsVersion parses `fuse-overlayfs --version`. Returns "1.0"
// when parsing fails so callers always get a non-empty string.
func fuseOverlayfsVersion() string {
	out, err := exec.Command("fuse-overlayfs", "--version").Output()
	if err != nil {
		return "1.0"
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, "fuse-overlayfs") {
			if parts := strings.SplitN(line, "version", 2); len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "1.0"
}

// overlayKernelModuleLoaded checks /proc/filesystems for the overlay
// entry, falling back to `modprobe -n overlay` for hosts that load it
// on demand. Shared between userns and root flavors.
func overlayKernelModuleLoaded(ctx context.Context) bool {
	if data, err := os.ReadFile("/proc/filesystems"); err == nil && strings.Contains(string(data), "overlay") {
		return true
	}
	if err := exec.CommandContext(ctx, "modprobe", "-n", "overlay").Run(); err == nil {
		return true
	}
	return false
}
