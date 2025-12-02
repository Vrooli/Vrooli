package generate

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access to suite generation endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new generation client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Create submits a new suite generation request.
func (c *Client) Create(req Request) (Response, []byte, error) {
	body, err := c.api.Request(http.MethodPost, "/api/v1/suite-requests", nil, req)
	if err != nil {
		return Response{}, nil, err
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return Response{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}
