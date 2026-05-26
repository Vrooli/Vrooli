package restores

// lifecycle.go is the Level-2 workflow model for a restore/verify operation
// (per docs/concepts/FLOWS.md): the legal state set, the transition function,
// and the invariants. It is pure — no I/O, no clock — so the service
// orchestrates effects through seams and calls these for state decisions.

// RestoreEvent names a transition trigger in the restore lifecycle.
type RestoreEvent string

const (
	EventStartRestore RestoreEvent = "start_restore" // requested -> restoring
	EventStartVerify  RestoreEvent = "start_verify"  // requested -> verifying
	EventRestored     RestoreEvent = "restored"      // restoring -> restored
	EventVerified     RestoreEvent = "verified"      // verifying -> verified
	EventFail         RestoreEvent = "fail"          // any non-terminal -> failed
)

// terminal reports whether s is a terminal restore state.
func terminal(s RestoreStatus) bool {
	switch s {
	case RestoreVerified, RestoreRestored, RestoreFailed:
		return true
	default:
		return false
	}
}

// Transition returns the next state for (from, event), and whether the
// transition is legal. Illegal transitions (e.g. moving to verified from
// restoring, or any transition out of a terminal state) return ok=false.
func Transition(from RestoreStatus, ev RestoreEvent) (RestoreStatus, bool) {
	if terminal(from) {
		return from, false
	}
	switch ev {
	case EventStartRestore:
		if from == RestoreRequested {
			return RestoreRestoring, true
		}
	case EventStartVerify:
		if from == RestoreRequested {
			return RestoreVerifying, true
		}
	case EventRestored:
		if from == RestoreRestoring {
			return RestoreRestored, true
		}
	case EventVerified:
		if from == RestoreVerifying {
			return RestoreVerified, true
		}
	case EventFail:
		if !terminal(from) {
			return RestoreFailed, true
		}
	}
	return from, false
}

// CheckInvariants validates a restore record against the lifecycle invariants.
// Returns nil when the record is internally consistent. The critical invariant:
// status=verified REQUIRES a non-zero last_verified_at and a non-empty
// checksum. A false "verified" is the worst possible bug (OT-P0-006).
func CheckInvariants(r Restore) error {
	if r.Status == RestoreVerified {
		if r.LastVerifiedAt.IsZero() {
			return &invariantError{msg: "status=verified but last_verified_at is zero"}
		}
		if r.Checksum == "" {
			return &invariantError{msg: "status=verified but checksum is empty"}
		}
	}
	if r.Status == RestoreFailed {
		if !r.LastVerifiedAt.IsZero() {
			return &invariantError{msg: "status=failed but last_verified_at is set"}
		}
	}
	return nil
}

type invariantError struct{ msg string }

func (e *invariantError) Error() string { return "restore invariant violated: " + e.msg }
