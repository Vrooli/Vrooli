package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"test-genie/cli/internal/apijson"

	"github.com/vrooli/cli-core/cliutil"

	execTypes "test-genie/cli/internal/execute"
)

// Note: os import is still needed for TEST_GENIE_EXECUTION_TIMEOUT env var check

// Default timeout for test execution (15 minutes to handle long test suites)
const defaultExecutionTimeout = 15 * time.Minute

// Client provides API access to execution endpoints.
type Client struct {
	api             *cliutil.APIClient
	httpClient      *cliutil.HTTPClient
	executionClient *http.Client
	timeout         time.Duration
}

// NewClient creates a new execution client.
func NewClient(api *cliutil.APIClient, httpClient *cliutil.HTTPClient) *Client {
	timeout := defaultExecutionTimeout
	// Allow override via environment variable
	if envTimeout := os.Getenv("TEST_GENIE_EXECUTION_TIMEOUT"); envTimeout != "" {
		if secs, err := strconv.Atoi(envTimeout); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return &Client{
		api:             api,
		httpClient:      httpClient,
		executionClient: &http.Client{Timeout: timeout},
		timeout:         timeout,
	}
}

// Run submits an execution request with an extended timeout suitable for test suites.
func (c *Client) Run(req Request) (Response, []byte, error) {
	// Get base URL from the api client
	baseURL := c.resolveBaseURL()
	if baseURL == "" {
		return Response{}, nil, fmt.Errorf("api base URL not configured")
	}

	// Marshal request body
	payload, err := json.Marshal(req)
	if err != nil {
		return Response{}, nil, fmt.Errorf("encode request: %w", err)
	}

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/executions", bytes.NewReader(payload))
	if err != nil {
		return Response{}, nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute with the extended timeout client
	resp, err := c.executionClient.Do(httpReq)
	if err != nil {
		return Response{}, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return Response{}, body, fmt.Errorf("api error (%d): %s", resp.StatusCode, extractErrorMessage(body))
	}

	result, err := parseRunResponse(body)
	if err != nil {
		return Response{}, body, err
	}
	return result, body, nil
}

// resolveBaseURL gets the base URL from the configured HTTP client.
// The HTTP client is initialized by cli-core with proper port detection via vrooli.
func (c *Client) resolveBaseURL() string {
	if c.httpClient != nil {
		if base := c.httpClient.BaseURL(); base != "" {
			return base
		}
	}
	if c.api != nil {
		return c.api.BaseURL()
	}
	return ""
}

// extractErrorMessage pulls an error message from a JSON response
func extractErrorMessage(data []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		parts := make([]string, 0, 2)
		if errObj, ok := parsed["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				parts = append(parts, msg)
			}
		}
		if msg, ok := parsed["message"].(string); ok {
			parts = append(parts, msg)
		}
		if msg, ok := parsed["error"].(string); ok {
			parts = append(parts, msg)
		}
		if details, ok := parsed["errors"].([]interface{}); ok {
			for _, detail := range details {
				if msg, ok := detail.(string); ok && msg != "" {
					parts = append(parts, msg)
				}
			}
		}
		if len(parts) > 0 {
			return joinUnique(parts, ": ")
		}
	}
	if len(data) > 200 {
		return string(data[:200]) + "..."
	}
	return string(data)
}

func joinUnique(parts []string, sep string) string {
	seen := make(map[string]struct{}, len(parts))
	unique := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		unique = append(unique, part)
	}
	return strings.Join(unique, sep)
}

func parseRunResponse(body []byte) (Response, error) {
	return apijson.Parse[Response](body, "parse response")
}

// PreviewPlan resolves the actual phase plan and timing guidance for an execution request.
func (c *Client) PreviewPlan(req Request) (execTypes.PlanPreview, error) {
	baseURL := c.resolveBaseURL()
	if baseURL == "" {
		return execTypes.PlanPreview{}, fmt.Errorf("api base URL not configured")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return execTypes.PlanPreview{}, fmt.Errorf("encode request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/executions/plan", bytes.NewReader(payload))
	if err != nil {
		return execTypes.PlanPreview{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.executionClient.Do(httpReq)
	if err != nil {
		return execTypes.PlanPreview{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return execTypes.PlanPreview{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return execTypes.PlanPreview{}, fmt.Errorf("api error (%d): %s", resp.StatusCode, extractErrorMessage(body))
	}

	return apijson.Parse[execTypes.PlanPreview](body, "parse execution plan preview")
}
