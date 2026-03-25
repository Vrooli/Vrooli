// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
//
// Auto-trigger decision logic for workshop rounds. These are pure functions
// with no side effects — they decide whether to auto-advance or auto-initialize,
// but do not perform the actual spawning.
package workshop

// AutoAdvanceResult holds the decision and reason from ShouldAutoAdvance.
type AutoAdvanceResult struct {
	Advance bool
	Reason  string // "not_ready", "ready", "max_rounds", "pending_decisions", "no_rounds"
}

// ShouldAutoAdvance decides whether the next workshop round should be
// auto-triggered after saving round responses. Returns true only when:
//   - The latest round exists and has no pending (unanswered) decisions
//   - The item is not yet ready (effective scores < 3 on at least one dimension)
//   - The round count is below maxAutoRounds
func ShouldAutoAdvance(latestRound *Round, roundCount int, kind string, maxAutoRounds int) AutoAdvanceResult {
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
	if roundCount >= maxAutoRounds {
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

// DependencyStatus holds a resolved dependency's ref and status.
type DependencyStatus struct {
	Ref    string // "kind/name"
	Status string // backlog status string, or "" if unresolved
	Found  bool   // false if dependency couldn't be loaded
}

// WorkshopBlockResult holds the workshop dependency-check decision.
type WorkshopBlockResult struct {
	Blocked      bool
	BlockingDeps []string // refs that are blocking
	Reason       string   // "deps_not_ready", "no_deps", "deps_ready"
}

// IsWorkshopReady returns true if a dependency status indicates its plan
// is sufficiently developed to not block downstream workshops.
func IsWorkshopReady(status string) bool {
	switch status {
	case "backlog", "researching":
		return false
	default:
		return true
	}
}

// CheckWorkshopDependencies decides whether an item's workshop should be
// blocked based on its dependency statuses. Dependencies that could not
// be loaded (Found=false) are treated as non-blocking (fail-open).
func CheckWorkshopDependencies(deps []DependencyStatus) WorkshopBlockResult {
	if len(deps) == 0 {
		return WorkshopBlockResult{Blocked: false, Reason: "no_deps"}
	}
	var blocking []string
	for _, d := range deps {
		if !d.Found {
			continue
		}
		if !IsWorkshopReady(d.Status) {
			blocking = append(blocking, d.Ref)
		}
	}
	if len(blocking) > 0 {
		return WorkshopBlockResult{Blocked: true, BlockingDeps: blocking, Reason: "deps_not_ready"}
	}
	return WorkshopBlockResult{Blocked: false, Reason: "deps_ready"}
}
