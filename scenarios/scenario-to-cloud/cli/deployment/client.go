package deployment

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for deployment operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new deployment client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Plan generates a deployment plan from a manifest.
func (c *Client) Plan(manifest map[string]interface{}) ([]byte, PlanResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/plan", nil, manifest)
	if err != nil {
		return nil, PlanResponse{}, err
	}
	var resp PlanResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, PlanResponse{}, err
	}
	return body, resp, nil
}

// Create creates a new deployment from a manifest.
func (c *Client) Create(req CreateRequest) ([]byte, CreateResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/deployments", nil, req)
	if err != nil {
		return nil, CreateResponse{}, err
	}
	var resp CreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, CreateResponse{}, err
	}
	return body, resp, nil
}

// List returns all deployments with optional filtering.
func (c *Client) List(opts ListOptions) ([]byte, ListResponse, error) {
	query := url.Values{}
	if opts.Status != "" {
		query.Set("status", opts.Status)
	}
	if opts.ScenarioID != "" {
		query.Set("scenario_id", opts.ScenarioID)
	}

	body, err := c.api.Get("/api/v1/deployments", query)
	if err != nil {
		return nil, ListResponse{}, err
	}
	var resp ListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ListResponse{}, err
	}
	return body, resp, nil
}

// Get returns a single deployment by ID.
func (c *Client) Get(id string) ([]byte, GetResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s", id), nil)
	if err != nil {
		return nil, GetResponse{}, err
	}
	var resp GetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, GetResponse{}, err
	}
	return body, resp, nil
}

// Delete removes a deployment by ID.
func (c *Client) Delete(id string, opts DeleteOptions) ([]byte, DeleteResponse, error) {
	query := url.Values{}
	if opts.Stop {
		query.Set("stop", "true")
	}
	if opts.Cleanup {
		query.Set("cleanup", "true")
	}

	body, err := c.api.Request("DELETE", fmt.Sprintf("/api/v1/deployments/%s", id), query, nil)
	if err != nil {
		return nil, DeleteResponse{}, err
	}
	var resp DeleteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeleteResponse{}, err
	}
	return body, resp, nil
}

// Execute starts the deployment pipeline.
func (c *Client) Execute(id string, req ExecuteRequest) ([]byte, ExecuteResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/execute", id), nil, req)
	if err != nil {
		return nil, ExecuteResponse{}, err
	}
	var resp ExecuteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ExecuteResponse{}, err
	}
	return body, resp, nil
}

// Start resumes a stopped deployment.
func (c *Client) Start(id string, req ExecuteRequest) ([]byte, StartResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/start", id), nil, req)
	if err != nil {
		return nil, StartResponse{}, err
	}
	var resp StartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, StartResponse{}, err
	}
	return body, resp, nil
}

// Stop stops a running deployment.
func (c *Client) Stop(id string) ([]byte, StopResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/stop", id), nil, nil)
	if err != nil {
		return nil, StopResponse{}, err
	}
	var resp StopResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, StopResponse{}, err
	}
	return body, resp, nil
}

// History returns the deployment history events.
func (c *Client) History(id string) ([]byte, HistoryResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/history", id), nil)
	if err != nil {
		return nil, HistoryResponse{}, err
	}
	var resp HistoryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, HistoryResponse{}, err
	}
	return body, resp, nil
}
