package runs

import "fmt"

// lifecycle.go is the Level-2 workflow model for the backup run (per
// docs/concepts/FLOWS.md): the legal state set, the transition function, and
// the invariants. It is pure — no I/O, no clock — so the service orchestrates
// effects through seams and calls these for state decisions. Promotion to the
// Level-5 flow-verifier model (flow.json + generated checked model + replay)
// is a deferred follow-up recorded in docs/internal/PROBLEMS.md.

// RunEvent names a transition trigger in the run lifecycle.
type RunEvent string

const (
	EventStartCapture  RunEvent = "start_capture"  // pending -> capturing
	EventStartSnapshot RunEvent = "start_snapshot" // capturing -> snapshotting
	EventComplete      RunEvent = "complete"       // -> completed (all succeeded)
	EventPartialFail   RunEvent = "partial_fail"   // -> partial_failed (some failed/blocked)
	EventFail          RunEvent = "fail"           // -> failed (all failed/blocked)
)

// terminal reports whether s is a terminal run state.
func terminal(s RunStatus) bool {
	switch s {
	case RunCompleted, RunPartialFailed, RunFailed:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether a run status is final.
func IsTerminal(s RunStatus) bool { return terminal(s) }

// Transition returns the next state for (from, event), and whether the
// transition is legal. Illegal transitions (e.g. snapshotting before
// capturing, or any transition out of a terminal state) return ok=false.
func Transition(from RunStatus, ev RunEvent) (RunStatus, bool) {
	if terminal(from) {
		return from, false
	}
	switch ev {
	case EventStartCapture:
		if from == RunPending {
			return RunCapturing, true
		}
	case EventStartSnapshot:
		if from == RunCapturing {
			return RunSnapshotting, true
		}
	case EventComplete:
		if from == RunSnapshotting {
			return RunCompleted, true
		}
	case EventPartialFail:
		if from == RunCapturing || from == RunSnapshotting {
			return RunPartialFailed, true
		}
	case EventFail:
		if from == RunPending || from == RunCapturing || from == RunSnapshotting {
			return RunFailed, true
		}
	}
	return from, false
}

// TransitionStatus applies one legal event to a run. Persisted repositories
// invoke this owner before writing a state change.
func (r *Run) TransitionStatus(event RunEvent) error {
	if r == nil {
		return fmt.Errorf("cannot transition a nil run")
	}
	next, ok := Transition(r.Status, event)
	if !ok {
		return fmt.Errorf("invalid run transition %q + %q", r.Status, event)
	}
	r.Status = next
	return nil
}

// classifyTerminal picks the terminal status for a run from its outcome tally.
// This is the load-bearing rule: a single target failure does NOT fail the
// whole run — it becomes partial_failed; only all-non-succeeded is failed.
func classifyTerminal(succeeded, failed, blocked int) RunStatus {
	switch {
	case failed == 0 && blocked == 0:
		return RunCompleted
	case succeeded == 0:
		return RunFailed
	default:
		return RunPartialFailed
	}
}

// CheckInvariants validates a run record against the lifecycle invariants.
// Returns nil when the run is internally consistent. The repository calls it
// before terminal persistence; lifecycle_test.go exercises it directly.
func CheckInvariants(r Run) error {
	if !terminal(r.Status) {
		return nil // mid-flight runs are not checked here
	}
	var succeeded, failed, blocked int
	var warning bool
	for _, o := range r.Outcomes {
		switch o.Status {
		case OutcomeSucceeded:
			succeeded++
		case OutcomeFailed:
			failed++
		case OutcomeBlocked:
			blocked++
		}
		warning = warning || o.Warning != ""
	}
	want := classifyTerminal(succeeded, failed, blocked)
	if want == RunCompleted && (warning || r.Error != "") {
		want = RunPartialFailed
	}
	// A run with no outcomes that "completed" is vacuously fine (empty plan).
	if len(r.Outcomes) == 0 {
		return nil
	}
	if r.Status != want {
		return &invariantError{got: r.Status, want: want}
	}
	// A completed run must have a recorded success for every outcome.
	if r.Status == RunCompleted && succeeded != len(r.Outcomes) {
		return &invariantError{got: r.Status, want: RunPartialFailed}
	}
	return nil
}

type invariantError struct{ got, want RunStatus }

func (e *invariantError) Error() string {
	return "run invariant violated: status " + string(e.got) + " inconsistent with outcomes (expected " + string(e.want) + ")"
}
