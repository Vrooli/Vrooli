package aisearch

import "strings"

// relevance.go owns the regime-aware "weak match" decision boundary. The
// numeric meaning of a hit's score depends on *which scoring regime produced
// it* — a cross-encoder sigmoid, a dense cosine similarity, and an LLM 0..1
// relevance judge occupy different bands — so a single fixed weak threshold is
// fragile. The active regime is detectable at runtime: it is the reranker leg
// the SearchResponse already reports (Reranker == "cross-encoder:…" / "llm:…" /
// "none"). LabelWeak reads that leg and picks the right band, so every adopter
// gets correct weak-labeling with no per-scenario configuration.
//
// This is a named, single-home decision boundary (see docs/internal SEAMS /
// decision-points): "is this hit a weak match?" is decided exactly here.

// scoreRegime identifies the scoring regime that produced a hit's score.
type scoreRegime int

const (
	// regimeCosine is dense cosine similarity — the rerank-off DENSE path, and the
	// safe fallback for any unrecognized leg.
	regimeCosine scoreRegime = iota
	// regimeCrossEncoder is the cross-encoder relevance score (sigmoid 0..1):
	// junk collapses toward 0, direct answers sit high.
	regimeCrossEncoder
	// regimeLLM is the LLM listwise relevance judge (0..1).
	regimeLLM
	// regimeFusion is a server-side RRF (reciprocal-rank-fusion) score — the
	// rerank-off HYBRID path. It is an uncalibrated ranking signal, not a 0..1
	// relevance probability, so it occupies its own band: the cosine cutoffs (a
	// 0.55 weak line, a 0.35 hard floor) would wrongly judge real fused hits weak
	// or annihilate them. Distinguished from cosine by the retrieval method, not
	// the leg name (both report Reranker=="none"), so it needs regimeFor.
	regimeFusion
)

// Regime-keyed weak thresholds. A hit is "weak" when its score is below the
// threshold for the active leg's regime. These are package constants, not
// per-scenario levers (the regime is auto-detected); a consumer that genuinely
// needs to override does so via WeakThresholdOverride on its config, but the
// out-of-the-box values are correct for the live cli-health corpus and were
// chosen from the Track-C eval matrix (see docs/reference/configuration.md).
const (
	// WeakThresholdCosine is the legacy dense-cosine cutoff, preserved for the
	// rerank-off path (gibberish 0.50–0.55 overlaps weak-but-real 0.54–0.70 in
	// the dense band, so 0.55 is the practical "uncertain" line).
	WeakThresholdCosine = 0.55
	// WeakThresholdCrossEncoder: the cross-encoder drives junk to ~0 and real
	// answers high, so anything it scores below ~0.3 is genuinely uncertain.
	WeakThresholdCrossEncoder = 0.30
	// WeakThresholdLLM: the LLM judge emits 0 (irrelevant) .. 1 (directly
	// answers), so the midpoint is the natural weak/strong divide.
	WeakThresholdLLM = 0.50
	// WeakThresholdFusion: an RRF fused score is an uncalibrated *ranking* signal,
	// so an absolute weak line is inherently approximate here — the relative MaxGap
	// floor (FusionRelevanceMaxGap) is the primary tail gate. This line is
	// deliberately conservative: real fused hits on the live KO docs corpus sit
	// ≥0.35, so 0.20 keeps genuine answers strong while still flagging the deep
	// near-zero tail a fused query always returns. Tune from a fusion gibberish A/B
	// (search-hub eval) if a fused adopter surfaces the weak flag.
	WeakThresholdFusion = 0.20
)

// regimeFor determines the scoring regime from the retrieval method
// ("dense"/"hybrid"/"text") and the active reranker leg. A reranker, when it
// ran, dictates the regime regardless of method (it rescored every hit). With no
// reranker the method decides: an RRF-fused hybrid leg is its own band (fusion),
// while a dense/text leg is cosine. This is the single home for "which scoring
// regime produced these hits?" — both the floor and the weak-label read it.
func regimeFor(method, leg string) scoreRegime {
	switch {
	case strings.HasPrefix(leg, "cross-encoder:"):
		return regimeCrossEncoder
	case strings.HasPrefix(leg, "llm:"):
		return regimeLLM
	case method == "hybrid":
		return regimeFusion
	default:
		return regimeCosine
	}
}

// regimeForLeg is the method-agnostic regime lookup (a reranker leg → its
// regime, anything else → cosine). Prefer regimeFor when the retrieval method is
// known, so a rerank-off RRF-fused leg is classified as fusion rather than
// cosine. Retained for callers (and back-compat) that only have a leg name.
func regimeForLeg(leg string) scoreRegime {
	return regimeFor("", leg)
}

func weakThresholdForRegime(r scoreRegime) float64 {
	switch r {
	case regimeCrossEncoder:
		return WeakThresholdCrossEncoder
	case regimeLLM:
		return WeakThresholdLLM
	case regimeFusion:
		return WeakThresholdFusion
	default:
		return WeakThresholdCosine
	}
}

// WeakThreshold returns the score below which a hit is a weak match, for the
// regime of the given reranker leg (method-agnostic). Pure, no I/O. Prefer
// WeakThresholdForMethod when the retrieval method is known.
func WeakThreshold(leg string) float64 {
	return weakThresholdForRegime(regimeForLeg(leg))
}

// WeakThresholdForMethod is the method-aware weak threshold: it classifies a
// rerank-off RRF-fused leg (method "hybrid") into the fusion band instead of the
// cosine band, so a fused adopter does not mislabel real hits weak.
func WeakThresholdForMethod(method, leg string) float64 {
	return weakThresholdForRegime(regimeFor(method, leg))
}

// LabelWeak reports whether a hit with the given score, produced by the given
// reranker leg, is a weak match (method-agnostic). This is the one place the
// "weak vs strong" decision lives; the service computes it once and carries the
// bool to every consumer so CLI and UI never re-derive (and never drift on) a
// threshold. Prefer LabelWeakForMethod when the retrieval method is known.
func LabelWeak(leg string, score float64) bool {
	return score < WeakThreshold(leg)
}

// LabelWeakForMethod is the method-aware weak label; the service uses it so the
// rerank-off hybrid path is judged on the fusion band, not the cosine band.
func LabelWeakForMethod(method, leg string, score float64) bool {
	return score < WeakThresholdForMethod(method, leg)
}
