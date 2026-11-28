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

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
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
var _ ManagedDNSClient = (*cfDNSClient)(nil)

// EnsureManagedRecord implements the provider-neutral record contract while
// retaining the same ownership safety as the tunnel-specific path: an
// existing record with different content is a conflict and is never
// overwritten automatically.
func (c *cfDNSClient) EnsureManagedRecord(ctx context.Context, spec DNSRecordSpec) (DNSResult, error) {
	if err := validateDNSRecordSpec(spec); err != nil {
		return DNSResult{}, err
	}
	zoneID, err := c.zoneID(ctx, apexOf(spec.Hostname))
	if err != nil {
		return DNSResult{}, err
	}
	existing, err := c.findRecordByType(ctx, zoneID, spec.Hostname, spec.Type)
	if err != nil {
		return DNSResult{}, err
	}
	if existing.ID != "" {
		if existing.Content != spec.Content || existing.Priority != spec.Priority || existing.Proxied != spec.Proxied {
			return DNSResult{}, fmt.Errorf("dns: record %q conflicts with existing %s record", spec.Hostname, spec.Type)
		}
		return DNSResult{RecordID: existing.ID}, nil
	}
	payload := map[string]any{"type": spec.Type, "name": spec.Hostname, "ttl": spec.TTL, "proxied": spec.Proxied}
	if spec.Type == "MX" {
		payload["data"] = map[string]any{"type": "MX", "name": spec.Hostname, "content": spec.Content, "priority": spec.Priority}
	} else {
		payload["content"] = spec.Content
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return DNSResult{}, fmt.Errorf("dns: marshal managed record: %w", err)
	}
	respBody, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/zones/%s/dns_records", c.baseURL, url.PathEscape(zoneID)), body)
	if err != nil {
		return DNSResult{}, fmt.Errorf("dns: create managed record %q: %w", spec.Hostname, err)
	}
	return DNSResult{RecordID: parseRecordID(respBody), Created: true}, nil
}

// UpdateManagedRecord updates one record in place after the caller has
// explicitly verified ownership/content. It is separate from EnsureManagedRecord
// so an additive deployment reconcile never overwrites an operator's record.
func (c *cfDNSClient) UpdateManagedRecord(ctx context.Context, spec DNSRecordSpec) (DNSResult, error) {
	if err := validateDNSRecordSpec(spec); err != nil {
		return DNSResult{}, err
	}
	zoneID, err := c.zoneID(ctx, apexOf(spec.Hostname))
	if err != nil {
		return DNSResult{}, err
	}
	existing, err := c.findRecordByType(ctx, zoneID, spec.Hostname, spec.Type)
	if err != nil {
		return DNSResult{}, err
	}
	if existing.ID == "" {
		return c.EnsureManagedRecord(ctx, spec)
	}
	payload := map[string]any{"type": spec.Type, "name": spec.Hostname, "ttl": spec.TTL, "proxied": spec.Proxied}
	if spec.Type == "MX" {
		payload["data"] = map[string]any{"type": "MX", "name": spec.Hostname, "content": spec.Content, "priority": spec.Priority}
	} else {
		payload["content"] = spec.Content
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return DNSResult{}, fmt.Errorf("dns: marshal managed update: %w", err)
	}
	if _, err := c.do(ctx, http.MethodPut, fmt.Sprintf("%s/zones/%s/dns_records/%s", c.baseURL, url.PathEscape(zoneID), url.PathEscape(existing.ID)), body); err != nil {
		return DNSResult{}, fmt.Errorf("dns: update managed record %q: %w", spec.Hostname, err)
	}
	return DNSResult{RecordID: existing.ID, Created: false}, nil
}

// EnsureSPFRecord merges one provider mechanism into the single existing SPF
// TXT record. It refuses ambiguous zones with multiple SPF records.
func (c *cfDNSClient) EnsureSPFRecord(ctx context.Context, providerProfile, hostname, mechanism string, ttl int, dryRun bool) (DNSResult, string, error) {
	result, merged, _, err := c.ensureSPFRecord(ctx, providerProfile, hostname, mechanism, ttl, dryRun)
	return result, merged, err
}

func (c *cfDNSClient) MergeSPFRecord(ctx context.Context, providerProfile, hostname, mechanism string, ttl int, dryRun bool) (SPFResult, error) {
	result, merged, cost, err := c.ensureSPFRecord(ctx, providerProfile, hostname, mechanism, ttl, dryRun)
	return SPFResult{DNSResult: result, MergedValue: merged, LookupCost: cost}, err
}

func (c *cfDNSClient) ensureSPFRecord(ctx context.Context, providerProfile, hostname, mechanism string, ttl int, dryRun bool) (DNSResult, string, int, error) {
	if strings.TrimSpace(providerProfile) == "" {
		return DNSResult{}, "", 0, fmt.Errorf("provider profile is required")
	}
	zoneID, err := c.zoneID(ctx, apexOf(hostname))
	if err != nil {
		return DNSResult{}, "", 0, err
	}
	records, err := c.findRecordsByType(ctx, zoneID, strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), "."), "TXT")
	if err != nil {
		return DNSResult{}, "", 0, err
	}
	values := make([]string, 0, len(records))
	for _, record := range records {
		values = append(values, record.Content)
	}
	merged, err := MergeSPFMechanism(values, mechanism)
	if err != nil {
		return DNSResult{}, "", 0, err
	}
	spfReport := maildns.New(cloudflareMailResolver{client: c, proposedHost: strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), "."), proposed: merged}).Verify(ctx, hostname, []maildns.ProviderRequirement{{Name: "spf-merge", SPFMechanism: mechanism}})
	if len(spfReport.Providers) == 0 || spfReport.Providers[0].Status != maildns.Pass {
		detail := "SPF merge would exceed the lookup budget"
		if len(spfReport.Providers) > 0 {
			detail = spfReport.Providers[0].Detail
		}
		return DNSResult{}, merged, firstSPFCost(spfReport), fmt.Errorf("dns: %s (lookup cost %d)", detail, firstSPFCost(spfReport))
	}
	if dryRun {
		return DNSResult{}, merged, firstSPFCost(spfReport), nil
	}
	var existingSPF *cfDNSRecord
	for i := range records {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(records[i].Content)), "v=spf1") {
			existingSPF = &records[i]
			break
		}
	}
	if existingSPF == nil {
		result, createErr := c.EnsureManagedRecord(ctx, DNSRecordSpec{ProviderProfile: providerProfile, Hostname: hostname, Type: "TXT", Content: merged, TTL: ttl})
		return result, merged, firstSPFCost(spfReport), createErr
	}
	payload, marshalErr := json.Marshal(map[string]any{"type": "TXT", "name": hostname, "content": merged, "ttl": ttl, "proxied": false})
	if marshalErr != nil {
		return DNSResult{}, merged, firstSPFCost(spfReport), marshalErr
	}
	if _, updateErr := c.do(ctx, http.MethodPut, fmt.Sprintf("%s/zones/%s/dns_records/%s", c.baseURL, url.PathEscape(zoneID), url.PathEscape(existingSPF.ID)), payload); updateErr != nil {
		return DNSResult{}, merged, firstSPFCost(spfReport), fmt.Errorf("dns: update SPF record %q: %w", hostname, updateErr)
	}
	return DNSResult{RecordID: existingSPF.ID}, merged, firstSPFCost(spfReport), nil
}

func firstSPFCost(report maildns.Report) int {
	if len(report.Providers) == 0 {
		return 0
	}
	return report.Providers[0].Cost
}

type cloudflareMailResolver struct {
	client       *cfDNSClient
	proposedHost string
	proposed     string
}

func (r cloudflareMailResolver) LookupTXT(ctx context.Context, host string) ([]string, error) {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == r.proposedHost {
		return []string{r.proposed}, nil
	}
	zoneID, err := r.client.zoneID(ctx, apexOf(host))
	if err != nil {
		return nil, err
	}
	records, err := r.client.findRecordsByType(ctx, zoneID, host, "TXT")
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(records))
	for _, record := range records {
		values = append(values, record.Content)
	}
	return values, nil
}

func (r cloudflareMailResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	zoneID, err := r.client.zoneID(ctx, apexOf(host))
	if err != nil {
		return "", err
	}
	records, err := r.client.findRecordsByType(ctx, zoneID, host, "CNAME")
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", fmt.Errorf("CNAME %s not found", host)
	}
	return records[0].Content, nil
}

func validateDNSRecordSpec(spec DNSRecordSpec) error {
	if strings.TrimSpace(spec.ProviderProfile) == "" {
		return fmt.Errorf("provider profile is required")
	}
	spec.Hostname = strings.TrimSpace(strings.ToLower(spec.Hostname))
	spec.Type = strings.ToUpper(strings.TrimSpace(spec.Type))
	spec.Content = strings.TrimSpace(spec.Content)
	if spec.Hostname == "" || spec.Content == "" {
		return fmt.Errorf("dns: hostname and content are required")
	}
	switch spec.Type {
	case "A", "AAAA", "CNAME", "TXT", "MX":
	default:
		return fmt.Errorf("dns: record type %q is not supported", spec.Type)
	}
	if spec.TTL < 0 || spec.TTL > 86400 {
		return fmt.Errorf("dns: ttl %d is outside 0..86400", spec.TTL)
	}
	if spec.Proxied && spec.Type != "A" && spec.Type != "AAAA" && spec.Type != "CNAME" {
		return fmt.Errorf("dns: proxied record type %q is not supported", spec.Type)
	}
	if spec.Type == "MX" && spec.Priority < 0 {
		return fmt.Errorf("dns: MX priority must not be negative")
	}
	return nil
}

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
	ID       string `json:"id"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	Proxied  bool   `json:"proxied"`
	Priority int    `json:"priority"`
}

func (c *cfDNSClient) findRecordByType(ctx context.Context, zoneID, hostname, recordType string) (cfDNSRecord, error) {
	records, err := c.findRecordsByType(ctx, zoneID, hostname, recordType)
	if err != nil {
		return cfDNSRecord{}, err
	}
	if len(records) == 0 {
		return cfDNSRecord{}, nil
	}
	return records[0], nil
}

func (c *cfDNSClient) findRecordsByType(ctx context.Context, zoneID, hostname, recordType string) ([]cfDNSRecord, error) {
	u := fmt.Sprintf("%s/zones/%s/dns_records?type=%s&name=%s", c.baseURL, url.PathEscape(zoneID), url.QueryEscape(recordType), url.QueryEscape(hostname))
	body, err := c.do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("dns: list %s records for %q: %w", recordType, hostname, err)
	}
	var env struct {
		Success bool          `json:"success"`
		Result  []cfDNSRecord `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("dns: parse %s records for %q: %w", recordType, hostname, err)
	}
	return env.Result, nil
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
// so "react-component-library.example.invalid" -> "example.invalid". A bare apex
// (no subdomain) is returned unchanged.
func apexOf(hostname string) string {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	i := strings.IndexByte(hostname, '.')
	if i < 0 {
		return hostname
	}
	rest := hostname[i+1:]
	if !strings.Contains(rest, ".") {
		// hostname was already an apex like "example.invalid".
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
