package health

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

// fakeClient overrides only GetSelfHealth; the embedded interface satisfies the
// rest of the RunsServiceClient surface.
type fakeClient struct {
	runs_v1connect.RunsServiceClient
	resp *runspb.GetSelfHealthResponse
}

func (f *fakeClient) GetSelfHealth(context.Context, *connect.Request[runspb.GetSelfHealthRequest]) (*connect.Response[runspb.GetSelfHealthResponse], error) {
	return connect.NewResponse(f.resp), nil
}

func withFakeClient(t *testing.T, fc runs_v1connect.RunsServiceClient) {
	t.Helper()
	prev := newClient
	newClient = func(*cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) { return fc, nil }
	t.Cleanup(func() { newClient = prev })
}

func sampleResponse() *runspb.GetSelfHealthResponse {
	return &runspb.GetSelfHealthResponse{
		SelfHealth: &runspb.SelfHealth{
			Catalog: &runspb.CatalogSummary{TotalPhases: 12, DelegatedPhases: 12, NativePhases: 0},
			Conformance: []*runspb.ProviderConformance{
				{Phase: "proto", Provider: "proto-health", Reachable: true, ContractValid: true, IdentityOk: true, SpecValid: true, MetricsAdopted: true, AdoptionScore: 1.0},
				{Phase: "contracts", Provider: "cli-health", Reachable: true, ContractValid: true, IdentityOk: true, SpecValid: true, MetricsAdopted: false, AdoptionScore: 0.8},
			},
			ConformanceFreshness: "live",
			Ledger: &runspb.ReliabilityLedger{
				WindowDays:   30,
				RunCount:     10,
				Availability: 0.9,
				RunOutcomes:  []*runspb.RunOutcomeCount{{Outcome: "passed", Count: 8}, {Outcome: "errored", Count: 1}, {Outcome: "failed", Count: 1}},
				Phases: []*runspb.PhaseReliability{
					{Phase: "proto", Provider: "proto-health", TotalObservations: 5, Passed: 4, Failed: 1, Availability: 1, FailureRate: 0.2, Duration: &runspb.DurationStats{P50: 10, P95: 30}},
				},
			},
		},
	}
}

func TestHealthHumanSummary(t *testing.T) {
	withFakeClient(t, &fakeClient{resp: sampleResponse()})
	var buf bytes.Buffer
	if err := run(&cliutil.APIClient{}, nil, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Test Genie self-health", "12 phases", "Conformance (live)", "availability=90.0%", "proto"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestHealthJSON(t *testing.T) {
	withFakeClient(t, &fakeClient{resp: sampleResponse()})
	var buf bytes.Buffer
	if err := run(&cliutil.APIClient{}, []string{"--json"}, &buf); err != nil {
		t.Fatalf("run --json: %v", err)
	}
	out := buf.String()
	// proto JSON is camelCase and emits the wrapper + nested ledger fields.
	for _, want := range []string{`"selfHealth"`, `"conformanceFreshness"`, `"ledger"`, `"availability"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("json missing %q:\n%s", want, out)
		}
	}
}
