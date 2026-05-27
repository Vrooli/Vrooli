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
			{Signal: "fake-1", Domain: "manifest", Value: 0.85, Evidence: []signals.Evidence{{Kind: "demo", Summary: "x"}}},
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

func TestAggregator_DropsEmptyEvidence(t *testing.T) {
	sig := &mocks.FakeSignal{
		NameValue: "fake-1",
		Weight:    1.0,
		Available: true,
		Returns: []signals.Score{{
			Signal: "fake-1", Domain: "graph", Value: 0.99,
			// No evidence — should be dropped.
		}},
	}
	agg := signals.NewAggregator(newReg(sig), nil)
	v := agg.Aggregate(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, emptyDomainMap()), graph.Chunk{ID: "c1"})
	if len(v.Scores) != 0 {
		t.Fatalf("evidence-less score should be dropped, got %+v", v.Scores)
	}
	if v.Tier != signals.TierConflict {
		t.Fatalf("expected conflict when no scores survive, got %s", v.Tier)
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
