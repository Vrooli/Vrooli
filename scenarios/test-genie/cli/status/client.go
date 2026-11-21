package status

import (
	"test-genie/cli/internal/apijson"

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
	resp, err := apijson.Parse[Response](body, "parse health response")
	if err != nil {
		return body, Response{}, err
	}
	return body, resp, nil
}
