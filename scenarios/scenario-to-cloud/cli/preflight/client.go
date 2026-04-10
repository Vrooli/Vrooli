package preflight

import (
	"encoding/json"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for preflight operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new preflight client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// APIClient returns the underlying API client.
func (c *Client) APIClient() *cliutil.APIClient {
	return c.api
}

// Run executes preflight checks for a manifest.
// Returns raw bytes for JSON output and parsed response for formatted output.
func (c *Client) Run(manifest map[string]interface{}) ([]byte, Response, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight", nil, manifest)
	if err != nil {
		return nil, Response{}, err
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, Response{}, err
	}
	return body, resp, nil
}

// FixPorts fixes port conflicts.
func (c *Client) FixPorts(req FixPortsRequest) ([]byte, FixResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/fix/ports", nil, req)
	if err != nil {
		return nil, FixResponse{}, err
	}
	var resp FixResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FixResponse{}, err
	}
	return body, resp, nil
}

// FixFirewall fixes firewall rules.
func (c *Client) FixFirewall(req FixFirewallRequest) ([]byte, FixFirewallResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/fix/firewall", nil, req)
	if err != nil {
		return nil, FixFirewallResponse{}, err
	}
	var resp FixFirewallResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FixFirewallResponse{}, err
	}
	return body, resp, nil
}

// FixProcesses stops conflicting processes.
func (c *Client) FixProcesses(req FixProcessesRequest) ([]byte, FixProcessesResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/fix/stop-processes", nil, req)
	if err != nil {
		return nil, FixProcessesResponse{}, err
	}
	var resp FixProcessesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FixProcessesResponse{}, err
	}
	return body, resp, nil
}

// DiskUsage returns disk usage information.
func (c *Client) DiskUsage(req DiskUsageRequest) ([]byte, DiskUsageResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/disk/usage", nil, req)
	if err != nil {
		return nil, DiskUsageResponse{}, err
	}
	var resp DiskUsageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DiskUsageResponse{}, err
	}
	return body, resp, nil
}

// DiskCleanup cleans up disk space.
func (c *Client) DiskCleanup(req DiskCleanupRequest) ([]byte, DiskCleanupResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/disk/cleanup", nil, req)
	if err != nil {
		return nil, DiskCleanupResponse{}, err
	}
	var resp DiskCleanupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DiskCleanupResponse{}, err
	}
	return body, resp, nil
}

// Requirements returns canonical VPS requirements.
func (c *Client) Requirements() ([]byte, RequirementsResponse, error) {
	body, err := c.api.Get("/api/v1/preflight/requirements", nil)
	if err != nil {
		return nil, RequirementsResponse{}, err
	}
	var resp RequirementsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, RequirementsResponse{}, err
	}
	return body, resp, nil
}
