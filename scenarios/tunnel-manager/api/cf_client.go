package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// CFClient interacts with the Cloudflare API v4 for tunnel management. [REQ:CFAPI-001]
type CFClient struct {
	apiToken  string
	accountID string
	tunnelID  string
	baseURL   string
	client    *http.Client
}

type CFClientOption func(*CFClient)

func WithCFBaseURL(url string) CFClientOption {
	return func(c *CFClient) { c.baseURL = url }
}

func WithCFHTTPClient(hc *http.Client) CFClientOption {
	return func(c *CFClient) { c.client = hc }
}

// NewCFClient creates a Cloudflare API client. [REQ:CFAPI-001]
// Token is sourced from the CF_API_TOKEN environment variable or passed directly.
func NewCFClient(opts ...CFClientOption) (*CFClient, error) {
	c := &CFClient{
		apiToken:  os.Getenv("CF_API_TOKEN"),
		accountID: os.Getenv("CF_ACCOUNT_ID"),
		tunnelID:  os.Getenv("CF_TUNNEL_ID"),
		baseURL:   "https://api.cloudflare.com/client/v4",
		client:    &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.apiToken == "" {
		return nil, fmt.Errorf("CF_API_TOKEN is required")
	}
	if c.accountID == "" {
		return nil, fmt.Errorf("CF_ACCOUNT_ID is required")
	}
	if c.tunnelID == "" {
		return nil, fmt.Errorf("CF_TUNNEL_ID is required")
	}
	return c, nil
}

// CFTunnelConfig represents the tunnel configuration from the API. [REQ:CFAPI-002]
type CFTunnelConfig struct {
	Config struct {
		Ingress []CFIngressRule `json:"ingress"`
	} `json:"config"`
}

// CFIngressRule represents a Cloudflare ingress rule.
type CFIngressRule struct {
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service"`
}

// CFTunnelStatus represents the tunnel status from the API. [REQ:CFAPI-005]
type CFTunnelStatus struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	Connections []struct {
		ColoName    string `json:"colo_name"`
		IsAlive     bool   `json:"is_alive"`
		OpenedAt    string `json:"opened_at"`
		OriginIP    string `json:"origin_ip"`
		ConnectorID string `json:"connector_id"`
	} `json:"connections"`
}

// tunnelURL builds the Cloudflare API tunnel URL with an optional suffix path.
func (c *CFClient) tunnelURL(suffix string) string {
	return fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s%s", c.baseURL, c.accountID, c.tunnelID, suffix)
}

// doGet performs a GET request, unmarshals the standard CF response envelope,
// and checks the success field.
func doGet[T any](c *CFClient, ctx context.Context, url, label string) (*T, error) {
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	var envelope struct {
		Success bool `json:"success"`
		Result  T    `json:"result"`
	}
	if err := json.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parse %s: %w", label, err)
	}
	if !envelope.Success {
		return nil, fmt.Errorf("cloudflare API returned success=false for %s", label)
	}
	return &envelope.Result, nil
}

// ReadConfig fetches the current tunnel configuration. [REQ:CFAPI-002]
func (c *CFClient) ReadConfig(ctx context.Context) (*CFTunnelConfig, error) {
	return doGet[CFTunnelConfig](c, ctx, c.tunnelURL("/configurations"), "tunnel config")
}

// PushConfig updates the tunnel ingress configuration. [REQ:CFAPI-003]
// Uses read-modify-write pattern with catch-all 404 as last rule.
func (c *CFClient) PushConfig(ctx context.Context, rules []CFIngressRule) error {
	// Ensure catch-all rule is last
	hasCatchAll := false
	for _, r := range rules {
		if r.Hostname == "" {
			hasCatchAll = true
			break
		}
	}
	if !hasCatchAll {
		rules = append(rules, CFIngressRule{Service: "http_status:404"})
	}

	payload := map[string]any{
		"config": map[string]any{
			"ingress": rules,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	_, err = c.doRequest(ctx, "PUT", c.tunnelURL("/configurations"), body)
	if err != nil {
		return fmt.Errorf("push tunnel config: %w", err)
	}
	return nil
}

// GetTunnelStatus fetches the tunnel's current status. [REQ:CFAPI-005]
func (c *CFClient) GetTunnelStatus(ctx context.Context) (*CFTunnelStatus, error) {
	return doGet[CFTunnelStatus](c, ctx, c.tunnelURL(""), "tunnel status")
}

// RoutesToCFRules converts routes to Cloudflare ingress rules. [REQ:CFAPI-003]
func RoutesToCFRules(routes []Route) []CFIngressRule {
	var rules []CFIngressRule
	for _, r := range routes {
		if !r.Enabled {
			continue
		}
		hostname := r.Subdomain + ".vrooli.com"
		if r.PublicURL != "" {
			hostname = extractHostname(r.PublicURL)
		}
		rules = append(rules, CFIngressRule{
			Hostname: hostname,
			Service:  fmt.Sprintf("http://localhost:%d", r.LocalPort),
		})
	}
	// Catch-all
	rules = append(rules, CFIngressRule{Service: "http_status:404"})
	return rules
}

func (c *CFClient) doRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}
