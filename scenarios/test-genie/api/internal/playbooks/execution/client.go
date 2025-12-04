package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"test-genie/internal/playbooks/types"
)

const (
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 15 * time.Second
	// HealthCheckTimeout is the timeout for health check requests.
	HealthCheckTimeout = 5 * time.Second
	// HealthCheckWaitTimeout is how long to wait for BAS to become healthy.
	HealthCheckWaitTimeout = 45 * time.Second
	// WorkflowExecutionTimeout is how long to wait for a workflow to complete.
	WorkflowExecutionTimeout = 3 * time.Minute
)

// Client defines the interface for BAS API operations.
type Client interface {
	// Health checks if the BAS API is healthy.
	Health(ctx context.Context) error
	// WaitForHealth waits until BAS becomes healthy or timeout.
	WaitForHealth(ctx context.Context) error
	// ExecuteWorkflow starts a workflow execution and returns the execution ID.
	ExecuteWorkflow(ctx context.Context, definition map[string]any, name string) (string, error)
	// GetStatus retrieves the status of an execution.
	GetStatus(ctx context.Context, executionID string) (types.ExecutionStatus, error)
	// WaitForCompletion waits for a workflow to complete.
	WaitForCompletion(ctx context.Context, executionID string) error
	// GetTimeline retrieves the timeline data for an execution.
	GetTimeline(ctx context.Context, executionID string) ([]byte, error)
}

// HTTPClient is the HTTP-based BAS client implementation.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new BAS HTTP client.
func NewClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
}

// WithHTTPClient sets a custom HTTP client (for testing).
func (c *HTTPClient) WithHTTPClient(client *http.Client) *HTTPClient {
	c.httpClient = client
	return c
}

// Health checks if the BAS API is healthy.
func (c *HTTPClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed: status %s", resp.Status)
	}
	return nil
}

// WaitForHealth waits until BAS becomes healthy or timeout.
func (c *HTTPClient) WaitForHealth(ctx context.Context) error {
	// Check immediately first - BAS might already be healthy
	if err := c.Health(ctx); err == nil {
		return nil
	}

	deadline := time.Now().Add(HealthCheckWaitTimeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("health check timeout after %s", HealthCheckWaitTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.Health(ctx); err == nil {
				return nil
			}
		}
	}
}

// ExecuteWorkflow starts a workflow execution and returns the execution ID.
func (c *HTTPClient) ExecuteWorkflow(ctx context.Context, definition map[string]any, name string) (string, error) {
	payload := map[string]any{
		"flow_definition":     definition,
		"parameters":          map[string]any{},
		"wait_for_completion": false,
		"metadata": map[string]any{
			"name": name,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/workflows/execute-adhoc", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("workflow execution failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	var result struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ExecutionID == "" {
		return "", fmt.Errorf("execution_id missing in response: %s", strings.TrimSpace(string(data)))
	}

	return result.ExecutionID, nil
}

// GetStatus retrieves the status of an execution.
func (c *HTTPClient) GetStatus(ctx context.Context, executionID string) (types.ExecutionStatus, error) {
	var status types.ExecutionStatus

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s", c.baseURL, executionID), nil)
	if err != nil {
		return status, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return status, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return status, fmt.Errorf("status lookup failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	if err := json.Unmarshal(data, &status); err != nil {
		return status, fmt.Errorf("failed to decode status: %w", err)
	}

	return status, nil
}

// WaitForCompletion waits for a workflow to complete.
func (c *HTTPClient) WaitForCompletion(ctx context.Context, executionID string) error {
	// Helper to check status and return appropriate result
	checkStatus := func() (done bool, err error) {
		status, err := c.GetStatus(ctx, executionID)
		if err != nil {
			return true, err
		}

		normalized := strings.ToLower(status.Status)
		switch normalized {
		case "completed", "success":
			return true, nil
		case "failed", "error", "errored":
			if status.FailureReason != "" {
				return true, fmt.Errorf("workflow failed: %s", status.FailureReason)
			}
			if status.Error != "" {
				return true, fmt.Errorf("workflow failed: %s", status.Error)
			}
			return true, fmt.Errorf("workflow failed with status %s", status.Status)
		}
		return false, nil
	}

	// Check immediately first - workflow might already be done
	if done, err := checkStatus(); done {
		return err
	}

	deadline := time.Now().Add(WorkflowExecutionTimeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("workflow execution timed out after %s", WorkflowExecutionTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if done, err := checkStatus(); done {
				return err
			}
		}
	}
}

// GetTimeline retrieves the timeline data for an execution.
func (c *HTTPClient) GetTimeline(ctx context.Context, executionID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s/timeline", c.baseURL, executionID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("timeline fetch failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	return data, nil
}

// SummarizeTimeline extracts a summary from timeline data.
func SummarizeTimeline(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var doc struct {
		Frames []struct {
			StepType string `json:"step_type"`
			Status   string `json:"status"`
		} `json:"frames"`
	}

	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}

	totalSteps := len(doc.Frames)
	totalAsserts := 0
	assertPassed := 0

	for _, frame := range doc.Frames {
		if frame.StepType == "assert" {
			totalAsserts++
			if strings.EqualFold(frame.Status, "completed") {
				assertPassed++
			}
		}
	}

	if totalAsserts > 0 {
		return fmt.Sprintf(" (%d steps, %d/%d assertions passed)", totalSteps, assertPassed, totalAsserts)
	}
	if totalSteps > 0 {
		return fmt.Sprintf(" (%d steps)", totalSteps)
	}
	return ""
}
