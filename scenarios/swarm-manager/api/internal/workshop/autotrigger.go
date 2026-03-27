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
