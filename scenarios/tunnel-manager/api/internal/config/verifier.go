package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"tunnel-manager/internal/httpc"
)

// cfVerifier is the production CredentialVerifier. It performs read-only
// Cloudflare API v4 calls over the httpc.Doer seam to turn a present-but-
// possibly-wrong credential set into a per-check verdict. It NEVER writes
// account state (no DNS record create, no ingress push) and never returns a
// secret value — only a coarse verdict plus remediation.
//
// Why a probe per scope rather than parsing token policies: Cloudflare's
// /user/tokens/verify confirms the token is active but does not enumerate
// resource scopes in a form we can map to "can edit DNS for this zone". A
// cheap read against each resource the token must reach (account, tunnel,
// zone, zone DNS records) distinguishes "authenticated" from "authorized for
// what TM needs", which is exactly the gap that made the live URL go dead.
type cfVerifier struct {
	doer    httpc.Doer
	baseURL string
}

// NewCFVerifier builds the production credential verifier over the shared
// outbound HTTP seam. A nil doer falls back to http.DefaultClient.
func NewCFVerifier(doer httpc.Doer) CredentialVerifier {
	if doer == nil {
		doer = http.DefaultClient
	}
	return &cfVerifier{doer: doer, baseURL: "https://api.cloudflare.com/client/v4"}
}

var _ CredentialVerifier = (*cfVerifier)(nil)

func (v *cfVerifier) Verify(ctx context.Context, cfg CFConfig, apexes []string, accessRequired bool) (CredentialVerification, error) {
	checks := make([]CredentialCheck, 0, 6+2*len(apexes))

	// 1. Token present + authenticates.
	tokenCheck := v.checkToken(ctx, cfg.APIToken)
	checks = append(checks, tokenCheck)
	tokenUsable := tokenCheck.State == CheckOK

	// 2. Account reachable (also confirms account-scoped read).
	checks = append(checks, v.checkAccount(ctx, cfg, tokenUsable))

	// 3. Tunnel reachable (confirms Account:Cloudflare Tunnel scope).
	checks = append(checks, v.checkTunnel(ctx, cfg, tokenUsable))

	// 4. One zone-lookup + DNS-edit probe per distinct apex.
	for _, apex := range dedupeNonEmpty(apexes) {
		checks = append(checks, v.checkZoneDNS(ctx, cfg.APIToken, apex, tokenUsable)...)
	}

	// 5. Access application metadata. This is always probed so operators can
	// distinguish gated-auth discovery failures from tunnel/DNS failures. The
	// application read is only required for core readiness when public exposure
	// is enabled; a GET cannot prove the write permission used by reconciliation.
	access := v.checkAccessScope(ctx, cfg, tokenUsable, accessRequired)
	checks = append(checks, access)

	// 6. Access organization metadata. This is the read needed to discover the
	// shared authentication domain for gated scenarios. It is informational for
	// generic tunnel-manager readiness because VerifyCredentials has no specific
	// scenario route, but its failure must be visible and actionable.
	checks = append(checks, v.checkAccessOrganization(ctx, cfg, tokenUsable))

	ready := true
	allChecksOK := true
	for _, c := range checks {
		if c.State != CheckOK {
			allChecksOK = false
		}
		if c.Required && c.State != CheckOK {
			ready = false
		}
	}
	return CredentialVerification{
		Checks:       checks,
		Capabilities: summarizeCapabilities(checks),
		Ready:        ready,
		AllChecksOK:  allChecksOK,
	}, nil
}

// checkAccessScope probes the account-scoped Access apps list. A successful
// GET proves application metadata is readable; it deliberately does not claim
// that a later POST/DELETE will be authorized.
func (v *cfVerifier) checkAccessScope(ctx context.Context, cfg CFConfig, tokenUsable, required bool) CredentialCheck {
	c := newCredentialCheck(CheckNameAccessScope, CapabilityAccessAppMetadata, required)
	if strings.TrimSpace(cfg.AccountID) == "" {
		c.State = CheckMissing
		c.Remediation = "Set the account id via `config credentials-set --account-id <id>`."
		return c
	}
	if !tokenUsable {
		c.State = CheckUnspecified
		c.Detail = "skipped: token not usable"
		return c
	}
	u := fmt.Sprintf("%s/accounts/%s/access/apps?per_page=1", v.baseURL, url.PathEscape(cfg.AccountID))
	status, _, err := v.get(ctx, cfg.APIToken, u)
	c = classifyResourceCheck(c, status, err,
		"Access application metadata read failed",
		"Grant the tunnel-manager credential read access to Access applications; public-exposure writes require a separate write permission.",
		"Access applications are not available for this account.")
	if c.State == CheckOK {
		c.Detail = "Access application metadata readable; write permission not tested"
	}
	return c
}

// checkAccessOrganization probes the organization endpoint used to discover
// the shared Access authentication domain. A 401/403 is intentionally
// classified separately from an invalid API token: token verification and the
// other account resources may still be healthy.
func (v *cfVerifier) checkAccessOrganization(ctx context.Context, cfg CFConfig, tokenUsable bool) CredentialCheck {
	c := newCredentialCheck(CheckNameAccessOrganization, CapabilityGatedUIAuthentication, false)
	if strings.TrimSpace(cfg.AccountID) == "" {
		c.State = CheckMissing
		c.Remediation = "Set the account id via `config credentials-set --account-id <id>`."
		return c
	}
	if !tokenUsable {
		c.State = CheckUnspecified
		c.Detail = "skipped: token not usable"
		return c
	}
	u := fmt.Sprintf("%s/accounts/%s/access/organizations", v.baseURL, url.PathEscape(cfg.AccountID))
	status, body, err := v.get(ctx, cfg.APIToken, u)
	c = classifyResourceCheck(c, status, err,
		"Access organization metadata read failed",
		"Grant the tunnel-manager credential read access to Access organization metadata, or configure VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN once on tunnel-manager.",
		"Access organization metadata is unavailable for this account; verify that Access is enabled.")
	if c.State == CheckOK {
		var org struct {
			Result struct {
				AuthDomain string `json:"auth_domain"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &org); err != nil {
			c.State = CheckInvalid
			c.Detail = "Access organization metadata response was not understood"
			c.Remediation = "Verify the Cloudflare Access organization response and retry."
		} else if strings.TrimSpace(org.Result.AuthDomain) == "" {
			c.State = CheckInvalid
			c.Detail = "Access organization has no authentication domain"
			c.Remediation = "Configure the Access authentication domain or set VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN on tunnel-manager."
		} else {
			c.Detail = "Access authentication domain readable"
		}
	}
	return c
}

func (v *cfVerifier) checkToken(ctx context.Context, token string) CredentialCheck {
	c := newCredentialCheck(CheckNameToken, CapabilityCloudflareAuthentication, true)
	if strings.TrimSpace(token) == "" {
		c.State = CheckMissing
		c.Remediation = "Set the Cloudflare API token via `printf '%s' <token> | config credentials-set --api-token-stdin`."
		return c
	}
	status, _, err := v.get(ctx, token, v.baseURL+"/user/tokens/verify")
	switch {
	case err != nil:
		c.State = CheckUnavailable
		c.Detail = "token verify request failed"
		c.Remediation = "Check network connectivity to api.cloudflare.com and retry."
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		c.State = CheckInvalid
		c.Remediation = "The token is rejected (expired/revoked). Issue a new token and provide it with `printf '%s' <token> | config credentials-set --api-token-stdin`."
	case status == http.StatusTooManyRequests || status >= 500:
		c.State = CheckUnavailable
		c.Detail = fmt.Sprintf("Cloudflare token verification returned HTTP %d", status)
		c.Remediation = "Cloudflare did not provide a stable verification result; retry later."
	case status >= 400:
		c.State = CheckInvalid
		c.Detail = fmt.Sprintf("HTTP %d", status)
		c.Remediation = "The token did not verify. Re-issue it in the Cloudflare dashboard."
	default:
		c.State = CheckOK
		c.Detail = "token active"
	}
	return c
}

func (v *cfVerifier) checkAccount(ctx context.Context, cfg CFConfig, tokenUsable bool) CredentialCheck {
	c := newCredentialCheck(CheckNameAccount, CapabilityRemoteIngress, true)
	if strings.TrimSpace(cfg.AccountID) == "" {
		c.State = CheckMissing
		c.Remediation = "Set the account id via `config credentials-set --account-id <id>`."
		return c
	}
	if !tokenUsable {
		c.State = CheckUnspecified
		c.Detail = "skipped: token not usable"
		return c
	}
	status, _, err := v.get(ctx, cfg.APIToken, fmt.Sprintf("%s/accounts/%s", v.baseURL, url.PathEscape(cfg.AccountID)))
	return classifyResourceCheck(c, status, err,
		"account read failed",
		"The token cannot read this account; confirm the account id and that the token is scoped to it.",
		"Account id not found; correct it via `config credentials-set --account-id <id>`.")
}

func (v *cfVerifier) checkTunnel(ctx context.Context, cfg CFConfig, tokenUsable bool) CredentialCheck {
	c := newCredentialCheck(CheckNameTunnel, CapabilityRemoteIngress, true)
	if strings.TrimSpace(cfg.TunnelID) == "" {
		c.State = CheckMissing
		c.Remediation = "Set the tunnel id via `config credentials-set --tunnel-id <id>`."
		return c
	}
	if strings.TrimSpace(cfg.AccountID) == "" || !tokenUsable {
		c.State = CheckUnspecified
		c.Detail = "skipped: token/account not usable"
		return c
	}
	u := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s", v.baseURL, url.PathEscape(cfg.AccountID), url.PathEscape(cfg.TunnelID))
	status, _, err := v.get(ctx, cfg.APIToken, u)
	return classifyResourceCheck(c, status, err,
		"tunnel read failed",
		"The token lacks Account:Cloudflare Tunnel scope. Add it and re-issue the token.",
		"Tunnel id not found; correct it via `config credentials-set --tunnel-id <id>`.")
}

// checkZoneDNS resolves the apex zone by name then probes DNS-records read as a
// proxy for Zone:DNS:Edit. A read is the cheapest non-mutating proof the token
// can reach the zone's DNS surface; a token that lost DNS access entirely (the
// regression this guards) returns 403 here instead of producing a dead URL.
func (v *cfVerifier) checkZoneDNS(ctx context.Context, token, apex string, tokenUsable bool) []CredentialCheck {
	lookup := newCredentialCheck(CheckNameZoneLookup, CapabilityDNSAutomation, true)
	lookup.Detail = apex
	dns := newCredentialCheck(CheckNameDNSScope, CapabilityDNSAutomation, true)
	dns.Detail = apex
	if !tokenUsable {
		lookup.State = CheckUnspecified
		lookup.Detail = "skipped: token not usable"
		dns.State = CheckUnspecified
		dns.Detail = "skipped: token not usable"
		return []CredentialCheck{lookup, dns}
	}

	zoneURL := fmt.Sprintf("%s/zones?name=%s", v.baseURL, url.QueryEscape(apex))
	status, body, err := v.get(ctx, token, zoneURL)
	switch {
	case err != nil:
		lookup.State = CheckInvalid
		lookup.Remediation = "Zone lookup failed; check connectivity to api.cloudflare.com."
		dns.State = CheckUnspecified
		dns.Detail = "skipped: zone unresolved"
		return []CredentialCheck{lookup, dns}
	case status == http.StatusForbidden || status == http.StatusUnauthorized:
		lookup.State = CheckInsufficientScope
		lookup.Remediation = "Add Zone:Read scope to the token so TM can resolve the apex zone for DNS automation."
		dns.State = CheckInsufficientScope
		dns.Detail = "skipped: zone unreadable"
		dns.Remediation = "Add Zone:Read + Zone:DNS:Edit scopes to the token."
		return []CredentialCheck{lookup, dns}
	}
	zoneID := parseZoneLookup(body)
	if zoneID == "" {
		lookup.State = CheckInvalid
		lookup.Remediation = fmt.Sprintf("No zone named %q is visible to this token; confirm the apex and that the token has Zone:Read.", apex)
		dns.State = CheckUnspecified
		dns.Detail = "skipped: zone not found"
		return []CredentialCheck{lookup, dns}
	}
	lookup.State = CheckOK

	// DNS-records read as a non-mutating proxy for Zone:DNS:Edit reach. A 403
	// here surfaces a DNS-scope regression distinctly from a zone-read failure.
	recURL := fmt.Sprintf("%s/zones/%s/dns_records?per_page=1", v.baseURL, url.PathEscape(zoneID))
	dStatus, _, dErr := v.get(ctx, token, recURL)
	dns = classifyResourceCheck(dns, dStatus, dErr,
		"dns records read failed",
		"The token cannot read DNS records for this zone; add Zone:DNS:Edit and re-issue it.",
		"DNS records endpoint not found for the resolved zone.")
	return []CredentialCheck{lookup, dns}
}

// get issues an authenticated GET and returns (status, body, error). A non-nil
// error is reserved for transport failures; HTTP status codes are returned for
// the caller to classify.
func (v *cfVerifier) get(ctx context.Context, token, rawURL string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.doer.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}

// classifyResourceCheck maps a single resource read into a verdict: transport
// error or provider 429/5xx → UNAVAILABLE, 403/401 → INSUFFICIENT_SCOPE,
// 404/other 4xx → INVALID, 2xx → OK. The distinction matters: an unavailable
// provider should not be presented as a bad token or missing permission.
func classifyResourceCheck(c CredentialCheck, status int, err error, errDetail, scopeRemediation, notFoundRemediation string) CredentialCheck {
	switch {
	case err != nil:
		c.State = CheckUnavailable
		c.Detail = errDetail
		c.Remediation = "Check connectivity to api.cloudflare.com and retry."
	case status == http.StatusForbidden || status == http.StatusUnauthorized:
		c.State = CheckInsufficientScope
		c.Detail = fmt.Sprintf("Cloudflare rejected the read (HTTP %d)", status)
		c.Remediation = scopeRemediation
	case status == http.StatusNotFound:
		c.State = CheckInvalid
		c.Remediation = notFoundRemediation
	case status == http.StatusTooManyRequests || status >= 500:
		c.State = CheckUnavailable
		c.Detail = fmt.Sprintf("Cloudflare returned HTTP %d", status)
		c.Remediation = "Cloudflare did not provide a stable result; retry later."
	case status >= 400:
		c.State = CheckInvalid
		c.Detail = fmt.Sprintf("HTTP %d", status)
		c.Remediation = notFoundRemediation
	default:
		c.State = CheckOK
	}
	return c
}

func newCredentialCheck(name, capability string, required bool) CredentialCheck {
	return CredentialCheck{Name: name, Capability: capability, Required: required}
}

// summarizeCapabilities preserves a stable display order while grouping the
// checks that establish each operation. A capability can be not-ready even
// when it is not a blocker for Tunnel Manager's core remote/DNS readiness.
func summarizeCapabilities(checks []CredentialCheck) []CredentialCapability {
	order := []string{
		CapabilityCloudflareAuthentication,
		CapabilityRemoteIngress,
		CapabilityDNSAutomation,
		CapabilityAccessAppMetadata,
		CapabilityGatedUIAuthentication,
	}
	byCapability := make(map[string]*CredentialCapability, len(order))
	for _, name := range order {
		byCapability[name] = &CredentialCapability{Name: name, Ready: true}
	}
	for _, check := range checks {
		if check.Capability == "" {
			continue
		}
		capability, ok := byCapability[check.Capability]
		if !ok {
			capability = &CredentialCapability{Name: check.Capability, Ready: true}
			byCapability[check.Capability] = capability
			order = append(order, check.Capability)
		}
		capability.CheckNames = append(capability.CheckNames, check.Name)
		capability.Required = capability.Required || check.Required
		if check.State != CheckOK {
			capability.Ready = false
			if capability.Reason == "" {
				if check.Remediation != "" {
					capability.Reason = check.Remediation
				} else if check.Detail != "" {
					capability.Reason = check.Detail
				} else {
					capability.Reason = "one or more checks did not pass"
				}
			}
		}
	}
	result := make([]CredentialCapability, 0, len(order))
	for _, name := range order {
		capability, ok := byCapability[name]
		if !ok || len(capability.CheckNames) == 0 {
			continue
		}
		if capability.Ready {
			capability.Reason = "all checks passed"
		}
		result = append(result, *capability)
	}
	return result
}

func dedupeNonEmpty(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// parseZoneLookup extracts the first zone id from a GET /zones?name=<apex>
// response envelope, returning "" when the lookup found no visible zone.
func parseZoneLookup(body []byte) string {
	var env struct {
		Success bool `json:"success"`
		Result  []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return ""
	}
	if !env.Success || len(env.Result) == 0 {
		return ""
	}
	return env.Result[0].ID
}
