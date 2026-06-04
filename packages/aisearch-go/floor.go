package aisearch

// FloorConfig tunes ApplyRelevanceFloor. Both knobs are consumer-configurable
// (LoadConfig reads RELEVANCE_MAX_GAP / RELEVANCE_HARD_FLOOR) so the heuristic
// can be tuned per corpus without a rebuild.
type FloorConfig struct {
	// MaxGap is the relative cutoff: any hit scoring more than MaxGap below the
	// query's own best hit is dropped. This adapts to each query's score band,
	// unlike a constant floor (verified bands show weak-real 0.53–0.64 and
	// gibberish 0.52–0.54 overlap, so a fixed cut would hide correct answers).
	// Non-positive falls back to DefaultRelevanceMaxGap.
	MaxGap float64
	// HardFloor is a garbage-only safety net, not the primary gate: hits below it
	// are dropped even when within MaxGap of the top. Non-positive disables it.
	HardFloor float64
}

// ApplyRelevanceFloor drops weak/garbage hits without hiding correct answers to
// legitimately-sparse queries. It always keeps the single best hit, then drops
// any hit scoring below max(topScore-MaxGap, HardFloor). Input order is
// preserved; the slice is assumed score-ranked but correctness does not depend
// on it. Reusable, consumer-agnostic math — the calibrated long-term answer is
// the reranker (it ships first because it is dependency-free).
func ApplyRelevanceFloor(hits []SearchResult, cfg FloorConfig) []SearchResult {
	if len(hits) <= 1 {
		return hits
	}
	maxGap := cfg.MaxGap
	if maxGap <= 0 {
		maxGap = DefaultRelevanceMaxGap
	}
	hardFloor := cfg.HardFloor
	if hardFloor < 0 {
		hardFloor = 0
	}

	bestIdx := 0
	for i := range hits {
		if hits[i].Score > hits[bestIdx].Score {
			bestIdx = i
		}
	}
	cutoff := hits[bestIdx].Score - maxGap
	if hardFloor > cutoff {
		cutoff = hardFloor
	}

	out := make([]SearchResult, 0, len(hits))
	for i := range hits {
		if i == bestIdx || hits[i].Score >= cutoff {
			out = append(out, hits[i])
		}
	}
	return out
}
