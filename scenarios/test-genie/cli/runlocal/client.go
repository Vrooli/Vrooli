package runlocal

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/internal/apijson"
)

// Client provides API access to local test runner endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new runlocal client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Run triggers a local test run for the specified scenario.
func (c *Client) Run(scenario string, req Request) (Response, []byte, error) {
	path := fmt.Sprintf("/api/v1/scenarios/%s/run-tests", url.PathEscape(scenario))
	body, err := c.api.Request(http.MethodPost, path, nil, req)
	if err != nil {
		return Response{}, nil, err
	}
	resp, err := apijson.Parse[Response](body, "parse response")
	if err != nil {
		return Response{}, body, err
	}
	return resp, body, nil
}
