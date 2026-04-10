package status

import (
	"encoding/json"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for status operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new status client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Check fetches the health status from the API.
// Returns raw bytes for JSON output and parsed response for formatted output.
func (c *Client) Check() ([]byte, Response, error) {
	body, err := c.api.Get("/health", url.Values{})
	if err != nil {
		return nil, Response{}, err
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, Response{}, err
	}
	return body, resp, nil
}
