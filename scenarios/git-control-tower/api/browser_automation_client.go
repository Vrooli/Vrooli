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

// BrowserAutomationClient is a lightweight HTTP client for browser-automation-studio APIs.
type BrowserAutomationClient struct {
	httpClient *http.Client
	resolver   *discovery.Resolver
}

// NewBrowserAutomationClient creates a new BAS client with the given timeout.
func NewBrowserAutomationClient(timeout time.Duration) *BrowserAutomationClient {
	return &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// GetScreenshotData fetches raw screenshot bytes and content-type using the artifact's URL.
func (c *BrowserAutomationClient) GetScreenshotData(ctx context.Context, screenshotURL string) ([]byte, string, error) {
	return c.doRaw(ctx, screenshotURL)
}

// ExecuteAdhocWorkflow calls POST /api/v1/workflows/execute-adhoc on BAS.
// The adhoc endpoint always returns immediately — callers must poll for completion.
func (c *BrowserAutomationClient) ExecuteAdhocWorkflow(ctx context.Context, req BASExecuteAdhocRequest, requiresVideo bool) (*BASExecuteResponse, error) {
	path := "/api/v1/workflows/execute-adhoc"
	if requiresVideo {
		path += "?requires_video=true"
	}
	var result BASExecuteResponse
	err := c.doJSON(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetExecutionStatus calls GET /api/v1/executions/{id} on BAS.
func (c *BrowserAutomationClient) GetExecutionStatus(ctx context.Context, executionID string) (*BASExecutionDetail, error) {
	var result BASExecutionDetail
	err := c.doGet(ctx, "/api/v1/executions/"+executionID, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PollExecutionCompletion polls BAS until the execution reaches a terminal
// status (completed, failed, cancelled) or the context is cancelled.
func (c *BrowserAutomationClient) PollExecutionCompletion(ctx context.Context, executionID string, pollInterval time.Duration) (*BASExecutionDetail, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			detail, err := c.GetExecutionStatus(ctx, executionID)
			if err != nil {
				return nil, fmt.Errorf("poll execution %s: %w", executionID, err)
			}
			if isTerminalBASStatus(detail.Status) {
				return detail, nil
			}
		}
	}
}

// isTerminalBASStatus returns true for BAS execution statuses that indicate completion.
func isTerminalBASStatus(status string) bool {
	switch status {
	case "EXECUTION_STATUS_COMPLETED", "EXECUTION_STATUS_FAILED", "EXECUTION_STATUS_CANCELLED":
		return true
	default:
		return false
	}
}

// GetScreenshots calls GET /api/v1/executions/{id}/screenshots on BAS.
func (c *BrowserAutomationClient) GetScreenshots(ctx context.Context, executionID string) (*BASScreenshotsResponse, error) {
	var result BASScreenshotsResponse
	err := c.doGet(ctx, "/api/v1/executions/"+executionID+"/screenshots", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRecordedVideos calls GET /api/v1/executions/{id}/recorded-videos on BAS.
func (c *BrowserAutomationClient) GetRecordedVideos(ctx context.Context, executionID string) (*BASRecordedVideosResponse, error) {
	var result BASRecordedVideosResponse
	err := c.doGet(ctx, "/api/v1/executions/"+executionID+"/recorded-videos", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetVideoData fetches raw video bytes and content-type using the artifact's storage URL.
func (c *BrowserAutomationClient) GetVideoData(ctx context.Context, storageURL string) ([]byte, string, error) {
	return c.doRaw(ctx, storageURL)
}

func (c *BrowserAutomationClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.resolver != nil {
		return c.resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
	}
	return discovery.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
}

func (c *BrowserAutomationClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve BAS url: %w", err)
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
		return fmt.Errorf("BAS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseBASError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *BrowserAutomationClient) doGet(ctx context.Context, path string, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve BAS url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("BAS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseBASError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *BrowserAutomationClient) doRaw(ctx context.Context, path string) ([]byte, string, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("resolve BAS url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("BAS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", parseBASError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func parseBASError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("BAS error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("BAS error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("BAS error: status %d, body: %s", resp.StatusCode, string(body))
}
