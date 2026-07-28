// Package driver. overlay.go: the unified overlay driver that backs all
// three overlay-flavored DriverIDs (overlayfs-userns, overlayfs-root,
// fuse-overlayfs). The bodies of every method are identical across
// flavors — only the mount backend, the availability probe, the
// version string, and the required containment level differ. Those vary
// points are captured as fields on a single struct.
package driver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/driver/changedetect"
	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/process"
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
	backend      fsmount.Backend
	mounter      fsmount.Mounter
	starter      process.Starter
	availability availabilityFunc
	version      func() string
	isolation    ContainmentLevel
	clock        clock.Clock
}

// NewOverlayfsUserNSDriver builds the kernel-overlayfs flavor that runs
// inside an unprivileged user namespace (Linux 5.11+). All Deps fields
// are required; nil panics with a structured message.
func NewOverlayfsUserNSDriver(cfg Config, deps Deps) *OverlayDriver {
	deps.Validate("driver.NewOverlayfsUserNSDriver")
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	d := &OverlayDriver{
		id:        DriverOverlayfsUserNS,
		config:    cfg,
		backend:   fsmount.BackendKernelOverlay,
		mounter:   deps.Mounter,
		starter:   deps.Starter,
		isolation: ContainmentRequired,
		version:   func() string { return "1.0" },
		clock:     deps.Clock,
	}
	d.availability = func(ctx context.Context) (bool, error) {
		if runtime.GOOS != "linux" {
			return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
		}
		if !overlayKernelModuleLoaded(ctx, deps.Starter) {
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
func NewOverlayfsRootDriver(cfg Config, deps Deps) *OverlayDriver {
	deps.Validate("driver.NewOverlayfsRootDriver")
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	d := &OverlayDriver{
		id:        DriverOverlayfsRoot,
		config:    cfg,
		backend:   fsmount.BackendKernelOverlay,
		mounter:   deps.Mounter,
		starter:   deps.Starter,
		isolation: ContainmentRequired,
		version:   func() string { return "1.0" },
		clock:     deps.Clock,
	}
	d.availability = func(ctx context.Context) (bool, error) {
		if runtime.GOOS != "linux" {
			return false, fmt.Errorf("overlayfs driver requires Linux (current OS: %s)", runtime.GOOS)
		}
		if !overlayKernelModuleLoaded(ctx, deps.Starter) {
			return false, fmt.Errorf("overlayfs module not available")
		}
		if InUserNamespace() {
			return false, fmt.Errorf("overlayfs-root requires running on the host (not inside a user namespace)")
		}
		if os.Geteuid() == 0 || checkCapSysAdmin(deps.Starter) {
			return true, nil
		}
		return false, fmt.Errorf("overlayfs-root requires root or CAP_SYS_ADMIN")
	}
	return d
}

// NewFuseOverlayfsDriver builds the userspace fuse-overlayfs flavor.
// Available without a user namespace as long as the fuse-overlayfs
// binary, fusermount, and /dev/fuse exist.
func NewFuseOverlayfsDriver(cfg Config, deps Deps) *OverlayDriver {
	deps.Validate("driver.NewFuseOverlayfsDriver")
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	d := &OverlayDriver{
		id:        DriverFuseOverlayfs,
		config:    cfg,
		backend:   fsmount.BackendFuseOverlayfs,
		mounter:   deps.Mounter,
		starter:   deps.Starter,
		isolation: ContainmentPreferred,
		version:   func() string { return fuseOverlayfsVersion(deps.Starter) },
		clock:     deps.Clock,
	}
	d.availability = func(ctx context.Context) (bool, error) {
		if _, err := deps.Starter.LookPath("fuse-overlayfs"); err != nil {
			return false, fmt.Errorf("fuse-overlayfs not found in PATH: %w", err)
		}
		if _, err := deps.Starter.LookPath("fusermount"); err != nil {
			if _, err := deps.Starter.LookPath("fusermount3"); err != nil {
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
func NewOverlayfsDriver(cfg Config, deps Deps) *OverlayDriver {
	if InUserNamespace() {
		return NewOverlayfsUserNSDriver(cfg, deps)
	}
	return NewOverlayfsRootDriver(cfg, deps)
}

// --- Driver interface ---

func (d *OverlayDriver) ID() DriverID                          { return d.id }
func (d *OverlayDriver) Version() string                       { return d.version() }
func (d *OverlayDriver) RequiredContainment() ContainmentLevel { return d.isolation }
func (d *OverlayDriver) BaseDir() string                       { return d.config.BaseDir }
func (d *OverlayDriver) HomeOverlayBaseDir() string            { return d.config.HomeOverlayBaseDir }

// Capabilities reports the overlay driver's static contract.
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
	paths, err := mountProjectOverlay(ctx, d.mounter, d.backend, sandboxDir, s.ScopePath)
	if err != nil {
		return nil, err
	}
	hostHome := os.Getenv("HOME")
	if hostHome == "" {
		s.HomeOverlayState = types.HomeOverlayNotRequested
		return paths, nil
	}
	lower, upper, work, merged, err := mountHomeOverlay(ctx, d.mounter, d.backend, d.config.HomeOverlayBaseDir, s.ID, hostHome)
	if err != nil {
		// Boundary-of-responsibility: the driver mounts what it can; the
		// caller (handler) decides whether absence is fatal based on the
		// active profile's HomeOverlayRequirement value.
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
	unmountHomeOverlay(ctx, d.mounter, d.config.HomeOverlayBaseDir, s.ID)
	if err := removeHomeOverlayDir(d.config.HomeOverlayBaseDir, s.ID); err != nil {
		fmt.Fprintf(os.Stderr, "home overlay dir cleanup: %v\n", err)
	}
	if !d.mounter.IsMountPoint(s.MergedDir) {
		return nil
	}
	return d.mounter.Unmount(ctx, s.MergedDir, false)
}

func (d *OverlayDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return cleanupSandboxDirAll(ctx, d.mounter, d.config.BaseDir, d.config.HomeOverlayBaseDir, s.ID)
}

func (d *OverlayDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	a, errA := listSandboxDirsInBase(d.config.BaseDir)
	b, errB := listHomeOverlayDirs(d.config.HomeOverlayBaseDir)
	return mergeUUIDLists(a, errA, b, errB)
}

func (d *OverlayDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	return cleanupSandboxDirAll(ctx, d.mounter, d.config.BaseDir, d.config.HomeOverlayBaseDir, id)
}

func (d *OverlayDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	if s.UpperDir == "" {
		return nil, fmt.Errorf("sandbox upper directory not set")
	}
	return changedetect.Walk(ctx,
		changedetect.WalkOpts{Lower: s.LowerDir, Upper: s.UpperDir, SandboxID: s.ID, IgnoreMatcher: diff.NewGitIgnoreMatcher(s.ProjectRoot, diff.NewExecCommandRunner(d.starter))},
		&changedetect.OverlayStrategy{FileIDFn: StableFileID},
		d.clock.Now(),
	)
}

func (d *OverlayDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

func (d *OverlayDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(d.mounter, s)
}

// =============================================================================
// Helpers that consume the Starter for binary version/capability probes
// =============================================================================

// fuseOverlayfsVersion parses `fuse-overlayfs --version`. Returns "1.0"
// when parsing fails so callers always get a non-empty string.
func fuseOverlayfsVersion(s process.Starter) string {
	res, err := process.Run(context.Background(), s, process.StartOpts{
		Path: "fuse-overlayfs",
		Args: []string{"--version"},
	})
	if err != nil || res.Exit.ExitCode != 0 {
		return "1.0"
	}
	for _, line := range strings.Split(strings.TrimSpace(string(res.Stdout)), "\n") {
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
func overlayKernelModuleLoaded(ctx context.Context, s process.Starter) bool {
	if data, err := os.ReadFile("/proc/filesystems"); err == nil && strings.Contains(string(data), "overlay") {
		return true
	}
	res, err := process.Run(ctx, s, process.StartOpts{
		Path: "modprobe",
		Args: []string{"-n", "overlay"},
	})
	if err != nil {
		return false
	}
	return res.Exit.ExitCode == 0
}
