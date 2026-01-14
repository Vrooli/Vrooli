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
