package sweep

import (
	"math/rand"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// --- Guard 1: paired bootstrap CI ------------------------------------------

func TestPairedMarginCI_SignificantWinExcludesZero(t *testing.T) {
	// Every case improves by +1 → the entire bootstrap distribution is +1.
	winner := []float64{1, 1, 1, 1, 1, 1}
	incumbent := []float64{0, 0, 0, 0, 0, 0}
	mean, lo, hi := pairedMarginCI(winner, incumbent, 1000, rand.New(rand.NewSource(7)))
	if mean != 1 || lo <= 0 || hi != 1 {
		t.Fatalf("clear win: mean=%v lo=%v hi=%v; want mean=1, lo>0, hi=1", mean, lo, hi)
	}
}

func TestPairedMarginCI_NoiseOverlapsZero(t *testing.T) {
	// One of six cases improves; the rest tie → resamples frequently draw all
	// zeros, so the lower bound sits at 0 (not significant).
	winner := []float64{1, 0, 0, 0, 0, 0}
	incumbent := []float64{0, 0, 0, 0, 0, 0}
	mean, lo, _ := pairedMarginCI(winner, incumbent, 5000, rand.New(rand.NewSource(7)))
	if mean <= 0 {
		t.Fatalf("expected a positive point estimate, got %v", mean)
	}
	if lo > 0 {
		t.Fatalf("within-noise candidate must NOT be significant: lo=%v (want ≤ 0)", lo)
	}
}

func TestPairedMarginCI_EmptyOrMismatched(t *testing.T) {
	if m, lo, hi := pairedMarginCI(nil, nil, 100, nil); m != 0 || lo != 0 || hi != 0 {
		t.Fatalf("empty vectors must be non-significant zero, got %v/%v/%v", m, lo, hi)
	}
	if _, lo, _ := pairedMarginCI([]float64{1}, []float64{0, 0}, 100, nil); lo != 0 {
		t.Fatalf("mismatched lengths must be non-significant")
	}
}

// --- Guard 2: held-out split -----------------------------------------------

func TestSplitCases_Deterministic(t *testing.T) {
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	tun1, held1 := splitCases(ids, nil, 0.3)
	tun2, held2 := splitCases(ids, nil, 0.3)
	if len(held1) != 3 { // ceil(0.3*10)
		t.Fatalf("held-out size = %d, want 3", len(held1))
	}
	if len(tun1)+len(held1) != len(ids) {
		t.Fatalf("split lost cases: %d + %d != %d", len(tun1), len(held1), len(ids))
	}
	if !equalStrings(tun1, tun2) || !equalStrings(held1, held2) {
		t.Fatalf("split is not deterministic: %v/%v vs %v/%v", tun1, held1, tun2, held2)
	}
}

func TestSplitCases_GeneratedAlwaysHeldOut(t *testing.T) {
	ids := []string{"a", "b", "c", "d"}
	gen := map[string]bool{"a": true, "b": true}
	tuning, heldout := splitCases(ids, gen, 0.0) // fraction 0: only generated held out
	for _, id := range tuning {
		if gen[id] {
			t.Fatalf("generated case %q leaked into the tuning fold", id)
		}
	}
	if !contains(heldout, "a") || !contains(heldout, "b") {
		t.Fatalf("generated cases must be held out, got heldout=%v", heldout)
	}
}

func TestHeldoutHolds(t *testing.T) {
	heldout := []string{"a", "b", "c"}
	cand := map[string]float64{"a": 1, "b": 1, "c": 1}
	inc := map[string]float64{"a": 1, "b": 0, "c": 1}
	if ok, _ := heldoutHolds(cand, inc, heldout, 3); !ok {
		t.Fatalf("candidate that improves held-out should hold")
	}
	// Regression on held-out blocks.
	bad := map[string]float64{"a": 0, "b": 0, "c": 1}
	if ok, reason := heldoutHolds(bad, inc, heldout, 3); ok || reason == "" {
		t.Fatalf("held-out regression must block, got ok=%v reason=%q", ok, reason)
	}
	// Fold too small to validate blocks (conservative).
	if ok, reason := heldoutHolds(cand, inc, heldout, 5); ok || reason == "" {
		t.Fatalf("too-small fold must block, got ok=%v reason=%q", ok, reason)
	}
}

// --- Guard 3: multi-objective constraints ----------------------------------

func TestConstraintsFeasible(t *testing.T) {
	c := Constraints{MaxGibberish: 0.4, MaxP95Ms: 1000}
	ok, _ := c.feasible(&evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{MaxGibberishScore: 0.3, LatencyP95Ms: 500}})
	if !ok {
		t.Fatalf("within bounds must be feasible")
	}
	if ok, reason := c.feasible(&evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{MaxGibberishScore: 0.9}}); ok || reason == "" {
		t.Fatalf("gibberish over ceiling must be infeasible")
	}
	if ok, reason := c.feasible(&evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{LatencyP95Ms: 5000}}); ok || reason == "" {
		t.Fatalf("p95 over budget must be infeasible")
	}
	// p95 budget of 0 disables the latency constraint.
	cNoBudget := Constraints{MaxGibberish: 1, MaxP95Ms: 0}
	if ok, _ := cNoBudget.feasible(&evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{LatencyP95Ms: 99999}}); !ok {
		t.Fatalf("zero budget must disable the latency constraint")
	}
}

func TestDeriveConstraints_AnchoredToIncumbent(t *testing.T) {
	inc := &evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{MaxGibberishScore: 0.6, LatencyP95Ms: 200}}
	c := deriveConstraints(inc, 0.5, 3.0)
	if c.MaxGibberish != 0.6 { // incumbent gibberish above the absolute floor
		t.Fatalf("ceiling should track incumbent (0.6), got %v", c.MaxGibberish)
	}
	if c.MaxP95Ms != 600 { // 200 * 3
		t.Fatalf("p95 budget = %d, want 600", c.MaxP95Ms)
	}
	// When incumbent gibberish is below the absolute floor, the floor wins.
	inc2 := &evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{MaxGibberishScore: 0.1}}
	if c2 := deriveConstraints(inc2, 0.5, 3.0); c2.MaxGibberish != 0.5 {
		t.Fatalf("absolute floor should win, got %v", c2.MaxGibberish)
	}
}

// --- Guard 4: complexity tie-break -----------------------------------------

func TestComplexityOrdering(t *testing.T) {
	dense := aisearch.TuningConfig{Engine: aisearch.EngineDense}
	hybrid := aisearch.TuningConfig{Engine: aisearch.EngineHybrid}
	denseRerank := aisearch.TuningConfig{Engine: aisearch.EngineDense, RerankEnabled: true}
	if !(complexity(dense) < complexity(hybrid)) {
		t.Fatalf("dense must be simpler than hybrid")
	}
	if !(complexity(dense) < complexity(denseRerank)) {
		t.Fatalf("rerank-off must be simpler than rerank-on")
	}
	blend := aisearch.TuningConfig{Engine: aisearch.EngineDense, RerankEnabled: true, RerankBlend: true}
	if !(complexity(denseRerank) < complexity(blend)) {
		t.Fatalf("no-blend must be simpler than blend")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
