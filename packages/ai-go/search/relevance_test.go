package aisearch

import "testing"

func TestWeakThresholdPerRegime(t *testing.T) {
	cases := []struct {
		leg  string
		want float64
	}{
		{"cross-encoder:bge-reranker-v2-m3", WeakThresholdCrossEncoder},
		{"llm:rerank.llm_fallback", WeakThresholdLLM},
		{"none", WeakThresholdCosine},
		{"", WeakThresholdCosine},
		{"dense", WeakThresholdCosine},
		{"chain[cross-encoder:x>llm:y]", WeakThresholdCosine}, // chain Name(), not a leg → cosine
		{"something-unknown", WeakThresholdCosine},
	}
	for _, tc := range cases {
		if got := WeakThresholdForMethod("", tc.leg); got != tc.want {
			t.Errorf("WeakThresholdForMethod(%q,%q) = %g, want %g", "", tc.leg, got, tc.want)
		}
	}
}

func TestRegimeForMethodBlend(t *testing.T) {
	cases := []struct {
		name   string
		method string
		leg    string
		want   string
	}{
		// The blended leg embeds its inner reranker name; it must still resolve to
		// the fusion band (RRF rank-fusion scores), not the inner regime.
		{"blend over cross-encoder is fused", "dense", "blend:cross-encoder:bge-reranker-v2-m3", "fused"},
		{"blend over llm is fused", "dense", "blend:llm:judge", "fused"},
		// Unblended legs keep their own regime.
		{"pure cross-encoder", "dense", "cross-encoder:m", "cross-encoder"},
		{"pure llm", "dense", "llm:q", "llm"},
		{"rerank-off hybrid is fused", "hybrid", "none", "fused"},
		{"dense none is cosine", "dense", "none", "cosine"},
		{"text is cosine", "text", "", "cosine"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RegimeForMethod(tc.method, tc.leg); got != tc.want {
				t.Fatalf("RegimeForMethod(%q,%q) = %q, want %q", tc.method, tc.leg, got, tc.want)
			}
		})
	}
}

func TestWeakThresholdForMethod(t *testing.T) {
	cases := []struct {
		name   string
		method string
		leg    string
		want   float64
	}{
		// A reranker leg wins regardless of method (it rescored the hits).
		{"hybrid+cross-encoder uses xenc band", "hybrid", "cross-encoder:m", WeakThresholdCrossEncoder},
		{"hybrid+llm uses llm band", "hybrid", "llm:q", WeakThresholdLLM},
		// The differentiator: rerank-OFF hybrid is fusion, not cosine.
		{"hybrid+none is fusion", "hybrid", "none", WeakThresholdFusion},
		{"hybrid empty leg is fusion", "hybrid", "", WeakThresholdFusion},
		// Dense / text rerank-off stay cosine.
		{"dense+none is cosine", "dense", "none", WeakThresholdCosine},
		{"text leg is cosine", "text", "", WeakThresholdCosine},
		// Leg-only back-compat: no method => fused leg would look like cosine, which
		// is exactly why the service must pass the method.
		{"empty method+none is cosine", "", "none", WeakThresholdCosine},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WeakThresholdForMethod(tc.method, tc.leg); got != tc.want {
				t.Fatalf("WeakThresholdForMethod(%q,%q) = %g, want %g", tc.method, tc.leg, got, tc.want)
			}
		})
	}
}

func TestLabelWeakForMethodFusion(t *testing.T) {
	// A real fused doc hit (≥0.35 on the live KO corpus) must NOT be weak under the
	// fusion band, even though it sits below the cosine 0.55 line — this is the
	// latent bug the fusion regime fixes.
	if LabelWeakForMethod("hybrid", "none", 0.35) {
		t.Errorf("real fused hit 0.35 wrongly labeled weak under fusion band")
	}
	if !LabelWeakForMethod("dense", "none", 0.35) {
		t.Errorf("sanity: 0.35 SHOULD be weak under the legacy cosine band (0.55)")
	}
	// The deep near-zero tail a fused query always returns is still flagged weak.
	if !LabelWeakForMethod("hybrid", "none", 0.05) {
		t.Errorf("deep fused tail 0.05 should be weak")
	}
	if LabelWeakForMethod("hybrid", "none", WeakThresholdFusion) {
		t.Errorf("boundary is exclusive: == threshold is not weak")
	}
}

func TestLabelWeakForMethodBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		leg   string
		score float64
		want  bool
	}{
		// Cross-encoder: junk near 0 is weak, a high score is strong; the boundary
		// is exclusive (== threshold is NOT weak).
		{"xenc gibberish weak", "cross-encoder:m", 0.001, true},
		{"xenc just below", "cross-encoder:m", 0.29, true},
		{"xenc at boundary strong", "cross-encoder:m", WeakThresholdCrossEncoder, false},
		{"xenc strong", "cross-encoder:m", 0.92, false},
		// LLM 0..1 judge.
		{"llm low weak", "llm:rerank.llm_fallback", 0.40, true},
		{"llm at boundary strong", "llm:rerank.llm_fallback", WeakThresholdLLM, false},
		{"llm high strong", "llm:rerank.llm_fallback", 0.85, false},
		// Cosine (rerank-off) keeps the legacy 0.55 line.
		{"cosine weak-real-ish weak", "none", 0.54, true},
		{"cosine at boundary strong", "none", WeakThresholdCosine, false},
		{"cosine strong", "none", 0.80, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LabelWeakForMethod("", tc.leg, tc.score); got != tc.want {
				t.Fatalf("LabelWeakForMethod(%q, %q, %g) = %v, want %v", "", tc.leg, tc.score, got, tc.want)
			}
		})
	}
}
