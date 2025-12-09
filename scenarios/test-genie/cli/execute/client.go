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
	"time"

	"github.com/vrooli/cli-core/cliutil"

	execTypes "test-genie/cli/internal/execute"
	"test-genie/cli/internal/phases"
)

// Note: os import is still needed for TEST_GENIE_EXECUTION_TIMEOUT env var check

// Default timeout for test execution (15 minutes to handle long test suites)
const defaultExecutionTimeout = 15 * time.Minute

// Client provides API access to execution endpoints.
type Client struct {
	api             *cliutil.APIClient
	httpClient      *cliutil.HTTPClient
	executionClient *http.Client
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultExecutionTimeout)
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

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return Response{}, body, fmt.Errorf("parse response: %w", err)
	}
	return result, body, nil
}

type PhaseSettings struct {
	Items   []phases.Descriptor              `json:"items"`
	Toggles map[string]execTypes.PhaseToggle `json:"toggles"`
}

// ListPhases retrieves the phase catalog and toggle metadata from the server.
func (c *Client) ListPhases() (PhaseSettings, error) {
	var payload PhaseSettings
	body, err := c.api.Get("/api/v1/phases", nil)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return PhaseSettings{}, fmt.Errorf("parse phase descriptors: %w", err)
	}
	if payload.Toggles == nil {
		payload.Toggles = map[string]execTypes.PhaseToggle{}
	}
	return payload, nil
}

// resolveBaseURL gets the base URL from the configured HTTP client.
// The HTTP client is initialized by cli-core with proper port detection via vrooli.
func (c *Client) resolveBaseURL() string {
	if c.httpClient != nil {
		return c.httpClient.BaseURL()
	}
	return ""
}

// extractErrorMessage pulls an error message from a JSON response
func extractErrorMessage(data []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		if errObj, ok := parsed["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				return msg
			}
		}
		if msg, ok := parsed["message"].(string); ok {
			return msg
		}
	}
	if len(data) > 200 {
		return string(data[:200]) + "..."
	}
	return string(data)
}
