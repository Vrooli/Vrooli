package uismoke

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access to UI smoke test endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new UI smoke client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Run executes a UI smoke test for the specified scenario.
func (c *Client) Run(scenario string, req Request) (Response, []byte, error) {
	path := fmt.Sprintf("/api/v1/scenarios/%s/ui-smoke", url.PathEscape(scenario))
	body, err := c.api.Request(http.MethodPost, path, nil, req)
	if err != nil {
		return Response{}, nil, err
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return Response{}, body, fmt.Errorf("parse ui smoke response: %w", err)
	}
	return resp, body, nil
}
