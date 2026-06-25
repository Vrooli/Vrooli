package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"tunnel-manager/internal/httpc"
)

// cfDNSClient is the production DNSClient over the Cloudflare API v4 DNS-records
// endpoint. It creates the proxied CNAME that makes an exposed hostname
// publicly resolvable: <sub>.<apex> CNAME <tunnel-id>.cfargotunnel.com,
// proxied=true. This reverses the old "TM never touches DNS" non-goal that left
// every freshly-exposed hostname returning NXDOMAIN.
//
// It is additive and idempotent: EnsureRecord creates a record only when none
// exists for the hostname, and never rewrites a record pointing somewhere else
// (an out-of-band CNAME is left exactly as the operator set it). Zone ids are
// resolved by apex name and cached for the client's lifetime.
type cfDNSClient struct {
	doer     httpc.Doer
	apiToken string
	tunnelID string
	baseURL  string

	mu        sync.Mutex
	zoneCache map[string]string // apex -> zone id
}

// NewCFDNSClient builds the production DNSClient. It returns nil when any
// required credential is absent (mirroring NewCFClient), so DNS automation is
// simply skipped rather than erroring when the tunnel is not configured.
func NewCFDNSClient(doer httpc.Doer, cfg CFConfig) DNSClient {
	if cfg.APIToken == "" || cfg.TunnelID == "" {
		return nil
	}
	if doer == nil {
		doer = http.DefaultClient
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	return &cfDNSClient{
		doer:      doer,
		apiToken:  cfg.APIToken,
		tunnelID:  cfg.TunnelID,
		baseURL:   base,
		zoneCache: make(map[string]string),
	}
}

var _ DNSClient = (*cfDNSClient)(nil)

// tunnelTarget is the CNAME content every managed record points at.
func (c *cfDNSClient) tunnelTarget() string {
	return c.tunnelID + ".cfargotunnel.com"
}

func (c *cfDNSClient) EnsureRecord(ctx context.Context, hostname string) (DNSResult, error) {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	if hostname == "" {
		return DNSResult{}, fmt.Errorf("dns: empty hostname")
	}
	zoneID, err := c.zoneID(ctx, apexOf(hostname))
	if err != nil {
		return DNSResult{}, err
	}

	// Look up an existing record for the exact hostname first (idempotency).
	existing, err := c.findRecord(ctx, zoneID, hostname)
	if err != nil {
		return DNSResult{}, err
	}
	if existing.ID != "" {
		// A record already exists. Leave it untouched (additive: never clobber
		// an out-of-band record) and report it as pre-existing.
		return DNSResult{RecordID: existing.ID, Created: false}, nil
	}

	payload := map[string]any{
		"type":    "CNAME",
		"name":    hostname,
		"content": c.tunnelTarget(),
		"proxied": true,
		"ttl":     1, // 1 = automatic; required field for non-proxied, harmless for proxied.
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return DNSResult{}, fmt.Errorf("dns: marshal record: %w", err)
	}
	respBody, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/zones/%s/dns_records", c.baseURL, url.PathEscape(zoneID)), body)
	if err != nil {
		return DNSResult{}, fmt.Errorf("dns: create record for %q: %w", hostname, err)
	}
	id := parseRecordID(respBody)
	return DNSResult{RecordID: id, Created: true}, nil
}

func (c *cfDNSClient) RemoveRecord(ctx context.Context, hostname string) (bool, error) {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	if hostname == "" {
		return false, nil
	}
	zoneID, err := c.zoneID(ctx, apexOf(hostname))
	if err != nil {
		return false, err
	}
	existing, err := c.findRecord(ctx, zoneID, hostname)
	if err != nil {
		return false, err
	}
	if existing.ID == "" {
		return false, nil // already gone — idempotent.
	}
	_, err = c.do(ctx, http.MethodDelete, fmt.Sprintf("%s/zones/%s/dns_records/%s", c.baseURL, url.PathEscape(zoneID), url.PathEscape(existing.ID)), nil)
	if err != nil {
		return false, fmt.Errorf("dns: delete record for %q: %w", hostname, err)
	}
	return true, nil
}

type cfDNSRecord struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

func (c *cfDNSClient) findRecord(ctx context.Context, zoneID, hostname string) (cfDNSRecord, error) {
	u := fmt.Sprintf("%s/zones/%s/dns_records?type=CNAME&name=%s", c.baseURL, url.PathEscape(zoneID), url.QueryEscape(hostname))
	body, err := c.do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return cfDNSRecord{}, fmt.Errorf("dns: list records for %q: %w", hostname, err)
	}
	var env struct {
		Success bool          `json:"success"`
		Result  []cfDNSRecord `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return cfDNSRecord{}, fmt.Errorf("dns: parse records for %q: %w", hostname, err)
	}
	if len(env.Result) == 0 {
		return cfDNSRecord{}, nil
	}
	return env.Result[0], nil
}

func (c *cfDNSClient) zoneID(ctx context.Context, apex string) (string, error) {
	apex = strings.TrimSpace(strings.ToLower(apex))
	if apex == "" {
		return "", fmt.Errorf("dns: cannot resolve zone for empty apex")
	}
	c.mu.Lock()
	if id, ok := c.zoneCache[apex]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	body, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/zones?name=%s", c.baseURL, url.QueryEscape(apex)), nil)
	if err != nil {
		return "", fmt.Errorf("dns: resolve zone %q: %w", apex, err)
	}
	id := parseZoneLookup(body)
	if id == "" {
		return "", fmt.Errorf("dns: no zone named %q visible to this token (need Zone:Read)", apex)
	}
	c.mu.Lock()
	c.zoneCache[apex] = id
	c.mu.Unlock()
	return id, nil
}

func (c *cfDNSClient) do(ctx context.Context, method, rawURL string, body []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("cloudflare DNS API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// apexOf returns the registrable apex for a hostname by stripping the leading
// subdomain label. Route subdomains are single DNS labels (validated upstream),
// so "react-component-library.itsagitime.com" -> "itsagitime.com". A bare apex
// (no subdomain) is returned unchanged.
func apexOf(hostname string) string {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	i := strings.IndexByte(hostname, '.')
	if i < 0 {
		return hostname
	}
	rest := hostname[i+1:]
	if !strings.Contains(rest, ".") {
		// hostname was already an apex like "itsagitime.com".
		return hostname
	}
	return rest
}

func parseRecordID(body []byte) string {
	var env struct {
		Success bool `json:"success"`
		Result  struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return ""
	}
	return env.Result.ID
}
