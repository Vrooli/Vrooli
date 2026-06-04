package aisearch

import "testing"

func ids(results []SearchResult) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.ID
	}
	return out
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
