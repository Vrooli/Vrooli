package generate

import (
	"net/http"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/internal/apijson"
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
	resp, err := apijson.Parse[Response](body, "parse response")
	if err != nil {
		return Response{}, body, err
	}
	return resp, body, nil
}
