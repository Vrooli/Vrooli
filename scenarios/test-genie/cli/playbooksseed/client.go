package playbooksseed

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/internal/apijson"
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
	resp, err := apijson.Parse[ApplyResponse](body, "parse response")
	if err != nil {
		return ApplyResponse{}, body, err
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
	resp, err := apijson.Parse[CleanupResponse](body, "parse response")
	if err != nil {
		return CleanupResponse{}, body, err
	}
	return resp, body, nil
}
