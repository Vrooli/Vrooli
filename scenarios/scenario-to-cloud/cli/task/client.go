package task

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for task operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new task client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Create creates a new AI task for a deployment.
func (c *Client) Create(deploymentID string, req CreateRequest) ([]byte, CreateResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/tasks", deploymentID), nil, req)
	if err != nil {
		return nil, CreateResponse{}, err
	}
	var resp CreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, CreateResponse{}, err
	}
	return body, resp, nil
}

// List returns all tasks for a deployment.
func (c *Client) List(deploymentID string) ([]byte, ListResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/tasks", deploymentID), nil)
	if err != nil {
		return nil, ListResponse{}, err
	}
	var resp ListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ListResponse{}, err
	}
	return body, resp, nil
}

// Get returns a specific task.
func (c *Client) Get(deploymentID, taskID string) ([]byte, GetResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/tasks/%s", deploymentID, taskID), nil)
	if err != nil {
		return nil, GetResponse{}, err
	}
	var resp GetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, GetResponse{}, err
	}
	return body, resp, nil
}

// Stop stops a running task.
func (c *Client) Stop(deploymentID, taskID string) ([]byte, StopResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/tasks/%s/stop", deploymentID, taskID), nil, nil)
	if err != nil {
		return nil, StopResponse{}, err
	}
	var resp StopResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, StopResponse{}, err
	}
	return body, resp, nil
}

// AgentStatus returns the AI agent status.
func (c *Client) AgentStatus() ([]byte, AgentStatusResponse, error) {
	body, err := c.api.Get("/api/v1/agent-manager/status", nil)
	if err != nil {
		return nil, AgentStatusResponse{}, err
	}
	var resp AgentStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, AgentStatusResponse{}, err
	}
	return body, resp, nil
}
