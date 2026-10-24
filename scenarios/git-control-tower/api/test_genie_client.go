package main

import (
	"context"
	"time"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// TestGenieClient is a lightweight HTTP client for test-genie APIs.
type TestGenieClient struct {
	BaseClient
}

// NewTestGenieClient creates a new test-genie client with the given timeout.
func NewTestGenieClient(timeout time.Duration) *TestGenieClient {
	return &TestGenieClient{
		BaseClient: NewBaseClient("test-genie", timeout),
	}
}

// ExecuteSuite calls POST /api/v1/executions on test-genie.
func (c *TestGenieClient) ExecuteSuite(ctx context.Context, req TestExecutionRequest) (*TestExecutionResult, error) {
	var result TestExecutionResult
	err := c.doJSON(ctx, "/api/v1/executions", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadRun reads canonical durable history, retaining an exact requested run ID.
func (c *TestGenieClient) ReadRun(ctx context.Context, scenario, runID string) (*runspb.RunInfo, error) {
	client := evidenceRunsClient{httpClient: c.httpClient, resolveURL: c.resolveBaseURL}
	if runID != "" {
		result, err := client.GetRun(ctx, scenario, runID)
		if err != nil {
			return nil, err
		}
		return result.GetRun(), nil
	}
	runs, err := client.ListRuns(ctx, scenario, 1)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return runs[0], nil
}
