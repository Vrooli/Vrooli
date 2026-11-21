package sweep

import (
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
)

func TestQueryTimeArms_FullFactorialPrunedAndDeduped(t *testing.T) {
	// Incumbent: dense, rerank off. The full-factorial over (rerank_enabled,
	// rerank_blend) is 4; blend&&!rerank is pruned → {off/off, on/off, on/on};
	// off/off equals the incumbent and is excluded → 2 arms.
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	arms := queryTimeArms(inc)
	if len(arms) != 2 {
		t.Fatalf("query-time arms = %d, want 2; got %v", len(arms), tags(arms))
	}
	// None may be the invalid blend-without-rerank combo, and none may equal inc.
	for _, a := range arms {
		if a.RerankBlend && !a.RerankEnabled {
			t.Fatalf("invalid arm leaked: blend without rerank")
		}
		if canonical(a) == canonical(inc) {
			t.Fatalf("incumbent must be excluded from the arm set")
		}
		// Index-time fields must be untouched by the query-time tier.
		if a.Engine != inc.Engine || a.EmbedTaskPrefix != inc.EmbedTaskPrefix {
			t.Fatalf("query-time arm changed an index-time factor: %+v", a)
		}
	}
}

func TestQueryTimeArms_HybridEnumeratesFusionStrategies(t *testing.T) {
	inc := aisearch.TuningConfig{Engine: aisearch.EngineHybrid}.WithDefaults()
	ars := queryTimeArms(inc)
	if len(ars) != 5 {
		t.Fatalf("hybrid query-time arms = %d, want 5; got %v", len(ars), tags(ars))
	}
	seen := map[string]bool{}
	for _, arm := range ars {
		seen[arm.HybridFusion] = true
	}
	if !seen[aisearch.HybridFusionRRF] || !seen[aisearch.HybridFusionDBSF] {
		t.Fatalf("hybrid query-time arms omitted a fusion strategy: %v", seen)
	}
}

func TestIndexTimeArms_CoordinateAscentAndDropped(t *testing.T) {
	// Incumbent: dense, no task prefix. Coordinate-ascent moves ONE index-time
	// factor at a time → {hybrid}, {prefix=true} = 2 arms. The full index-time
	// grid is engine(2)×prefix(2)=4; explored = 2 arms + incumbent = 3; so 1
	// interaction (hybrid AND prefix) is deliberately dropped.
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense, EmbedTaskPrefix: false}.WithDefaults()
	arms, dropped := indexTimeArms(inc)
	if len(arms) != 2 {
		t.Fatalf("index-time arms = %d, want 2; got %v", len(arms), tags(arms))
	}
	if dropped != 1 {
		t.Fatalf("dropped interactions = %d, want 1", dropped)
	}
	// Each arm differs from the incumbent in exactly one index-time factor.
	for _, a := range arms {
		diffs := 0
		if a.Engine != inc.Engine {
			diffs++
		}
		if a.EmbedTaskPrefix != inc.EmbedTaskPrefix {
			diffs++
		}
		if diffs != 1 {
			t.Fatalf("coordinate-ascent arm changed %d factors, want 1: %+v", diffs, a)
		}
	}
}

func TestQueryTimeOverrides_ProjectsAllFields(t *testing.T) {
	t0 := aisearch.TuningConfig{
		Engine:          aisearch.EngineDense,
		RerankEnabled:   true,
		RerankBlend:     true,
		RerankShortlist: 42,
		Floor:           aisearch.FloorTuning{MaxGap: 0.1, HardFloor: 0.2},
		HybridFusion:    aisearch.HybridFusionDBSF,
	}
	ov := queryTimeOverrides(t0)
	if ov.RerankEnabled == nil || !*ov.RerankEnabled {
		t.Fatalf("rerank_enabled not projected")
	}
	if ov.RerankBlend == nil || !*ov.RerankBlend {
		t.Fatalf("rerank_blend not projected")
	}
	if ov.RerankShortlist == nil || *ov.RerankShortlist != 42 {
		t.Fatalf("rerank_shortlist not projected")
	}
	if ov.FloorMaxGap == nil || *ov.FloorMaxGap != 0.1 {
		t.Fatalf("floor_max_gap not projected")
	}
	if ov.FloorHardFloor == nil || *ov.FloorHardFloor != 0.2 {
		t.Fatalf("floor_hard_floor not projected")
	}
	if ov.HybridFusion == nil || *ov.HybridFusion != aisearch.HybridFusionDBSF {
		t.Fatalf("hybrid_fusion not projected")
	}
}

func TestFloorRegime(t *testing.T) {
	if floorRegime(aisearch.TuningConfig{Engine: aisearch.EngineDense}) != "cosine" {
		t.Fatalf("dense, no blend → cosine")
	}
	if floorRegime(aisearch.TuningConfig{Engine: aisearch.EngineHybrid}) != "fused" {
		t.Fatalf("hybrid → fused")
	}
	if floorRegime(aisearch.TuningConfig{Engine: aisearch.EngineDense, RerankEnabled: true, RerankBlend: true}) != "fused" {
		t.Fatalf("rerank-blend → fused")
	}
}

func tags(arms []aisearch.TuningConfig) []string {
	out := make([]string, len(arms))
	for i, a := range arms {
		out[i] = canonical(a)
	}
	return out
}
