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

	"test-genie/cli/internal/phases"
)

// Default timeout for test execution (15 minutes to handle long test suites)
const defaultExecutionTimeout = 15 * time.Minute

// Client provides API access to execution endpoints.
type Client struct {
	api             *cliutil.APIClient
	executionClient *http.Client
	baseURL         string
}

// NewClient creates a new execution client.
func NewClient(api *cliutil.APIClient) *Client {
	timeout := defaultExecutionTimeout
	// Allow override via environment variable
	if envTimeout := os.Getenv("TEST_GENIE_EXECUTION_TIMEOUT"); envTimeout != "" {
		if secs, err := strconv.Atoi(envTimeout); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return &Client{
		api:             api,
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

// ListPhases retrieves the phase catalog from the server.
func (c *Client) ListPhases() ([]phases.Descriptor, error) {
	body, err := c.api.Get("/api/v1/phases", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []phases.Descriptor `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse phase descriptors: %w", err)
	}
	return payload.Items, nil
}

// resolveBaseURL gets the base URL from environment or the api client
func (c *Client) resolveBaseURL() string {
	// Check common environment variables
	for _, envVar := range []string{"TEST_GENIE_API_BASE", "TEST_GENIE_API_URL"} {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
	}
	// Try to detect from port registry
	if port := os.Getenv("TEST_GENIE_API_PORT"); port != "" {
		return "http://localhost:" + port
	}
	// Fallback to default port
	return "http://localhost:17480"
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
