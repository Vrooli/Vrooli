package destinationreadiness

import (
	"strings"

	"data-backup-manager/internal/sysmounts"
)

// WriteVerdictInput is the complete evidence set for the write-safety state
// machine. Keeping the decision pure makes every branch testable and stops the
// verdict from being re-derived differently in preflight, health, and the UI.
type WriteVerdictInput struct {
	Mounted              bool
	ReadOnly             bool
	DeviceWriteProtected bool
	FilesystemState      sysmounts.FilesystemState
	OfflineCheckVerified bool
	Probe                WriteProbeResult
	// AllowDegraded is true only when another verified copy exists for every
	// selected target. It never upgrades the native unknown state to clean.
	AllowDegraded bool
}

// EvaluateWriteVerdict returns the explicit write verdict and a short reason.
//
// Precedence is deliberate:
//  1. Impossibility (not mounted, read-only, write-protected) blocks.
//  2. A failed probe blocks even when native metadata claims clean, because
//     the destination just demonstrated it cannot accept and return bytes.
//  3. A dirty/needs-check native state blocks; only a real check may clear it.
//  4. Clean native state is ready.
//  5. Verified offline-check evidence is ready despite an unknown native state.
//  6. Unknown native state is degraded only with an explicit policy and a
//     passed probe; otherwise it stays unknown.
func EvaluateWriteVerdict(in WriteVerdictInput) (WriteVerdict, string) {
	switch {
	case !in.Mounted:
		return WriteBlocked, "destination is not mounted"
	case in.ReadOnly:
		return WriteBlocked, "destination is mounted read-only"
	case in.DeviceWriteProtected:
		return WriteBlocked, "destination block device is write-protected"
	case in.Probe.Ran && !in.Probe.Passed:
		return WriteBlocked, "bounded write probe failed: " + strings.TrimSpace(in.Probe.Error)
	}

	switch in.FilesystemState {
	case sysmounts.FilesystemStateDirty:
		return WriteBlocked, "filesystem reports a dirty state"
	case sysmounts.FilesystemStateNeedsCheck:
		return WriteBlocked, "filesystem reports that a check is required"
	case sysmounts.FilesystemStateClean:
		return WriteReady, "filesystem reports a clean state"
	}

	if in.OfflineCheckVerified {
		return WriteReady, "offline consistency check passed for the current read/write mount episode"
	}
	if in.AllowDegraded {
		if !in.Probe.Ran {
			return WriteUnknown, "filesystem state is unobservable and no bounded write probe was run"
		}
		if in.Probe.Passed {
			return WriteDegraded, "filesystem state is unobservable; a bounded write probe passed and another verified copy exists"
		}
	}
	return WriteUnknown, "filesystem dirty/needs-check state is not exposed on this host"
}
