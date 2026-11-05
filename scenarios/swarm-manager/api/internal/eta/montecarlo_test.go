package eta

import (
	"strings"
	"testing"
)

// independentClosure builds n pending items of the given class with no
// dependencies between them.
func independentClosure(n int, class string) GoalClosureInput {
	in := GoalClosureInput{Deps: map[string][]string{}}
	for i := 0; i < n; i++ {
		ref := "task/" + string(rune('a'+i))
		in.Items = append(in.Items, ClosureItem{Ref: ref, EffortClass: class})
		in.Deps[ref] = nil
	}
	return in
}

func newSampledEstimator(t *testing.T, capacity int) *Estimator {
	t.Helper()
	// A confident live cluster for M near 48h.
	var samples []Sample
	for _, h := range []float64{44, 46, 48, 50, 52, 48, 47, 49} {
		samples = append(samples, Sample{EffortClass: "M", DurationHours: h, Origin: OriginLive})
	}
	return NewEstimator(samples, nil, capacity, DefaultTrials, DefaultSeed)
}

func TestEstimateGoal_EmptyClosureNotOK(t *testing.T) {
	e := NewEstimator(nil, nil, 1, DefaultTrials, DefaultSeed)
	if _, ok := e.EstimateGoal(GoalClosureInput{}); ok {
		t.Fatal("empty closure should return ok=false")
	}
}

func TestEstimateGoal_AllDoneIsZeroBand(t *testing.T) {
	e := NewEstimator(nil, nil, 1, DefaultTrials, DefaultSeed)
	in := GoalClosureInput{
		Items: []ClosureItem{{Ref: "task/a", EffortClass: "M", Done: true}},
		Deps:  map[string][]string{"task/a": nil},
	}
	band, ok := e.EstimateGoal(in)
	if !ok {
		t.Fatal("done closure should return ok=true")
	}
	if band.RemainingItems != 0 {
		t.Errorf("remaining = %d, want 0", band.RemainingItems)
	}
	if band.P50Hours != 0 || band.P80Hours != 0 {
		t.Errorf("done band should be zero, got p50=%v p80=%v", band.P50Hours, band.P80Hours)
	}
	if band.P50Label != "done" {
		t.Errorf("p50 label = %q, want done", band.P50Label)
	}
}

func TestEstimateGoal_P50NotAboveP80(t *testing.T) {
	e := newSampledEstimator(t, 2)
	band, ok := e.EstimateGoal(independentClosure(6, "M"))
	if !ok {
		t.Fatal("expected ok")
	}
	if band.P50Hours > band.P80Hours {
		t.Errorf("p50 %v must be <= p80 %v", band.P50Hours, band.P80Hours)
	}
}

func TestEstimateGoal_MoreCapacityShortensBand(t *testing.T) {
	// 8 independent items: throughput (sum/capacity) dominates the single-item
	// critical path, so raising capacity must shorten the band.
	closure := independentClosure(8, "M")

	narrow := newSampledEstimator(t, 1)
	wide := newSampledEstimator(t, 8)
	slow, _ := narrow.EstimateGoal(closure)
	fast, _ := wide.EstimateGoal(closure)

	if !(fast.P80Hours < slow.P80Hours) {
		t.Errorf("capacity 8 p80 %.1f must be shorter than capacity 1 p80 %.1f", fast.P80Hours, slow.P80Hours)
	}
	if fast.LaneCapacity != 8 {
		t.Errorf("lane capacity = %d, want 8", fast.LaneCapacity)
	}
}

func TestEstimateGoal_CriticalPathFloorsCapacityGain(t *testing.T) {
	// A dependency chain a→b→c: the critical path is the whole chain, so extra
	// lane capacity cannot shorten it below the chain length.
	chain := GoalClosureInput{
		Items: []ClosureItem{
			{Ref: "task/a", EffortClass: "M"},
			{Ref: "task/b", EffortClass: "M"},
			{Ref: "task/c", EffortClass: "M"},
		},
		Deps: map[string][]string{
			"task/a": nil,
			"task/b": {"task/a"},
			"task/c": {"task/b"},
		},
	}
	cap1, _ := newSampledEstimator(t, 1).EstimateGoal(chain)
	cap9, _ := newSampledEstimator(t, 9).EstimateGoal(chain)

	// With a strict chain, throughput = sum/cap; at cap 3+ the chain critical
	// path (~3×median) is the binding constraint, so cap1 and cap9 agree.
	if cap9.P50Hours != cap1.P50Hours {
		t.Errorf("chain-dominated band should not change with capacity: cap1=%.1f cap9=%.1f", cap1.P50Hours, cap9.P50Hours)
	}
	// Sanity: the chain band should be materially larger than one item.
	single, _ := newSampledEstimator(t, 1).EstimateGoal(independentClosure(1, "M"))
	if !(cap1.P50Hours > single.P50Hours*2) {
		t.Errorf("3-item chain p50 %.1f should exceed 2× single-item p50 %.1f", cap1.P50Hours, single.P50Hours)
	}
}

func TestEstimateGoal_GateWaitLengthensBand(t *testing.T) {
	e := newSampledEstimator(t, 1)
	ungated := GoalClosureInput{
		Items: []ClosureItem{{Ref: "task/a", EffortClass: "M", Gated: false}},
		Deps:  map[string][]string{"task/a": nil},
	}
	gated := GoalClosureInput{
		Items: []ClosureItem{{Ref: "task/a", EffortClass: "M", Gated: true}},
		Deps:  map[string][]string{"task/a": nil},
	}
	base, _ := e.EstimateGoal(ungated)
	withGate, _ := e.EstimateGoal(gated)

	if !(withGate.P50Hours > base.P50Hours) {
		t.Errorf("gated p50 %.1f must exceed ungated p50 %.1f", withGate.P50Hours, base.P50Hours)
	}
}

func TestEstimateGoal_BasisDegradesToPriorsOnly(t *testing.T) {
	// No samples anywhere: the goal band must read "priors only".
	e := NewEstimator(nil, nil, 2, DefaultTrials, DefaultSeed)
	band, _ := e.EstimateGoal(independentClosure(4, "M"))
	if band.BasisLabel != "priors only" {
		t.Errorf("basis label = %q, want %q", band.BasisLabel, "priors only")
	}
	if band.Confidence != ConfidenceLow {
		t.Errorf("confidence = %q, want low", band.Confidence)
	}
	if band.Basis != BasisDefault {
		t.Errorf("basis = %q, want default", band.Basis)
	}
}

func TestEstimateGoal_BasisReportsSampleCount(t *testing.T) {
	e := newSampledEstimator(t, 2) // 8 live M samples
	band, _ := e.EstimateGoal(independentClosure(3, "M"))
	if !strings.HasSuffix(band.BasisLabel, "samples") {
		t.Errorf("basis label = %q, want a sample count", band.BasisLabel)
	}
	if band.Basis != BasisLive {
		t.Errorf("basis = %q, want live", band.Basis)
	}
	if band.Confidence != ConfidenceHigh {
		t.Errorf("confidence = %q, want high", band.Confidence)
	}
}

func TestEstimateGoal_MixedBasisWhenClassesDiffer(t *testing.T) {
	// M has live samples; L has none → mixed basis across the pending items.
	var samples []Sample
	for _, h := range []float64{44, 46, 48, 50, 52, 48} {
		samples = append(samples, Sample{EffortClass: "M", DurationHours: h, Origin: OriginLive})
	}
	e := NewEstimator(samples, nil, 2, DefaultTrials, DefaultSeed)
	in := GoalClosureInput{
		Items: []ClosureItem{
			{Ref: "task/a", EffortClass: "M"},
			{Ref: "task/b", EffortClass: "L"},
		},
		Deps: map[string][]string{"task/a": nil, "task/b": nil},
	}
	band, _ := e.EstimateGoal(in)
	if band.Basis != BasisMixed {
		t.Errorf("basis = %q, want mixed", band.Basis)
	}
	// Weakest confidence (L is priors-only → low) wins.
	if band.Confidence != ConfidenceLow {
		t.Errorf("confidence = %q, want low (weakest)", band.Confidence)
	}
}

func TestEstimateGoal_Determinism(t *testing.T) {
	closure := independentClosure(5, "M")
	a, _ := newSampledEstimator(t, 2).EstimateGoal(closure)
	b, _ := newSampledEstimator(t, 2).EstimateGoal(closure)
	if a.P50Hours != b.P50Hours || a.P80Hours != b.P80Hours {
		t.Errorf("estimate should be deterministic: (%v,%v) vs (%v,%v)", a.P50Hours, a.P80Hours, b.P50Hours, b.P80Hours)
	}
}

func TestHumanizeHours(t *testing.T) {
	cases := []struct {
		hours float64
		want  string
	}{
		{0, "done"},
		{-5, "done"},
		{1, "~1 hour"},
		{5, "~5 hours"},
		{48, "~2 days"},
		{24 * 21, "~3 weeks"},
	}
	for _, c := range cases {
		if got := HumanizeHours(c.hours); got != c.want {
			t.Errorf("HumanizeHours(%v) = %q, want %q", c.hours, got, c.want)
		}
	}
}
