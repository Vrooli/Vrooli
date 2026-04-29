package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// SelectDriver returns the best available driver for the current system.
// Priority order:
//  1. Kernel overlayfs (UserNS variant when in a user namespace, Root
//     variant otherwise) — flat memory, no per-mount daemon.
//  2. fuse-overlayfs — userspace fallback when kernel overlayfs is unavailable.
//  3. Copy driver — cross-platform fallback, always available.
func SelectDriver(ctx context.Context, cfg Config) (Driver, error) {
	overlayDriver := NewOverlayfsDriver(cfg)
	if available, err := overlayDriver.IsAvailable(ctx); err == nil && available {
		log.Printf("driver: using kernel overlayfs (%s)", overlayDriver.ID())
		return overlayDriver, nil
	} else if err != nil {
		log.Printf("driver: kernel overlayfs not available: %v", err)
	}

	fuseDriver := NewFuseOverlayfsDriver(cfg)
	if available, err := fuseDriver.IsAvailable(ctx); err == nil && available {
		log.Printf("driver: using fuse-overlayfs (kernel overlayfs unavailable; daemon-per-mount)")
		return fuseDriver, nil
	} else if err != nil {
		log.Printf("driver: fuse-overlayfs not available: %v", err)
	}

	log.Printf("driver: falling back to copy driver (slower but universal)")
	log.Printf("driver: for better performance, install fuse-overlayfs or wrap startup with `unshare -U -m -r`")
	return NewCopyDriver(cfg), nil
}

// DriverInfo returns information about available drivers on the current system.
func DriverInfo(ctx context.Context, cfg Config) []Info {
	overlayDriver := NewOverlayfsDriver(cfg)
	overlayAvailable, _ := overlayDriver.IsAvailable(ctx)

	fuseDriver := NewFuseOverlayfsDriver(cfg)
	fuseAvailable, _ := fuseDriver.IsAvailable(ctx)

	copyDriver := NewCopyDriver(cfg)

	return []Info{
		{
			ID:          overlayDriver.ID(),
			Version:     overlayDriver.Version(),
			Description: "Linux overlayfs driver - efficient copy-on-write using kernel overlayfs",
			Available:   overlayAvailable,
		},
		{
			ID:          DriverFuseOverlayfs,
			Version:     fuseDriver.Version(),
			Description: "FUSE overlayfs driver - unprivileged overlayfs with direct filesystem access",
			Available:   fuseAvailable,
		},
		{
			ID:          DriverCopy,
			Version:     copyDriver.Version(),
			Description: "Cross-platform copy driver - works on any OS using file copies",
			Available:   true,
		},
	}
}

// --- Driver Preference Storage ---

const preferenceFileName = "driver-preference.json"

// DriverPreference stores the user's saved driver preference.
type DriverPreference struct {
	DriverID string `json:"driverId"`
}

// SaveDriverPreference saves the driver preference to a file under baseDir.
func SaveDriverPreference(baseDir, driverID string) error {
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

// SelectDriverWithPreference returns the best available driver, respecting
// any saved preference. A saved preference for an unavailable driver
// falls through to SelectDriver's normal priority.
func SelectDriverWithPreference(ctx context.Context, cfg Config) (Driver, error) {
	pref, err := LoadDriverPreference(cfg.BaseDir)
	if err == nil && pref != "" {
		if d, ok := tryPreferredDriver(ctx, cfg, DriverID(pref)); ok {
			return d, nil
		}
		log.Printf("driver: saved preference %q not available; falling through to auto-select", pref)
	}
	return SelectDriver(ctx, cfg)
}

// tryPreferredDriver constructs the driver for id and returns it when
// IsAvailable succeeds. Returns (nil, false) on unknown ID or unavailability.
func tryPreferredDriver(ctx context.Context, cfg Config, id DriverID) (Driver, bool) {
	d, err := NewDriverFor(cfg, id)
	if err != nil {
		return nil, false
	}
	available, err := d.IsAvailable(ctx)
	if err != nil || !available {
		return nil, false
	}
	log.Printf("driver: using %s (saved preference)", id)
	return d, true
}

// NewDriverFor returns a fresh driver for the given canonical ID. Used by
// SwitchDriver (in slot.go) when an operator hot-swaps drivers via
// /api/v1/driver/select.
func NewDriverFor(cfg Config, id DriverID) (Driver, error) {
	switch id {
	case DriverFuseOverlayfs:
		return NewFuseOverlayfsDriver(cfg), nil
	case DriverOverlayfsUserNS:
		return NewOverlayfsUserNSDriver(cfg), nil
	case DriverOverlayfsRoot:
		return NewOverlayfsRootDriver(cfg), nil
	case DriverCopy:
		return NewCopyDriver(cfg), nil
	}
	return nil, fmt.Errorf("unknown driver ID: %s", id)
}
