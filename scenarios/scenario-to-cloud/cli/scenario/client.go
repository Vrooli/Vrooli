package scenario

import (
	"encoding/json"
	"fmt"

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
