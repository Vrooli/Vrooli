// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// StableFileID generates a deterministic UUID for a file within a sandbox.
// This ensures the same file always gets the same ID across API calls,
// enabling reliable file-based operations like approve and discard.
func StableFileID(sandboxID uuid.UUID, filePath string) uuid.UUID {
	return uuid.NewSHA1(sandboxID, []byte(filePath))
}

// DriverType identifies which driver implementation is in use.
type DriverType string

const (
	DriverTypeOverlayfs     DriverType = "overlayfs"
	DriverTypeFuseOverlayfs DriverType = "fuse-overlayfs" // Unprivileged with direct access
	DriverTypeCopy          DriverType = "copy"           // [OT-P2-004] Cross-platform fallback
	DriverTypeNone          DriverType = "none"
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

	// --- Temporal Safety Methods ---

	// IsMounted verifies whether the sandbox overlay is currently mounted.
	// This enables validation before operations that require an active mount.
	// Returns true if mounted, false if not, and error if check fails.
	IsMounted(ctx context.Context, s *types.Sandbox) (bool, error)

	// VerifyMountIntegrity checks that the mount is healthy and accessible.
	// Returns nil if the mount is valid, or an error describing the problem.
	// This can detect issues like stale mounts or corrupted overlay state.
	VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error

	// --- Partial Approval Support (OT-P1-002) ---

	// RemoveFromUpper removes a file from the upper (writable) layer.
	// This is used after partial approval to clean up applied files
	// while preserving unapproved changes for follow-up approvals.
	// Returns nil if file doesn't exist (idempotent).
	RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error

	// --- Process Isolation Methods (OT-P0-003) ---

	// Exec executes a command inside the sandbox with process isolation.
	// Uses bubblewrap on Linux to provide filesystem constraints where
	// the canonical repo is read-only and only the overlay upper layer is writable.
	Exec(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error)

	// StartProcess starts a long-running process in the sandbox.
	// Returns the PID for tracking purposes.
	StartProcess(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error)
}

// MountState represents the current state of a sandbox mount.
type MountState struct {
	IsMounted    bool   `json:"isMounted"`
	IsHealthy    bool   `json:"isHealthy"`
	MergedDir    string `json:"mergedDir,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
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

// defaultBaseDir returns the default sandbox base directory.
// Uses XDG data directory (~/.local/share/workspace-sandbox) for unprivileged operation.
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
	Type        DriverType
	Version     string
	Description string
	Available   bool
}

// --- Driver Selection [OT-P2-004] ---

// SelectDriver returns the best available driver for the current system.
// It tests each driver in priority order and returns the first one that works.
//
// Selection order (when UseFuseOverlayfs is false):
//  1. OverlayfsDriver - native kernel overlayfs (best performance)
//     Works if: running in user namespace (kernel 5.11+) OR has CAP_SYS_ADMIN
//  2. FuseOverlayfsDriver - fuse-overlayfs (direct access, good performance)
//     Works if: fuse-overlayfs installed, /dev/fuse available
//  3. CopyDriver - cross-platform fallback (always works)
//     Uses file copies instead of overlayfs, slower but universal
//
// Selection order (when UseFuseOverlayfs is true):
//  1. FuseOverlayfsDriver - prioritized for direct filesystem access
//  2. OverlayfsDriver - fallback to native overlayfs
//  3. CopyDriver - cross-platform fallback
//
// The cfg parameter is used to configure whichever driver is selected.
// Logs are emitted explaining which driver was selected and why fallbacks occurred.
func SelectDriver(ctx context.Context, cfg Config) (Driver, error) {
	// If fuse-overlayfs is explicitly requested, try it first
	if cfg.UseFuseOverlayfs {
		fuseDriver := NewFuseOverlayfsDriver(cfg)
		available, err := fuseDriver.IsAvailable(ctx)
		if err == nil && available {
			log.Printf("driver: using fuse-overlayfs (direct access, configured preference)")
			return fuseDriver, nil
		}
		if err != nil {
			log.Printf("driver: fuse-overlayfs requested but not available: %v", err)
		}
		// Fall through to try other options
	}

	// Try native overlayfs first (best performance)
	overlayDriver := NewOverlayfsDriver(cfg)
	available, err := overlayDriver.IsAvailable(ctx)
	if err == nil && available {
		log.Printf("driver: using native overlayfs (optimal performance)")
		return overlayDriver, nil
	}

	// Log why overlayfs isn't available
	if err != nil {
		log.Printf("driver: overlayfs not available: %v", err)
	} else {
		log.Printf("driver: overlayfs not available (mount test failed)")
	}

	// Try fuse-overlayfs if not explicitly requested but native isn't available
	if !cfg.UseFuseOverlayfs {
		fuseDriver := NewFuseOverlayfsDriver(cfg)
		available, err := fuseDriver.IsAvailable(ctx)
		if err == nil && available {
			log.Printf("driver: using fuse-overlayfs (direct access, fallback from native overlayfs)")
			return fuseDriver, nil
		}
		if err != nil {
			log.Printf("driver: fuse-overlayfs not available: %v", err)
		}
	}

	// Fall back to copy driver
	log.Printf("driver: falling back to copy driver (slower but universal)")
	log.Printf("driver: for better performance, install fuse-overlayfs or ensure Linux kernel 5.11+ with user namespaces")
	copyDriver := NewCopyDriver(cfg)
	return copyDriver, nil
}

// DriverInfo returns information about available drivers on the current system.
// [OT-P2-004] Cross-Platform Driver Interface
func DriverInfo(ctx context.Context, cfg Config) []Info {
	var info []Info

	// Check overlayfs
	overlayDriver := NewOverlayfsDriver(cfg)
	overlayAvailable, _ := overlayDriver.IsAvailable(ctx)
	info = append(info, Info{
		Type:        DriverTypeOverlayfs,
		Version:     overlayDriver.Version(),
		Description: "Linux overlayfs driver - efficient copy-on-write using kernel overlayfs",
		Available:   overlayAvailable,
	})

	// Check fuse-overlayfs
	fuseDriver := NewFuseOverlayfsDriver(cfg)
	fuseAvailable, _ := fuseDriver.IsAvailable(ctx)
	info = append(info, Info{
		Type:        DriverTypeFuseOverlayfs,
		Version:     fuseDriver.Version(),
		Description: "FUSE overlayfs driver - unprivileged overlayfs with direct filesystem access",
		Available:   fuseAvailable,
	})

	// Copy driver is always available
	copyDriver := NewCopyDriver(cfg)
	info = append(info, Info{
		Type:        DriverTypeCopy,
		Version:     copyDriver.Version(),
		Description: "Cross-platform copy driver - works on any OS using file copies",
		Available:   true,
	})

	return info
}

// --- Driver Preference Storage ---

const preferenceFileName = "driver-preference.json"

// DriverPreference stores the user's driver preference.
type DriverPreference struct {
	// DriverID is the selected driver option ID (e.g., "fuse-overlayfs", "overlayfs-userns")
	DriverID string `json:"driverId"`
}

// SaveDriverPreference saves the driver preference to a file.
func SaveDriverPreference(baseDir, driverID string) error {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}

	pref := DriverPreference{DriverID: driverID}
	data, err := json.MarshalIndent(pref, "", "  ")
	if err != nil {
		return err
	}

	prefPath := filepath.Join(baseDir, preferenceFileName)
	return os.WriteFile(prefPath, data, 0o644)
}

// LoadDriverPreference loads the driver preference from a file.
// Returns empty string and error if no preference is set.
func LoadDriverPreference(baseDir string) (string, error) {
	prefPath := filepath.Join(baseDir, preferenceFileName)
	data, err := os.ReadFile(prefPath)
	if err != nil {
		return "", err
	}

	var pref DriverPreference
	if err := json.Unmarshal(data, &pref); err != nil {
		return "", err
	}

	return pref.DriverID, nil
}

// SelectDriverWithPreference returns the best available driver,
// respecting any saved preference.
func SelectDriverWithPreference(ctx context.Context, cfg Config) (Driver, error) {
	// Check for saved preference
	pref, err := LoadDriverPreference(cfg.BaseDir)
	if err == nil && pref != "" {
		// Map preference to config
		switch pref {
		case string(DriverOptionFuseOverlayfs):
			cfg.UseFuseOverlayfs = true
		case string(DriverOptionOverlayfsUserNS), string(DriverOptionOverlayfsRoot):
			cfg.UseFuseOverlayfs = false
		case string(DriverOptionCopy):
			// Force copy driver
			log.Printf("driver: using copy driver (saved preference)")
			return NewCopyDriver(cfg), nil
		}
	}

	return SelectDriver(ctx, cfg)
}
