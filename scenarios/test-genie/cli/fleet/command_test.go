package fleet

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// fakeClient overrides only GetFleetHealth; the embedded interface satisfies the
// rest of the RunsServiceClient surface.
type fakeClient struct {
	runs_v1connect.RunsServiceClient
	resp *runspb.GetFleetHealthResponse
}

func (f *fakeClient) GetFleetHealth(context.Context, *connect.Request[runspb.GetFleetHealthRequest]) (*connect.Response[runspb.GetFleetHealthResponse], error) {
	return connect.NewResponse(f.resp), nil
}

func withFakeClient(t *testing.T, fc runs_v1connect.RunsServiceClient) {
	t.Helper()
	prev := newClient
	newClient = func(*cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) { return fc, nil }
	t.Cleanup(func() { newClient = prev })
}

func sampleResponse() *runspb.GetFleetHealthResponse {
	return &runspb.GetFleetHealthResponse{
		FleetHealth: &runspb.FleetHealth{
			WindowDays:      30,
			CapturedAt:      "2026-06-20T12:00:00Z",
			ScenariosTested: 2,
			ScenariosTotal:  3,
			TotalRuns:       5,
			TotalIssues:     2,
			Scenarios: []*runspb.FleetScenarioHealth{
				{Scenario: "flaky", Runs: 3, PassedRuns: 1, FailedRuns: 2, FailureRate: 0.66, Issues: 2, LastOutcome: "failed", LastRunAt: "2026-06-20T11:00:00Z", AgeDays: 0.04},
				{Scenario: "healthy", Runs: 2, PassedRuns: 2, FailureRate: 0, LastOutcome: "passed", AgeDays: 1},
			},
			TopFindingSources:   []*runspb.FleetFindingSource{{Source: "standards", Issues: 2}},
			NeverTestedInWindow: []string{"untouched"},
		},
	}
}

func TestFleetStatusHumanSummary(t *testing.T) {
	withFakeClient(t, &fakeClient{resp: sampleResponse()})
	var buf bytes.Buffer
	if err := runStatus(&cliutil.APIClient{}, nil, &buf); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Fleet health", "2 tested / 3 total", "flaky", "standards=2", "Never tested in window (1)", "untouched"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	// healthy scenario should NOT appear in the most-errored list.
	if strings.Contains(out, "healthy") {
		t.Fatalf("clean scenario leaked into most-errored list:\n%s", out)
	}
}

func TestFleetStatusJSON(t *testing.T) {
	withFakeClient(t, &fakeClient{resp: sampleResponse()})
	var buf bytes.Buffer
	if err := runStatus(&cliutil.APIClient{}, []string{"--json"}, &buf); err != nil {
		t.Fatalf("runStatus --json: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\"fleetHealth\"") || !strings.Contains(out, "\"neverTestedInWindow\"") {
		t.Fatalf("json output missing expected fields:\n%s", out)
	}
}

func TestFleetUnknownSubcommand(t *testing.T) {
	if err := Run(&cliutil.APIClient{}, []string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if err := Run(&cliutil.APIClient{}, nil); err == nil {
		t.Fatal("expected usage error for missing subcommand")
	}
}
