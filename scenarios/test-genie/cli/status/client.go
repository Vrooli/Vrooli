package status

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access to health endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new status client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Check retrieves the current health status.
func (c *Client) Check() ([]byte, Response, error) {
	body, err := c.api.Get("/health", nil)
	if err != nil {
		return nil, Response{}, err
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, Response{}, fmt.Errorf("parse health response: %w", err)
	}
	return body, resp, nil
}
