package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// CFConfig carries the Cloudflare credentials the cfClient needs.
type CFConfig struct {
	APIToken  string
	AccountID string
	TunnelID  string
	TokenRef  string
	Source    string
	Missing   []string
	// BaseURL overrides the Cloudflare API base (tests/staging). Defaults
	// to the production v4 endpoint when empty.
	BaseURL string
}

// ResolveCloudflareEnv reads Cloudflare credentials from the process
// environment. Canonical CLOUDFLARE_* names win; legacy CF_* names remain a
// deterministic fallback for existing local tooling.
func ResolveCloudflareEnv(lookup func(string) string) CFConfig {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	accountID, accountSource := firstEnv(lookup, "CLOUDFLARE_ACCOUNT_ID", "CF_ACCOUNT_ID")
	tunnelID, tunnelSource := firstEnv(lookup, "CLOUDFLARE_TUNNEL_ID", "CF_TUNNEL_ID")
	apiToken, tokenSource := firstEnv(lookup, "CLOUDFLARE_API_TOKEN", "CF_API_TOKEN")

	missing := make([]string, 0, 3)
	if accountID == "" {
		missing = append(missing, "CLOUDFLARE_ACCOUNT_ID")
	}
	if tunnelID == "" {
		missing = append(missing, "CLOUDFLARE_TUNNEL_ID")
	}
	if apiToken == "" {
		missing = append(missing, "CLOUDFLARE_API_TOKEN")
	}

	source := credentialSource(accountSource, tunnelSource, tokenSource)
	tokenRef := ""
	if tokenSource != "" {
		tokenRef = "env:" + tokenSource
	}
	return CFConfig{
		APIToken:  apiToken,
		AccountID: accountID,
		TunnelID:  tunnelID,
		TokenRef:  tokenRef,
		Source:    source,
		Missing:   missing,
	}
}

func firstEnv(lookup func(string) string, canonical, legacy string) (string, string) {
	if v := lookup(canonical); v != "" {
		return v, canonical
	}
	if v := lookup(legacy); v != "" {
		return v, legacy
	}
	return "", ""
}

func credentialSource(fields ...string) string {
	hasCanonical := false
	hasLegacy := false
	for _, field := range fields {
		switch {
		case field == "":
		case len(field) >= len("CLOUDFLARE_") && field[:len("CLOUDFLARE_")] == "CLOUDFLARE_":
			hasCanonical = true
		case len(field) >= len("CF_") && field[:len("CF_")] == "CF_":
			hasLegacy = true
		}
	}
	switch {
	case hasCanonical && hasLegacy:
		return "env:mixed"
	case hasCanonical:
		return "env:CLOUDFLARE_*"
	case hasLegacy:
		return "env:CF_*"
	default:
		return "none"
	}
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
		rules = append(rules, IngressRule{Hostname: r.Hostname, Service: r.Service})
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
		out = append(out, cfIngressRule{Hostname: r.Hostname, Service: r.Service})
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
