package ssh

import (
	"encoding/json"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for SSH operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new SSH client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Keys lists all SSH keys.
func (c *Client) Keys() ([]byte, KeysResponse, error) {
	body, err := c.api.Get("/api/v1/ssh/keys", nil)
	if err != nil {
		return nil, KeysResponse{}, err
	}
	var resp KeysResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, KeysResponse{}, err
	}
	return body, resp, nil
}

// Generate creates a new SSH key.
func (c *Client) Generate(req GenerateRequest) ([]byte, GenerateResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/ssh/keys/generate", nil, req)
	if err != nil {
		return nil, GenerateResponse{}, err
	}
	var resp GenerateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, GenerateResponse{}, err
	}
	return body, resp, nil
}

// Delete removes an SSH key.
func (c *Client) Delete(name string) ([]byte, DeleteResponse, error) {
	query := url.Values{}
	query.Set("name", name)

	body, err := c.api.Request("DELETE", "/api/v1/ssh/keys", query, nil)
	if err != nil {
		return nil, DeleteResponse{}, err
	}
	var resp DeleteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeleteResponse{}, err
	}
	return body, resp, nil
}

// Test tests SSH connection to a host.
func (c *Client) Test(req TestRequest) ([]byte, TestResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/ssh/test", nil, req)
	if err != nil {
		return nil, TestResponse{}, err
	}
	var resp TestResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, TestResponse{}, err
	}
	return body, resp, nil
}

// CopyKey copies an SSH key to a remote host.
func (c *Client) CopyKey(req CopyKeyRequest) ([]byte, CopyKeyResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/ssh/copy-key", nil, req)
	if err != nil {
		return nil, CopyKeyResponse{}, err
	}
	var resp CopyKeyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, CopyKeyResponse{}, err
	}
	return body, resp, nil
}
