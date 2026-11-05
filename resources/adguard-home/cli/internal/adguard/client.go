// Package adguard contains the resource-local AdGuard Home control API client.
package adguard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/tuning"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	StatusEndpoint               = "/control/status"
	DNSInfoEndpoint              = "/control/dns_info"
	QueryLogConfigEndpoint       = "/control/querylog/config"
	QueryLogConfigUpdateEndpoint = "/control/querylog/config/update"
	LegacyQueryLogInfoEndpoint   = "/control/querylog_info"
	ClientsEndpoint              = "/control/clients"
	TestUpstreamDNSEndpoint      = "/control/test_upstream_dns"
	InstallCheckConfigEndpoint   = "/control/install/check_config"
	InstallConfigureEndpoint     = "/control/install/configure"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Credentials struct {
	Username string
	Password string
}

type Client struct {
	baseURL string
	creds   Credentials
	http    HTTPClient
}

type Option func(*Client)

func WithHTTPClient(client HTTPClient) Option {
	return func(c *Client) {
		c.http = client
	}
}

func NewClient(baseURL string, creds Credentials, opts ...Option) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	c := &Client{
		baseURL: baseURL,
		creds:   creds,
		http:    &http.Client{Timeout: tuning.ControlPlaneClientTimeout()},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
	}
	return c, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

type ServerStatus struct {
	Version          string `json:"version"`
	ProtectionStatus *bool  `json:"protection_status"`
	Protection       *bool  `json:"protection"`
	Running          *bool  `json:"running"`
}

type DNSInfo struct {
	UpstreamDNS []string       `json:"upstream_dns"`
	Raw         map[string]any `json:"-"`
}

type QueryLogConfig struct {
	Enabled           *bool    `json:"enabled,omitempty"`
	Interval          any      `json:"interval,omitempty"`
	Ignored           []string `json:"ignored,omitempty"`
	IgnoredEnabled    *bool    `json:"ignored_enabled,omitempty"`
	AnonymizeClientIP *bool    `json:"anonymize_client_ip,omitempty"`
	SizeMemory        any      `json:"size_memory,omitempty"`
}

type AddressInfo struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type InitialConfiguration struct {
	DNS      AddressInfo `json:"dns"`
	Web      AddressInfo `json:"web"`
	Username string      `json:"username"`
	Password string      `json:"password"`
}

type CheckConfigRequest struct {
	DNS         AddressInfo `json:"dns"`
	Web         AddressInfo `json:"web"`
	SetStaticIP bool        `json:"set_static_ip"`
}

type CheckConfigResponse struct {
	DNS      CheckConfigSection `json:"dns"`
	Web      CheckConfigSection `json:"web"`
	StaticIP any                `json:"static_ip,omitempty"`
}

type CheckConfigSection struct {
	Status     string `json:"status"`
	CanAutofix bool   `json:"can_autofix"`
}

type UpstreamTestRequest struct {
	UpstreamDNS []string `json:"upstream_dns"`
}

type UpstreamPreview struct {
	CurrentUpstreams  []string          `json:"current_upstreams"`
	ProposedUpstreams []string          `json:"proposed_upstreams"`
	Added             []string          `json:"added,omitempty"`
	Removed           []string          `json:"removed,omitempty"`
	Changed           bool              `json:"changed"`
	TestResults       map[string]string `json:"test_results,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	MutationRequired  bool              `json:"mutation_required"`
	ApprovalRequired  bool              `json:"approval_required"`
	RollbackSource    string            `json:"rollback_source"`
}

type ClientsReport struct {
	Configured []ClientEntry `json:"configured,omitempty"`
	Auto       []ClientEntry `json:"auto,omitempty"`
	Total      int           `json:"total"`
	Warnings   []string      `json:"warnings,omitempty"`
}

type ClientEntry struct {
	Name      string   `json:"name,omitempty"`
	IDs       []string `json:"ids,omitempty"`
	IP        string   `json:"ip,omitempty"`
	Source    string   `json:"source,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Upstreams []string `json:"upstreams,omitempty"`
}

type clientsPayload struct {
	Clients     []ClientEntry `json:"clients"`
	AutoClients []ClientEntry `json:"auto_clients"`
}

func (c *Client) Status(ctx context.Context) (ServerStatus, int, error) {
	return getJSON[ServerStatus](ctx, c, StatusEndpoint)
}

func (c *Client) DNSInfo(ctx context.Context) (DNSInfo, int, error) {
	var raw map[string]any
	raw, code, err := getJSON[map[string]any](ctx, c, DNSInfoEndpoint)
	if err != nil || code < 200 || code >= 300 {
		return DNSInfo{}, code, err
	}
	info := DNSInfo{Raw: raw}
	if upstreams, ok := raw["upstream_dns"].([]any); ok {
		for _, upstream := range upstreams {
			if value, ok := upstream.(string); ok {
				info.UpstreamDNS = append(info.UpstreamDNS, value)
			}
		}
	}
	return info, code, nil
}

func (c *Client) QueryLogConfig(ctx context.Context) (QueryLogConfig, string, int, error) {
	config, code, err := getJSON[QueryLogConfig](ctx, c, QueryLogConfigEndpoint)
	if err == nil && code >= 200 && code < 300 {
		return config, QueryLogConfigEndpoint, code, nil
	}
	if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		return config, QueryLogConfigEndpoint, code, err
	}
	config, legacyCode, legacyErr := getJSON[QueryLogConfig](ctx, c, LegacyQueryLogInfoEndpoint)
	return config, LegacyQueryLogInfoEndpoint, legacyCode, legacyErr
}

func (c *Client) Clients(ctx context.Context) (ClientsReport, int, error) {
	payload, code, err := getJSON[clientsPayload](ctx, c, ClientsEndpoint)
	if err != nil || code < 200 || code >= 300 {
		return ClientsReport{}, code, err
	}
	report := ClientsReport{
		Configured: cleanClients(payload.Clients),
		Auto:       cleanClients(payload.AutoClients),
	}
	report.Total = len(report.Configured) + len(report.Auto)
	if report.Total == 0 {
		report.Warnings = append(report.Warnings, "AdGuard Home returned no configured or automatically discovered clients.")
	}
	return report, code, nil
}

func (c *Client) PreviewUpstreams(ctx context.Context, proposed []string, runTest bool) (UpstreamPreview, error) {
	info, code, err := c.DNSInfo(ctx)
	if err != nil {
		return UpstreamPreview{}, err
	}
	if code < 200 || code >= 300 {
		return UpstreamPreview{}, fmt.Errorf("DNS info returned HTTP %d", code)
	}

	current := cleanStrings(info.UpstreamDNS)
	proposed = cleanStrings(proposed)
	preview := UpstreamPreview{
		CurrentUpstreams:  current,
		ProposedUpstreams: proposed,
		Added:             difference(proposed, current),
		Removed:           difference(current, proposed),
		MutationRequired:  true,
		ApprovalRequired:  true,
		RollbackSource:    DNSInfoEndpoint,
	}
	preview.Changed = len(preview.Added) > 0 || len(preview.Removed) > 0
	if len(proposed) == 0 {
		preview.Warnings = append(preview.Warnings, "No proposed upstreams were provided; preview is read-only.")
		preview.MutationRequired = false
		preview.ApprovalRequired = false
	}

	if runTest && len(proposed) > 0 {
		results, testCode, testErr := postJSON[map[string]string](ctx, c, TestUpstreamDNSEndpoint, UpstreamTestRequest{UpstreamDNS: proposed})
		if testErr != nil {
			preview.Warnings = append(preview.Warnings, "Upstream test failed: "+testErr.Error())
		} else if testCode < 200 || testCode >= 300 {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Upstream test returned HTTP %d.", testCode))
		} else {
			preview.TestResults = results
		}
	}
	return preview, nil
}

func (c *Client) CheckInitialConfig(ctx context.Context, cfg InitialConfiguration) (CheckConfigResponse, int, error) {
	return postJSON[CheckConfigResponse](ctx, c, InstallCheckConfigEndpoint, CheckConfigRequest{
		DNS:         cfg.DNS,
		Web:         cfg.Web,
		SetStaticIP: false,
	})
}

func (c *Client) ConfigureInitial(ctx context.Context, cfg InitialConfiguration) (int, error) {
	return c.requestStatus(ctx, http.MethodPost, InstallConfigureEndpoint, cfg)
}

func (c *Client) DisableQueryLog(ctx context.Context) (int, error) {
	config, _, code, err := c.QueryLogConfig(ctx)
	if err != nil {
		return code, err
	}
	if code < 200 || code >= 300 {
		return code, nil
	}
	disabled := false
	config.Enabled = &disabled
	return c.requestStatus(ctx, http.MethodPut, QueryLogConfigUpdateEndpoint, config)
}

func (c *Client) requestStatus(ctx context.Context, method, endpoint string, body any) (int, error) {
	resp, err := c.do(ctx, method, endpoint, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func getJSON[T any](ctx context.Context, c *Client, endpoint string) (T, int, error) {
	var zero T
	resp, err := c.do(ctx, http.MethodGet, endpoint, nil)
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

func postJSON[T any](ctx context.Context, c *Client, endpoint string, body any) (T, int, error) {
	return requestJSON[T](ctx, c, http.MethodPost, endpoint, body)
}

func requestJSON[T any](ctx context.Context, c *Client, method, endpoint string, body any) (T, int, error) {
	var zero T
	resp, err := c.do(ctx, method, endpoint, body)
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

func (c *Client) do(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := cliout.NewEncoder(&buf).Encode(body); err != nil {
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
	return c.http.Do(req)
}

func cleanClients(values []ClientEntry) []ClientEntry {
	out := make([]ClientEntry, 0, len(values))
	for _, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		value.IP = strings.TrimSpace(value.IP)
		value.Source = strings.TrimSpace(value.Source)
		value.IDs = cleanStrings(value.IDs)
		value.Tags = cleanStrings(value.Tags)
		value.Upstreams = cleanStrings(value.Upstreams)
		if value.Name != "" || value.IP != "" || len(value.IDs) > 0 {
			out = append(out, value)
		}
	}
	return out
}

func cleanStrings(values []string) []string {
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
