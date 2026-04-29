package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// --- Driver Selection ---
//
// SelectDriver returns the best available driver for the current system,
// honoring any saved preference. The priority order (post-Phase 5) is:
//   1. OverlayfsUserNS — kernel overlayfs wrapped in a user namespace.
//      No fuse-overlayfs daemon per mount → flat memory under load.
//   2. FuseOverlayfs — userspace fallback when the deployment isn't
//      wrapped in `unshare -U -m -r` or the kernel can't do unprivileged
//      overlayfs.
//   3. OverlayfsRoot — kernel overlayfs with CAP_SYS_ADMIN. Mostly an
//      escape hatch; rarely correct in single-tenant local dev.
//   4. Copy — cross-platform fallback. Always available; slowest.

// SelectDriver returns the best available driver for the current system.
// It tests each driver in priority order and returns the first one that
// reports IsAvailable. All non-trivial branches log so operators can see
// why a particular driver was chosen.
func SelectDriver(ctx context.Context, cfg Config) (Driver, error) {
	// 1. OverlayfsUserNS / OverlayfsRoot — both surface as OverlayfsDriver.
	overlayDriver := NewOverlayfsDriver(cfg)
	if available, err := overlayDriver.IsAvailable(ctx); err == nil && available {
		log.Printf("driver: using kernel overlayfs (optimal performance, no per-mount daemon)")
		return overlayDriver, nil
	} else if err != nil {
		log.Printf("driver: kernel overlayfs not available: %v", err)
	}

	// 2. FuseOverlayfs.
	fuseDriver := NewFuseOverlayfsDriver(cfg)
	if available, err := fuseDriver.IsAvailable(ctx); err == nil && available {
		log.Printf("driver: using fuse-overlayfs (kernel overlayfs unavailable; daemon-per-mount)")
		return fuseDriver, nil
	} else if err != nil {
		log.Printf("driver: fuse-overlayfs not available: %v", err)
	}

	// 3. Copy fallback.
	log.Printf("driver: falling back to copy driver (slower but universal)")
	log.Printf("driver: for better performance, install fuse-overlayfs or wrap startup with `unshare -U -m -r`")
	return NewCopyDriver(cfg), nil
}

// DriverInfo returns information about available drivers on the current system.
func DriverInfo(ctx context.Context, cfg Config) []Info {
	var info []Info

	overlayDriver := NewOverlayfsDriver(cfg)
	overlayAvailable, _ := overlayDriver.IsAvailable(ctx)
	info = append(info, Info{
		Type:        DriverTypeOverlayfs,
		Version:     overlayDriver.Version(),
		Description: "Linux overlayfs driver - efficient copy-on-write using kernel overlayfs",
		Available:   overlayAvailable,
	})

	fuseDriver := NewFuseOverlayfsDriver(cfg)
	fuseAvailable, _ := fuseDriver.IsAvailable(ctx)
	info = append(info, Info{
		Type:        DriverTypeFuseOverlayfs,
		Version:     fuseDriver.Version(),
		Description: "FUSE overlayfs driver - unprivileged overlayfs with direct filesystem access",
		Available:   fuseAvailable,
	})

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
// any saved preference. A saved preference for "copy" forces CopyDriver
// even when overlayfs is available; preference for kernel/fuse overlayfs
// short-circuits to that driver if available, otherwise we fall through
// to SelectDriver's normal priority.
func SelectDriverWithPreference(ctx context.Context, cfg Config) (Driver, error) {
	pref, err := LoadDriverPreference(cfg.BaseDir)
	if err == nil && pref != "" {
		switch DriverOptionID(pref) {
		case DriverOptionCopy:
			log.Printf("driver: using copy driver (saved preference)")
			return NewCopyDriver(cfg), nil
		case DriverOptionFuseOverlayfs:
			fuseDriver := NewFuseOverlayfsDriver(cfg)
			if available, err := fuseDriver.IsAvailable(ctx); err == nil && available {
				log.Printf("driver: using fuse-overlayfs (saved preference)")
				return fuseDriver, nil
			}
			log.Printf("driver: saved preference fuse-overlayfs not available; falling through to auto-select")
		case DriverOptionOverlayfsUserNS, DriverOptionOverlayfsRoot:
			overlayDriver := NewOverlayfsDriver(cfg)
			if available, err := overlayDriver.IsAvailable(ctx); err == nil && available {
				log.Printf("driver: using kernel overlayfs (saved preference: %s)", pref)
				return overlayDriver, nil
			}
			log.Printf("driver: saved preference %s not available; falling through to auto-select", pref)
		}
	}
	return SelectDriver(ctx, cfg)
}

// NewDriverFor returns a fresh driver for the given option ID. Used by
// SwitchDriver (in slot.go) when an operator hot-swaps drivers via
// /api/v1/driver/select.
func NewDriverFor(cfg Config, optionID DriverOptionID) (Driver, error) {
	switch optionID {
	case DriverOptionFuseOverlayfs:
		return NewFuseOverlayfsDriver(cfg), nil
	case DriverOptionOverlayfsUserNS, DriverOptionOverlayfsRoot:
		return NewOverlayfsDriver(cfg), nil
	case DriverOptionCopy:
		return NewCopyDriver(cfg), nil
	}
	return nil, fmt.Errorf("unknown driver option: %s", optionID)
}
