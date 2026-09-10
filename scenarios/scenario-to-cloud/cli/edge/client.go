package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	edgev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge/edgev1connect"

	"scenario-to-cloud/cli/internal/transport"
)

// Client provides API access for edge and TLS operations: the generated
// EdgeService for the typed observation, the REST client for the legacy
// DNS/Caddy/TLS actions.
type Client struct {
	api         *cliutil.APIClient
	Service     edgev1connect.EdgeServiceClient
	Deployments deploymentsv1connect.DeploymentsServiceClient
}

// NewClient creates a new edge client over the shared transport.
func NewClient(tr transport.Transport) *Client {
	return &Client{
		api:         tr.API,
		Service:     edgev1connect.NewEdgeServiceClient(tr.HTTP, tr.BaseURL),
		Deployments: deploymentsv1connect.NewDeploymentsServiceClient(tr.HTTP, tr.BaseURL),
	}
}

// Observation returns the typed edge observation (routes, private
// listeners, DNS, TLS, readiness).
func (c *Client) Observation(ctx context.Context, deploymentID string) (*edgev1.GetEdgeObservationResponse, error) {
	resp, err := c.Service.GetEdgeObservation(ctx, connect.NewRequest(&edgev1.GetEdgeObservationRequest{DeploymentId: deploymentID}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// DNSCheck checks DNS configuration for a deployment.
func (c *Client) DNSCheck(deploymentID string) ([]byte, DNSCheckResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/edge/dns-check", deploymentID), nil)
	if err != nil {
		return nil, DNSCheckResponse{}, err
	}
	var resp DNSCheckResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DNSCheckResponse{}, err
	}
	return body, resp, nil
}

// DNSRecords lists DNS records for a deployment.
func (c *Client) DNSRecords(deploymentID string) ([]byte, DNSRecordsResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/edge/dns-records", deploymentID), nil)
	if err != nil {
		return nil, DNSRecordsResponse{}, err
	}
	var resp DNSRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DNSRecordsResponse{}, err
	}
	return body, resp, nil
}

// Caddy performs Caddy control actions.
func (c *Client) Caddy(deploymentID string, req CaddyRequest) ([]byte, CaddyResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/edge/caddy", deploymentID), nil, req)
	if err != nil {
		return nil, CaddyResponse{}, err
	}
	var resp CaddyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, CaddyResponse{}, err
	}
	return body, resp, nil
}

// TLSInfo retrieves TLS certificate information.
func (c *Client) TLSInfo(deploymentID string) ([]byte, TLSInfoResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/edge/tls", deploymentID), nil)
	if err != nil {
		return nil, TLSInfoResponse{}, err
	}
	var resp TLSInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, TLSInfoResponse{}, err
	}
	return body, resp, nil
}

// TLSRenew triggers TLS certificate renewal.
func (c *Client) TLSRenew(deploymentID string, domain string, force bool) ([]byte, TLSRenewResponse, error) {
	query := url.Values{}
	if domain != "" {
		query.Set("domain", domain)
	}
	if force {
		query.Set("force", "true")
	}

	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/edge/tls/renew", deploymentID), query, nil)
	if err != nil {
		return nil, TLSRenewResponse{}, err
	}
	var resp TLSRenewResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, TLSRenewResponse{}, err
	}
	return body, resp, nil
}
