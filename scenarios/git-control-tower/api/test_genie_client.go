package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// TestGenieClient is a lightweight HTTP client for test-genie APIs.
type TestGenieClient struct {
	httpClient *http.Client
	resolver   *discovery.Resolver
}

// NewTestGenieClient creates a new test-genie client with the given timeout.
func NewTestGenieClient(timeout time.Duration) *TestGenieClient {
	return &TestGenieClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *TestGenieClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.resolver != nil {
		return c.resolver.ResolveScenarioURLDefault(ctx, "test-genie")
	}
	return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
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

func (c *TestGenieClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve test-genie url: %w", err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("test-genie request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseTestGenieError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *TestGenieClient) doGet(ctx context.Context, path string, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve test-genie url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("test-genie request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseTestGenieError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseTestGenieError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("test-genie error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("test-genie error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("test-genie error: status %d, body: %s", resp.StatusCode, string(body))
}
