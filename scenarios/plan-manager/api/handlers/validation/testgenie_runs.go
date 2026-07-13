package validation

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	internalvalidation "plan-manager/internal/validation"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// testGenieRunClient is deliberately read-only. Native Test Genie wait remains
// the producer contract; this adapter only synchronizes its durable snapshot.
type testGenieRunClient struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http connect.HTTPClient
}

func newTestGenieRunClient() internalvalidation.TestRunClient {
	return testGenieRunClient{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: http.DefaultClient}
}

func (c testGenieRunClient) GetRun(ctx context.Context, scenario, runID string) (internalvalidation.TestRunEvidence, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return internalvalidation.TestRunEvidence{}, fmt.Errorf("resolve test-genie URL: %w", err)
	}
	client := runsconnect.NewRunsServiceClient(c.http, baseURL)
	resp, err := client.GetRun(ctx, connect.NewRequest(&runspb.GetRunRequest{Scenario: scenario, RunId: runID}))
	if err != nil {
		return internalvalidation.TestRunEvidence{}, fmt.Errorf("get test-genie run: %w", err)
	}
	run := resp.Msg.GetRun()
	return internalvalidation.TestRunEvidence{Scenario: run.GetScenario(), RunID: run.GetRunId(), Status: run.GetStatus(), Fingerprint: run.GetTreeDigest(), TerminalAt: run.GetCompletedAt(), Detail: "Test Genie run " + run.GetRunId() + " is " + run.GetStatus()}, nil
}
