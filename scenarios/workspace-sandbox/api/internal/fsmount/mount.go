// Package fsmount is the canonical mount/unmount seam for
// workspace-sandbox.
//
// Why this exists (Round 4 Phase 7):
//   - Driver code previously called syscall.Mount/syscall.Unmount and
//     spawned the fuse-overlayfs binary directly. Mid-mount failures
//     (the lower bind succeeds but the merge fails), partial
//     unmount-with-stale-mount, and "fuse-overlayfs daemon forks-and-
//     dies before reporting success" were all unreachable from go test.
//   - The Mounter interface concentrates every kernel-overlay,
//     fuse-overlayfs, fusermount, and mountpoint syscall into a single
//     place. Production code wires SystemMounter; tests inject
//     FakeMounter from internal/testutil/mocks.
//
// Per-backend behavior:
//   - BackendKernelOverlay uses syscall.Mount("overlay", ...). On
//     failure, it falls back to invoking `mount -t overlay` so the
//     error message surfaces useful kernel context (the same
//     `mount`-command fallback the previous driver code had).
//   - BackendFuseOverlayfs spawns the fuse-overlayfs binary. Unmount
//     prefers `fusermount -u`, falling back to `fusermount -u -z`
//     (lazy detach) when the daemon is wedged.
//
// IsMountPoint queries the `mountpoint` binary. Wrapping it here keeps
// the kernel-vs-userspace distinction local to the seam and lets tests
// stub mountpoint-presence cleanly.
//
// See docs/SEAMS.md "Mounter Seam (Round 4 Phase 7)".
package fsmount

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"workspace-sandbox/internal/process"
)

// Backend selects which kernel/userspace implementation handles the
// mount. Required on every MountOpts; the zero value is rejected.
type Backend int

const (
	// BackendUnset is the zero value; rejected at Mount time so callers
	// never silently fall through to a default.
	BackendUnset Backend = iota

	// BackendKernelOverlay uses the in-kernel overlayfs driver via
	// syscall.Mount. Available on every Linux 4.0+ kernel; requires
	// either CAP_SYS_ADMIN (root flavor) or wrapping in a user
	// namespace (userns flavor).
	BackendKernelOverlay

	// BackendFuseOverlayfs spawns the fuse-overlayfs userspace daemon.
	// Slower than the kernel overlay but works without CAP_SYS_ADMIN
	// outside a user namespace.
	BackendFuseOverlayfs
)

// String renders the backend name for error messages and tests.
func (b Backend) String() string {
	switch b {
	case BackendKernelOverlay:
		return "kernel-overlay"
	case BackendFuseOverlayfs:
		return "fuse-overlayfs"
	default:
		return fmt.Sprintf("backend(%d)", int(b))
	}
}

// MountOpts describes a single overlay mount. Only the overlayfs shape
// is supported today (every workspace-sandbox driver uses overlay
// semantics); future tmpfs/bind backends should add their own option
// struct rather than accreting fields here.
type MountOpts struct {
	// Backend selects the runtime implementation. Required.
	Backend Backend

	// Lower is the (potentially colon-joined) lower-layer source. For
	// the overlayfs drivers this is either the project ScopePath or
	// the host $HOME for the home-overlay.
	Lower string

	// Upper is the read-write upper layer.
	Upper string

	// Work is the overlayfs scratch directory.
	Work string

	// Merged is the mount target (the merged view).
	Merged string
}

// validate returns nil when opts is well-formed.
func (o MountOpts) validate() error {
	if o.Backend == BackendUnset {
		return errors.New("MountOpts.Backend is required")
	}
	if o.Lower == "" {
		return errors.New("MountOpts.Lower is required")
	}
	if o.Upper == "" {
		return errors.New("MountOpts.Upper is required")
	}
	if o.Work == "" {
		return errors.New("MountOpts.Work is required")
	}
	if o.Merged == "" {
		return errors.New("MountOpts.Merged is required")
	}
	return nil
}

// optsString renders the overlayfs option string the kernel and
// fuse-overlayfs both accept.
func (o MountOpts) optsString() string {
	return fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", o.Lower, o.Upper, o.Work)
}

// Mounter is the seam every production mount/unmount goes through.
// SystemMounter is the runtime impl; tests use FakeMounter from
// internal/testutil/mocks.
type Mounter interface {
	// Mount mounts an overlay per opts. Returns nil on success.
	Mount(ctx context.Context, opts MountOpts) error

	// Unmount tears down the mount at target. lazy=true requests a
	// MNT_DETACH-style unmount that succeeds even when files are open;
	// lazy=false attempts a clean unmount first and falls back to lazy
	// only when the clean path fails. Idempotent: returns nil when
	// target is not a mount point.
	Unmount(ctx context.Context, target string, lazy bool) error

	// IsMountPoint reports whether path is currently a mount point.
	IsMountPoint(path string) bool
}

// =============================================================================
// SystemMounter — production implementation
// =============================================================================

// SystemMounter is the production Mounter. It wraps syscall.Mount and
// shells out to fuse-overlayfs / fusermount / mountpoint via
// process.Starter so every exec call site stays inside the canonical
// seam.
type SystemMounter struct {
	starter process.Starter
}

// NewSystemMounter constructs the production mounter. starter is
// required (it backs every binary invocation); panics on nil so the
// startup wiring fails loud.
func NewSystemMounter(starter process.Starter) *SystemMounter {
	if starter == nil {
		panic("fsmount.NewSystemMounter: starter is required")
	}
	return &SystemMounter{starter: starter}
}

// Mount routes to the per-backend impl after validating opts.
func (m *SystemMounter) Mount(ctx context.Context, opts MountOpts) error {
	if err := opts.validate(); err != nil {
		return err
	}
	switch opts.Backend {
	case BackendKernelOverlay:
		return m.mountKernel(ctx, opts)
	case BackendFuseOverlayfs:
		return m.mountFuse(ctx, opts)
	default:
		return fmt.Errorf("fsmount: unsupported backend %s", opts.Backend)
	}
}

// mountKernel uses the kernel overlayfs driver via syscall.Mount, with a
// `mount -t overlay` fallback that surfaces a clearer error message.
func (m *SystemMounter) mountKernel(ctx context.Context, opts MountOpts) error {
	if err := syscall.Mount("overlay", opts.Merged, "overlay", 0, opts.optsString()); err == nil {
		return nil
	}
	res, err := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
		Path: "mount",
		Args: []string{"-t", "overlay", "overlay", "-o", opts.optsString(), opts.Merged},
	})
	if err != nil {
		return fmt.Errorf("kernel-overlay mount: %w", err)
	}
	if res.Exit.ExitCode != 0 {
		return fmt.Errorf("kernel-overlay mount: exit %d (output: %s)", res.Exit.ExitCode, strings.TrimSpace(string(res.Stdout)))
	}
	return nil
}

// mountFuse spawns the fuse-overlayfs binary.
func (m *SystemMounter) mountFuse(ctx context.Context, opts MountOpts) error {
	res, err := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
		Path: "fuse-overlayfs",
		Args: []string{"-o", opts.optsString(), opts.Merged},
	})
	if err != nil {
		return fmt.Errorf("fuse-overlayfs: %w", err)
	}
	if res.Exit.ExitCode != 0 {
		return fmt.Errorf("fuse-overlayfs: exit %d (output: %s)", res.Exit.ExitCode, strings.TrimSpace(string(res.Stdout)))
	}
	return nil
}

// Unmount tears down a mount at target. The lazy flag picks between
// kernel MNT_DETACH (lazy) and a `umount`/`fusermount` clean path with
// lazy fallback when fuse-overlayfs is in use.
func (m *SystemMounter) Unmount(ctx context.Context, target string, lazy bool) error {
	if target == "" {
		return errors.New("fsmount.Unmount: target is required")
	}
	if !m.IsMountPoint(target) {
		return nil
	}

	// Try kernel-style first: this works for both kernel overlayfs
	// mounts and (post-fusermount-detach) any leftover overlayfs
	// remnant the kernel still sees.
	flags := 0
	if lazy {
		flags = syscall.MNT_DETACH
	}
	if err := syscall.Unmount(target, flags); err == nil {
		return nil
	}

	// Userspace fallbacks: prefer `fusermount -u` for fuse mounts,
	// then `umount -l` for kernel mounts. Either may succeed where the
	// raw syscall failed (e.g. a non-root caller without CAP_SYS_ADMIN).
	if path, err := m.starter.LookPath("fusermount"); err == nil {
		if res, runErr := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
			Path: path,
			Args: []string{"-u", target},
		}); runErr == nil && res.Exit.ExitCode == 0 {
			return nil
		}
		if res, runErr := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
			Path: path,
			Args: []string{"-u", "-z", target},
		}); runErr == nil && res.Exit.ExitCode == 0 {
			return nil
		}
	} else if path, err := m.starter.LookPath("fusermount3"); err == nil {
		if res, runErr := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
			Path: path,
			Args: []string{"-u", target},
		}); runErr == nil && res.Exit.ExitCode == 0 {
			return nil
		}
		if res, runErr := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
			Path: path,
			Args: []string{"-u", "-z", target},
		}); runErr == nil && res.Exit.ExitCode == 0 {
			return nil
		}
	}

	// Last resort: umount -l. We only return an error when the path is
	// still a mount point afterward, since "already unmounted" is the
	// idempotent outcome we want.
	res, runErr := process.RunCombinedOutput(ctx, m.starter, process.StartOpts{
		Path: "umount",
		Args: []string{"-l", target},
	})
	if runErr != nil && m.IsMountPoint(target) {
		return fmt.Errorf("umount %s: %w", target, runErr)
	}
	if res.Exit.ExitCode != 0 && m.IsMountPoint(target) {
		return fmt.Errorf("umount %s: exit %d (output: %s)", target, res.Exit.ExitCode, strings.TrimSpace(string(res.Stdout)))
	}
	return nil
}

// IsMountPoint queries the `mountpoint` binary. Returns false when the
// binary is missing or returns non-zero — both states mean "not
// observably mounted via this seam," which is the safe side.
func (m *SystemMounter) IsMountPoint(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	res, err := process.Run(context.Background(), m.starter, process.StartOpts{
		Path: "mountpoint",
		Args: []string{"-q", path},
	})
	if err != nil {
		return false
	}
	return res.Exit.ExitCode == 0
}

// =============================================================================
// Probe helper (used by namespace package)
// =============================================================================

// ProbeKernelOverlayMount creates a temp tree, mounts an overlayfs at
// it, then unmounts and cleans up. Returns nil when the kernel-overlay
// path is currently usable. Used by the namespace probe to test
// "overlayfs in this user namespace works."
func ProbeKernelOverlayMount(ctx context.Context, m Mounter) error {
	tmpDir, err := os.MkdirTemp("", "overlay-test-")
	if err != nil {
		return fmt.Errorf("probe tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	lower := filepath.Join(tmpDir, "lower")
	upper := filepath.Join(tmpDir, "upper")
	work := filepath.Join(tmpDir, "work")
	merged := filepath.Join(tmpDir, "merged")
	for _, dir := range []string{lower, upper, work, merged} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("probe mkdir %s: %w", dir, err)
		}
	}
	opts := MountOpts{
		Backend: BackendKernelOverlay,
		Lower:   lower,
		Upper:   upper,
		Work:    work,
		Merged:  merged,
	}
	if err := m.Mount(ctx, opts); err != nil {
		return err
	}
	if err := m.Unmount(ctx, merged, false); err != nil {
		return fmt.Errorf("probe unmount: %w", err)
	}
	return nil
}
