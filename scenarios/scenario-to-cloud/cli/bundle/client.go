package bundle

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"

	"scenario-to-cloud/cli/internal/transport"
)

// Client provides API access for bundle operations; deployment-scoped
// bundle commands resolve their selector through the deployments service.
type Client struct {
	api         *cliutil.APIClient
	Deployments deploymentsv1connect.DeploymentsServiceClient
}

// NewClient creates a new bundle client over the shared transport.
func NewClient(tr transport.Transport) *Client {
	return &Client{api: tr.API, Deployments: deploymentsv1connect.NewDeploymentsServiceClient(tr.HTTP, tr.BaseURL)}
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

// DeploymentVPSList lists bundles on the VPS bundle cache for a specific deployment ID.
func (c *Client) DeploymentVPSList(deploymentID string) ([]byte, DeploymentVPSListResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/bundles/vps", deploymentID), nil)
	if err != nil {
		return nil, DeploymentVPSListResponse{}, err
	}
	var resp DeploymentVPSListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeploymentVPSListResponse{}, err
	}
	return body, resp, nil
}

// DeploymentVPSGC garbage-collects VPS bundle cache for a specific deployment ID.
func (c *Client) DeploymentVPSGC(deploymentID string, req VPSBundleGCRequest) ([]byte, VPSBundleGCResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/bundles/vps/gc", deploymentID), nil, req)
	if err != nil {
		return nil, VPSBundleGCResponse{}, err
	}
	var resp VPSBundleGCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, VPSBundleGCResponse{}, err
	}
	return body, resp, nil
}
