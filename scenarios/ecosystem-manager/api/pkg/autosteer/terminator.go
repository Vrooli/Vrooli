package autosteer

import (
	"fmt"
	"math"
	"strings"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// diminishingReturnsWindow is the trailing number of score samples over which
// the controller measures marginal improvement. It is the SAME measurement the
// v1 Layer-2 net-progress thrashing detector extends (see CONTROL-MODEL.md
// "Termination"); v1 adds the closed/introduced split, v0 uses net total only.
const diminishingReturnsWindow = 3

// Stop reasons (stable strings surfaced in the decision trace and history).
const (
	StopObjectiveMet       = "objective_met"
	StopDiminishingReturns = "diminishing_returns"
	StopBudgetExhausted    = "budget_exhausted"
	StopReasonContinue     = "continue"
	StopNothingActionable  = "nothing_actionable"
	StopThrashingCycle     = "thrashing_cycle"
	StopNoNetProgress      = "no_net_progress"
)

// severityRank maps a configured max-open-severity string to a comparable rank.
// A finding "above" the threshold (rank strictly greater) keeps the loop
// running. An empty/"none" threshold tolerates no open findings at all.
func severityRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none":
		return 0
	case "info", "low":
		return 1
	case "warning", "warn", "medium":
		return 2
	case "error", "high":
		return 3
	case "blocker", "critical":
		return 4
	default:
		return 0
	}
}

// findingRank maps a finding severity to the same comparable scale.
func findingRank(s architecturev1.FindingSeverity) int {
	switch s {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return 4
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return 3
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return 2
	default: // INFO and UNSPECIFIED
		return 1
	}
}

// Terminator implements the controller's global, gradient-based TERMINATE
// stage. Termination is never per-phase (there are no phases): it is one of
// objective-met, diminishing-returns, or budget-exhausted.
type Terminator struct{}

// NewTerminator creates a Terminator.
func NewTerminator() *Terminator { return &Terminator{} }

// objectiveMet reports whether the profile's targets are satisfied by the
// current state: no finding above max_open_severity AND (if configured) the
// operational-targets gap metric meets its threshold.
func objectiveMet(state *ProfileExecutionState, profile *AutoSteerProfile) (bool, string) {
	threshold := severityRank(profile.Objective.Targets.MaxOpenSeverity)
	for _, f := range state.Findings.Findings {
		if findingRank(f.Severity) > threshold {
			return false, ""
		}
	}

	// Operational-targets gate: only applies when the scenario actually declares
	// operational targets. A scenario with none is vacuously satisfied — treating
	// "no targets" as 0% would make objective-met permanently unreachable.
	if target := profile.Objective.Targets.OperationalTargetsPct; target > 0 && state.Metrics.OperationalTargetsTotal > 0 {
		if state.Metrics.OperationalTargetsPercentage < target {
			return false, ""
		}
	}

	return true, fmt.Sprintf(
		"objective met: no finding above severity %q and operational targets ≥ %.0f%%",
		strings.ToLower(strings.TrimSpace(profile.Objective.Targets.MaxOpenSeverity)),
		profile.Objective.Targets.OperationalTargetsPct,
	)
}

// meanImprovement returns the average per-iteration reduction in total weighted
// score over the trailing window, and whether enough samples exist to judge it.
// A positive value means the score is trending down (improving).
func meanImprovement(history []float64) (float64, bool) {
	n := len(history)
	if n < diminishingReturnsWindow {
		return 0, false
	}
	window := history[n-diminishingReturnsWindow:]
	total := window[0] - window[len(window)-1]
	steps := float64(len(window) - 1)
	if steps <= 0 {
		return 0, false
	}
	return total / steps, true
}

// ShouldStop evaluates global termination. Order: objective-met (best outcome),
// then the Layer-2 thrashing defenses (fingerprint cycle, then net-progress
// stall — caught on first recurrence rather than at the backstop), then the
// budget backstop, then diminishing returns.
func (t *Terminator) ShouldStop(state *ProfileExecutionState, profile *AutoSteerProfile) (bool, string) {
	if state == nil || profile == nil {
		return false, StopReasonContinue
	}

	if met, reason := objectiveMet(state, profile); met {
		return true, reason
	}

	if at := cycleRecurrence(state, profile.Budget.cycleWindow()); at > 0 {
		return true, fmt.Sprintf(
			"%s: current open-findings set recurred (last seen iteration %d) within a %d-iteration window",
			StopThrashingCycle, at, profile.Budget.cycleWindow(),
		)
	}

	if flow, ok := netFindingsFlow(state, profile.Budget.netProgressWindow()); ok &&
		math.Abs(float64(flow)) <= profile.Budget.netProgressFloor() {
		return true, fmt.Sprintf(
			"%s: net findings flow %+d over last %d iterations within floor %.0f",
			StopNoNetProgress, flow, profile.Budget.netProgressWindow(), profile.Budget.netProgressFloor(),
		)
	}

	if cap := effectiveMaxIterations(state, profile); cap > 0 && state.Iteration >= cap {
		if cap < profile.Budget.MaxIterations {
			return true, fmt.Sprintf(
				"%s: reached degraded-gate cap %d (halved from %d after the DTV gate degraded)",
				StopBudgetExhausted, cap, profile.Budget.MaxIterations,
			)
		}
		return true, fmt.Sprintf("%s: reached max %d iterations", StopBudgetExhausted, cap)
	}

	if floor := profile.Budget.DiminishingReturnsFloor; floor > 0 {
		if improvement, ok := meanImprovement(state.ScoreHistory); ok && improvement < floor {
			return true, fmt.Sprintf(
				"%s: mean weighted-score improvement %.4f/iter over last %d iterations < floor %.4f",
				StopDiminishingReturns, improvement, diminishingReturnsWindow, floor,
			)
		}
	}

	return false, StopReasonContinue
}

// firstDegradedIteration returns the iteration of the earliest trace entry whose
// Layer-1 DTV gate ran degraded (proceed-cap-flag, EM-P2), or 0 if none. The
// latch lives in the trace itself (no separate state column) so it survives
// persistence and the halving stays idempotent across iterations.
func firstDegradedIteration(state *ProfileExecutionState) int {
	for _, e := range state.Trace {
		if e.GateDegradedCause != "" {
			return e.Iteration
		}
	}
	return 0
}

// effectiveMaxIterations applies the proceed-cap-flag budget penalty: once the
// DTV gate has degraded, the remaining iteration budget (from the first degraded
// iteration onward) is halved — applied exactly once because it is derived from
// the fixed first-degraded iteration. Returns the profile's MaxIterations
// unchanged when the gate never degraded (or no cap is set).
func effectiveMaxIterations(state *ProfileExecutionState, profile *AutoSteerProfile) int {
	maxIters := profile.Budget.MaxIterations
	if maxIters <= 0 || state == nil {
		return maxIters
	}
	d := firstDegradedIteration(state)
	if d <= 0 || d >= maxIters {
		return maxIters
	}
	remaining := maxIters - d
	return d + remaining/2
}

// cycleRecurrence reports the iteration at which the current open-findings
// fingerprint was previously seen within the trailing window of k iterations, or
// 0 if there is no recurrence. An empty fingerprint (no open findings) never
// counts as a cycle — that path is handled by objective-met.
func cycleRecurrence(state *ProfileExecutionState, k int) int {
	fp := state.Findings.Fingerprint
	if fp == "" || k <= 0 {
		return 0
	}
	n := len(state.Trace)
	start := n - k
	if start < 0 {
		start = 0
	}
	for i := start; i < n; i++ {
		if state.Trace[i].Fingerprint == fp {
			return state.Trace[i].Iteration
		}
	}
	return 0
}

// netFindingsFlow sums (closed − introduced) across all dimensions over the
// trailing w completed iterations. The bool is false until w iterations exist.
func netFindingsFlow(state *ProfileExecutionState, w int) (int, bool) {
	n := len(state.Trace)
	if w <= 0 || n < w {
		return 0, false
	}
	flow := 0
	for i := n - w; i < n; i++ {
		e := state.Trace[i]
		for _, c := range e.ClosedByDimension {
			flow += c
		}
		for _, c := range e.IntroducedByDimension {
			flow -= c
		}
	}
	return flow, true
}
