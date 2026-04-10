package vps

import (
	"encoding/json"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for VPS operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new VPS client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// SetupPlan generates a VPS setup plan.
func (c *Client) SetupPlan(manifest map[string]interface{}, bundlePath string) ([]byte, SetupPlanResponse, error) {
	req := map[string]interface{}{
		"manifest":    manifest,
		"bundle_path": bundlePath,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/setup/plan", nil, req)
	if err != nil {
		return nil, SetupPlanResponse{}, err
	}
	var resp SetupPlanResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SetupPlanResponse{}, err
	}
	return body, resp, nil
}

// SetupApply executes VPS setup.
func (c *Client) SetupApply(manifest map[string]interface{}, bundlePath string) ([]byte, SetupApplyResponse, error) {
	req := map[string]interface{}{
		"manifest":    manifest,
		"bundle_path": bundlePath,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/setup/apply", nil, req)
	if err != nil {
		return nil, SetupApplyResponse{}, err
	}
	var resp SetupApplyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SetupApplyResponse{}, err
	}
	return body, resp, nil
}

// DeployPlan generates a VPS deploy plan.
func (c *Client) DeployPlan(manifest map[string]interface{}) ([]byte, DeployPlanResponse, error) {
	req := map[string]interface{}{
		"manifest": manifest,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/deploy/plan", nil, req)
	if err != nil {
		return nil, DeployPlanResponse{}, err
	}
	var resp DeployPlanResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeployPlanResponse{}, err
	}
	return body, resp, nil
}

// DeployApply executes VPS deploy.
func (c *Client) DeployApply(manifest map[string]interface{}) ([]byte, DeployApplyResponse, error) {
	req := map[string]interface{}{
		"manifest": manifest,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/deploy/apply", nil, req)
	if err != nil {
		return nil, DeployApplyResponse{}, err
	}
	var resp DeployApplyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeployApplyResponse{}, err
	}
	return body, resp, nil
}
