package signals_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/mocks"
)

func newReg(sigs ...signals.Signal) *signals.Registry {
	return signals.NewRegistry(sigs...)
}

func TestAggregator_TierAutoPlaceAboveThreshold(t *testing.T) {
	sig := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-1", Domain: "graph", Value: 0.95,
			Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}},
		}},
	}
	agg := signals.NewAggregator(newReg(sig), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if v.Tier != signals.TierAutoPlace {
		t.Fatalf("want auto_place, got %s (verdict=%+v)", v.Tier, v)
	}
}

func TestAggregator_TierSuggest(t *testing.T) {
	sig := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-1", Domain: "graph", Value: 0.70,
			Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}},
		}},
	}
	agg := signals.NewAggregator(newReg(sig), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if v.Tier != signals.TierSuggest {
		t.Fatalf("want suggest, got %s", v.Tier)
	}
}

func TestAggregator_TierConflictBelowSuggest(t *testing.T) {
	sig := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-1", Domain: "graph", Value: 0.30,
			Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}},
		}},
	}
	agg := signals.NewAggregator(newReg(sig), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if v.Tier != signals.TierConflict {
		t.Fatalf("want conflict, got %s", v.Tier)
	}
}

func TestAggregator_TiedTopTwoConflicts(t *testing.T) {
	sig := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{
			{Signal: "fake-1", Domain: "graph", Value: 0.90, Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}}},
			{Signal: "fake-1", Domain: "conflicts", Value: 0.85, Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}}},
		},
	}
	agg := signals.NewAggregator(newReg(sig), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if !v.Tied {
		t.Fatal("expected tied verdict")
	}
	if v.Tier != signals.TierConflict {
		t.Fatalf("tied verdict should be conflict, got %s", v.Tier)
	}
}

func TestAggregator_DropsEmptyEvidenceAndSynthesizesBrokenContractAbstention(t *testing.T) {
	sig := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-1", Domain: "graph", Value: 0.99,
			// No evidence — should be dropped; aggregator records a
			// broken-contract abstention so the failure surfaces.
		}},
	}
	agg := signals.NewAggregator(newReg(sig), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if len(v.Scores) != 0 {
		t.Fatalf("evidence-less score should be dropped, got %+v", v.Scores)
	}
	if len(v.Abstentions) != 1 || v.Abstentions[0].Signal != "fake-1" {
		t.Fatalf("expected broken-contract abstention for fake-1, got %+v", v.Abstentions)
	}
	if v.Tier != signals.TierConflict {
		t.Fatalf("expected conflict when no scores survive, got %s", v.Tier)
	}
}

func TestAggregator_AbstainingSignalDoesNotDiluteDirectionValue(t *testing.T) {
	// Scored signal weight=2.0 emits domain=graph value=1.0.
	// Abstaining signal weight=1.0 emits no score.
	// Direction should be 1.0 because only scoring signals normalize it;
	// confidence should record participation as 2/3.
	scored := &mocks.FakeSignal{
		NameValue: "fake-scored",
		Weight:    2.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-scored", Domain: "graph", Value: 1.0,
			Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}},
		}},
	}
	abstaining := &mocks.FakeSignal{
		NameValue: "fake-abstain",
		Weight:    1.0,
		Available: true,
		Abstain: &signals.Abstention{
			Signal:   "fake-abstain",
			Reason:   "no data",
			Evidence: []signals.Evidence{{Kind: "abstain", Summary: "no data"}},
		},
	}
	agg := signals.NewAggregator(newReg(scored, abstaining), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if got := v.TopValue; got != 1.0 {
		t.Fatalf("abstaining signal must not dilute direction value: got %v want 1.0", got)
	}
	if got := v.Confidence; got < 0.66 || got > 0.67 {
		t.Fatalf("confidence should capture participation: got %v want ~0.667", got)
	}
	if len(v.Abstentions) != 1 || v.Abstentions[0].Signal != "fake-abstain" {
		t.Fatalf("expected one abstention for fake-abstain, got %+v", v.Abstentions)
	}
	if v.Tier != signals.TierAutoPlace {
		t.Fatalf("expected auto_place with direction=1.0 and confidence above high quorum, got %s", v.Tier)
	}
}

func TestAggregator_SingleSignalCannotAutoPlaceWithoutHighQuorum(t *testing.T) {
	scored := &mocks.FakeSignal{
		NameValue: "fake-scored",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-scored", Domain: "graph", Value: 1.0,
			Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}},
		}},
	}
	abstainA := &mocks.FakeSignal{NameValue: "fake-abstain-a", Weight: 1.0, Available: true, Abstain: &signals.Abstention{
		Signal: "fake-abstain-a", Reason: "no data", Evidence: []signals.Evidence{{Kind: "abstain", Summary: "no data"}},
	}}
	abstainB := &mocks.FakeSignal{NameValue: "fake-abstain-b", Weight: 1.0, Available: true, Abstain: &signals.Abstention{
		Signal: "fake-abstain-b", Reason: "no data", Evidence: []signals.Evidence{{Kind: "abstain", Summary: "no data"}},
	}}
	abstainC := &mocks.FakeSignal{NameValue: "fake-abstain-c", Weight: 1.0, Available: true, Abstain: &signals.Abstention{
		Signal: "fake-abstain-c", Reason: "no data", Evidence: []signals.Evidence{{Kind: "abstain", Summary: "no data"}},
	}}
	agg := signals.NewAggregator(newReg(scored, abstainA, abstainB, abstainC), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if v.TopValue != 1.0 {
		t.Fatalf("direction should remain unanimous at 1.0, got %v", v.TopValue)
	}
	if v.Confidence != 0.25 {
		t.Fatalf("confidence = %v, want 0.25", v.Confidence)
	}
	if v.Tier != signals.TierConflict {
		t.Fatalf("low-quorum unanimous verdict should conflict, got %s", v.Tier)
	}
	if v.QuorumMet {
		t.Fatal("quorum should not be met below low quorum")
	}
}

func TestAggregator_SkipsUnavailableSignal(t *testing.T) {
	good := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-1", Domain: "graph", Value: 0.90,
			Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}},
		}},
	}
	disabled := &mocks.FakeSignal{
		NameValue: "fake-2",
		Weight:    2.0,
		Available: false,
	}
	agg := signals.NewAggregator(newReg(good, disabled), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if disabled.ScoreCalls.Load() != 0 {
		t.Fatal("disabled signal should not be called")
	}
	if v.TopDomain != "graph" {
		t.Fatalf("unexpected top: %+v", v)
	}
}

func TestRegistry_DeterministicOrder(t *testing.T) {
	r := signals.NewRegistry(
		&mocks.FakeSignal{NameValue: "z"},
		&mocks.FakeSignal{NameValue: "a"},
		&mocks.FakeSignal{NameValue: "m"},
	)
	names := []string{}
	for _, s := range r.All() {
		names = append(names, s.Name())
	}
	want := []string{"a", "m", "z"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order mismatch: got %v want %v", names, want)
		}
	}
}
