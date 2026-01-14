package secrets

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for secrets operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new secrets client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Get retrieves secrets for a scenario.
func (c *Client) Get(scenarioID string, reveal bool) ([]byte, GetResponse, error) {
	query := url.Values{}
	if reveal {
		query.Set("reveal", "true")
	}

	body, err := c.api.Get(fmt.Sprintf("/api/v1/secrets/%s", scenarioID), query)
	if err != nil {
		return nil, GetResponse{}, err
	}
	var resp GetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, GetResponse{}, err
	}
	return body, resp, nil
}
