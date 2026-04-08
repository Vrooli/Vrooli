package main

import (
	"context"
	"fmt"
	"time"
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

// ListExecutions calls GET /api/v1/executions?scenario=<name>&limit=<n> on test-genie.
func (c *TestGenieClient) ListExecutions(ctx context.Context, scenario string, limit int) (*TestExecutionListResponse, error) {
	path := fmt.Sprintf("/api/v1/executions?scenario=%s&limit=%d", scenario, limit)
	var result TestExecutionListResponse
	err := c.doGet(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetExecution calls GET /api/v1/executions/{id} on test-genie.
func (c *TestGenieClient) GetExecution(ctx context.Context, id string) (*TestExecutionResult, error) {
	var result TestExecutionResult
	err := c.doGet(ctx, "/api/v1/executions/"+id, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
