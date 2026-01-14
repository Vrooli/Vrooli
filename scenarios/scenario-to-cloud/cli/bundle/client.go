package bundle

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for bundle operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new bundle client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Build creates a mini-Vrooli bundle from a manifest.
// Returns raw bytes for JSON output and parsed response for formatted output.
func (c *Client) Build(manifest map[string]interface{}) ([]byte, BuildResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/bundle/build", nil, manifest)
	if err != nil {
		return nil, BuildResponse{}, err
	}
	var resp BuildResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, BuildResponse{}, err
	}
	return body, resp, nil
}

// List returns all stored bundles.
func (c *Client) List() ([]byte, ListResponse, error) {
	body, err := c.api.Get("/api/v1/bundles", nil)
	if err != nil {
		return nil, ListResponse{}, err
	}
	var resp ListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ListResponse{}, err
	}
	return body, resp, nil
}

// Stats returns bundle storage statistics.
func (c *Client) Stats() ([]byte, StatsResponse, error) {
	body, err := c.api.Get("/api/v1/bundles/stats", nil)
	if err != nil {
		return nil, StatsResponse{}, err
	}
	var resp StatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, StatsResponse{}, err
	}
	return body, resp, nil
}

// Delete removes a bundle by SHA256.
func (c *Client) Delete(sha256 string) ([]byte, DeleteResponse, error) {
	body, err := c.api.Request("DELETE", fmt.Sprintf("/api/v1/bundles/%s", sha256), nil, nil)
	if err != nil {
		return nil, DeleteResponse{}, err
	}
	var resp DeleteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeleteResponse{}, err
	}
	return body, resp, nil
}

// Cleanup removes old or orphaned bundles.
func (c *Client) Cleanup(req CleanupRequest) ([]byte, CleanupResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/bundles/cleanup", nil, req)
	if err != nil {
		return nil, CleanupResponse{}, err
	}
	var resp CleanupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, CleanupResponse{}, err
	}
	return body, resp, nil
}

// VPSList lists bundles on the VPS for a deployment.
func (c *Client) VPSList(manifest map[string]interface{}) ([]byte, VPSListResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/bundles/vps/list", nil, manifest)
	if err != nil {
		return nil, VPSListResponse{}, err
	}
	var resp VPSListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, VPSListResponse{}, err
	}
	return body, resp, nil
}

// VPSDelete deletes bundles from the VPS.
func (c *Client) VPSDelete(manifest map[string]interface{}, req VPSDeleteRequest) ([]byte, VPSDeleteResponse, error) {
	payload := map[string]interface{}{
		"manifest": manifest,
		"options":  req,
	}
	body, err := c.api.Request("POST", "/api/v1/bundles/vps/delete", nil, payload)
	if err != nil {
		return nil, VPSDeleteResponse{}, err
	}
	var resp VPSDeleteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, VPSDeleteResponse{}, err
	}
	return body, resp, nil
}
