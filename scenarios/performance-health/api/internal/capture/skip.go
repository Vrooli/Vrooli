package capture

import "performance-health/internal/readiness"

// skipReason reports whether the capture should be cleanly skipped before any
// restart, and why. Capture is impossible when the scenario can reach no browser
// tier or the required seams are not wired.
func skipReason(tier readiness.Tier, bas BASClient, build BuildController) (string, bool) {
	if tier == readiness.TierNone {
		return "scenario has no UI surface; browser perf is skipped", true
	}
	if bas == nil {
		return "BAS perf-capture client is not wired", true
	}
	if build == nil {
		return "profile-mode build controller is not wired", true
	}
	return "", false
}

// reachedTier is the tier actually achieved by a capture: a Tier-1-eligible
// scenario whose page emitted no ⚛ marks falls back to Tier 0 (not an error).
func reachedTier(eligible readiness.Tier, hasComponentMarks bool) readiness.Tier {
	if eligible == readiness.Tier1 && !hasComponentMarks {
		return readiness.Tier0
	}
	return eligible
}
