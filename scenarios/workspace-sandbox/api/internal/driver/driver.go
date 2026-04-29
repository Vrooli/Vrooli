// Package driver provides sandbox driver interfaces and implementations.
//
// The package is split into orthogonal capability interfaces (MountDriver,
// ChangeTracker, MountVerifier) so each driver implementation declares
// exactly what it supports. Process execution lives in the sub-package
// driver/exec; drivers do NOT implement Exec/StartProcess directly.
//
// File map:
//
//	driver.go         interfaces + types (this file)
//	select.go         SelectDriver + preference IO + NewDriverFor
//	slot.go           atomic.Pointer wrapper + SwitchDriver
//	options.go        DriverOption capability matrix
//	probe.go          host capability probes
//	helpers.go        shared overlayfs/cleanup helpers
//	overlay.go        unified OverlayDriver (overlayfs-userns, overlayfs-root, fuse-overlayfs)
//	copy.go           cross-platform fallback
//	exec/             process isolation (Exec, StartProcess, BwrapConfig, …)
package driver

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// StableFileID generates a deterministic UUID for a file within a sandbox.
// The same file always gets the same ID across API calls, enabling
// reliable file-based operations (approve, discard, partial approval).
func StableFileID(sandboxID uuid.UUID, filePath string) uuid.UUID {
	return uuid.NewSHA1(sandboxID, []byte(filePath))
}

// DriverID is the canonical identifier across DB, wire, preference file,
// and Go code. The four values are fixed.
type DriverID string

const (
	DriverOverlayfsUserNS DriverID = "overlayfs-userns"
	DriverOverlayfsRoot   DriverID = "overlayfs-root"
	DriverFuseOverlayfs   DriverID = "fuse-overlayfs"
	DriverCopy            DriverID = "copy"
)

// IsolationMode is the single decision boundary for "how isolated should
// this exec be". Each driver declares its required mode via
// MountDriver.RequiresBwrap; callers pass the result to exec.Exec /
// exec.StartProcess. Defining the type here (instead of in driver/exec)
// keeps the exec package free of a back-reference cycle.
type IsolationMode int

const (
	// ModeNone runs the command directly in s.MergedDir with no namespace
	// isolation. Used by the copy driver, which has no real mount.
	ModeNone IsolationMode = iota

	// ModeBwrapPreferred uses bwrap when available; falls back to direct
	// execution when bwrap isn't installed. Used by fuse-overlayfs whose
	// mount is host-visible — direct execution still operates against the
	// merged dir, just without process isolation.
	ModeBwrapPreferred

	// ModeBwrapRequired hard-errors when bwrap is missing. Used by kernel
	// overlayfs whose mount lives inside the API's mount namespace — a
	// direct child won't see the merged dir, so falling back would return
	// the host filesystem and silently produce wrong results.
	ModeBwrapRequired
)

// MountPaths contains the paths used for overlay mounting.
//
// HomeLowerDir/HomeUpperDir/HomeWorkDir/HomeMergedDir are populated when
// the driver mounts a per-sandbox fuse-overlayfs over the host $HOME. The
// merged dir is bind-mounted at /home/<user> inside the bwrap namespace
// so agent CLIs find their host config while writes go to the upper
// layer (per-run, ephemeral). Zero when $HOME is not set or the driver
// chose not to set up a home overlay (e.g. CopyDriver, tests).
type MountPaths struct {
	LowerDir  string // Read-only layer (canonical repo)
	UpperDir  string // Writable layer (changes)
	WorkDir   string // Overlayfs work directory
	MergedDir string // Merged mount point

	HomeLowerDir  string // Host $HOME, read-only via overlay
	HomeUpperDir  string // Per-sandbox writable layer for $HOME writes
	HomeWorkDir   string // fuse-overlayfs scratch dir for the home overlay
	HomeMergedDir string // Merged $HOME mount point on the host side
}

// DriverCapabilities is the pure (no-I/O) declaration of what a driver
// supports. Used by handlers to decide whether a profile's requirements
// can be satisfied — see IsolationProfile.HomeOverlayRequirement.
//
// DOC: home-overlay seam — driver-side capability declaration. See
// docs/internal/SEAMS.md.
type DriverCapabilities struct {
	// HomeOverlay is true when the driver can mount a per-sandbox overlay
	// over the host $HOME. False for the copy driver. True for both
	// overlayfs variants and fuse-overlayfs.
	HomeOverlay bool
	// CoW is true when changes are stored copy-on-write. False for the
	// copy driver (full copies, not CoW).
	CoW bool
	// NamespaceIsolation is the isolation guarantee this driver provides
	// when paired with bwrap. Mirrors RequiresBwrap() but is a struct
	// field (one decision) rather than a method (one capability).
	NamespaceIsolation IsolationMode
}

// MountDriver is the base interface every driver implements: mount
// lifecycle plus orphan reconciliation.
type MountDriver interface {
	// ID returns the canonical driver ID. Used in DB columns, wire
	// payloads, preference files, and every internal switch.
	ID() DriverID

	// Version returns the driver version.
	Version() string

	// IsAvailable checks if this driver can be used on the current system.
	IsAvailable(ctx context.Context) (bool, error)

	// Mount creates the overlay mount for a sandbox.
	Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error)

	// Unmount removes the overlay mount.
	Unmount(ctx context.Context, s *types.Sandbox) error

	// Cleanup removes all sandbox artifacts (dirs, mounts).
	Cleanup(ctx context.Context, s *types.Sandbox) error

	// ListSandboxDirs returns the IDs of all sandbox directories on disk
	// under BaseDir. Used by the filesystem orphan reconciler to detect
	// dirs the repository does not know about.
	//
	// Implementations must skip non-UUID entries silently (driver
	// bookkeeping like the preference file lives there too) and treat a
	// missing BaseDir as "no orphans".
	ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error)

	// CleanupOrphan releases an orphaned sandbox by ID alone. Idempotent:
	// missing dirs and already-unmounted overlays are not errors.
	CleanupOrphan(ctx context.Context, id uuid.UUID) error

	// RequiresBwrap declares which IsolationMode this driver requires.
	// Callers pass the result to exec.Exec / exec.StartProcess. Replaces
	// the prior central exec.DriverModeFor type-switch — adding a new
	// driver no longer requires editing a central dispatcher.
	RequiresBwrap() IsolationMode

	// Capabilities declares what features this driver supports. Pure
	// (no I/O); each driver returns a static struct. Used by handlers
	// to decide whether a requested profile can be satisfied.
	//
	// DOC: home-overlay seam.
	Capabilities() DriverCapabilities
}

// ChangeTracker captures the change-detection + partial-approval seam.
// All three drivers satisfy it.
type ChangeTracker interface {
	// GetChangedFiles returns the list of files changed in the upper layer.
	GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error)

	// RemoveFromUpper removes a file from the upper (writable) layer.
	// Idempotent: returns nil if file doesn't exist.
	RemoveFromUpper(ctx context.Context, s *types.Sandbox, relPath string) error
}

// MountVerifier is implemented only by drivers backed by an actual mount
// (overlayfs, fuse-overlayfs). CopyDriver does NOT implement it; callers
// that don't know which driver is active should use VerifyIfSupported.
type MountVerifier interface {
	VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error
}

// Driver is the composite that the service layer holds: every driver
// supports mount lifecycle and change tracking. MountVerifier is opt-in.
type Driver interface {
	MountDriver
	ChangeTracker
}

// VerifyIfSupported returns nil for drivers without a mount to verify
// (CopyDriver). For mount-backed drivers it delegates to
// VerifyMountIntegrity. Keeps callers branchless.
func VerifyIfSupported(ctx context.Context, d Driver, s *types.Sandbox) error {
	if v, ok := d.(MountVerifier); ok {
		return v.VerifyMountIntegrity(ctx, s)
	}
	return nil
}

// Config holds driver configuration.
type Config struct {
	// BaseDir is the root directory for sandbox artifacts.
	BaseDir string

	// HomeOverlayBaseDir is the directory that holds per-sandbox
	// home-{upper,work,merged} dirs. MUST be outside $HOME. Resolved by
	// config.ResolveHomeOverlayBaseDir at startup.
	HomeOverlayBaseDir string

	// MaxSandboxes limits the total number of active sandboxes.
	MaxSandboxes int

	// MaxSizeMB limits the size of a single sandbox.
	MaxSizeMB int64

	// UseFuseOverlayfs is the legacy preference flag; SelectDriver
	// preserves it but the post-Phase-5 default is kernel overlayfs in a
	// user namespace. Operators can still flip drivers via the
	// /api/v1/driver/select endpoint (see SwitchDriver in slot.go).
	UseFuseOverlayfs bool
}

// defaultBaseDir returns the default sandbox base directory.
// Uses XDG data directory (~/.local/share/workspace-sandbox).
func defaultBaseDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "workspace-sandbox")
	}
	return "/var/lib/workspace-sandbox"
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseDir:          defaultBaseDir(),
		MaxSandboxes:     1000,
		MaxSizeMB:        10240, // 10 GB
		UseFuseOverlayfs: false,
	}
}

// Info contains metadata about a driver.
type Info struct {
	ID          DriverID
	Version     string
	Description string
	Available   bool
}
