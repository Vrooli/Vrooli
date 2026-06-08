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

	aisearch "github.com/vrooli/aisearch-go"
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

func (c *httpProviderClient) Search(ctx context.Context, d *registryv1.ProviderDescriptor, query string, limit int32, opts internaleval.SearchCallOptions) ([]*routingv1.SearchHit, error) {
	hj := d.GetEndpoint().GetHttpJson()
	if hj == nil {
		return nil, fmt.Errorf("provider %q: only http_json endpoints are callable", d.GetProviderId())
	}
	base, err := c.resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", hj.GetScenarioId(), err)
	}
	body, err := providers.RenderBody(hj.GetBodyTemplate(), query, limit, d.GetType())
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
