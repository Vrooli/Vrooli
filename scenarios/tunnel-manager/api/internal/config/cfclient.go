package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tunnel-manager/internal/httpc"
)

// cfClient is the production IngressClient over the Cloudflare API v4
// tunnel configurations endpoint. It is built over httpc.Doer (an
// *http.Client in production) plus the tunnel credentials, so service
// tests substitute a fake IngressClient instead of a fake transport.
//
// Ported from the old adapter/cloudflare.go (ReadConfig/PushConfig against
// accounts/<acct>/cfd_tunnel/<tid>/configurations) and re-wrapped behind
// the IngressClient seam.
type cfClient struct {
	doer      httpc.Doer
	apiToken  string
	accountID string
	tunnelID  string
	baseURL   string
}

type cloudflareBootstrapClient struct {
	doer    httpc.Doer
	baseURL string
}

// NewCloudflareBootstrapAPI exposes the account/tunnel bootstrap calls behind
// the same HTTP seam as ingress management. The API token is supplied per
// operation and is never retained in the client configuration.
func NewCloudflareBootstrapAPI(doer httpc.Doer, baseURL string) CloudflareBootstrapAPI {
	if doer == nil {
		doer = http.DefaultClient
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.cloudflare.com/client/v4"
	}
	return &cloudflareBootstrapClient{doer: doer, baseURL: strings.TrimRight(baseURL, "/")}
}

// CFConfig carries the Cloudflare credentials the cfClient needs.
type CFConfig struct {
	APIToken       string // #nosec G101 -- field carries an in-memory value from env/file; no hardcoded credential.
	AccountID      string
	TunnelID       string
	ConnectorToken string
	TokenRef       string
	Source         string
	Missing        []string
	// BaseURL overrides the Cloudflare API base (tests/staging). Defaults
	// to the production v4 endpoint when empty.
	BaseURL string
}

const (
	cloudflareAccountIDField      = "CLOUDFLARE_ACCOUNT_ID"
	cloudflareTunnelIDField       = "CLOUDFLARE_TUNNEL_ID"
	cloudflareAPITokenField       = "CLOUDFLARE_API_TOKEN"       // #nosec G101 -- env var name only, not a credential value.
	cloudflareConnectorTokenField = "CLOUDFLARE_CONNECTOR_TOKEN" // #nosec G101 -- env var name only, not a credential value.
)

var cloudflareCredentialFields = []string{
	cloudflareAccountIDField,
	cloudflareTunnelIDField,
	cloudflareAPITokenField,
	cloudflareConnectorTokenField,
}

// NewCFClient builds the production IngressClient. It returns nil when any
// credential is absent — callers pass that nil through to the service,
// where remote operations then return ErrRemoteUnavailable. This keeps
// "creds missing" a single, typed, testable failure mode rather than a
// scattering of env checks.
func NewCFClient(doer httpc.Doer, cfg CFConfig) IngressClient {
	if cfg.APIToken == "" || cfg.AccountID == "" || cfg.TunnelID == "" {
		return nil
	}
	if doer == nil {
		doer = http.DefaultClient
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	return &cfClient{
		doer:      doer,
		apiToken:  cfg.APIToken,
		accountID: cfg.AccountID,
		tunnelID:  cfg.TunnelID,
		baseURL:   base,
	}
}

// Compile-time guarantee.
var _ IngressClient = (*cfClient)(nil)

// cfIngressRule mirrors the Cloudflare ingress wire shape.
type cfIngressRule struct {
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service"`
}

func (c *cfClient) configURL() string {
	return fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s/configurations", c.baseURL, c.accountID, c.tunnelID)
}

func (c *cfClient) ReadIngress(ctx context.Context) ([]IngressRule, error) {
	body, err := c.do(ctx, http.MethodGet, c.configURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("read ingress: %w", err)
	}
	var envelope struct {
		Success bool `json:"success"`
		Result  struct {
			Config struct {
				Ingress []cfIngressRule `json:"ingress"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse ingress: %w", err)
	}
	if !envelope.Success {
		return nil, fmt.Errorf("cloudflare API returned success=false reading ingress")
	}
	rules := make([]IngressRule, 0, len(envelope.Result.Config.Ingress))
	for _, r := range envelope.Result.Config.Ingress {
		rules = append(rules, IngressRule(r))
	}
	return rules, nil
}

func (c *cfClient) PushIngress(ctx context.Context, rules []IngressRule) error {
	wire := toCFRules(rules)
	payload := map[string]any{"config": map[string]any{"ingress": wire}}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ingress: %w", err)
	}
	if _, err := c.do(ctx, http.MethodPut, c.configURL(), body); err != nil {
		return fmt.Errorf("push ingress: %w", err)
	}
	return nil
}

// toCFRules converts domain rules to the wire shape, guaranteeing a
// trailing catch-all 404 (the old PushConfig invariant).
func toCFRules(rules []IngressRule) []cfIngressRule {
	out := make([]cfIngressRule, 0, len(rules)+1)
	hasCatchAll := false
	for _, r := range rules {
		out = append(out, cfIngressRule(r))
		if r.Hostname == "" {
			hasCatchAll = true
		}
	}
	if !hasCatchAll {
		out = append(out, cfIngressRule{Service: "http_status:404"})
	}
	return out
}

func (c *cfClient) do(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *cloudflareBootstrapClient) VerifyToken(ctx context.Context, token string) error {
	body, err := c.do(ctx, http.MethodGet, c.baseURL+"/user/tokens/verify", token, nil)
	if err != nil {
		return err
	}
	var response struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse token verification: %w", err)
	}
	if !response.Success {
		return fmt.Errorf("Cloudflare token verification was unsuccessful")
	}
	return nil
}

func (c *cloudflareBootstrapClient) ListAccounts(ctx context.Context, token string) ([]CloudflareAccount, error) {
	body, err := c.do(ctx, http.MethodGet, c.baseURL+"/accounts", token, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Success bool `json:"success"`
		Result  []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse accounts: %w", err)
	}
	if !response.Success {
		return nil, fmt.Errorf("Cloudflare accounts request was unsuccessful")
	}
	accounts := make([]CloudflareAccount, 0, len(response.Result))
	for _, account := range response.Result {
		accounts = append(accounts, CloudflareAccount{ID: account.ID})
	}
	return accounts, nil
}

func (c *cloudflareBootstrapClient) ListTunnels(ctx context.Context, token, accountID string) ([]CloudflareTunnel, error) {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel?is_deleted=false", c.baseURL, accountID)
	body, err := c.do(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Success bool `json:"success"`
		Result  []struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			DeletedAt *string `json:"deleted_at"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse tunnels: %w", err)
	}
	if !response.Success {
		return nil, fmt.Errorf("Cloudflare tunnels request was unsuccessful")
	}
	tunnels := make([]CloudflareTunnel, 0, len(response.Result))
	for _, tunnel := range response.Result {
		tunnels = append(tunnels, CloudflareTunnel{ID: tunnel.ID, Name: tunnel.Name, Deleted: tunnel.DeletedAt != nil})
	}
	return tunnels, nil
}

func (c *cloudflareBootstrapClient) CreateTunnel(ctx context.Context, token, accountID, name string) (CloudflareTunnel, error) {
	body, err := json.Marshal(map[string]string{"name": name, "config_src": "cloudflare"})
	if err != nil {
		return CloudflareTunnel{}, err
	}
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel", c.baseURL, accountID)
	data, err := c.do(ctx, http.MethodPost, url, token, body)
	if err != nil {
		return CloudflareTunnel{}, err
	}
	var response struct {
		Success bool             `json:"success"`
		Result  CloudflareTunnel `json:"result"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return CloudflareTunnel{}, fmt.Errorf("parse created tunnel: %w", err)
	}
	if !response.Success {
		return CloudflareTunnel{}, fmt.Errorf("Cloudflare tunnel creation was unsuccessful")
	}
	return response.Result, nil
}

func (c *cloudflareBootstrapClient) ConnectorToken(ctx context.Context, token, accountID, tunnelID string) (string, error) {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s/token", c.baseURL, accountID, tunnelID)
	data, err := c.do(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return "", err
	}
	var response struct {
		Success bool   `json:"success"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("parse connector token: %w", err)
	}
	if !response.Success {
		return "", fmt.Errorf("Cloudflare connector-token request was unsuccessful")
	}
	return response.Result, nil
}

func (c *cloudflareBootstrapClient) do(ctx context.Context, method, url, token string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Cloudflare API error %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
