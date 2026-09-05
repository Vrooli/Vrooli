package focus

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

type focusTestGenieResolver struct{ base string }

func (r focusTestGenieResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.base, nil
}

type focusTestGenieRuns struct {
	runsconnect.UnimplementedRunsServiceHandler
	health *runsv1.FleetHealth
}

func (f focusTestGenieRuns) GetFleetHealth(context.Context, *connect.Request[runsv1.GetFleetHealthRequest]) (*connect.Response[runsv1.GetFleetHealthResponse], error) {
	return connect.NewResponse(&runsv1.GetFleetHealthResponse{FleetHealth: f.health}), nil
}

func TestTestGenieGapSourceEmitsTypedFleetEvidence(t *testing.T) {
	_, handler := runsconnect.NewRunsServiceHandler(focusTestGenieRuns{health: &runsv1.FleetHealth{
		WindowDays:             30,
		TotalRuns:              10,
		Scenarios:              []*runsv1.FleetScenarioHealth{{Scenario: "demo", Runs: 10, PassedRuns: 2, FailedRuns: 8}},
		FailureClassifications: []*runsv1.FailureClassificationCount{{Classification: "maturity_contract", Count: 6}, {Classification: "system", Count: 2}},
	}})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	source := &testGenieGapSource{resolver: focusTestGenieResolver{base: server.URL}, http: server.Client(), deadline: time.Second}
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 1 {
		t.Fatalf("gaps = %+v, want one fleet gap", gaps)
	}
	if gaps[0].Axis != AxisEmpirical || gaps[0].EvidenceSource != "test-genie" || gaps[0].Recurrence != 8 {
		t.Fatalf("gap = %+v, want empirical test-genie recurrence 8", gaps[0])
	}
	if len(gaps[0].Notes) != 3 {
		t.Fatalf("notes = %v, want summary plus two typed classifications", gaps[0].Notes)
	}
}

func TestTestGenieGapSourceOmitsCleanFleet(t *testing.T) {
	_, handler := runsconnect.NewRunsServiceHandler(focusTestGenieRuns{health: &runsv1.FleetHealth{
		TotalRuns: 2,
		Scenarios: []*runsv1.FleetScenarioHealth{{Scenario: "demo", Runs: 2, PassedRuns: 2}},
	}})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	source := &testGenieGapSource{resolver: focusTestGenieResolver{base: server.URL}, http: server.Client(), deadline: time.Second}
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 0 {
		t.Fatalf("clean fleet gaps = %+v, want none", gaps)
	}
}
