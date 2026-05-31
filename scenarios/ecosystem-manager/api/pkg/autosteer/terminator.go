package autosteer

import (
	"fmt"
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

	if target := profile.Objective.Targets.OperationalTargetsPct; target > 0 {
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
// then budget backstop, then diminishing returns.
func (t *Terminator) ShouldStop(state *ProfileExecutionState, profile *AutoSteerProfile) (bool, string) {
	if state == nil || profile == nil {
		return false, StopReasonContinue
	}

	if met, reason := objectiveMet(state, profile); met {
		return true, reason
	}

	if profile.Budget.MaxIterations > 0 && state.Iteration >= profile.Budget.MaxIterations {
		return true, fmt.Sprintf("%s: reached max %d iterations", StopBudgetExhausted, profile.Budget.MaxIterations)
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
