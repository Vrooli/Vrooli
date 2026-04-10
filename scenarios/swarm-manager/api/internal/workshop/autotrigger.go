// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
//
// Auto-trigger decision logic for workshop rounds. These are pure functions
// with no side effects — they decide whether to auto-advance or auto-initialize,
// but do not perform the actual spawning.
package workshop

// AutoAdvanceResult holds the decision and reason from ShouldAutoAdvance.
type AutoAdvanceResult struct {
	Advance  bool
	Reason   string // "not_ready", "finalizing", "max_rounds", "pending_decisions", "disabled", "no_rounds"
	NextMode string // "workshop" | "finalize" | ""
}

// ShouldAutoAdvance decides whether the next workshop step should be
// auto-triggered after saving round responses. Returns true only when:
//   - Auto-advance is enabled globally
//   - The latest round exists and has no pending (unanswered) decisions
//   - Either the item is ready and should finalize, or it is not yet ready and
//     the round count is below maxAutoRounds
func ShouldAutoAdvance(enabled bool, latestRound *Round, roundCount int, kind string, maxAutoRounds int) AutoAdvanceResult {
	if !enabled {
		return AutoAdvanceResult{Advance: false, Reason: "disabled"}
	}
	if latestRound == nil {
		return AutoAdvanceResult{Advance: false, Reason: "no_rounds"}
	}
	if CountPendingDecisions(latestRound) > 0 {
		return AutoAdvanceResult{Advance: false, Reason: "pending_decisions"}
	}
	effective := ComputeEffectiveScores(latestRound.Readiness, roundCount, kind)
	if IsReady(effective) {
		return AutoAdvanceResult{Advance: true, Reason: "finalizing", NextMode: "finalize"}
	}
	if roundCount >= maxAutoRounds {
		return AutoAdvanceResult{Advance: false, Reason: "max_rounds", NextMode: "workshop"}
	}
	return AutoAdvanceResult{Advance: true, Reason: "not_ready", NextMode: "workshop"}
}

// ShouldAutoInitialize decides whether a newly-created backlog item should
// have its first workshop round auto-triggered. Controlled by the global
// auto_initialize_workshop setting.
func ShouldAutoInitialize(enabled bool) bool {
	return enabled
}

// ShouldCascade decides whether dependency-resolved items should auto-trigger
// workshops for their dependents. Controlled by the global
// auto_cascade_workshop setting.
func ShouldCascade(enabled bool) bool {
	return enabled
}
