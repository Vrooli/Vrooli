package lifecycle

import (
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	operationViewStart = "start"
)

// Reader-side evaluation of a start-operation record (plan Phase 3 step 4):
// classify the record honestly (a running record with a dead initiator is
// abandoned, never trusted), derive an ETA from recorded phase-duration
// history (unknown history renders unknown — never a fabricated number), and
// apply the test-genie recommended-next-check policy so agents get a single
// sanctioned re-check cadence instead of inventing poll loops.

// StartOperationView is what status/wait readers surface for the latest
// start operation of an instance.
type StartOperationView struct {
	OperationID string `json:"operation_id"`
	Scenario    string `json:"scenario"`
	Variant     string `json:"variant,omitempty"`
	Operation   string `json:"operation"`
	// Status is running|succeeded|failed|abandoned. A stored "running" whose
	// initiator PID is dead is reported abandoned.
	Status string `json:"status"`
	// InitiatorPID is the orchestrating process recorded on the operation, 0
	// when the record carries none. Surfaced so a reader that sees an
	// in-flight operation can name the same process the lock-contention error
	// names (ScenarioBusyError.HolderPID) instead of hunting for it with ps.
	InitiatorPID int `json:"initiator_pid,omitempty"`
	// Initiator provenance answers "who started this?" once the initiating
	// process is gone — which, for a CLI-driven start, is almost immediately.
	// Empty on records written before provenance was captured, or on hosts
	// that cannot report a given field.
	InitiatorArgv       string                               `json:"initiator_argv,omitempty"`
	InitiatorParentPID  int                                  `json:"initiator_parent_pid,omitempty"`
	InitiatorParentArgv string                               `json:"initiator_parent_argv,omitempty"`
	InitiatorScope      string                               `json:"initiator_scope,omitempty"`
	Verdict             string                               `json:"verdict,omitempty"`
	Error               string                               `json:"error,omitempty"`
	CurrentStep         string                               `json:"current_step,omitempty"`
	DependencyCurrent   string                               `json:"dependency_current,omitempty"`
	DependencyIndex     int                                  `json:"dependency_index,omitempty"`
	DependencyTotal     int                                  `json:"dependency_total,omitempty"`
	StartedAt           time.Time                            `json:"started_at"`
	FinishedAt          *time.Time                           `json:"finished_at,omitempty"`
	ElapsedSeconds      int                                  `json:"elapsed_seconds"`
	Steps               []scenarioruntime.StartOperationStep `json:"steps,omitempty"`
	// ETAKnown is false when the phase-duration history cannot honestly
	// estimate the remaining time; ETASeconds is meaningful only when true.
	ETAKnown   bool `json:"eta_known"`
	ETASeconds int  `json:"eta_seconds,omitempty"`
	// RecommendedNextCheckSeconds mirrors the test-genie policy: 0 when
	// terminal (stop checking), 30 when the ETA is unknown, otherwise the
	// estimated remaining time clamped to [5, 60].
	RecommendedNextCheckSeconds int `json:"recommended_next_check_seconds"`
}

// TransitionLine renders the one-line attach-side heartbeat for a step or
// dependency transition, "" when there is nothing worth printing. Shared by
// the in-process attacher and the `scenario wait` handler so the two
// surfaces cannot drift.
func (v StartOperationView) TransitionLine() string {
	switch {
	case v.DependencyCurrent != "":
		return fmt.Sprintf("%s: in-flight start on dependency %s (%d/%d)", v.Scenario, v.DependencyCurrent, v.DependencyIndex, v.DependencyTotal)
	case v.CurrentStep != "":
		return fmt.Sprintf("%s: in-flight start in %s step", v.Scenario, v.CurrentStep)
	}
	return ""
}

// InFlightSummary renders the one-line description a status surface shows for
// a lifecycle operation that is still running; "" when the operation is
// terminal and nothing is in flight.
//
// Runtime status alone cannot express this. A scenario three seconds into a
// start reports "stopped" exactly like one nobody is touching, so the operator
// reaches for `scenario restart` and the per-scenario lock refuses it, naming
// only a bare PID. Reporting the in-flight operation — and the same initiator
// the ScenarioBusyError names — is what connects the two surfaces.
func (v StartOperationView) InFlightSummary() string {
	if v.Terminal() {
		return ""
	}
	operation := v.Operation
	if operation == "" {
		operation = operationViewStart
	}
	line := operation + " in progress"
	if v.InitiatorPID > 0 {
		line += fmt.Sprintf(" (pid %d)", v.InitiatorPID)
	}
	switch {
	case v.DependencyCurrent != "":
		line += fmt.Sprintf(" — dependency %s (%d/%d)", v.DependencyCurrent, v.DependencyIndex, v.DependencyTotal)
	case v.CurrentStep != "":
		line += fmt.Sprintf(" — %s step", v.CurrentStep)
	}
	line += fmt.Sprintf(", %ds elapsed", v.ElapsedSeconds)
	if v.ETAKnown {
		line += fmt.Sprintf(", ~%ds remaining", v.ETASeconds)
	}
	return line
}

// Terminal reports whether the viewed operation reached a final state
// (including abandoned — there is nothing to keep watching).
func (v StartOperationView) Terminal() bool {
	switch v.Status {
	case scenarioruntime.StartOperationStatusSucceeded,
		scenarioruntime.StartOperationStatusFailed,
		scenarioruntime.StartOperationStatusAbandoned:
		return true
	}
	return false
}

const (
	recommendedNextCheckUnknown = 30
	recommendedNextCheckMin     = 5
	recommendedNextCheckMax     = 60
)

// EvaluateStartOperation builds the reader view from a stored record. Pure:
// PID liveness, the clock, and duration history are injected.
func EvaluateStartOperation(op scenarioruntime.StartOperation, isPIDRunning func(int) bool, now time.Time, estimates map[string]time.Duration) StartOperationView {
	view := StartOperationView{
		OperationID:       op.OperationID,
		Scenario:          op.Scenario,
		Variant:           op.Variant,
		Operation:         op.Operation,
		Status:            op.Status,
		Verdict:           op.Verdict,
		Error:             op.Error,
		CurrentStep:       op.CurrentStep,
		DependencyCurrent: op.DependencyCurrent,
		DependencyIndex:   op.DependencyIndex,
		DependencyTotal:   op.DependencyTotal,
		StartedAt:         op.StartedAt,
		FinishedAt:        op.FinishedAt,
		Steps:             op.Steps(),
	}
	if op.InitiatorPID != nil {
		view.InitiatorPID = *op.InitiatorPID
	}
	if op.InitiatorParentPID != nil {
		view.InitiatorParentPID = *op.InitiatorParentPID
	}
	view.InitiatorArgv = op.InitiatorArgv
	view.InitiatorParentArgv = op.InitiatorParentArgv
	view.InitiatorScope = op.InitiatorScope
	end := now
	if op.FinishedAt != nil {
		end = *op.FinishedAt
	}
	if elapsed := end.Sub(op.StartedAt); elapsed > 0 {
		view.ElapsedSeconds = int(elapsed / time.Second)
	}
	// Dead-initiator staleness: the record claims running but nobody is
	// orchestrating. Report abandoned; the next start supersedes the row.
	if view.Status == scenarioruntime.StartOperationStatusRunning &&
		(op.InitiatorPID == nil || isPIDRunning == nil || !isPIDRunning(*op.InitiatorPID)) {
		view.Status = scenarioruntime.StartOperationStatusAbandoned
	}
	if view.Terminal() {
		view.RecommendedNextCheckSeconds = 0
		return view
	}

	remaining, known := estimateRemaining(view.Steps, op, now, estimates)
	view.ETAKnown = known
	if known {
		view.ETASeconds = int(remaining / time.Second)
		next := view.ETASeconds
		if next < recommendedNextCheckMin {
			next = recommendedNextCheckMin
		}
		if next > recommendedNextCheckMax {
			next = recommendedNextCheckMax
		}
		view.RecommendedNextCheckSeconds = next
		return view
	}
	view.RecommendedNextCheckSeconds = recommendedNextCheckUnknown
	return view
}

// estimateRemaining sums duration estimates for the steps still ahead of an
// in-flight operation. Honesty rules:
//   - a phase with recorded history contributes its smoothed estimate
//     (minus time already spent when it is the running step, floored at 0);
//   - develop and health always run, so missing history for either makes the
//     whole ETA unknown;
//   - setup runs conditionally: it counts when it is already running or when
//     the operation is a restart (setup is forced); a operationViewStart that has not
//     entered setup may skip it, so it is excluded rather than guessed.
func estimateRemaining(steps []scenarioruntime.StartOperationStep, op scenarioruntime.StartOperation, now time.Time, estimates map[string]time.Duration) (time.Duration, bool) {
	stepState := map[string]scenarioruntime.StartOperationStep{}
	for _, step := range steps {
		stepState[step.Name] = step
	}
	var remaining time.Duration
	for _, phase := range []string{startStepSetup, startStepDevelop, startStepHealth} {
		step, seen := stepState[phase]
		if seen && step.Status != scenarioruntime.StartStepRunning {
			continue // already finished
		}
		if phase == startStepSetup && !seen && op.Operation != "restart" {
			continue // may legitimately be skipped; do not guess
		}
		estimate, ok := estimates[phase]
		if !ok {
			return 0, false
		}
		if seen {
			spent := now.Sub(step.StartedAt)
			if spent > 0 {
				estimate -= spent
			}
			if estimate < 0 {
				estimate = 0
			}
		}
		remaining += estimate
	}
	return remaining, true
}
