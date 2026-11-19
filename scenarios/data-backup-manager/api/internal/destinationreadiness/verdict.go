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
//  6. Unknown native state is degraded (warning) when the bounded probe passed,
//     and unknown (blocked) when no probe ran.
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
	// An unobservable state is not evidence of corruption. The bounded probe is
	// the physical proof that this mount can be written, so a passing probe is
	// enough to admit the write with a warning. Without a probe there is no
	// evidence either way, and the verdict stays unknown.
	//
	// Note what is NOT part of this decision: how many other copies exist. That
	// is a drive-loss risk, reported by coverage as a single-copy warning. It
	// must never block a backup, because refusing the write leaves the data with
	// fewer copies rather than more.
	if !in.Probe.Ran {
		return WriteUnknown, "filesystem state is unobservable and no bounded write probe was run"
	}
	if in.Probe.Passed {
		return WriteDegraded, "filesystem state is not exposed by this filesystem; a bounded write/read/fsync probe passed"
	}
	return WriteUnknown, "filesystem state is unobservable and the bounded write probe did not pass"
}
