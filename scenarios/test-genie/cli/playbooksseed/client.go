package playbooksseed

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access to playbooks seed lifecycle endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new seed client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Apply requests playbooks seed apply for a scenario.
func (c *Client) Apply(scenario string, req ApplyRequest) (ApplyResponse, []byte, error) {
	path := fmt.Sprintf("/api/v1/scenarios/%s/playbooks/seed/apply", url.PathEscape(scenario))
	body, err := c.api.Request(http.MethodPost, path, nil, req)
	if err != nil {
		return ApplyResponse{}, nil, err
	}
	var resp ApplyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ApplyResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}

// Cleanup requests playbooks seed cleanup for a scenario.
func (c *Client) Cleanup(scenario string, req CleanupRequest) (CleanupResponse, []byte, error) {
	path := fmt.Sprintf("/api/v1/scenarios/%s/playbooks/seed/cleanup", url.PathEscape(scenario))
	body, err := c.api.Request(http.MethodPost, path, nil, req)
	if err != nil {
		return CleanupResponse{}, nil, err
	}
	var resp CleanupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return CleanupResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}
