// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"

	"workspace-sandbox/internal/types"
)

// DriverType identifies which driver implementation is in use.
type DriverType string

const (
	DriverTypeOverlayfs DriverType = "overlayfs"
	DriverTypeNone      DriverType = "none"
)

// MountPaths contains the paths used for overlay mounting.
type MountPaths struct {
	LowerDir  string // Read-only layer (canonical repo)
	UpperDir  string // Writable layer (changes)
	WorkDir   string // Overlayfs work directory
	MergedDir string // Merged mount point
}

// Driver is the interface for sandbox driver implementations.
type Driver interface {
	// Type returns the driver type.
	Type() DriverType

	// Version returns the driver version.
	Version() string

	// IsAvailable checks if this driver can be used on the current system.
	IsAvailable(ctx context.Context) (bool, error)

	// Mount creates the overlay mount for a sandbox.
	Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error)

	// Unmount removes the overlay mount.
	Unmount(ctx context.Context, s *types.Sandbox) error

	// GetChangedFiles returns the list of files changed in the upper layer.
	GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error)

	// Cleanup removes all sandbox artifacts (dirs, mounts).
	Cleanup(ctx context.Context, s *types.Sandbox) error
}

// Config holds driver configuration.
type Config struct {
	// BaseDir is the root directory for sandbox artifacts.
	BaseDir string

	// MaxSandboxes limits the total number of active sandboxes.
	MaxSandboxes int

	// MaxSizeMB limits the size of a single sandbox.
	MaxSizeMB int64

	// UseFuseOverlayfs uses fuse-overlayfs instead of kernel overlayfs.
	// This allows unprivileged operation but may be slower.
	UseFuseOverlayfs bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseDir:          "/var/lib/workspace-sandbox",
		MaxSandboxes:     1000,
		MaxSizeMB:        10240, // 10 GB
		UseFuseOverlayfs: false,
	}
}

// Info contains metadata about a driver.
type Info struct {
	Type        DriverType
	Version     string
	Description string
	Available   bool
}
