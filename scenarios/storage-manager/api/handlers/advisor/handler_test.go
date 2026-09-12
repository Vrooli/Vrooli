package advisor

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internaladvisor "storage-manager/internal/advisor"

	advisorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/advisor"
)

type fakeReader struct {
	facts map[string]internaladvisor.ScenarioFacts
}

func (f fakeReader) Read(_ context.Context, s string) (internaladvisor.ScenarioFacts, error) {
	return f.facts[s], nil
}

func newTestHandler() *Handler {
	rd := fakeReader{facts: map[string]internaladvisor.ScenarioFacts{
		"alpha": {Scenario: "alpha", Engines: []string{"postgres"}, StorageStage: "greenfield", DebtNotes: []string{"premature migrations"}},
		"beta":  {Scenario: "beta", Engines: []string{"sqlite"}, StorageStage: "production", HasMigrations: true},
	}}
	return NewHandler(internaladvisor.NewService(rd, nil), nil)
}

func TestAnalyzeMigrationsHandler(t *testing.T) {
	h := newTestHandler()
	resp, err := h.AnalyzeMigrations(context.Background(), connect.NewRequest(&advisorv1.AnalyzeMigrationsRequest{Scenarios: []string{"alpha", "beta"}}))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if resp.Msg.GetScenarioCount() != 2 {
		t.Fatalf("scenario_count: got %d want 2", resp.Msg.GetScenarioCount())
	}
	if resp.Msg.GetDebtCount() != 1 {
		t.Fatalf("debt_count: got %d want 1", resp.Msg.GetDebtCount())
	}
	if resp.Msg.GetWithMigrationsCount() != 1 {
		t.Fatalf("with_migrations: got %d want 1", resp.Msg.GetWithMigrationsCount())
	}
}

func TestAdviseEnginesHandler(t *testing.T) {
	h := newTestHandler()
	resp, err := h.AdviseEngines(context.Background(), connect.NewRequest(&advisorv1.AdviseEnginesRequest{Scenarios: []string{"alpha", "beta"}}))
	if err != nil {
		t.Fatalf("advise: %v", err)
	}
	// Only alpha (postgres) is a candidate.
	if len(resp.Msg.GetCandidates()) != 1 {
		t.Fatalf("candidates: got %d want 1", len(resp.Msg.GetCandidates()))
	}
	c := resp.Msg.GetCandidates()[0]
	if c.GetScenario() != "alpha" || c.GetRecommendedEngine() != "sqlite" {
		t.Fatalf("unexpected candidate: %+v", c)
	}
}
