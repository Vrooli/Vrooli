package manifest

import (
	"encoding/json"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for manifest operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new manifest client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Validate sends a manifest to the API for validation.
// Returns raw bytes for JSON output and parsed response for formatted output.
func (c *Client) Validate(manifest map[string]interface{}) ([]byte, ValidateResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/manifest/validate", nil, manifest)
	if err != nil {
		return nil, ValidateResponse{}, err
	}
	var resp ValidateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ValidateResponse{}, err
	}
	return body, resp, nil
}
