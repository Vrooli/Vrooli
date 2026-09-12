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

// FloorForMethodLeg returns the regime-appropriate FloorConfig for the given
// retrieval method + active reranker leg, with any explicitly-set field of
// override winning. The regime is auto-detected (cross-encoder / llm / fusion /
// cosine), so an adopter gets a correct floor with no per-scenario tuning; an
// operator who sets <PREFIX>_RELEVANCE_MAX_GAP / _RELEVANCE_HARD_FLOOR still
// overrides the regime default. Override precedence: MaxGap > 0 and
// HardFloor != 0 are treated as "set" (HardFloor is set to a negative value to
// deliberately disable it).
//
// The fusion band (rerank-off hybrid) disables the absolute HardFloor and keeps
// only the relative MaxGap, which is why ServiceOptions.ApplyFloor is now safe to
// leave on for a fused/doc adopter: the cosine 0.35 HardFloor that used to
// annihilate real RRF hits no longer applies to them.
//
// This is the single home for "which floor band applies to these scores?" — the
// floor *math* lives in ApplyRelevanceFloor; the *policy* of which band lives here.
func FloorForMethodLeg(method, leg string, override FloorConfig) FloorConfig {
	base := regimeFloor(regimeFor(method, leg))
	if override.MaxGap > 0 {
		base.MaxGap = override.MaxGap
	}
	if override.HardFloor != 0 {
		base.HardFloor = override.HardFloor
	}
	return base
}

// regimeFloor is the regime→FloorConfig table.
func regimeFloor(r scoreRegime) FloorConfig {
	switch r {
	case regimeCrossEncoder:
		return FloorConfig{MaxGap: CrossEncoderRelevanceMaxGap, HardFloor: CrossEncoderRelevanceHardFloor}
	case regimeLLM:
		return FloorConfig{MaxGap: LLMRelevanceMaxGap, HardFloor: LLMRelevanceHardFloor}
	case regimeFusion:
		return FloorConfig{MaxGap: FusionRelevanceMaxGap, HardFloor: FusionRelevanceHardFloor}
	default:
		return FloorConfig{MaxGap: DefaultRelevanceMaxGap, HardFloor: DefaultRelevanceHardFloor}
	}
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
