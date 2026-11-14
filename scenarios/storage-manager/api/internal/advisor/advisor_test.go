package advisor

import (
	"context"
	"errors"
	"testing"
)

type fakeReader struct {
	facts map[string]ScenarioFacts
	fail  map[string]bool
}

func (f fakeReader) Read(_ context.Context, scenario string) (ScenarioFacts, error) {
	if f.fail[scenario] {
		return ScenarioFacts{}, errors.New("boom")
	}
	return f.facts[scenario], nil
}

type fakeEnum struct{ list []string }

func (f fakeEnum) List(context.Context) ([]string, error) { return f.list, nil }

func TestAnalyzeMigrations(t *testing.T) {
	rd := fakeReader{
		facts: map[string]ScenarioFacts{
			"deployed": {Scenario: "deployed", Engines: []string{"postgres"}, StorageStage: "production", HasMigrations: false, DebtNotes: []string{"no migration path"}},
			"clean":    {Scenario: "clean", Engines: []string{"sqlite"}, StorageStage: "greenfield", HasMigrations: false},
			"brown":    {Scenario: "brown", Engines: []string{"sqlite"}, StorageStage: "production", HasMigrations: true},
		},
		fail: map[string]bool{"broken": true},
	}
	svc := NewService(rd, fakeEnum{})
	res, err := svc.AnalyzeMigrations(context.Background(), []string{"deployed", "clean", "brown", "broken"})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if res.ScenarioCount != 3 {
		t.Fatalf("scenario_count: got %d want 3", res.ScenarioCount)
	}
	if res.WithMigrationsCount != 1 {
		t.Fatalf("with_migrations: got %d want 1", res.WithMigrationsCount)
	}
	if res.DebtCount != 1 {
		t.Fatalf("debt_count: got %d want 1", res.DebtCount)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("errors: got %+v want 1", res.Errors)
	}
}

func TestAdviseEnginesRanking(t *testing.T) {
	rd := fakeReader{
		facts: map[string]ScenarioFacts{
			"green-pg": {Scenario: "green-pg", Engines: []string{"postgres"}, StorageStage: "greenfield"},
			"prod-pg":  {Scenario: "prod-pg", Engines: []string{"postgres"}, StorageStage: "production"},
			"sqlite":   {Scenario: "sqlite", Engines: []string{"sqlite"}, StorageStage: "production"},
		},
	}
	svc := NewService(rd, nil)
	res, err := svc.AdviseEngines(context.Background(), []string{"green-pg", "prod-pg", "sqlite"})
	if err != nil {
		t.Fatalf("advise: %v", err)
	}
	// Only the two Postgres scenarios are candidates; SQLite one is not.
	if len(res.Candidates) != 2 {
		t.Fatalf("candidates: got %d want 2", len(res.Candidates))
	}
	// greenfield ranks above production.
	if res.Candidates[0].Scenario != "green-pg" {
		t.Fatalf("expected green-pg ranked first, got %+v", res.Candidates)
	}
	if res.Candidates[0].FitnessScore <= res.Candidates[1].FitnessScore {
		t.Fatalf("greenfield should outscore production: %v vs %v", res.Candidates[0].FitnessScore, res.Candidates[1].FitnessScore)
	}
	// Production candidate carries a blocker.
	if len(res.Candidates[1].Blockers) == 0 {
		t.Fatal("production candidate should carry a blocker")
	}
	for _, c := range res.Candidates {
		if c.RecommendedEngine != "sqlite" || c.CurrentEngine != "postgres" || c.Autofixable {
			t.Fatalf("unexpected candidate shape: %+v", c)
		}
	}
}

func TestAdviseEnginesEnumeratorFallback(t *testing.T) {
	rd := fakeReader{facts: map[string]ScenarioFacts{"a": {Scenario: "a", Engines: []string{"postgres"}, StorageStage: "pilot"}}}
	svc := NewService(rd, fakeEnum{list: []string{"a"}})
	res, err := svc.AdviseEngines(context.Background(), nil)
	if err != nil {
		t.Fatalf("advise: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("want 1 candidate, got %d", len(res.Candidates))
	}
}

func TestUnwiredService(t *testing.T) {
	var svc *Service
	if _, err := svc.AnalyzeMigrations(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := svc.AdviseEngines(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected error")
	}
}
