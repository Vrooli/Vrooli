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
func (c *Client) FixFirewall(req FixFirewallRequest) ([]byte, FixResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/fix/firewall", nil, req)
	if err != nil {
		return nil, FixResponse{}, err
	}
	var resp FixResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FixResponse{}, err
	}
	return body, resp, nil
}

// FixProcesses stops conflicting processes.
func (c *Client) FixProcesses(req FixProcessesRequest) ([]byte, FixResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/fix/stop-processes", nil, req)
	if err != nil {
		return nil, FixResponse{}, err
	}
	var resp FixResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FixResponse{}, err
	}
	return body, resp, nil
}

// DiskUsage returns disk usage information.
func (c *Client) DiskUsage() ([]byte, DiskUsageResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/preflight/disk/usage", nil, nil)
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
