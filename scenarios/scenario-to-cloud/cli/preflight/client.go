package preflight

import (
	"encoding/json"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/transport"
)

// Client provides API access for preflight operations and resolves the
// deployment a fix targets through the deployments client.
type Client struct {
	api         *cliutil.APIClient
	Deployments *deployment.Client
}

// NewClient creates a new preflight client over the shared transport.
func NewClient(tr transport.Transport) *Client {
	return &Client{api: tr.API, Deployments: deployment.NewClient(tr)}
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
