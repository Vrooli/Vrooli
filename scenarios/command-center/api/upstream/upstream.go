// Package upstream provides read-only clients for the three data sources
// command-center aggregates: Swarm Manager, Vrooli Core, and LPBS.
package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	swarmstatsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats"
	swarmstatsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats/stats_v1connect"
)

// ErrNotAvailable is returned when an upstream is not configured or cannot
// be reached. Callers should fall back to gap-mode data.
var ErrNotAvailable = errors.New("upstream not available")

// Client is the shared read interface for every upstream source.
type Client interface {
	// Name returns the source identifier (e.g. "swarm", "vrooli", "lpbs").
	Name() string
	// Fetch performs a GET against the upstream and returns the raw body.
	Fetch(ctx context.Context, path string) (json.RawMessage, error)
}

// NewRESTResolved creates a lifecycle-aware JSON client for a producer whose
// read surface is REST. It is intentionally small: the dashboard owns
// interpretation and validation of the returned contract.
func NewRESTResolved(name string, resolve func() string, bearerToken string) Client {
	c := newResolved(name, resolve)
	if bearerToken != "" {
		c.authFn = func(req *http.Request) { req.Header.Set("Authorization", "Bearer "+bearerToken) }
	}
	return c
}

type jsonConnectClient struct {
	name     string
	resolve  func() string
	path     string
	http     *http.Client
	features map[string]string
}

func NewJSONConnectResolved(name string, resolve func() string, path string, features ...map[string]string) Client {
	return &jsonConnectClient{name: name, resolve: resolve, path: path, http: &http.Client{Timeout: 5 * time.Second}, features: optionalFeatureSet(features)}
}

func (c *jsonConnectClient) Name() string { return c.name }

func (c *jsonConnectClient) Fetch(ctx context.Context, path string) (json.RawMessage, error) {
	if path == "/health" {
		return newResolved(c.name, c.resolve).Fetch(ctx, path)
	}
	if path != c.path {
		return nil, fmt.Errorf("%s JSON Connect client does not expose path %q", c.name, path)
	}
	base := c.resolve()
	if base == "" {
		return nil, ErrNotAvailable
	}
	body, err := json.Marshal(map[string]any{})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("%s http %d", c.name, response.StatusCode)
	}
	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%s read body: %w", c.name, err)
	}
	return json.RawMessage(result), nil
}

func (c *jsonConnectClient) ProbeFeatures(ctx context.Context) (map[string]string, map[string]string) {
	if _, err := c.Fetch(ctx, c.path); err != nil {
		return nil, nil
	}
	statuses, reasons := map[string]string{}, map[string]string{}
	for feature := range c.features {
		statuses[feature] = "compatible"
		reasons[feature] = "JSON Connect producer projection returned successfully"
	}
	return statuses, reasons
}

// FeatureProbe is implemented by typed producer clients that can prove a
// feature contract independently of the producer's cheap health endpoint.
// Legacy REST fallbacks intentionally do not implement this interface.
type FeatureProbe interface {
	ProbeFeatures(context.Context) (map[string]string, map[string]string)
}

// baseClient is the shared HTTP plumbing used by every upstream.
type baseClient struct {
	name    string
	resolve func() string
	http    *http.Client
	authFn  func(req *http.Request)
}

func newBase(name, baseURL string) *baseClient {
	return newResolved(name, func() string { return baseURL })
}

// newResolved builds a client whose base URL is looked up on every fetch, so
// a source whose port is assigned at runtime by the lifecycle manager can be
// found after it restarts without restarting this API.
func newResolved(name string, resolve func() string) *baseClient {
	return &baseClient{
		name:    name,
		resolve: resolve,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *baseClient) Name() string { return c.name }

func (c *baseClient) Fetch(ctx context.Context, path string) (json.RawMessage, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		body, err := c.fetchOnce(ctx, path)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryableHTTPError(ctx, err) || attempt == 1 {
			break
		}
		if err := waitBeforeRetry(ctx); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *baseClient) fetchOnce(ctx context.Context, path string) (json.RawMessage, error) {
	baseURL := c.resolve()
	if baseURL == "" {
		return nil, ErrNotAvailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.authFn != nil {
		c.authFn(req)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s read body: %w", c.name, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotAvailable
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s http %d: %s", c.name, resp.StatusCode, truncate(string(body), 200))
	}
	return json.RawMessage(body), nil
}

func retryableHTTPError(ctx context.Context, err error) bool {
	if err == nil || errors.Is(err, ErrNotAvailable) || ctx.Err() != nil {
		return false
	}
	var networkErr net.Error
	return errors.As(err, &networkErr)
}

func waitBeforeRetry(ctx context.Context) error {
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func retryableConnectError(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	switch connect.CodeOf(err) {
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeResourceExhausted:
		return true
	default:
		return false
	}
}

func retryConnect(ctx context.Context, call func() error) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		lastErr = call()
		if lastErr == nil || !retryableConnectError(ctx, lastErr) || attempt == 1 {
			return lastErr
		}
		if err := waitBeforeRetry(ctx); err != nil {
			return err
		}
	}
	return lastErr
}

// typedClient owns the shared lifecycle for typed producer clients. Producers
// provide only their resolver, feature probe, and selector-envelope projection.
type typedClient struct {
	name     string
	resolve  func() string
	http     *http.Client
	token    string
	features map[string]string
	fetch    func(context.Context, *typedClient, string) (json.RawMessage, error)
	probe    func(context.Context, *typedClient) (map[string]string, map[string]string)
}

func optionalFeatureSet(sets []map[string]string) map[string]string {
	if len(sets) == 0 || sets[0] == nil {
		return map[string]string{}
	}
	return sets[0]
}

func newTypedClient(name string, resolve func() string, token string, features map[string]string, fetch func(context.Context, *typedClient, string) (json.RawMessage, error), probe func(context.Context, *typedClient) (map[string]string, map[string]string)) *typedClient {
	return &typedClient{name: name, resolve: resolve, http: &http.Client{Timeout: 5 * time.Second}, token: token, features: features, fetch: fetch, probe: probe}
}

func (c *typedClient) Name() string { return c.name }

func (c *typedClient) ProbeFeatures(ctx context.Context) (map[string]string, map[string]string) {
	if c.probe == nil {
		return nil, nil
	}
	return c.probe(ctx, c)
}

func (c *typedClient) Fetch(ctx context.Context, path string) (json.RawMessage, error) {
	if path == "/health" {
		health := newResolved(c.name, c.resolve)
		if c.token != "" {
			health.authFn = func(req *http.Request) { req.Header.Set("Authorization", "Bearer "+c.token) }
		}
		return health.Fetch(ctx, path)
	}
	if c.fetch == nil {
		return nil, fmt.Errorf("%s typed client has no projection", c.name)
	}
	return c.fetch(ctx, c, path)
}

// NewSwarm returns a client for the Swarm Manager scenario's REST API.
func NewSwarm(baseURL string) Client {
	return newBase("swarm", baseURL)
}

// NewSwarmResolved returns a Swarm Manager client whose base URL is resolved
// at fetch time.
func NewSwarmResolved(resolve func() string) Client {
	return newResolved("swarm", resolve)
}

// NewSwarmTypedResolved uses Swarm Manager's generated StatsService for the
// canonical portfolio projection. REST remains an explicit operational probe
// path only; it is not a fallback for the typed projection.
func NewSwarmTypedResolved(resolve func() string, features ...map[string]string) Client {
	return newTypedClient("swarm", resolve, "", optionalFeatureSet(features), swarmFetch, swarmProbe)
}

func swarmProbe(ctx context.Context, c *typedClient) (map[string]string, map[string]string) {
	baseURL := c.resolve()
	if baseURL == "" {
		return nil, nil
	}
	var err error
	retryErr := retryConnect(ctx, func() error {
		baseURL := c.resolve()
		if baseURL == "" {
			return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
		}
		_, err = swarmstatsconnect.NewStatsServiceClient(c.http, baseURL).GetPortfolioStats(ctx, connect.NewRequest(&swarmstatsv1.GetPortfolioStatsRequest{}))
		return err
	})
	if retryErr != nil {
		return nil, nil
	}
	features := map[string]string{}
	reasons := map[string]string{}
	for feature := range c.features {
		features[feature] = "compatible"
		reasons[feature] = "StatsService.GetPortfolioStats returned the typed producer projection"
	}
	return features, reasons
}

func swarmFetch(ctx context.Context, c *typedClient, path string) (json.RawMessage, error) {
	if path == "/api/v1/goals" {
		return newResolved("swarm", c.resolve).Fetch(ctx, path)
	}
	if path != "/api/v1/stats" {
		return nil, fmt.Errorf("swarm typed client does not expose path %q", path)
	}
	baseURL := c.resolve()
	if baseURL == "" {
		return nil, ErrNotAvailable
	}
	var response *connect.Response[swarmstatsv1.PortfolioStats]
	err := retryConnect(ctx, func() error {
		baseURL := c.resolve()
		if baseURL == "" {
			return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
		}
		var callErr error
		response, callErr = swarmstatsconnect.NewStatsServiceClient(c.http, baseURL).GetPortfolioStats(ctx, connect.NewRequest(&swarmstatsv1.GetPortfolioStatsRequest{}))
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("swarm typed stats: %w", err)
	}
	observedAt := ""
	if timestamp := response.Msg.GetObservedAt(); timestamp != nil {
		observedAt = timestamp.AsTime().UTC().Format(time.RFC3339)
	}
	payload := map[string]any{
		"observed_at":      observedAt,
		"contract_version": "legacy.v1",
		"units": map[string]string{
			"swarm_throughput":     "count",
			"throughput_stats":     "count",
			"swarm_active_agents":  "count",
			"agent_stats":          "percent",
			"timing_stats":         "minutes",
			"blocking_stats":       "count",
			"dashboard_stats":      "count",
			"review_stats":         "count",
			"scope_stats":          "count",
			"composite_throughput": "count",
		},
		"throughput": map[string]any{
			"completed_last_7_days": response.Msg.GetSwarmThroughput(),
			"created_last_7_days":   response.Msg.GetThroughputStats(),
		},
		"agent": map[string]any{
			"total_executions":      response.Msg.GetSwarmActiveAgents(),
			"success_rate":          response.Msg.GetAgentStats(),
			"avg_execution_minutes": response.Msg.GetTimingStats(),
		},
		"sample_size": response.Msg.GetAgentStatsSampleSize(),
		"blocking": map[string]any{
			"currently_blocked": response.Msg.GetBlockingStats(),
		},
		"dashboard": map[string]any{
			"total_backlog_size":       response.Msg.GetDashboardStats(),
			"total_completed_all_time": response.Msg.GetCompositeThroughput(),
		},
		"review": map[string]any{
			"rounds_completed": response.Msg.GetReviewStats(),
		},
		"scope_stats": response.Msg.GetScopeStats(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("swarm typed stats encode: %w", err)
	}
	return encoded, nil
}

// NewVrooli returns a client for the Vrooli core's REST API.
func NewVrooli(baseURL string) Client {
	return newBase("vrooli", baseURL)
}

// NewVrooliTypedResolved uses the control plane's generated scenario-list
// contract for the canonical inventory projection. REST remains available to
// callers that explicitly use an operational endpoint such as /health.
func NewVrooliTypedResolved(resolve func() string, features ...map[string]string) Client {
	return newTypedClient("vrooli", resolve, "", optionalFeatureSet(features), vrooliFetch, vrooliProbe)
}

func vrooliProbe(ctx context.Context, c *typedClient) (map[string]string, map[string]string) {
	response, err := c.list(ctx)
	if err != nil || response.Msg.GetObservedAt() == nil {
		return nil, nil
	}
	features := make(map[string]string, len(c.features))
	for feature := range c.features {
		features[feature] = "compatible"
	}
	reasons := make(map[string]string, len(features))
	for feature := range features {
		reasons[feature] = "ScenarioControlPlaneService.ListScenarios returned the typed producer projection"
	}
	return features, reasons
}

func vrooliFetch(ctx context.Context, c *typedClient, path string) (json.RawMessage, error) {
	if path != "/scenarios" {
		return nil, fmt.Errorf("vrooli typed client does not expose path %q", path)
	}
	response, err := c.list(ctx)
	if err != nil {
		return nil, fmt.Errorf("vrooli typed scenarios: %w", err)
	}
	rows := make([]map[string]any, 0, len(response.Msg.GetScenarios()))
	for _, scenario := range response.Msg.GetScenarios() {
		ports := make(map[string]any, len(scenario.GetPorts()))
		for _, port := range scenario.GetPorts() {
			ports[port.GetKey()] = port.GetPort()
		}
		rows = append(rows, map[string]any{
			"name":          scenario.GetName(),
			"description":   scenario.GetDescription(),
			"status":        scenario.GetStatus(),
			"health_status": scenario.GetHealthStatus(),
			"ports":         ports,
		})
	}
	observedAt := ""
	if timestamp := response.Msg.GetObservedAt(); timestamp != nil {
		observedAt = timestamp.AsTime().UTC().Format(time.RFC3339)
	}
	payload := map[string]any{
		"observed_at":      observedAt,
		"contract_version": "legacy.v1",
		"units": map[string]string{
			"active_scenarios":        "count",
			"scenario_health":         "count",
			"scenario_health_detail":  "count",
			"total_scenarios":         "count",
			"scenario_completeness":   "count",
			"scenario_ports":          "count",
			"composite_system_health": "count",
			"composite_portfolio":     "count",
		},
		"data": rows,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("vrooli typed scenarios encode: %w", err)
	}
	return encoded, nil
}

func (c *typedClient) list(ctx context.Context) (*connect.Response[cliv1.ScenarioListResponse], error) {
	baseURL := c.resolve()
	if baseURL == "" {
		return nil, ErrNotAvailable
	}
	var response *connect.Response[cliv1.ScenarioListResponse]
	err := retryConnect(ctx, func() error {
		baseURL := c.resolve()
		if baseURL == "" {
			return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
		}
		var callErr error
		response, callErr = cliv1connect.NewScenarioControlPlaneServiceClient(c.http, baseURL).ListScenarios(ctx, connect.NewRequest(&cliv1.ListScenariosRequest{IncludePorts: true}))
		return callErr
	})
	return response, err
}

// NewLPBS returns a client for the LPBS scenario's admin dashboard API.
// The bearer token is attached to every request when non-empty so that
// /api/v1/admin/dashboard/* is accessible once LPBS ships those handlers.
// On 404 (endpoints not yet implemented) the client returns ErrNotAvailable
// so the caller can fall through to gap-mode data from the registry.
func NewLPBS(baseURL, bearerToken string) Client {
	return NewLPBSResolved(func() string { return baseURL }, bearerToken)
}

// NewLPBSResolved is NewLPBS with a base URL resolved at fetch time.
func NewLPBSResolved(resolve func() string, bearerToken string) Client {
	c := newResolved("lpbs", resolve)
	if bearerToken != "" {
		c.authFn = func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}
	return c
}

// NewLPBSTypedResolved uses LPBS's generated MetricsService for its canonical
// analytics summary. REST remains an explicit operational probe path only;
// it is not a fallback for the typed projection.
func NewLPBSTypedResolved(resolve func() string, bearerToken string, features ...map[string]string) Client {
	return newTypedClient("lpbs", resolve, bearerToken, optionalFeatureSet(features), lpbsFetch, lpbsProbe)
}

func lpbsProbe(ctx context.Context, c *typedClient) (map[string]string, map[string]string) {
	baseURL := c.resolve()
	if baseURL == "" {
		return nil, nil
	}
	httpClient := c.http
	if c.token != "" {
		httpClient = &http.Client{Timeout: c.http.Timeout, Transport: bearerTransport{base: c.http.Transport, token: c.token}}
	}
	summaryErr := retryConnect(ctx, func() error {
		baseURL := c.resolve()
		if baseURL == "" {
			return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
		}
		_, callErr := lpbsconnect.NewMetricsServiceClient(httpClient, baseURL).GetAnalyticsSummary(ctx, connect.NewRequest(&lpbsv1.GetAnalyticsSummaryRequest{}))
		return callErr
	})
	revenueErr := retryConnect(ctx, func() error {
		baseURL := c.resolve()
		if baseURL == "" {
			return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
		}
		_, callErr := lpbsconnect.NewAdminRevenueServiceClient(httpClient, baseURL).GetRevenueSummary(ctx, connect.NewRequest(&lpbsv1.GetRevenueSummaryRequest{}))
		return callErr
	})
	if summaryErr != nil && revenueErr != nil {
		return nil, nil
	}
	features := map[string]string{}
	if summaryErr == nil {
		for feature, kind := range c.features {
			if kind != "analytics" {
				continue
			}
			features[feature] = "compatible"
		}
	}
	if revenueErr == nil {
		for feature, kind := range c.features {
			if kind != "revenue" {
				continue
			}
			features[feature] = "compatible"
		}
	}
	reasons := map[string]string{}
	for feature := range features {
		reasons[feature] = "LPBS typed producer projection returned a compatible contract"
	}
	return features, reasons
}

func lpbsFetch(ctx context.Context, c *typedClient, path string) (json.RawMessage, error) {
	if path != "/api/v1/admin/dashboard/summary" && path != "/api/v1/admin/dashboard/revenue" {
		return nil, fmt.Errorf("lpbs typed client does not expose path %q", path)
	}
	baseURL := c.resolve()
	if baseURL == "" {
		return nil, ErrNotAvailable
	}
	httpClient := c.http
	if c.token != "" {
		httpClient = &http.Client{Timeout: c.http.Timeout, Transport: bearerTransport{base: c.http.Transport, token: c.token}}
	}
	if path == "/api/v1/admin/dashboard/revenue" {
		var response *connect.Response[lpbsv1.RevenueSummary]
		err := retryConnect(ctx, func() error {
			baseURL := c.resolve()
			if baseURL == "" {
				return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
			}
			var callErr error
			response, callErr = lpbsconnect.NewAdminRevenueServiceClient(httpClient, baseURL).GetRevenueSummary(ctx, connect.NewRequest(&lpbsv1.GetRevenueSummaryRequest{}))
			return callErr
		})
		if err != nil {
			return nil, fmt.Errorf("lpbs typed revenue: %w", err)
		}
		observedAt := ""
		if timestamp := response.Msg.GetObservedAt(); timestamp != nil {
			observedAt = timestamp.AsTime().UTC().Format(time.RFC3339)
		}
		payload := map[string]any{
			"observed_at": observedAt, "contract_version": "revenue-summary.v1",
			"units": map[string]string{"revenue_mrr": "currency", "revenue_today": "currency", "revenue_rollup": "currency", "composite_revenue": "currency"},
			"revenue": map[string]any{
				"mrr":   float64(response.Msg.GetMrrMinor()) / 100,
				"today": float64(response.Msg.GetRevenueTodayMinor()) / 100,
				"month": float64(response.Msg.GetRevenueWindowMinor()) / 100,
			},
			"sample_size": response.Msg.GetSampleSize(), "currency": response.Msg.GetCurrency(),
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("lpbs typed revenue encode: %w", err)
		}
		return encoded, nil
	}
	var response *connect.Response[lpbsv1.AnalyticsSummary]
	err := retryConnect(ctx, func() error {
		baseURL := c.resolve()
		if baseURL == "" {
			return connect.NewError(connect.CodeUnavailable, ErrNotAvailable)
		}
		var callErr error
		response, callErr = lpbsconnect.NewMetricsServiceClient(httpClient, baseURL).GetAnalyticsSummary(ctx, connect.NewRequest(&lpbsv1.GetAnalyticsSummaryRequest{}))
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("lpbs typed metrics: %w", err)
	}
	// Keep the selector envelope stable while making the producer contract
	// typed. Preserve the producer-owned observation timestamp; the qualifier
	// grades freshness from it rather than from consumer fetch time.
	var ctaClicks, conversions, variantCount int64
	for _, stat := range response.Msg.GetVariantStats() {
		ctaClicks += stat.GetCtaClicks()
		conversions += stat.GetConversions()
		variantCount++
	}
	observedAt := ""
	if timestamp := response.Msg.GetObservedAt(); timestamp != nil {
		observedAt = timestamp.AsTime().UTC().Format(time.RFC3339)
	}
	payload := map[string]any{
		"observed_at":      observedAt,
		"contract_version": "legacy.v1",
		"units": map[string]string{
			"visitors":    "count",
			"cta_clicks":  "count",
			"conversions": "count",
			"variant_ab":  "percent",
		},
		"visitors":    response.Msg.GetTotalVisitors(),
		"cta_clicks":  ctaClicks,
		"conversions": conversions,
		"variant_ab":  variantCount,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("lpbs typed metrics encode: %w", err)
	}
	return encoded, nil
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return base.RoundTrip(clone)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
