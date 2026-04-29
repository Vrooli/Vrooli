package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// CandidateReport is one driver's evaluation in SelectionReport.
type CandidateReport struct {
	ID         DriverID `json:"id"`
	Available  bool     `json:"available"`
	Reason     string   `json:"reason,omitempty"`     // why available/unavailable
	Selected   bool     `json:"selected,omitempty"`   // true on the chosen driver
	Preference bool     `json:"preference,omitempty"` // true if a saved preference picked it
}

// SelectionReport explains the driver-selection decision: every candidate
// considered, why each was kept or rejected, and which one was finally
// chosen. Logged at startup and surfaced via /api/v1/driver/options for
// UI diagnostics.
type SelectionReport struct {
	Selected         DriverID          `json:"selected"`
	InUserNamespace  bool              `json:"inUserNamespace"`
	PreferenceFile   string            `json:"preferenceFile,omitempty"` // path that was checked
	PreferenceValue  string            `json:"preferenceValue,omitempty"`
	PreferenceUsed   bool              `json:"preferenceUsed"`
	Candidates       []CandidateReport `json:"candidates"`
}

// SelectDriver returns the best available driver for the current system.
// Priority order:
//  1. Kernel overlayfs (UserNS variant when in a user namespace, Root
//     variant otherwise) — flat memory, no per-mount daemon.
//  2. fuse-overlayfs — userspace fallback when kernel overlayfs is unavailable.
//  3. Copy driver — cross-platform fallback, always available.
//
// Returns a SelectionReport so callers can log every candidate's
// evaluation. The report includes one entry per candidate, in priority
// order; the chosen driver has Selected=true.
func SelectDriver(ctx context.Context, cfg Config) (Driver, *SelectionReport, error) {
	report := &SelectionReport{InUserNamespace: InUserNamespace()}

	candidates := []struct {
		id     DriverID
		ctor   func() Driver
	}{
		{DriverOverlayfsUserNS, func() Driver { return NewOverlayfsUserNSDriver(cfg) }},
		{DriverOverlayfsRoot, func() Driver { return NewOverlayfsRootDriver(cfg) }},
		{DriverFuseOverlayfs, func() Driver { return NewFuseOverlayfsDriver(cfg) }},
		{DriverCopy, func() Driver { return NewCopyDriver(cfg) }},
	}

	// Skip the inappropriate kernel-overlay variant for the current
	// environment. UserNS variant only applies when we are in a userns;
	// Root variant only applies when we are NOT in a userns. This matches
	// the prior NewOverlayfsDriver auto-pick behavior, but here every
	// candidate gets explicitly recorded.
	inNS := report.InUserNamespace

	var selected Driver
	for _, c := range candidates {
		entry := CandidateReport{ID: c.id}
		switch c.id {
		case DriverOverlayfsUserNS:
			if !inNS {
				entry.Available = false
				entry.Reason = "skipped: API is not running inside a user namespace (kernel overlayfs userns variant requires `unshare -U -m -r`)"
				report.Candidates = append(report.Candidates, entry)
				continue
			}
		case DriverOverlayfsRoot:
			if inNS {
				entry.Available = false
				entry.Reason = "skipped: API is inside a user namespace; the userns overlayfs variant takes precedence"
				report.Candidates = append(report.Candidates, entry)
				continue
			}
		}

		d := c.ctor()
		available, err := d.IsAvailable(ctx)
		entry.Available = available
		if err != nil {
			entry.Reason = err.Error()
		} else if available {
			entry.Reason = "available"
		} else {
			entry.Reason = "not available"
		}

		if available && selected == nil {
			selected = d
			entry.Selected = true
		}
		report.Candidates = append(report.Candidates, entry)
	}

	if selected == nil {
		// Copy is unconditionally available, so this branch is unreachable
		// in practice. Defensive fallback to keep the contract honest.
		copy := NewCopyDriver(cfg)
		selected = copy
		report.Candidates = append(report.Candidates, CandidateReport{
			ID:        DriverCopy,
			Available: true,
			Reason:    "fallback (defensive: no other driver was available)",
			Selected:  true,
		})
	}
	report.Selected = selected.ID()
	return selected, report, nil
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
// falls through to SelectDriver's normal priority. The SelectionReport
// describes both the preference attempt (PreferenceFile/PreferenceValue/
// PreferenceUsed) and every candidate's evaluation.
func SelectDriverWithPreference(ctx context.Context, cfg Config) (Driver, *SelectionReport, error) {
	prefPath := filepath.Join(cfg.BaseDir, preferenceFileName)
	pref, err := LoadDriverPreference(cfg.BaseDir)
	if err == nil && pref != "" {
		if d, ok := tryPreferredDriver(ctx, cfg, DriverID(pref)); ok {
			report := &SelectionReport{
				Selected:        d.ID(),
				InUserNamespace: InUserNamespace(),
				PreferenceFile:  prefPath,
				PreferenceValue: pref,
				PreferenceUsed:  true,
				Candidates: []CandidateReport{{
					ID:         d.ID(),
					Available:  true,
					Reason:     "selected via saved preference",
					Selected:   true,
					Preference: true,
				}},
			}
			return d, report, nil
		}
		log.Printf("driver: saved preference %q not available; falling through to auto-select", pref)
	}
	d, report, err := SelectDriver(ctx, cfg)
	if report != nil {
		report.PreferenceFile = prefPath
		report.PreferenceValue = pref
		report.PreferenceUsed = false
	}
	return d, report, err
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

// LogSelectionReport writes the selection report as a structured log line
// for operator diagnostics. Called once at startup; SelectDriver itself
// does not log because a few callers (tests, options endpoint) want the
// data structurally instead.
func LogSelectionReport(report *SelectionReport) {
	if report == nil {
		return
	}
	for _, c := range report.Candidates {
		state := "skipped"
		if c.Selected {
			state = "selected"
		} else if c.Available {
			state = "available"
		} else {
			state = "unavailable"
		}
		log.Printf("driver: candidate=%s state=%s reason=%q", c.ID, state, c.Reason)
	}
	log.Printf("driver: selected=%s inUserNamespace=%t preferenceUsed=%t preferenceValue=%q",
		report.Selected, report.InUserNamespace, report.PreferenceUsed, report.PreferenceValue)
}
