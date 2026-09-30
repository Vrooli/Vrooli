package aisearch

import "testing"

func ids(results []SearchResult) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.ID
	}
	return out
}

func TestFloorForMethodLegOverrides(t *testing.T) {
	t.Run("cross-encoder regime default", func(t *testing.T) {
		got := FloorForMethodLeg("", "cross-encoder:bge-reranker-v2-m3", FloorConfig{})
		if got.MaxGap != CrossEncoderRelevanceMaxGap || got.HardFloor != CrossEncoderRelevanceHardFloor {
			t.Fatalf("got %+v, want cross-encoder regime floor", got)
		}
	})
	t.Run("llm regime default", func(t *testing.T) {
		got := FloorForMethodLeg("", "llm:rerank.llm_fallback", FloorConfig{})
		if got.MaxGap != LLMRelevanceMaxGap || got.HardFloor != LLMRelevanceHardFloor {
			t.Fatalf("got %+v, want llm regime floor", got)
		}
	})
	t.Run("unknown leg falls back to cosine", func(t *testing.T) {
		got := FloorForMethodLeg("", "none", FloorConfig{})
		if got.MaxGap != DefaultRelevanceMaxGap || got.HardFloor != DefaultRelevanceHardFloor {
			t.Fatalf("got %+v, want cosine regime floor", got)
		}
	})
	t.Run("override wins per field", func(t *testing.T) {
		// MaxGap overridden, HardFloor left 0 (unset) → regime HardFloor kept.
		got := FloorForMethodLeg("", "cross-encoder:m", FloorConfig{MaxGap: 0.42})
		if got.MaxGap != 0.42 {
			t.Errorf("MaxGap = %g, want override 0.42", got.MaxGap)
		}
		if got.HardFloor != CrossEncoderRelevanceHardFloor {
			t.Errorf("HardFloor = %g, want regime default (unset override)", got.HardFloor)
		}
	})
	t.Run("negative hard floor override disables", func(t *testing.T) {
		got := FloorForMethodLeg("", "cross-encoder:m", FloorConfig{HardFloor: -1})
		if got.HardFloor != -1 {
			t.Errorf("HardFloor = %g, want -1 (operator-disabled)", got.HardFloor)
		}
	})
}

func TestFloorForMethodLeg(t *testing.T) {
	t.Run("rerank-off hybrid is the fusion band (no hard floor)", func(t *testing.T) {
		got := FloorForMethodLeg("hybrid", "none", FloorConfig{})
		if got.MaxGap != FusionRelevanceMaxGap || got.HardFloor != FusionRelevanceHardFloor {
			t.Fatalf("got %+v, want fusion regime floor (MaxGap=%g HardFloor=%g)", got, FusionRelevanceMaxGap, FusionRelevanceHardFloor)
		}
		if got.HardFloor != 0 {
			t.Fatalf("fusion HardFloor must be disabled (0), got %g", got.HardFloor)
		}
	})
	t.Run("rerank-off dense stays cosine", func(t *testing.T) {
		got := FloorForMethodLeg("dense", "none", FloorConfig{})
		if got.MaxGap != DefaultRelevanceMaxGap || got.HardFloor != DefaultRelevanceHardFloor {
			t.Fatalf("got %+v, want cosine regime floor", got)
		}
	})
	t.Run("reranker leg wins over hybrid method", func(t *testing.T) {
		got := FloorForMethodLeg("hybrid", "cross-encoder:m", FloorConfig{})
		if got.MaxGap != CrossEncoderRelevanceMaxGap || got.HardFloor != CrossEncoderRelevanceHardFloor {
			t.Fatalf("got %+v, want cross-encoder regime floor (rerank rescored the fused hits)", got)
		}
	})
}

func TestFusionFloorKeepsRealHits(t *testing.T) {
	// The end-to-end point of the fusion band: a real RRF result far below the top
	// (here 0.30 vs top 0.56) survives, where the cosine 0.35 HardFloor would have
	// annihilated it.
	in := []SearchResult{
		{ID: "top", Score: 0.56},
		{ID: "mid", Score: 0.40},
		{ID: "low-real", Score: 0.30},
	}
	fusion := FloorForMethodLeg("hybrid", "none", FloorConfig{})
	got := ids(ApplyRelevanceFloor(in, fusion))
	if len(got) != 3 {
		t.Fatalf("fusion floor must keep the real low hit, got %v", got)
	}
	// Contrast: the cosine band would drop low-real on its 0.35 hard floor.
	cosine := FloorForMethodLeg("dense", "none", FloorConfig{})
	if dropped := ids(ApplyRelevanceFloor(in, cosine)); len(dropped) == 3 {
		t.Fatalf("expected cosine band to drop the 0.30 hit, kept %v", dropped)
	}
}

func TestApplyRelevanceFloor(t *testing.T) {
	cfg := FloorConfig{MaxGap: 0.15, HardFloor: 0.35}

	t.Run("empty input safe", func(t *testing.T) {
		if got := ApplyRelevanceFloor(nil, cfg); len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})

	t.Run("single result always kept", func(t *testing.T) {
		in := []SearchResult{{ID: "a", Score: 0.10}} // below hard floor
		got := ApplyRelevanceFloor(in, cfg)
		if len(got) != 1 || got[0].ID != "a" {
			t.Fatalf("single result must be kept regardless of score, got %v", ids(got))
		}
	})

	t.Run("mid-gap dropped, top kept", func(t *testing.T) {
		in := []SearchResult{
			{ID: "top", Score: 0.80},
			{ID: "near", Score: 0.70}, // within 0.15 gap
			{ID: "far", Score: 0.50},  // gap 0.30 > 0.15 -> dropped
		}
		got := ids(ApplyRelevanceFloor(in, cfg))
		if len(got) != 2 || got[0] != "top" || got[1] != "near" {
			t.Fatalf("expected [top near], got %v", got)
		}
	})

	t.Run("all-high kept", func(t *testing.T) {
		in := []SearchResult{
			{ID: "a", Score: 0.81},
			{ID: "b", Score: 0.78},
			{ID: "c", Score: 0.75},
		}
		if got := ids(ApplyRelevanceFloor(in, cfg)); len(got) != 3 {
			t.Fatalf("tightly-banded strong hits must all survive, got %v", got)
		}
	})

	t.Run("hard floor drops weak even within gap", func(t *testing.T) {
		// Top is weak (0.40); a hit at 0.34 is within the 0.15 gap but below the
		// 0.35 hard floor -> dropped. Top kept (always).
		in := []SearchResult{
			{ID: "top", Score: 0.40},
			{ID: "weak", Score: 0.34},
		}
		got := ids(ApplyRelevanceFloor(in, cfg))
		if len(got) != 1 || got[0] != "top" {
			t.Fatalf("expected only [top], got %v", got)
		}
	})

	t.Run("default max gap applied when unset", func(t *testing.T) {
		in := []SearchResult{
			{ID: "top", Score: 0.80},
			{ID: "far", Score: 0.60}, // 0.20 gap > default 0.15
		}
		got := ids(ApplyRelevanceFloor(in, FloorConfig{}))
		if len(got) != 1 || got[0] != "top" {
			t.Fatalf("expected default gap to drop far hit, got %v", got)
		}
	})
}
