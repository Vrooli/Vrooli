package vps

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for VPS operations.
type Client struct {
	api *cliutil.APIClient
}

func (c *Client) InstancePlan(request map[string]interface{}) ([]byte, error) {
	return c.api.Request("POST", "/api/v1/instances/plan", nil, request)
}

func (c *Client) InstanceCreate(request map[string]interface{}) ([]byte, error) {
	return c.api.Request("POST", "/api/v1/instances", nil, request)
}

func (c *Client) InstanceAction(id, action, snapshot string) ([]byte, error) {
	path := "/api/v1/instances/" + id + "/" + action
	if snapshot != "" {
		path += "?name=" + url.QueryEscape(snapshot)
	}
	return c.api.Request("POST", path, nil, map[string]any{})
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

// SetupApply executes VPS setup. planDigest is the digest returned by the
// plan endpoint; when empty the plan is compiled first and its digest is
// submitted, so apply is a thin alias for "plan, then apply what was shown".
// A stale digest is refused by the API with plan_digest_mismatch.
func (c *Client) SetupApply(manifest map[string]interface{}, bundlePath, planDigest string) ([]byte, SetupApplyResponse, error) {
	if planDigest == "" {
		_, plan, err := c.SetupPlan(manifest, bundlePath)
		if err != nil {
			return nil, SetupApplyResponse{}, err
		}
		if plan.PlanDigest == "" {
			return nil, SetupApplyResponse{}, fmt.Errorf("setup plan response carried no plan_digest; cannot apply without a reviewed plan")
		}
		planDigest = plan.PlanDigest
	}
	req := map[string]interface{}{
		"manifest":    manifest,
		"bundle_path": bundlePath,
		"plan_digest": planDigest,
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

// DeployApply executes VPS deploy. See SetupApply for the planDigest rule.
func (c *Client) DeployApply(manifest map[string]interface{}, planDigest string) ([]byte, DeployApplyResponse, error) {
	if planDigest == "" {
		_, plan, err := c.DeployPlan(manifest)
		if err != nil {
			return nil, DeployApplyResponse{}, err
		}
		if plan.PlanDigest == "" {
			return nil, DeployApplyResponse{}, fmt.Errorf("deploy plan response carried no plan_digest; cannot apply without a reviewed plan")
		}
		planDigest = plan.PlanDigest
	}
	req := map[string]interface{}{
		"manifest":    manifest,
		"plan_digest": planDigest,
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
