package edge

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for edge and TLS operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new edge client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
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
