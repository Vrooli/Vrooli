package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for scenario operations.
type Client struct {
	api *cliutil.APIClient
}

var resolveAnalyzerBaseURL = defaultResolveAnalyzerBaseURL

// SetResolveAnalyzerBaseURLForTest overrides analyzer URL resolution for tests.
func SetResolveAnalyzerBaseURLForTest(fn func(ctx context.Context) (string, error)) {
	resolveAnalyzerBaseURL = fn
}

// ResolveAnalyzerBaseURLForTest returns the current analyzer URL resolver.
func ResolveAnalyzerBaseURLForTest() func(ctx context.Context) (string, error) {
	return resolveAnalyzerBaseURL
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
	baseURL, err := resolveAnalyzerBaseURL(ctx)
	if err != nil {
		return nil, DeploymentReportResponse{}, err
	}

	endpoint := fmt.Sprintf("%s/api/v1/scenarios/%s/deployment?refresh=true", strings.TrimSuffix(baseURL, "/"), url.PathEscape(scenarioID))
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

func defaultResolveAnalyzerBaseURL(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "scenario-dependency-analyzer", "API_PORT")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve analyzer port: %w", err)
	}
	portStr := strings.TrimSpace(string(out))
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return "", fmt.Errorf("invalid analyzer port: %s", portStr)
	}
	return fmt.Sprintf("http://localhost:%d", port), nil
}
