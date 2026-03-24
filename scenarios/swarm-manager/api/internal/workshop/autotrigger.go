// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
//
// Auto-trigger decision logic for workshop rounds. These are pure functions
// with no side effects — they decide whether to auto-advance or auto-initialize,
// but do not perform the actual spawning.
package workshop

// MaxAutoRounds is the safety cap on auto-triggered workshop rounds.
// The boost formula guarantees convergence for raw scores >= 2 well before
// this limit; this cap exists as a safety net for items with persistently
// low raw scores.
const MaxAutoRounds = 10

// AutoAdvanceResult holds the decision and reason from ShouldAutoAdvance.
type AutoAdvanceResult struct {
	Advance bool
	Reason  string // "not_ready", "ready", "max_rounds", "pending_decisions", "no_rounds"
}

// ShouldAutoAdvance decides whether the next workshop round should be
// auto-triggered after saving round responses. Returns true only when:
//   - The latest round exists and has no pending (unanswered) decisions
//   - The item is not yet ready (effective scores < 3 on at least one dimension)
//   - The round count is below MaxAutoRounds
func ShouldAutoAdvance(latestRound *Round, roundCount int, kind string) AutoAdvanceResult {
	if latestRound == nil {
		return AutoAdvanceResult{Advance: false, Reason: "no_rounds"}
	}
	if CountPendingDecisions(latestRound) > 0 {
		return AutoAdvanceResult{Advance: false, Reason: "pending_decisions"}
	}
	effective := ComputeEffectiveScores(latestRound.Readiness, roundCount, kind)
	if IsReady(effective) {
		return AutoAdvanceResult{Advance: false, Reason: "ready"}
	}
	if roundCount >= MaxAutoRounds {
		return AutoAdvanceResult{Advance: false, Reason: "max_rounds"}
	}
	return AutoAdvanceResult{Advance: true, Reason: "not_ready"}
}

// ShouldAutoInitialize decides whether a newly-created backlog item should
// have its first workshop round auto-triggered. The default is true (when
// autoWorkshop is nil). Callers can opt out by passing a pointer to false.
func ShouldAutoInitialize(autoWorkshop *bool) bool {
	if autoWorkshop == nil {
		return true
	}
	return *autoWorkshop
}
