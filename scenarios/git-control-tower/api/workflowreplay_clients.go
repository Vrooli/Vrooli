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

// workflowReplayRunsClient is the concrete RunsClient the WorkflowReplayService
// handler depends on. It wraps test-genie's RunsService Connect-RPC, resolving
// the endpoint through service discovery on every call so it survives test-genie
// restarts (same pattern as baselineRunsClient).
type workflowReplayRunsClient struct {
	httpClient *http.Client
	resolveURL func(ctx context.Context) (string, error)
}

func newWorkflowReplayRunsClient(timeout time.Duration) workflowReplayRunsClient {
	return workflowReplayRunsClient{
		httpClient: &http.Client{Timeout: timeout},
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
	}
}

func (c workflowReplayRunsClient) client(ctx context.Context) (runs_v1connect.RunsServiceClient, error) {
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie url: %w", err)
	}
	return runs_v1connect.NewRunsServiceClient(c.httpClient, baseURL), nil
}

func (c workflowReplayRunsClient) ListRuns(ctx context.Context, scenario string, limit int) ([]*runspb.RunInfo, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cl.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{
		Scenario: scenario, Limit: int32(limit),
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetRuns(), nil
}

func (c workflowReplayRunsClient) GetRun(ctx context.Context, scenario, runID string) (*runspb.RunInfo, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cl.GetRun(ctx, connect.NewRequest(&runspb.GetRunRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetRun(), nil
}

func (c workflowReplayRunsClient) ListRunVideos(ctx context.Context, scenario, runID string) ([]*runspb.RunVideo, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cl.ListRunVideos(ctx, connect.NewRequest(&runspb.ListRunVideosRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetVideos(), nil
}
