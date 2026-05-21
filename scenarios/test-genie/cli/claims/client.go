package claims

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"test-genie/cli/internal/apijson"

	"github.com/vrooli/cli-core/cliutil"
)

// ErrNotFound indicates no claim exists for the scenario.
var ErrNotFound = errors.New("playbooks claim not found")

// Client provides API access to the playbooks claim endpoints.
type Client struct {
	api *cliutil.APIClient
}

// NewClient constructs a Client.
func NewClient(api *cliutil.APIClient) *Client { return &Client{api: api} }

// List returns every active claim.
func (c *Client) List() ([]Claim, error) {
	body, err := c.api.Request(http.MethodGet, "/api/v1/playbooks/claims", nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := apijson.Parse[listResponse](body, "parse claims list")
	if err != nil {
		return nil, err
	}
	return resp.Claims, nil
}

// Get returns the active claim for a scenario, or nil if none is held.
func (c *Client) Get(scenario string) (*Claim, error) {
	path := fmt.Sprintf("/api/v1/playbooks/claims/%s", url.PathEscape(scenario))
	body, err := c.api.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := apijson.Parse[getResponse](body, "parse claim")
	if err != nil {
		return nil, err
	}
	return resp.Claim, nil
}

// Release force-breaks the active claim for a scenario.
// The actor argument is reserved for future audit-header support and is
// currently only echoed back in CLI output.
func (c *Client) Release(scenario, actor string) (*Claim, error) {
	_ = actor
	path := fmt.Sprintf("/api/v1/playbooks/claims/%s/release", url.PathEscape(scenario))
	body, err := c.api.Request(http.MethodPost, path, nil, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, ErrNotFound
		}
		return nil, err
	}
	resp, err := apijson.Parse[releaseResponse](body, "parse release")
	if err != nil {
		return nil, err
	}
	return &resp.Released, nil
}
