package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for scenario operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new scenario client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// List returns all available scenarios.
func (c *Client) List() ([]byte, ListResponse, error) {
	body, err := c.api.Get("/api/v1/scenarios", nil)
	if err != nil {
		return nil, ListResponse{}, err
	}
	var resp ListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ListResponse{}, err
	}
	return body, resp, nil
}

// Ports returns port allocations for a scenario.
func (c *Client) Ports(scenarioID string) ([]byte, PortsResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/scenarios/%s/ports", scenarioID), nil)
	if err != nil {
		return nil, PortsResponse{}, err
	}
	var resp PortsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, PortsResponse{}, err
	}
	return body, resp, nil
}

// Dependencies returns dependencies for a scenario.
func (c *Client) Dependencies(scenarioID string) ([]byte, DepsResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/scenarios/%s/dependencies", scenarioID), nil)
	if err != nil {
		return nil, DepsResponse{}, err
	}
	var resp DepsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DepsResponse{}, err
	}
	return body, resp, nil
}

// DeploymentReport returns deployment sizing and dependency graph metadata for a scenario.
func (c *Client) DeploymentReport(ctx context.Context, scenarioID string) ([]byte, DeploymentReportResponse, error) {
	baseURL, err := cliutil.ResolveScenarioAPIBaseURL(ctx, cliutil.ScenarioAPIBaseOptions{
		Target:  "scenario-dependency-analyzer",
		EnvVars: []string{"SCENARIO_DEPENDENCY_ANALYZER_API_BASE_URL"},
	})
	if err != nil {
		return nil, DeploymentReportResponse{}, err
	}

	endpoint, err := url.JoinPath(baseURL, "api/v1/scenarios", url.PathEscape(scenarioID), "deployment")
	if err != nil {
		return nil, DeploymentReportResponse{}, fmt.Errorf("build analyzer endpoint: %w", err)
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return nil, DeploymentReportResponse{}, fmt.Errorf("parse analyzer endpoint: %w", err)
	}
	query := parsedEndpoint.Query()
	query.Set("refresh", "true")
	parsedEndpoint.RawQuery = query.Encode()
	endpoint = parsedEndpoint.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, DeploymentReportResponse{}, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, DeploymentReportResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, DeploymentReportResponse{}, fmt.Errorf("analyzer returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, DeploymentReportResponse{}, err
	}

	var report DeploymentReportResponse
	if err := json.Unmarshal(body, &report); err != nil {
		return body, DeploymentReportResponse{}, err
	}
	return body, report, nil
}
