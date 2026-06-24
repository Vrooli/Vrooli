package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	adGuardStatusEndpoint         = "/control/status"
	adGuardDNSInfoEndpoint        = "/control/dns_info"
	adGuardQueryLogConfigEndpoint = "/control/querylog/config"
	adGuardLegacyQueryLogEndpoint = "/control/querylog_info"
)

type ResourceBackedAdGuardClient struct {
	Secrets SecretResolver
	HTTP    http.RoundTripper
	Timeout time.Duration
}

func NewResourceBackedAdGuardClient(secrets SecretResolver) ResourceBackedAdGuardClient {
	if secrets == nil {
		secrets = NewVaultSecretResolver()
	}
	return ResourceBackedAdGuardClient{Secrets: secrets, Timeout: 10 * time.Second}
}

func (c ResourceBackedAdGuardClient) Check(ctx context.Context, cfg BackendConfig) (ClientStatus, error) {
	client, errStatus := c.client(ctx, cfg)
	if errStatus != nil {
		return errStatus.status, nil
	}

	status, code, err := getAdGuardJSON[adGuardServerStatus](ctx, client, adGuardStatusEndpoint)
	if err != nil {
		return ClientStatus{
			Status:   "unreachable",
			Warnings: []string{"AdGuard Home control API was not reachable."},
			Checks:   []string{"Control API request failed before AdGuard Home could be verified."},
		}, nil
	}
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ClientStatus{
			Status:   "auth_failed",
			Warnings: []string{"AdGuard Home rejected the configured credentials."},
			Checks:   []string{"Control API responded but authentication failed."},
		}, nil
	case http.StatusNotFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return ClientStatus{
			Status:   "setup_required",
			Warnings: []string{"AdGuard Home setup appears incomplete or the control API is not mounted yet."},
			Checks:   []string{"Admin endpoint responded, but control status is not available."},
		}, nil
	}
	if code < 200 || code >= 300 {
		return ClientStatus{
			Status:   "degraded",
			Warnings: []string{fmt.Sprintf("AdGuard Home control status returned HTTP %d.", code)},
			Checks:   []string{"Control API returned a non-success status."},
		}, nil
	}

	out := ClientStatus{
		Status:           "healthy",
		FilteringEnabled: protectionEnabled(status),
		Checks:           []string{"Control status endpoint returned successfully."},
	}
	if !out.FilteringEnabled {
		out.Status = "degraded"
		out.Warnings = append(out.Warnings, "AdGuard Home protection/filtering is disabled or unknown.")
	}

	if dnsInfo, dnsCode, dnsErr := getAdGuardJSON[adGuardDNSInfo](ctx, client, adGuardDNSInfoEndpoint); dnsErr == nil && dnsCode >= 200 && dnsCode < 300 {
		out.Upstreams = cleanUniqueStrings(dnsInfo.UpstreamDNS)
		out.Checks = append(out.Checks, "DNS info endpoint returned successfully.")
	} else if dnsErr != nil {
		out.Warnings = append(out.Warnings, "DNS info unavailable.")
	} else {
		out.Warnings = append(out.Warnings, fmt.Sprintf("DNS info returned HTTP %d.", dnsCode))
	}

	queryLog, endpoint, queryCode, queryErr := c.queryLogConfig(ctx, client)
	if queryErr == nil && queryCode >= 200 && queryCode < 300 {
		if queryLog.Enabled != nil && *queryLog.Enabled {
			out.Status = "degraded"
			out.Warnings = append(out.Warnings, "Query log is enabled; Network Manager will not expose query-level DNS history.")
		} else if queryLog.Enabled != nil {
			out.Checks = append(out.Checks, fmt.Sprintf("Query log is disabled according to %s.", endpoint))
		}
	} else if queryErr != nil {
		out.Warnings = append(out.Warnings, "Query log posture unavailable.")
	} else {
		out.Warnings = append(out.Warnings, fmt.Sprintf("Query log posture returned HTTP %d.", queryCode))
	}

	return out, nil
}

func (c ResourceBackedAdGuardClient) PreviewUpstreams(ctx context.Context, cfg BackendConfig, upstreams []string) ([]string, error) {
	client, errStatus := c.client(ctx, cfg)
	if errStatus != nil {
		return nil, fmt.Errorf("cannot preview AdGuard upstreams: %s", firstWarning(errStatus.status))
	}
	info, code, err := getAdGuardJSON[adGuardDNSInfo](ctx, client, adGuardDNSInfoEndpoint)
	if err != nil {
		return nil, fmt.Errorf("read current AdGuard upstreams: %w", err)
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return nil, fmt.Errorf("AdGuard Home rejected the configured credentials")
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("AdGuard Home DNS info returned HTTP %d", code)
	}

	current := cleanUniqueStrings(info.UpstreamDNS)
	proposed := cleanUniqueStrings(upstreams)
	return []string{
		"Previewed AdGuard Home upstream update; no resolver changes were applied.",
		"Current upstreams: " + joinUpstreams(current),
		"Requested upstreams: " + joinUpstreams(proposed),
		"Added: " + joinUpstreams(difference(proposed, current)),
		"Removed: " + joinUpstreams(difference(current, proposed)),
		"Persistent upstream changes require the policy approval and rollback path.",
	}, nil
}

func (c ResourceBackedAdGuardClient) UpdateUpstreams(context.Context, BackendConfig, []string) (ClientStatus, []string, error) {
	return ClientStatus{}, nil, fmt.Errorf("%w: persistent AdGuard upstream updates require policy rollback support", ErrClientUnsupported)
}

type adGuardClient struct {
	baseURL   string
	creds     Credentials
	transport http.RoundTripper
	timeout   time.Duration
}

type clientErrorStatus struct {
	status ClientStatus
}

func (c ResourceBackedAdGuardClient) client(ctx context.Context, cfg BackendConfig) (*adGuardClient, *clientErrorStatus) {
	creds, err := c.Secrets.ResolveAdGuardCredentials(ctx, cfg)
	if err != nil {
		return nil, &clientErrorStatus{status: ClientStatus{
			Status:   "auth_failed",
			Warnings: []string{"AdGuard Home credential secret could not be resolved through resource-vault."},
			Checks:   []string{"Credential resolution failed before contacting AdGuard Home."},
		}}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := c.HTTP
	if transport == nil {
		transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	return &adGuardClient{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		creds:     creds,
		transport: transport,
		timeout:   timeout,
	}, nil
}

func getAdGuardJSON[T any](ctx context.Context, client *adGuardClient, endpoint string) (T, int, error) {
	var zero T
	resp, err := client.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return zero, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, resp.StatusCode, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(&zero); err != nil {
		return zero, resp.StatusCode, fmt.Errorf("decode %s: %w", endpoint, err)
	}
	return zero, resp.StatusCode, nil
}

func (c ResourceBackedAdGuardClient) queryLogConfig(ctx context.Context, client *adGuardClient) (adGuardQueryLogConfig, string, int, error) {
	config, code, err := getAdGuardJSON[adGuardQueryLogConfig](ctx, client, adGuardQueryLogConfigEndpoint)
	if err == nil && code >= 200 && code < 300 {
		return config, adGuardQueryLogConfigEndpoint, code, nil
	}
	if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		return config, adGuardQueryLogConfigEndpoint, code, err
	}
	config, legacyCode, legacyErr := getAdGuardJSON[adGuardQueryLogConfig](ctx, client, adGuardLegacyQueryLogEndpoint)
	return config, adGuardLegacyQueryLogEndpoint, legacyCode, legacyErr
}

func (c *adGuardClient) do(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var reader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		reader = &buf
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.creds.Username != "" || c.creds.Password != "" {
		req.SetBasicAuth(c.creds.Username, c.creds.Password)
	}
	return c.transport.RoundTrip(req)
}

type adGuardServerStatus struct {
	Version          string `json:"version"`
	ProtectionStatus *bool  `json:"protection_status"`
	Protection       *bool  `json:"protection"`
	Running          *bool  `json:"running"`
}

type adGuardDNSInfo struct {
	UpstreamDNS []string `json:"upstream_dns"`
}

type adGuardQueryLogConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func protectionEnabled(status adGuardServerStatus) bool {
	if status.ProtectionStatus != nil {
		return *status.ProtectionStatus
	}
	if status.Protection != nil {
		return *status.Protection
	}
	if status.Running != nil {
		return *status.Running
	}
	return false
}

func cleanUniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func difference(left, right []string) []string {
	blocked := make(map[string]struct{}, len(right))
	for _, value := range right {
		blocked[value] = struct{}{}
	}
	out := make([]string, 0)
	for _, value := range left {
		if _, ok := blocked[value]; !ok {
			out = append(out, value)
		}
	}
	return out
}

func firstWarning(status ClientStatus) string {
	if len(status.Warnings) > 0 {
		return status.Warnings[0]
	}
	return status.Status
}
