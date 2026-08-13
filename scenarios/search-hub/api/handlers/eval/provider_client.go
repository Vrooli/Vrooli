package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	internaleval "search-hub/internal/eval"
	"search-hub/internal/httpc"
	"search-hub/internal/providers"
	internalvalidation "search-hub/internal/validation"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// URLResolver turns a logical scenario_id into the scenario's live API base URL
// at call-time — the same backend cross-scenario resolution the router uses
// (never client-computed). Declared at the consumer (seam-discovery).
type URLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioID string) (baseURL string, err error)
}

const (
	evalCallTimeout   = 15 * time.Second
	maxResponseBytes  = 8 << 20 // 8 MiB — mirrors the router's response cap.
	evalStatusTimeout = 5 * time.Second
)

// httpProviderClient is the production internaleval.ProviderClient: it resolves
// a provider's base URL, calls its registered Search endpoint, and maps the
// response through the shared providers.MapResults adapter — exactly the
// router's call path, reused so the eval runner reaches a provider identically.
type httpProviderClient struct {
	resolver URLResolver
	doer     httpc.Doer
}

func newHTTPProviderClient(resolver URLResolver, doer httpc.Doer) *httpProviderClient {
	return &httpProviderClient{resolver: resolver, doer: doer}
}

// NewDefaultProviderClient returns the production provider client used by both
// eval execution and validation-time live corpus probing.
func NewDefaultProviderClient() internaleval.ProviderClient {
	return newHTTPProviderClient(newScenarioResolver(), httpc.NewDefault())
}

// NewDefaultStatusProbe returns the same descriptor-driven HTTP seam used by
// eval validation, exposed separately so maturity validation can report an
// absent index timestamp without importing routing internals.
func NewDefaultStatusProbe() internalvalidation.StatusProbe {
	return newHTTPProviderClient(newScenarioResolver(), httpc.NewDefault())
}

func (c *httpProviderClient) Search(ctx context.Context, d *registryv1.ProviderDescriptor, query string, limit int32, opts internaleval.SearchCallOptions) ([]*routingv1.SearchHit, error) {
	hj := d.GetEndpoint().GetHttpJson()
	if hj == nil {
		return nil, fmt.Errorf("provider %q: only http_json endpoints are callable", d.GetProviderId())
	}
	base, err := c.resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", hj.GetScenarioId(), err)
	}
	body, err := providers.RenderBodyWithScope(hj.GetBodyTemplate(), query, limit, d.GetType(), opts.Scope)
	if err != nil {
		return nil, fmt.Errorf("render body: %w", err)
	}

	cctx, cancel := context.WithTimeout(ctx, evalCallTimeout)
	defer cancel()
	url := strings.TrimRight(base, "/") + hj.GetPath()
	req, err := http.NewRequestWithContext(cctx, providers.HTTPMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	providers.ApplyHeaders(req, hj.GetHeaders())
	if err := applyOverrideHeaders(req, opts); err != nil {
		return nil, err
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	hits, err := providers.MapResults(d, raw)
	if err != nil {
		return nil, fmt.Errorf("map results: %w", err)
	}
	if int32(len(hits)) > limit && limit > 0 {
		hits = hits[:limit]
	}
	return hits, nil
}

// applyOverrideHeaders sets the query-time override + control-token headers on
// req when opts carries non-zero overrides. It is a no-op for the baseline call
// (nil/zero overrides), so an ordinary eval run sends no extra headers and the
// provider's public search path is untouched. The header names + JSON shape are
// the shared aisearch contract (override_transport.go) both sides import, so they
// cannot drift.
func applyOverrideHeaders(req *http.Request, opts internaleval.SearchCallOptions) error {
	if opts.Overrides == nil || opts.Overrides.IsZero() {
		return nil
	}
	value, err := aisearch.MarshalOverridesHeader(*opts.Overrides)
	if err != nil {
		return fmt.Errorf("encode search overrides: %w", err)
	}
	if value == "" {
		return nil
	}
	req.Header.Set(aisearch.OverridesHeader, value)
	if opts.ControlToken != "" {
		req.Header.Set(aisearch.ControlTokenHeader, opts.ControlToken)
	}
	return nil
}

// Snapshot probes the provider's status_endpoint (if registered) and extracts
// the well-known config fields generically — no provider-specific code. Fields
// it can't find are left zero (honest). Best-effort: any failure yields an empty
// snapshot rather than an error, so a status-less or down provider never fails a
// run (the reranker_leg simply reads "unknown").
func (c *httpProviderClient) Snapshot(ctx context.Context, d *registryv1.ProviderDescriptor) *evalv1.ConfigSnapshot {
	snap := &evalv1.ConfigSnapshot{}
	hj := d.GetStatusEndpoint().GetHttpJson()
	if hj == nil {
		return snap
	}
	base, err := c.resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return snap
	}
	body := strings.TrimSpace(hj.GetBodyTemplate())
	if body == "" {
		body = "{}"
	}
	cctx, cancel := context.WithTimeout(ctx, evalStatusTimeout)
	defer cancel()
	url := strings.TrimRight(base, "/") + hj.GetPath()
	req, err := http.NewRequestWithContext(cctx, providers.HTTPMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return snap
	}
	providers.ApplyHeaders(req, hj.GetHeaders())
	resp, err := c.doer.Do(req)
	if err != nil {
		return snap
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return snap
	}
	return parseSnapshot(raw)
}

func (c *httpProviderClient) ProbeIndexTimestamp(ctx context.Context, d *registryv1.ProviderDescriptor) (time.Time, error) {
	hj := d.GetStatusEndpoint().GetHttpJson()
	if hj == nil {
		return time.Time{}, nil
	}
	base, err := c.resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return time.Time{}, err
	}
	body := strings.TrimSpace(hj.GetBodyTemplate())
	if body == "" {
		body = "{}"
	}
	cctx, cancel := context.WithTimeout(ctx, evalStatusTimeout)
	defer cancel()
	url := strings.TrimRight(base, "/") + hj.GetPath()
	req, err := http.NewRequestWithContext(cctx, providers.HTTPMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return time.Time{}, err
	}
	providers.ApplyHeaders(req, hj.GetHeaders())
	resp, err := c.doer.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return time.Time{}, fmt.Errorf("status endpoint returned HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return time.Time{}, err
	}
	return parseIndexTimestamp(raw, d.GetIndexTimestampField()), nil
}

func parseIndexTimestamp(raw []byte, declaredField string) time.Time {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}
	}
	if timestamp := parseTimestampValue(fieldValue(payload, declaredField)); !timestamp.IsZero() {
		return timestamp
	}
	for _, key := range []string{"last_indexed_at", "lastIndexedAt", "last_index_at", "lastIndexAt", "index_updated_at", "indexUpdatedAt", "indexed_at", "indexedAt", "last_reindex_at", "lastReindexAt"} {
		if timestamp := parseTimestampValue(payload[key]); !timestamp.IsZero() {
			return timestamp
		}
	}
	return time.Time{}
}

func fieldValue(payload map[string]any, field string) any {
	var current any = payload
	for _, part := range strings.Split(strings.TrimSpace(field), ".") {
		if part == "" {
			return nil
		}
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = object[part]
		if !ok {
			return nil
		}
	}
	return current
}

func parseTimestampValue(value any) time.Time {
	text, ok := value.(string)
	if !ok {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05Z07:00"} {
		if timestamp, err := time.Parse(layout, text); err == nil {
			return timestamp
		}
	}
	return time.Time{}
}

// parseSnapshot maps a status JSON body onto a ConfigSnapshot using a small set
// of well-known keys (Connect's protojson uses lowerCamelCase; some surfaces use
// snake_case — both accepted). reranker → reranker_leg; indexed_count /
// indexedCount → indexed_count; embed_model / model → embed_model;
// rerank_enabled derived from a non-"none" leg.
func parseSnapshot(raw []byte) *evalv1.ConfigSnapshot {
	snap := &evalv1.ConfigSnapshot{}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return snap
	}
	if s := firstString(m, "reranker", "reranker_leg", "rerankerLeg"); s != "" {
		snap.RerankerLeg = s
		snap.RerankEnabled = s != "none" && s != ""
	}
	if s := firstString(m, "embed_model", "embedModel", "model"); s != "" {
		snap.EmbedModel = s
	}
	if n, ok := firstNumber(m, "indexed_count", "indexedCount"); ok {
		snap.IndexedCount = int32(n)
	}
	return snap
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func firstNumber(m map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			return v, true
		}
	}
	return 0, false
}
