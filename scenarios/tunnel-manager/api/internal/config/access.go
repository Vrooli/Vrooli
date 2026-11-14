package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"tunnel-manager/internal/httpc"
)

// cfAccessClient is the production AccessClient over the Cloudflare API v4
// Access apps/policies endpoints. It enforces the /public-asset convention at
// the edge: exactly one self_hosted Bypass-Everyone Access app scoped to
// <host>/public per active exposed host, so anonymous system fetchers (iOS
// A2HS, OG crawlers) can read branding/PWA assets while every other path stays
// gated by the operator's primary Access application.
//
// It is additive, idempotent, and ownership-guarded, mirroring cfDNSClient:
// EnsurePublicBypass looks up an existing TM-owned app first and never modifies
// an app it did not create; RemovePublicBypass deletes only an app whose domain
// is exactly <host>/public AND whose name carries the TM marker. The service
// gates removal further through the access ledger.
//
// HARD WRITE SCOPE CEILING: the only path it ever scopes an app to is /public,
// and the only decision it ever writes is bypass. Both are asserted in code
// (assertPublicBypassDomain / assertBypassDecision) and unit-tested. Read-only
// primary-app metadata discovery is exposed through AccessMetadataReader; it
// does not add any Access mutation capability.
type cfAccessClient struct {
	doer      httpc.Doer
	apiToken  string
	accountID string
	baseURL   string
}

// cloudflareAccessHTTPError preserves only the provider status class. The
// response body may contain provider-specific detail and is intentionally not
// carried into user-facing errors or logs.
type cloudflareAccessHTTPError struct {
	status int
}

func (e cloudflareAccessHTTPError) Error() string {
	return fmt.Sprintf("Cloudflare Access API rejected the request (HTTP %d)", e.status)
}

const (
	// publicBypassPath is the ONLY URL path TM will ever scope an Access app
	// to. The convention contract (docs/concepts/PUBLIC_ASSETS.md).
	publicBypassPath = "/public"
	// accessAppMarker tags every Access app TM creates so lookup/removal never
	// touches an app the operator configured out of band.
	accessAppMarker = "(TM public bypass)"
	// bypassDecision is the ONLY Access policy decision TM will ever write.
	bypassDecision = "bypass"
	// accessSessionDuration is the app session duration; immaterial for a
	// bypass app (no session is minted) but required by the API shape.
	accessSessionDuration = "24h"
)

// NewCFAccessClient builds the production AccessClient. It returns nil when the
// token or account id is absent (mirroring NewCFDNSClient), so the Access
// feature is simply disabled rather than erroring when unconfigured. Note the
// account id is REQUIRED here (Access apps are account-scoped), unlike DNS.
func NewCFAccessClient(doer httpc.Doer, cfg CFConfig) AccessClient {
	if cfg.APIToken == "" || cfg.AccountID == "" {
		return nil
	}
	if doer == nil {
		doer = http.DefaultClient
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	return &cfAccessClient{
		doer:      doer,
		apiToken:  cfg.APIToken,
		accountID: cfg.AccountID,
		baseURL:   base,
	}
}

// NewCFAccessMetadataReader builds the read-only Access metadata client using
// the same credential/configuration rules as the /public client.
func NewCFAccessMetadataReader(doer httpc.Doer, cfg CFConfig) AccessMetadataReader {
	if cfg.APIToken == "" || cfg.AccountID == "" {
		return nil
	}
	if doer == nil {
		doer = http.DefaultClient
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	return &cfAccessClient{doer: doer, apiToken: cfg.APIToken, accountID: cfg.AccountID, baseURL: base}
}

var (
	_ AccessClient         = (*cfAccessClient)(nil)
	_ AccessMetadataReader = (*cfAccessClient)(nil)
)

// publicBypassDomain builds the Access app domain for a host: <host>/public. It
// refuses anything that is not a bare hostname (a host carrying a path, query,
// or whitespace) so a caller can never widen the scope past /public.
func publicBypassDomain(host string) (string, error) {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return "", fmt.Errorf("access: empty host")
	}
	if strings.ContainsAny(host, "/ \t?#") {
		return "", fmt.Errorf("access: host %q must be a bare hostname with no path", host)
	}
	return host + publicBypassPath, nil
}

// assertPublicBypassDomain is the belt-and-suspenders guardrail the create path
// runs before any mutation: the domain must be exactly <non-empty-host>/public.
// It refuses empty / "/" / "/*" / a non-/public suffix outright.
func assertPublicBypassDomain(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" || domain == "/" || domain == "/*" {
		return fmt.Errorf("access: refusing to manage an Access app for path %q (only %q is allowed)", domain, publicBypassPath)
	}
	host, ok := strings.CutSuffix(domain, publicBypassPath)
	if !ok || host == "" || strings.ContainsAny(host, "/?#") {
		return fmt.Errorf("access: refusing domain %q — TM only ever scopes Access apps to <host>%s", domain, publicBypassPath)
	}
	return nil
}

// assertBypassDecision refuses any policy decision other than bypass — TM is a
// public-exemption manager and must never write an Allow/Block/identity policy.
func assertBypassDecision(decision string) error {
	if decision != bypassDecision {
		return fmt.Errorf("access: refusing policy decision %q — TM only ever writes %q", decision, bypassDecision)
	}
	return nil
}

func accessAppName(host string) string {
	return fmt.Sprintf("%s%s %s", host, publicBypassPath, accessAppMarker)
}

func accessPolicyName(host string) string {
	return fmt.Sprintf("%s%s bypass %s", host, publicBypassPath, accessAppMarker)
}

func (c *cfAccessClient) appsURL() string {
	return fmt.Sprintf("%s/accounts/%s/access/apps", c.baseURL, url.PathEscape(c.accountID))
}

// buildPublicBypassApp constructs the create payload for a <host>/public
// self_hosted app with one inline Bypass-Everyone policy. It routes through the
// guardrails so a bad host can never produce a mutation. (Inline policy is the
// plan's accepted shape: one POST creates the app + its bypass policy, with no
// cross-app ordering to get wrong.)
func buildPublicBypassApp(host string) (map[string]any, error) {
	domain, err := publicBypassDomain(host)
	if err != nil {
		return nil, err
	}
	if err := assertPublicBypassDomain(domain); err != nil {
		return nil, err
	}
	if err := assertBypassDecision(bypassDecision); err != nil {
		return nil, err
	}
	return map[string]any{
		"name":                 accessAppName(host),
		"type":                 "self_hosted",
		"domain":               domain,
		"session_duration":     accessSessionDuration,
		"app_launcher_visible": false,
		"policies": []map[string]any{{
			"name":       accessPolicyName(host),
			"decision":   bypassDecision,
			"precedence": 1,
			"include":    []map[string]any{{"everyone": map[string]any{}}},
		}},
	}, nil
}

func (c *cfAccessClient) EnsurePublicBypass(ctx context.Context, host string) (AccessResult, error) {
	existing, found, err := c.LookupPublicBypass(ctx, host)
	if err != nil {
		return AccessResult{}, err
	}
	if found {
		// A TM-owned app already exists — leave it untouched (additive).
		return AccessResult{AppID: existing.AppID, PolicyID: existing.PolicyID, Created: false}, nil
	}

	payload, err := buildPublicBypassApp(host)
	if err != nil {
		return AccessResult{}, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AccessResult{}, fmt.Errorf("access: marshal app: %w", err)
	}
	respBody, err := c.do(ctx, http.MethodPost, c.appsURL(), body)
	if err != nil {
		return AccessResult{}, fmt.Errorf("access: create bypass app for %q: %w", host, err)
	}
	app := parseAccessApp(respBody)
	return AccessResult{AppID: app.AppID, PolicyID: app.PolicyID, Created: true}, nil
}

func (c *cfAccessClient) RemovePublicBypass(ctx context.Context, host string) (bool, error) {
	existing, found, err := c.LookupPublicBypass(ctx, host)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil // already gone — idempotent.
	}
	delURL := fmt.Sprintf("%s/%s", c.appsURL(), url.PathEscape(existing.AppID))
	if _, err := c.do(ctx, http.MethodDelete, delURL, nil); err != nil {
		return false, fmt.Errorf("access: delete bypass app for %q: %w", host, err)
	}
	return true, nil
}

func (c *cfAccessClient) LookupPublicBypass(ctx context.Context, host string) (AccessApp, bool, error) {
	domain, err := publicBypassDomain(host)
	if err != nil {
		return AccessApp{}, false, err
	}
	// List apps and match on the exact <host>/public domain AND the TM name
	// marker, so a foreign Access app for the same host/path (operator-created)
	// is never mistaken for ours.
	body, err := c.do(ctx, http.MethodGet, c.appsURL()+"?per_page=1000", nil)
	if err != nil {
		return AccessApp{}, false, fmt.Errorf("access: list apps: %w", err)
	}
	var env struct {
		Success bool `json:"success"`
		Result  []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Domain   string `json:"domain"`
			Policies []struct {
				ID string `json:"id"`
			} `json:"policies"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return AccessApp{}, false, fmt.Errorf("access: parse apps: %w", err)
	}
	for _, a := range env.Result {
		if strings.EqualFold(strings.TrimSpace(a.Domain), domain) && strings.Contains(a.Name, accessAppMarker) {
			app := AccessApp{AppID: a.ID, Domain: a.Domain}
			if len(a.Policies) > 0 {
				app.PolicyID = a.Policies[0].ID
			}
			return app, true, nil
		}
	}
	return AccessApp{}, false, nil
}

// ResolveRuntimeBinding discovers the exact primary Access application for a
// route and the account's Access authentication domain. It is intentionally
// read-only: the response contains only the public verifier coordinates that
// an origin needs, never the management token or an Access assertion.
func (c *cfAccessClient) ResolveRuntimeBinding(ctx context.Context, host string) (AccessRuntimeBinding, error) {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || strings.ContainsAny(host, "/ \t?#") {
		return AccessRuntimeBinding{}, fmt.Errorf("access: host %q must be a bare hostname with no path", host)
	}

	body, err := c.do(ctx, http.MethodGet, c.appsURL()+"?per_page=1000", nil)
	if err != nil {
		return AccessRuntimeBinding{}, fmt.Errorf("access: list primary apps: %w", err)
	}
	var apps struct {
		Result []struct {
			Domain   string `json:"domain"`
			Audience string `json:"aud"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &apps); err != nil {
		return AccessRuntimeBinding{}, fmt.Errorf("access: parse primary apps: %w", err)
	}
	var audience string
	bestRank := 0
	matches := 0
	for _, app := range apps.Result {
		rank := accessPrimaryDomainRank(app.Domain, host)
		if rank == 0 || rank < bestRank {
			continue
		}
		if rank > bestRank {
			bestRank = rank
			matches = 0
			audience = ""
		}
		matches++
		if strings.TrimSpace(app.Audience) != "" {
			audience = strings.TrimSpace(app.Audience)
		}
	}
	if matches == 0 {
		return AccessRuntimeBinding{}, fmt.Errorf("access: no primary application found for %q", host)
	}
	if matches > 1 {
		return AccessRuntimeBinding{}, fmt.Errorf("access: multiple primary applications found for %q", host)
	}
	if audience == "" {
		return AccessRuntimeBinding{}, fmt.Errorf("access: primary application for %q has no audience tag", host)
	}

	teamDomain := ""
	orgBody, orgErr := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/accounts/%s/access/organizations", c.baseURL, url.PathEscape(c.accountID)), nil)
	if orgErr == nil {
		var org struct {
			Result struct {
				AuthDomain string `json:"auth_domain"`
			} `json:"result"`
		}
		if err := json.Unmarshal(orgBody, &org); err != nil {
			return AccessRuntimeBinding{}, fmt.Errorf("access: parse organization: %w", err)
		}
		teamDomain = normalizeAccessAuthDomain(org.Result.AuthDomain)
	}
	// Some least-privilege Access tokens can read applications but not the
	// organization endpoint. In that case the owner may set the one-time,
	// non-secret team origin on tunnel-manager itself. It is intentionally read
	// from this central process environment, never from each scenario.
	if teamDomain == "" {
		teamDomain = normalizeAccessAuthDomain(os.Getenv("VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN"))
	}
	if teamDomain == "" {
		if orgErr != nil {
			var providerErr cloudflareAccessHTTPError
			if errors.As(orgErr, &providerErr) && (providerErr.status == http.StatusUnauthorized || providerErr.status == http.StatusForbidden) {
				return AccessRuntimeBinding{}, ErrRemoteUnavailable{Reason: fmt.Sprintf("Cloudflare Access organization metadata was rejected (HTTP %d); grant the tunnel-manager credential read access or configure VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN once on tunnel-manager", providerErr.status)}
			}
			return AccessRuntimeBinding{}, ErrRemoteUnavailable{Reason: "Cloudflare Access organization metadata is unavailable; configure VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN once on tunnel-manager or grant organization read access"}
		}
		return AccessRuntimeBinding{}, fmt.Errorf("access: organization has no authentication domain")
	}
	return AccessRuntimeBinding{
		TeamDomain:  teamDomain,
		Audience:    audience,
		RecoveryURL: teamDomain,
	}, nil
}

// accessPrimaryDomainRank accepts the Access application shapes that can
// protect a hostname while excluding path-specific exceptions such as
// <host>/public. Exact host coverage wins over host/* and wildcard coverage;
// equal-specificity matches are rejected by ResolveRuntimeBinding.
func accessPrimaryDomainRank(rawDomain, host string) int {
	domain := strings.TrimSpace(strings.ToLower(rawDomain))
	host = strings.TrimSpace(strings.ToLower(host))
	switch {
	case domain == host:
		return 3
	case domain == host+"/*":
		return 2
	case strings.HasPrefix(domain, "*.") && strings.HasSuffix(host, strings.TrimPrefix(domain, "*")):
		return 1
	default:
		return 0
	}
}

func normalizeAccessAuthDomain(raw string) string {
	value := strings.TrimSpace(strings.TrimRight(raw, "/"))
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	return "https://" + strings.ToLower(parsed.Host)
}

// parseAccessApp extracts the app id + first policy id from a create response.
func parseAccessApp(body []byte) AccessApp {
	var env struct {
		Success bool `json:"success"`
		Result  struct {
			ID       string `json:"id"`
			Policies []struct {
				ID string `json:"id"`
			} `json:"policies"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return AccessApp{}
	}
	app := AccessApp{AppID: env.Result.ID}
	if len(env.Result.Policies) > 0 {
		app.PolicyID = env.Result.Policies[0].ID
	}
	return app
}

func (c *cfAccessClient) do(ctx context.Context, method, rawURL string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
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
		return nil, cloudflareAccessHTTPError{status: resp.StatusCode}
	}
	return data, nil
}
