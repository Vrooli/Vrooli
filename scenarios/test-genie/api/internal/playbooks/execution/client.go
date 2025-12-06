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

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
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

var (
	protoJSONUnmarshal = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	protoJSONMarshal = protojson.MarshalOptions{
		UseProtoNames:   true, // Use snake_case field names to match BAS expectations
		EmitUnpopulated: true, // Emit empty arrays (e.g., edges:[]) required by BAS schema
	}
)

// ProgressCallback is called periodically during workflow execution with status updates.
// It receives the current status and elapsed time. Return an error to abort waiting.
// Status is the proto Execution message returned by BAS.
type ProgressCallback func(status *basv1.Execution, elapsed time.Duration) error

// Client defines the interface for BAS API operations.
type Client interface {
	// Health checks if the BAS API is healthy.
	Health(ctx context.Context) error
	// WaitForHealth waits until BAS becomes healthy or timeout.
	WaitForHealth(ctx context.Context) error
	// ValidateResolved validates a resolved workflow before execution.
	// Returns validation issues or nil if the workflow is valid.
	ValidateResolved(ctx context.Context, definition map[string]any) (*ValidationResult, error)
	// ExecuteWorkflow starts a workflow execution and returns the execution ID.
	ExecuteWorkflow(ctx context.Context, definition map[string]any, name string) (string, error)
	// GetStatus retrieves the status of an execution.
	GetStatus(ctx context.Context, executionID string) (*basv1.Execution, error)
	// WaitForCompletion waits for a workflow to complete.
	WaitForCompletion(ctx context.Context, executionID string) error
	// WaitForCompletionWithProgress waits for completion and calls the progress callback periodically.
	WaitForCompletionWithProgress(ctx context.Context, executionID string, callback ProgressCallback) error
	// GetTimeline retrieves the timeline data for an execution.
	GetTimeline(ctx context.Context, executionID string) (*basv1.ExecutionTimeline, []byte, error)
	// GetScreenshots retrieves screenshot metadata for an execution.
	GetScreenshots(ctx context.Context, executionID string) ([]Screenshot, error)
	// DownloadAsset downloads an asset (screenshot, artifact) by URL.
	DownloadAsset(ctx context.Context, assetURL string) ([]byte, error)
	// BaseURL returns the base URL of the BAS API (for constructing asset URLs).
	BaseURL() string
}

// ValidationResult represents the result of BAS workflow validation.
type ValidationResult struct {
	Valid         bool              `json:"valid"`
	Errors        []ValidationIssue `json:"errors,omitempty"`
	Warnings      []ValidationIssue `json:"warnings,omitempty"`
	SchemaVersion string            `json:"schema_version,omitempty"`
}

// ValidationIssue represents a single validation issue.
type ValidationIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	NodeID   string `json:"node_id,omitempty"`
	NodeType string `json:"node_type,omitempty"`
	Field    string `json:"field,omitempty"`
	Pointer  string `json:"pointer,omitempty"`
	Hint     string `json:"hint,omitempty"`
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

// ValidateResolved validates a resolved workflow before execution.
// This is the pre-flight check that catches unresolved tokens and schema errors.
func (c *HTTPClient) ValidateResolved(ctx context.Context, definition map[string]any) (*ValidationResult, error) {
	payload := map[string]any{
		"workflow": definition,
		"strict":   true, // Use strict mode to catch all issues
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/workflows/validate-resolved", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("validation request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	var result ValidationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode validation response: %w", err)
	}

	return &result, nil
}

// ExecuteWorkflow starts a workflow execution and returns the execution ID.
// It uses proto serialization for type-safe request formatting.
func (c *HTTPClient) ExecuteWorkflow(ctx context.Context, definition map[string]any, name string) (string, error) {
	// Build proto request for type safety
	protoReq, err := BuildAdhocRequest(definition, name)
	if err != nil {
		return "", fmt.Errorf("failed to build proto request: %w", err)
	}

	// Marshal using protojson for consistent field naming
	body, err := protoJSONMarshal.Marshal(protoReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proto request: %w", err)
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

	var result basv1.ExecuteAdhocResponse
	if err := protoJSONUnmarshal.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to decode response (proto violation): %w", err)
	}

	if strings.TrimSpace(result.GetExecutionId()) == "" {
		return "", fmt.Errorf("execution_id missing in response: %s", strings.TrimSpace(string(data)))
	}

	return result.GetExecutionId(), nil
}

// GetStatus retrieves the status of an execution.
func (c *HTTPClient) GetStatus(ctx context.Context, executionID string) (*basv1.Execution, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s", c.baseURL, executionID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status lookup failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	var status basv1.Execution
	if err := protoJSONUnmarshal.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to decode status (proto violation): %w", err)
	}

	return &status, nil
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
	checkStatus := func() (done bool, status *basv1.Execution, err error) {
		status, err = c.GetStatus(ctx, executionID)
		if err != nil {
			return true, status, err
		}

		normalized := strings.ToLower(status.GetStatus())
		switch normalized {
		case "completed", "success":
			return true, status, nil
		case "failed", "error", "errored":
			if status != nil && status.Error != nil {
				if msg := strings.TrimSpace(status.GetError()); msg != "" {
					return true, status, fmt.Errorf("workflow failed: %s", msg)
				}
			}
			return true, status, fmt.Errorf("workflow failed with status %s", status.GetStatus())
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

// GetTimeline retrieves the timeline data for an execution and validates it against the proto contract.
// It returns the parsed proto message alongside the raw JSON for artifact persistence.
func (c *HTTPClient) GetTimeline(ctx context.Context, executionID string) (*basv1.ExecutionTimeline, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s/timeline", c.baseURL, executionID), nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, data, fmt.Errorf("timeline fetch failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}

	var timeline basv1.ExecutionTimeline
	if err := protoJSONUnmarshal.Unmarshal(data, &timeline); err != nil {
		return nil, data, fmt.Errorf("failed to decode timeline (proto violation): %w", err)
	}

	return &timeline, data, nil
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

// ParseTimeline parses timeline data and returns a summary using the proto contract.
// Returns a TimelineParseError if the data cannot be parsed, allowing callers
// to save the raw data for debugging.
func ParseTimeline(data []byte) (TimelineSummary, error) {
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		return TimelineSummary{}, err
	}
	if parsed == nil {
		return TimelineSummary{}, nil
	}
	return parsed.Summary, nil
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
