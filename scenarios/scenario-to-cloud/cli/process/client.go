package process

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for process control operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new process client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Kill sends a signal to a process.
func (c *Client) Kill(deploymentID string, req KillRequest) ([]byte, KillResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/actions/kill", deploymentID), nil, req)
	if err != nil {
		return nil, KillResponse{}, err
	}
	var resp KillResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, KillResponse{}, err
	}
	return body, resp, nil
}

// Restart restarts a scenario or resource.
func (c *Client) Restart(deploymentID string, req RestartRequest) ([]byte, RestartResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/actions/restart", deploymentID), nil, req)
	if err != nil {
		return nil, RestartResponse{}, err
	}
	var resp RestartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, RestartResponse{}, err
	}
	return body, resp, nil
}

// Control performs process control actions (start/stop/restart).
func (c *Client) Control(deploymentID string, req ControlRequest) ([]byte, ControlResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/actions/process", deploymentID), nil, req)
	if err != nil {
		return nil, ControlResponse{}, err
	}
	var resp ControlResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ControlResponse{}, err
	}
	return body, resp, nil
}

// VPSAction performs VPS-level actions (reboot, shutdown, etc.).
func (c *Client) VPSAction(deploymentID string, req VPSActionRequest) ([]byte, VPSActionResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/actions/vps", deploymentID), nil, req)
	if err != nil {
		return nil, VPSActionResponse{}, err
	}
	var resp VPSActionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, VPSActionResponse{}, err
	}
	return body, resp, nil
}
