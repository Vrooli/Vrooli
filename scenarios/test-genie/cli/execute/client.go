package execute

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/internal/phases"
)

// Client provides API access to execution endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new execution client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Run submits an execution request.
func (c *Client) Run(req Request) (Response, []byte, error) {
	body, err := c.api.Request(http.MethodPost, "/api/v1/executions", nil, req)
	if err != nil {
		return Response{}, nil, err
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return Response{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}

// ListPhases retrieves the phase catalog from the server.
func (c *Client) ListPhases() ([]phases.Descriptor, error) {
	body, err := c.api.Get("/api/v1/phases", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []phases.Descriptor `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse phase descriptors: %w", err)
	}
	return payload.Items, nil
}
