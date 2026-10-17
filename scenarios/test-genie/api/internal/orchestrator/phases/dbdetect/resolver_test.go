package dbdetect_test

import (
	"context"
	"errors"
	"testing"

	"test-genie/internal/orchestrator/phases/dbdetect"
)

// fixedCollector is a test collector that returns a canned observation list.
type fixedCollector struct {
	name string
	obs  []dbdetect.Observation
	err  error
}

func (c fixedCollector) Name() string { return c.name }
func (c fixedCollector) Collect(_ context.Context, _ dbdetect.ScenarioInputs) ([]dbdetect.Observation, error) {
	return c.obs, c.err
}

func mkProfile(db string, sources ...dbdetect.ProfileSource) dbdetect.Profile {
	return dbdetect.Profile{DB: db, Sources: sources}
}

func TestResolverNoEvidence(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{fixedCollector{name: "manifest"}},
		[]dbdetect.Profile{mkProfile("postgres",
			dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"},
		)},
	)
	rep := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if rep.Required("postgres") {
		t.Fatalf("expected not required")
	}
	if rep.Results["postgres"].Decision != nil {
		t.Fatalf("expected nil decision")
	}
}

func TestResolverSingleSource(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{fixedCollector{name: "manifest", obs: []dbdetect.Observation{{Collector: "manifest", Value: "postgres"}}}},
		[]dbdetect.Profile{mkProfile("postgres",
			dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"},
		)},
	)
	rep := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if !rep.Required("postgres") {
		t.Fatalf("expected required")
	}
	if rep.Results["postgres"].Decision == nil || rep.Results["postgres"].Decision.Source != "m" {
		t.Fatalf("decision wrong: %+v", rep.Results["postgres"].Decision)
	}
	if len(rep.Results["postgres"].Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", rep.Results["postgres"].Conflicts)
	}
}

func TestResolverHighestPriorityWins(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{
			fixedCollector{name: "manifest", obs: []dbdetect.Observation{{Collector: "manifest", Value: "postgres"}}},
			fixedCollector{name: "godeps", obs: []dbdetect.Observation{{Collector: "godeps", Value: "github.com/lib/pq"}}},
		},
		[]dbdetect.Profile{mkProfile("postgres",
			dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"},
			dbdetect.ProfileSource{Collector: "godeps", Match: dbdetect.ImportPrefixes("github.com/lib/pq"), Priority: dbdetect.PriorityMedium, Label: "g"},
		)},
	)
	rep := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if rep.Results["postgres"].Decision.Source != "m" {
		t.Fatalf("expected manifest decision, got %+v", rep.Results["postgres"].Decision)
	}
	if len(rep.Results["postgres"].Corroborating) != 1 || rep.Results["postgres"].Corroborating[0].Source != "g" {
		t.Fatalf("expected godeps corroborating, got %+v", rep.Results["postgres"].Corroborating)
	}
}

func TestResolverMissingCorroborationConflict(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{
			fixedCollector{name: "manifest", obs: []dbdetect.Observation{{Collector: "manifest", Value: "postgres"}}},
			fixedCollector{name: "godeps"},
		},
		[]dbdetect.Profile{mkProfile("postgres",
			dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"},
			dbdetect.ProfileSource{Collector: "godeps", Match: dbdetect.ImportPrefixes("github.com/lib/pq"), Priority: dbdetect.PriorityMedium, Label: "g"},
		)},
	)
	rep := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if !rep.Required("postgres") {
		t.Fatalf("expected required")
	}
	if len(rep.Results["postgres"].Conflicts) != 1 || rep.Results["postgres"].Conflicts[0].Kind != "missing-corroboration" {
		t.Fatalf("expected one missing-corroboration conflict, got %+v", rep.Results["postgres"].Conflicts)
	}
}

func TestResolverCollectorErrorCapturedAsConflict(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{
			fixedCollector{name: "manifest", obs: []dbdetect.Observation{{Collector: "manifest", Value: "postgres"}}},
			fixedCollector{name: "godeps", err: errors.New("boom")},
		},
		[]dbdetect.Profile{mkProfile("postgres",
			dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"},
			dbdetect.ProfileSource{Collector: "godeps", Match: dbdetect.ImportPrefixes("github.com/lib/pq"), Priority: dbdetect.PriorityMedium, Label: "g"},
		)},
	)
	rep := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if !rep.Required("postgres") {
		t.Fatalf("expected required despite collector error")
	}
	var sawErr bool
	for _, c := range rep.Results["postgres"].Conflicts {
		if c.Kind == "collector-error" {
			sawErr = true
		}
	}
	if !sawErr {
		t.Fatalf("expected collector-error conflict, got %+v", rep.Results["postgres"].Conflicts)
	}
}

func TestResolverMultiDB(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{
			fixedCollector{name: "manifest", obs: []dbdetect.Observation{{Collector: "manifest", Value: "postgres"}}},
			fixedCollector{name: "godeps", obs: []dbdetect.Observation{{Collector: "godeps", Value: "modernc.org/sqlite"}}},
		},
		[]dbdetect.Profile{
			mkProfile("postgres", dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"}),
			mkProfile("sqlite", dbdetect.ProfileSource{Collector: "godeps", Match: dbdetect.ImportPrefixes("modernc.org/sqlite"), Priority: dbdetect.PriorityHigh, Label: "g"}),
		},
	)
	rep := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if !rep.Required("postgres") || !rep.Required("sqlite") {
		t.Fatalf("expected both required, got %+v", rep.Results)
	}
}

func TestResolverUnknownCollectorAtConstruction(t *testing.T) {
	_, err := dbdetect.NewResolver(
		[]dbdetect.Collector{fixedCollector{name: "manifest"}},
		[]dbdetect.Profile{mkProfile("postgres", dbdetect.ProfileSource{Collector: "nope", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"})},
	)
	if err == nil {
		t.Fatalf("expected error for unknown collector")
	}
}

func TestResolverDeterministicOrder(t *testing.T) {
	r := mustResolver(t,
		[]dbdetect.Collector{fixedCollector{name: "manifest", obs: []dbdetect.Observation{
			{Collector: "manifest", Value: "postgres"},
			{Collector: "manifest", Value: "redis"},
		}}},
		[]dbdetect.Profile{
			mkProfile("postgres", dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("postgres"), Priority: dbdetect.PriorityHigh, Label: "m"}),
			mkProfile("redis", dbdetect.ProfileSource{Collector: "manifest", Match: dbdetect.ManifestType("redis"), Priority: dbdetect.PriorityHigh, Label: "m"}),
		},
	)
	rep1 := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	rep2 := r.Resolve(context.Background(), dbdetect.ScenarioInputs{})
	if !equalStrings(rep1.Order, rep2.Order) {
		t.Fatalf("order not deterministic: %v vs %v", rep1.Order, rep2.Order)
	}
	if !equalStrings(rep1.Order, []string{"postgres", "redis"}) {
		t.Fatalf("expected [postgres, redis], got %v", rep1.Order)
	}
}

func mustResolver(t *testing.T, cs []dbdetect.Collector, ps []dbdetect.Profile) *dbdetect.Resolver {
	t.Helper()
	r, err := dbdetect.NewResolver(cs, ps)
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	return r
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
