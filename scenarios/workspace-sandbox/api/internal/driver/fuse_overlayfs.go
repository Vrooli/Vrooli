// Package driver. fuse_overlayfs.go: unprivileged overlayfs via the
// fuse-overlayfs userspace daemon. Slower than the kernel driver but
// produces a host-visible merged dir (no userns required).
package driver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

var (
	_ Driver        = (*FuseOverlayfsDriver)(nil)
	_ MountVerifier = (*FuseOverlayfsDriver)(nil)
)

// FuseOverlayfsDriver implements Driver via the fuse-overlayfs binary.
type FuseOverlayfsDriver struct {
	config Config
}

func NewFuseOverlayfsDriver(cfg Config) *FuseOverlayfsDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &FuseOverlayfsDriver{config: cfg}
}

func (d *FuseOverlayfsDriver) ID() DriverID                { return DriverFuseOverlayfs }
func (d *FuseOverlayfsDriver) RequiresBwrap() IsolationMode { return ModeBwrapPreferred }
func (d *FuseOverlayfsDriver) BaseDir() string             { return d.config.BaseDir }
func (d *FuseOverlayfsDriver) HomeOverlayBaseDir() string  { return d.config.HomeOverlayBaseDir }

// Capabilities reports the fuse-overlayfs driver's contract.
// DOC: home-overlay seam.
func (d *FuseOverlayfsDriver) Capabilities() DriverCapabilities {
	return DriverCapabilities{
		HomeOverlay:        true,
		CoW:                true,
		NamespaceIsolation: ModeBwrapPreferred,
	}
}

// Version parses `fuse-overlayfs --version` output. Returns "1.0" if
// parsing fails. Format example: "fuse-overlayfs: version 1.13".
func (d *FuseOverlayfsDriver) Version() string {
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

// IsAvailable requires fuse-overlayfs, fusermount(3), and /dev/fuse.
func (d *FuseOverlayfsDriver) IsAvailable(ctx context.Context) (bool, error) {
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

func fusermountBin() string {
	if _, err := exec.LookPath("fusermount"); err == nil {
		return "fusermount"
	}
	return "fusermount3"
}

func (d *FuseOverlayfsDriver) mount(ctx context.Context, target, opts string) error {
	out, err := exec.CommandContext(ctx, "fuse-overlayfs", "-o", opts, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fuse-overlayfs: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// unmount tries fusermount -u, falls back to lazy -u -z. Idempotent.
func (d *FuseOverlayfsDriver) unmount(ctx context.Context, target string) error {
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

func (d *FuseOverlayfsDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
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
		slog.Warn("home overlay mount failed",
			"sandboxId", s.ID.String(),
			"driver", DriverFuseOverlayfs,
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

func (d *FuseOverlayfsDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
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

func (d *FuseOverlayfsDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return getOverlayChangedFiles(s)
}

func (d *FuseOverlayfsDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

func (d *FuseOverlayfsDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return cleanupSandboxDirAll(ctx, d.config.BaseDir, d.config.HomeOverlayBaseDir, s.ID, d.unmount)
}

func (d *FuseOverlayfsDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	a, errA := listSandboxDirsInBase(d.config.BaseDir)
	b, errB := listHomeOverlayDirs(d.config.HomeOverlayBaseDir)
	return mergeUUIDLists(a, errA, b, errB)
}

func (d *FuseOverlayfsDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	return cleanupSandboxDirAll(ctx, d.config.BaseDir, d.config.HomeOverlayBaseDir, id, d.unmount)
}

func (d *FuseOverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(s)
}
