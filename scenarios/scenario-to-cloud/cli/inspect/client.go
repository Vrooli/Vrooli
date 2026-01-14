package inspect

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for inspection operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new inspect client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Plan generates an inspection plan.
func (c *Client) Plan(manifest map[string]interface{}, opts Options) ([]byte, PlanResponse, error) {
	req := map[string]interface{}{
		"manifest": manifest,
		"options":  opts,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/inspect/plan", nil, req)
	if err != nil {
		return nil, PlanResponse{}, err
	}
	var resp PlanResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, PlanResponse{}, err
	}
	return body, resp, nil
}

// Apply executes the inspection.
func (c *Client) Apply(manifest map[string]interface{}, opts Options) ([]byte, ApplyResponse, error) {
	req := map[string]interface{}{
		"manifest": manifest,
		"options":  opts,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/inspect/apply", nil, req)
	if err != nil {
		return nil, ApplyResponse{}, err
	}
	var resp ApplyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ApplyResponse{}, err
	}
	return body, resp, nil
}

// LiveState retrieves the live state of a deployment.
func (c *Client) LiveState(id string) ([]byte, LiveStateResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/live-state", id), nil)
	if err != nil {
		return nil, LiveStateResponse{}, err
	}
	var resp LiveStateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LiveStateResponse{}, err
	}
	return body, resp, nil
}

// Drift retrieves drift detection results for a deployment.
func (c *Client) Drift(id string) ([]byte, DriftResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/drift", id), nil)
	if err != nil {
		return nil, DriftResponse{}, err
	}
	var resp DriftResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DriftResponse{}, err
	}
	return body, resp, nil
}

// Logs retrieves aggregated logs for a deployment.
func (c *Client) Logs(id string, opts LogsOptions) ([]byte, LogsResponse, error) {
	query := url.Values{}
	if opts.Source != "" {
		query.Set("source", opts.Source)
	}
	if opts.Level != "" {
		query.Set("level", opts.Level)
	}
	if opts.Search != "" {
		query.Set("search", opts.Search)
	}
	if opts.Tail > 0 {
		query.Set("tail", strconv.Itoa(opts.Tail))
	}
	if opts.Since != "" {
		query.Set("since", opts.Since)
	}

	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/logs", id), query)
	if err != nil {
		return nil, LogsResponse{}, err
	}
	var resp LogsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LogsResponse{}, err
	}
	return body, resp, nil
}

// Files lists files or reads file content from a deployment.
func (c *Client) Files(id string, opts FilesOptions) ([]byte, FilesResponse, error) {
	query := url.Values{}
	if opts.Path != "" {
		query.Set("path", opts.Path)
	}
	if opts.Content {
		query.Set("content", "true")
	}

	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/files", id), query)
	if err != nil {
		return nil, FilesResponse{}, err
	}
	var resp FilesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FilesResponse{}, err
	}
	return body, resp, nil
}
