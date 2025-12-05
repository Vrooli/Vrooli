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

// ProgressCallback is called periodically during workflow execution with status updates.
// It receives the current status and elapsed time. Return an error to abort waiting.
// Use types.ExecutionStatus as the status parameter type.
type ProgressCallback func(status types.ExecutionStatus, elapsed time.Duration) error

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
	// WaitForCompletionWithProgress waits for completion and calls the progress callback periodically.
	WaitForCompletionWithProgress(ctx context.Context, executionID string, callback ProgressCallback) error
	// GetTimeline retrieves the timeline data for an execution.
	GetTimeline(ctx context.Context, executionID string) ([]byte, error)
	// GetScreenshots retrieves screenshot metadata for an execution.
	GetScreenshots(ctx context.Context, executionID string) ([]Screenshot, error)
	// DownloadAsset downloads an asset (screenshot, artifact) by URL.
	DownloadAsset(ctx context.Context, assetURL string) ([]byte, error)
	// BaseURL returns the base URL of the BAS API (for constructing asset URLs).
	BaseURL() string
}

// Screenshot represents screenshot metadata from BAS.
type Screenshot struct {
	ID           string `json:"id"`
	ExecutionID  string `json:"execution_id"`
	StepName     string `json:"step_name"`
	StepIndex    int    `json:"step_index,omitempty"`
	Timestamp    string `json:"timestamp"`
	StorageURL   string `json:"storage_url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
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
	return c.WaitForCompletionWithProgress(ctx, executionID, nil)
}

// WaitForCompletionWithProgress waits for completion and calls the progress callback periodically.
// The callback is invoked approximately every 5 seconds with the current status.
func (c *HTTPClient) WaitForCompletionWithProgress(ctx context.Context, executionID string, callback ProgressCallback) error {
	start := time.Now()

	// Helper to check status and return appropriate result
	checkStatus := func() (done bool, status types.ExecutionStatus, err error) {
		status, err = c.GetStatus(ctx, executionID)
		if err != nil {
			return true, status, err
		}

		normalized := strings.ToLower(status.Status)
		switch normalized {
		case "completed", "success":
			return true, status, nil
		case "failed", "error", "errored":
			if status.FailureReason != "" {
				return true, status, fmt.Errorf("workflow failed: %s", status.FailureReason)
			}
			if status.Error != "" {
				return true, status, fmt.Errorf("workflow failed: %s", status.Error)
			}
			return true, status, fmt.Errorf("workflow failed with status %s", status.Status)
		}
		return false, status, nil
	}

	// Check immediately first - workflow might already be done
	if done, status, err := checkStatus(); done {
		if callback != nil {
			_ = callback(status, time.Since(start))
		}
		return err
	}

	deadline := time.Now().Add(WorkflowExecutionTimeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Track last progress report time for throttling
	lastProgressReport := time.Now()
	progressInterval := 5 * time.Second

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("workflow execution timed out after %s", WorkflowExecutionTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			done, status, err := checkStatus()

			// Report progress if callback provided and interval elapsed
			if callback != nil && time.Since(lastProgressReport) >= progressInterval {
				if callbackErr := callback(status, time.Since(start)); callbackErr != nil {
					return fmt.Errorf("progress callback aborted: %w", callbackErr)
				}
				lastProgressReport = time.Now()
			}

			if done {
				// Final progress report
				if callback != nil {
					_ = callback(status, time.Since(start))
				}
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

// TimelineSummary contains parsed timeline statistics.
type TimelineSummary struct {
	TotalSteps    int
	TotalAsserts  int
	AssertsPassed int
}

// String returns a human-readable summary string.
func (s TimelineSummary) String() string {
	if s.TotalAsserts > 0 {
		return fmt.Sprintf(" (%d steps, %d/%d assertions passed)", s.TotalSteps, s.AssertsPassed, s.TotalAsserts)
	}
	if s.TotalSteps > 0 {
		return fmt.Sprintf(" (%d steps)", s.TotalSteps)
	}
	return ""
}

// TimelineParseError indicates timeline data could not be parsed.
type TimelineParseError struct {
	RawData []byte
	Cause   error
}

func (e *TimelineParseError) Error() string {
	return fmt.Sprintf("failed to parse timeline data (%d bytes): %v", len(e.RawData), e.Cause)
}

func (e *TimelineParseError) Unwrap() error {
	return e.Cause
}

// SummarizeTimeline extracts a summary from timeline data.
// Deprecated: Use ParseTimeline for better error handling.
func SummarizeTimeline(data []byte) string {
	summary, _ := ParseTimeline(data)
	return summary.String()
}

// ParseTimeline parses timeline data and returns a summary.
// Returns a TimelineParseError if the data cannot be parsed, allowing callers
// to save the raw data for debugging.
func ParseTimeline(data []byte) (TimelineSummary, error) {
	var summary TimelineSummary

	if len(data) == 0 {
		return summary, nil
	}

	var doc struct {
		Frames []struct {
			StepType string `json:"step_type"`
			Status   string `json:"status"`
		} `json:"frames"`
	}

	if err := json.Unmarshal(data, &doc); err != nil {
		return summary, &TimelineParseError{
			RawData: data,
			Cause:   err,
		}
	}

	summary.TotalSteps = len(doc.Frames)

	for _, frame := range doc.Frames {
		if frame.StepType == "assert" {
			summary.TotalAsserts++
			if strings.EqualFold(frame.Status, "completed") {
				summary.AssertsPassed++
			}
		}
	}

	return summary, nil
}

// BaseURL returns the base URL of the BAS API.
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

// GetScreenshots retrieves screenshot metadata for an execution.
func (c *HTTPClient) GetScreenshots(ctx context.Context, executionID string) ([]Screenshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s/screenshots", c.baseURL, executionID), nil)
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
		return nil, fmt.Errorf("screenshots fetch failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	var result struct {
		Screenshots []Screenshot `json:"screenshots"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode screenshots response: %w", err)
	}

	return result.Screenshots, nil
}

// DownloadAsset downloads an asset by URL. The URL can be absolute or relative to the BAS API.
func (c *HTTPClient) DownloadAsset(ctx context.Context, assetURL string) ([]byte, error) {
	// Handle relative URLs by prepending base URL (without /api/v1 suffix)
	fullURL := assetURL
	if !strings.HasPrefix(assetURL, "http://") && !strings.HasPrefix(assetURL, "https://") {
		// Remove /api/v1 suffix from baseURL if present for asset downloads
		baseForAssets := strings.TrimSuffix(c.baseURL, "/api/v1")
		if strings.HasPrefix(assetURL, "/") {
			fullURL = baseForAssets + assetURL
		} else {
			fullURL = baseForAssets + "/" + assetURL
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	// Use a longer timeout for asset downloads
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("asset download failed: status=%s url=%s", resp.Status, fullURL)
	}

	return io.ReadAll(resp.Body)
}
