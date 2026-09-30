package convergence

// deriveTier grades a template's four-lens fitness into a coarse advisory tier.
// The thresholds come from REFERENCE_PATTERN_FITNESS.md's rules of thumb:
// per-replica infrastructure > ~20 lines is a candidate; coordinated edits > 5
// is a finding; comment-only contracts and drift surfaces are debt at scale. The
// tier is ADVISORY ONLY — the team decides substrate/nomination; this just ranks.
func deriveTier(tf TemplateFitness) FitnessTier {
	penalty := 0
	if tf.CoordinatedEditCount > 5 {
		penalty += tf.CoordinatedEditCount - 5
	}
	penalty += tf.CommentOnlyContractCount
	penalty += tf.DriftSurfaceCount
	// Per-replica cost contributes mildly (every ~500 LOC of shipped template is
	// one penalty point); it is the least decisive lens on its own.
	penalty += tf.PerReplicaCost / 500

	switch {
	case penalty <= 5:
		return TierStrong
	case penalty <= 15:
		return TierFair
	default:
		return TierWeak
	}
}

// deriveEligibility grades a reference's gold-star eligibility from its health
// signals (REFERENCE_SCENARIOS.md promotion rules): clean on all tools, ≥60d
// stability, not stale-from-template, and reasonable breadth.
func deriveEligibility(h ReferenceHealth) ReferenceEligibility {
	switch {
	case h.CleanOnAllTools && h.StabilityDays >= 60 && !h.StaleFromTemplate && h.Breadth >= 3:
		return EligibilityEligible
	case h.StabilityDays >= 60 && h.Breadth >= 2 && !h.StaleFromTemplate:
		return EligibilityCandidate
	default:
		return EligibilityIneligible
	}
}
