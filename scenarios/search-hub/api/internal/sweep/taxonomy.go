package sweep

import (
	"fmt"

	aisearch "github.com/vrooli/ai-go/search"
)

// taxonomy.go enumerates the candidate configurations ("arms") the sweep tries.
// It reads the factor TIERS from aisearch-go (the SSOT) — query-time factors are
// swept FULL-FACTORIAL (cheap, per-request overrides, no reindex); index-time
// factors are swept by COORDINATE-ASCENT (one factor moved at a time, to bound
// the reindex cost). Both build on the incumbent: every arm differs from the
// provider's current tuning only in the factors being swept, so a result is
// attributable.
//
// The candidate VALUE sets here are intentionally narrower than the taxonomy's
// legal domain: the taxonomy says what is *legal*, this says what is *worth
// trying by default*. The booleans go full-factorial; the fine-tuning int/float
// knobs (rerank_shortlist, the floor band) are pinned to the incumbent — moving
// them is a follow-up that needs only an extra candidate value, and pinning them
// is REPORTED (Result.note), never a silent cap.

// queryTimeArms returns the full-factorial query-time candidates built on the
// incumbent, EXCLUDING the incumbent itself (the orchestrator evaluates the
// incumbent separately as the baseline). Invalid combinations (blend without
// rerank — TuningConfig.Validate rejects it) are pruned. Hybrid engines also
// enumerate both legal dense+sparse fusion strategies. Results are de-duped by
// their canonical tuning so a pinned-knob collision can never double-run an arm.
func queryTimeArms(incumbent aisearch.TuningConfig) []aisearch.TuningConfig {
	incumbent = incumbent.WithDefaults()
	var raw []aisearch.TuningConfig
	fusions := []string{incumbent.HybridFusion}
	if incumbent.Engine == aisearch.EngineHybrid {
		fusions = []string{aisearch.HybridFusionRRF, aisearch.HybridFusionDBSF}
	}
	for _, fusion := range fusions {
		for _, rerank := range []bool{false, true} {
			for _, blend := range []bool{false, true} {
				t := incumbent // copy: keeps index-time + pinned fields fixed
				t.HybridFusion = fusion
				t.RerankEnabled = rerank
				t.RerankBlend = blend
				if t.Validate() != nil { // prunes blend && !rerank
					continue
				}
				raw = append(raw, t)
			}
		}
	}
	return dedupExcluding(raw, incumbent)
}

// indexTimeArms returns the coordinate-ascent index-time candidates built on the
// incumbent (one index-time factor moved per arm), plus the count of
// full-factorial INTERACTIONS deliberately NOT explored (so the orchestrator can
// report it — no silent cap). embed_model is pinned (only the installed model is
// safe to embed with), so the explored grid is engine × embed_task_prefix.
func indexTimeArms(incumbent aisearch.TuningConfig) (arms []aisearch.TuningConfig, droppedInteractions int) {
	incumbent = incumbent.WithDefaults()

	for _, engine := range []string{aisearch.EngineDense, aisearch.EngineHybrid} {
		if engine == incumbent.Engine {
			continue
		}
		t := incumbent
		t.Engine = engine
		if t.Validate() == nil {
			arms = append(arms, t)
		}
	}
	for _, prefix := range []bool{false, true} {
		if prefix == incumbent.EmbedTaskPrefix {
			continue
		}
		t := incumbent
		t.EmbedTaskPrefix = prefix
		if t.Validate() == nil {
			arms = append(arms, t)
		}
	}

	// Full-factorial index-time grid = engine(2) × embed_model(1, pinned) ×
	// embed_task_prefix(2) = 4. Coordinate-ascent explores the incumbent + the
	// single-factor moves; the remainder are the un-explored interactions.
	const fullFactorial = 2 * 1 * 2
	explored := len(arms) + 1 // + the incumbent
	if droppedInteractions = fullFactorial - explored; droppedInteractions < 0 {
		droppedInteractions = 0
	}
	return arms, droppedInteractions
}

// queryTimeOverrides projects a tuning's QUERY-TIME factors into the per-request
// override carrier. Every query-time field is set explicitly (not just deltas)
// so the arm fully determines the provider's query-time behavior regardless of
// its live config.
func queryTimeOverrides(t aisearch.TuningConfig) aisearch.SearchOverrides {
	return aisearch.SearchOverrides{
		RerankEnabled:   aisearch.OverrideBool(t.RerankEnabled),
		RerankBlend:     aisearch.OverrideBool(t.RerankBlend),
		RerankShortlist: aisearch.OverrideInt(t.RerankShortlist),
		FloorMaxGap:     aisearch.OverrideFloat(t.Floor.MaxGap),
		FloorHardFloor:  aisearch.OverrideFloat(t.Floor.HardFloor),
		HybridFusion:    aisearch.OverrideString(t.HybridFusion),
	}
}

// floorRegime labels which score regime the relevance floor applies in for a
// tuning: a fused regime (RRF — hybrid engine or rerank-blend; floor near 0) or
// a single-leg cosine regime (a real floor). It is descriptive metadata stamped
// onto each arm's ConfigSnapshot so a stored run records the floor band in force.
func floorRegime(t aisearch.TuningConfig) string {
	if t.Engine == aisearch.EngineHybrid || t.RerankBlend {
		return "fused"
	}
	return "cosine"
}

// canonical is the de-dup / tag key for a tuning: every factor, so two arms are
// "the same arm" iff every swept and pinned factor matches.
func canonical(t aisearch.TuningConfig) string {
	t = t.WithDefaults()
	return fmt.Sprintf("engine=%s,embed_model=%s,embed_task_prefix=%t,rerank_enabled=%t,rerank_blend=%t,shortlist=%d,rerank_preference=%s,floor=%g/%g,hybrid_fusion=%s",
		t.Engine, t.EmbedModel, t.EmbedTaskPrefix, t.RerankEnabled, t.RerankBlend, t.RerankShortlist, t.RerankPreference, t.Floor.MaxGap, t.Floor.HardFloor, t.HybridFusion)
}

// armTag is the stable, human-legible run tag for an arm in a tier.
func armTag(tier string, t aisearch.TuningConfig) string {
	return fmt.Sprintf("sweep:%s:%s", tier, canonical(t))
}

// dedupExcluding removes duplicates (by canonical) and any tuning equal to
// exclude, preserving first-seen order.
func dedupExcluding(in []aisearch.TuningConfig, exclude aisearch.TuningConfig) []aisearch.TuningConfig {
	seen := map[string]bool{canonical(exclude): true}
	out := make([]aisearch.TuningConfig, 0, len(in))
	for _, t := range in {
		k := canonical(t)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t)
	}
	return out
}
