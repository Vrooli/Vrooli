package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// evidenceRunsClient preserves Test Genie's canonical proto records at GCT's
// generic evidence boundary. It resolves Test Genie for every call so reads
// survive provider restarts without caching a stale endpoint.
type evidenceRunsClient struct {
	httpClient *http.Client
	resolveURL func(context.Context) (string, error)
}

func newEvidenceRunsClient(timeout time.Duration) evidenceRunsClient {
	return evidenceRunsClient{
		httpClient: &http.Client{Timeout: timeout},
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
	}
}

func (c evidenceRunsClient) client(ctx context.Context) (runs_v1connect.RunsServiceClient, error) {
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie url: %w", err)
	}
	return runs_v1connect.NewRunsServiceClient(c.httpClient, baseURL), nil
}

func (c evidenceRunsClient) ListRuns(ctx context.Context, scenario string, limit int) ([]*runspb.RunInfo, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{Scenario: scenario, Limit: int32(limit)}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetRuns(), nil
}

func (c evidenceRunsClient) StartRun(ctx context.Context, request *runspb.StartRunRequest) (*runspb.StartRunResponse, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.StartRun(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c evidenceRunsClient) GetRun(ctx context.Context, scenario, runID string) (*runspb.GetRunResponse, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetRun(ctx, connect.NewRequest(&runspb.GetRunRequest{Scenario: scenario, RunId: runID}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c evidenceRunsClient) ListRunArtifacts(ctx context.Context, scenario, runID string, kinds []string) (*runspb.ListRunArtifactsResponse, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListRunArtifacts(ctx, connect.NewRequest(&runspb.ListRunArtifactsRequest{
		Scenario: scenario, RunId: runID, Kinds: kinds,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
