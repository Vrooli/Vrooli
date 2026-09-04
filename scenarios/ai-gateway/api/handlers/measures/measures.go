// Package measures exposes AI Gateway's declared route-analytics measures over
// the route_events table, surfaced two ways over a SINGLE shared compute path so
// the two can never disagree:
//
//   - the packages/measures-go serve registry, mounted at /measures
//     (GET /measures/declarations + POST /measures/execute) — the contract the
//     behavioral probe and the search-hub central index call;
//   - the Connect-RPC MeasuresService (RegisterRoutes) — the typed CLI/UI surface.
//
// Values are real SQL aggregates over route_events (internal/routing), never a
// list-and-filter, and never read prompt/response text. Measures are analytics
// only — they must not become hidden route scoring.
//
// Proto: packages/proto/schemas/ai-gateway/v1/measures/measures.proto
package measures

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	measures "github.com/vrooli/measures-go"

	aigwmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"
	aigwmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures/measures_v1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"

	"ai-gateway/internal/routing"
)

// Metrics is the narrow route_events aggregate surface the measures compute
// against. The production *routing.SQLRepository satisfies it; tests inject a
// fake.
type Metrics interface {
	Aggregate(ctx context.Context, from, to time.Time) (routing.RouteAggregate, error)
	LatencyP95(ctx context.Context, from, to time.Time) (int64, error)
}

// computeFn computes one measure's scalar value over [from, to) and returns the
// formatted value plus executed-query provenance. Shared by the serve registry
// and the Connect RPC so a measure and its RPC report identical numbers.
type computeFn func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error)

type spec struct {
	decl    measures.MeasureDeclaration
	compute computeFn
}

// Measure names — the conventional "<domain>.<command>" identifiers. They MUST
// equal the manifest's domain.command so the behavioral probe's /execute call
// resolves them. The domain is route_events so the measures-health substrate
// coverage check for the route_events table is satisfied.
const (
	MeasureRouteTotal            = "route_events.total"
	MeasureRouteSuccessRate      = "route_events.success-rate"
	MeasureRouteFallbackRate     = "route_events.fallback-rate"
	MeasureRouteFailureRate      = "route_events.failure-rate"
	MeasureRouteBreakerOpen      = "route_events.breaker-open"
	MeasureRouteCapacityRejected = "route_events.capacity-rejections"
	MeasureRouteLatencyP95       = "route_events.latency-p95"
	MeasureRouteCost             = "route_events.cost"
	MeasureRouteTokens           = "route_events.tokens"
	MeasureRouteLocalShare       = "route_events.local-share"
)

func specs() []spec {
	return []spec{
		{
			decl: countDecl(MeasureRouteTotal, "CountRouteEvents",
				"How many AI routes were executed in a time window.",
				[]string{
					"how many ai routes ran this week",
					"gateway route count last 30 days",
					"how many gateway requests executed this month",
				}, "routes", "{count} routes executed ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return strconv.FormatInt(agg.Total, 10), countQuery("*", from, to), err
			},
		},
		{
			decl: rateDecl(MeasureRouteSuccessRate, "RouteSuccessRate",
				"What fraction of AI routes succeeded in a time window.",
				[]string{
					"gateway route success rate this week",
					"how often did ai routing succeed last month",
					"route success ratio in the last 7 days",
				}, "{rate} route success rate ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return formatRate(agg.Succeeded, agg.Total), rateQuery("status = 'succeeded'", from, to), err
			},
		},
		{
			decl: rateDecl(MeasureRouteFallbackRate, "RouteFallbackRate",
				"What fraction of successful AI routes used a fallback provider in a time window.",
				[]string{
					"gateway fallback rate this week",
					"how often did routing fall back to another provider last month",
					"route fallback ratio in the last 30 days",
				}, "{rate} of successful routes used a fallback ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return formatRate(agg.FallbackUsed, agg.Succeeded), rateQuery("fallback_used = 1", from, to), err
			},
		},
		{
			decl: rateDecl(MeasureRouteFailureRate, "RouteFailureRate",
				"What fraction of AI routes failed in a time window.",
				[]string{
					"gateway route failure rate this week",
					"how often did ai routing fail last month",
					"provider failure ratio in the last 7 days",
				}, "{rate} route failure rate ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return formatRate(agg.Failed, agg.Total), rateQuery("status = 'failed'", from, to), err
			},
		},
		{
			decl: countDecl(MeasureRouteBreakerOpen, "CountBreakerOpenRoutes",
				"How many AI routes were blocked by an open provider circuit breaker in a time window.",
				[]string{
					"how many routes hit an open breaker this week",
					"gateway breaker-open count last 30 days",
					"how often was a provider suppressed by its breaker this month",
				}, "routes", "{count} routes blocked by an open breaker ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return strconv.FormatInt(agg.BreakerOpen, 10), countQuery("rejection_reason = 'provider_breaker_open'", from, to), err
			},
		},
		{
			decl: countDecl(MeasureRouteCapacityRejected, "CountCapacityRejections",
				"How many local AI routes were rejected for insufficient capacity in a time window.",
				[]string{
					"how many local routes lacked capacity this week",
					"gateway capacity rejection count last 30 days",
					"how often did local routing fail for capacity this month",
				}, "routes", "{count} local routes rejected for capacity ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return strconv.FormatInt(agg.CapacityRejected, 10), countQuery("capacity_verdict = 'insufficient_capacity'", from, to), err
			},
		},
		{
			decl: latencyDecl(MeasureRouteLatencyP95, "RouteLatencyP95",
				"The p95 AI route latency in milliseconds over a time window.",
				[]string{
					"gateway p95 route latency this week",
					"95th percentile routing latency last month",
					"route latency p95 in the last 7 days",
				}, "{latency_ms} ms p95 route latency ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				p95, err := m.LatencyP95(ctx, from, to)
				return strconv.FormatInt(p95, 10), latencyQuery(from, to), err
			},
		},
		{
			decl: scalarDecl(MeasureRouteCost, "RouteCost", "Total priced AI route cost in USD over a time window.", []string{"total gateway cost this week", "how much did AI routing cost last month", "route cost in dollars"}, "cost_usd", "usd", "{cost_usd} USD route cost ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				if err != nil {
					return "", costQuery(from, to), err
				}
				if agg.PricedRows == 0 {
					return "", costQuery(from, to), fmt.Errorf("route cost unavailable: window contains no priced route rows")
				}
				return strconv.FormatFloat(agg.CostUSD, 'f', 6, 64), costQuery(from, to), nil
			},
		},
		{
			decl: scalarDecl(MeasureRouteTokens, "RouteTokens", "Total input and output tokens used by AI routes in a time window.", []string{"how many AI tokens were used this week", "gateway token usage last month", "route input and output token totals"}, "input_tokens", "tokens", "{input_tokens} input tokens and {output_tokens} output tokens ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return strconv.FormatInt(agg.InputTokens+agg.OutputTokens, 10), tokenQuery(from, to), err
			},
		},
		{
			decl: scalarDecl(MeasureRouteLocalShare, "RouteLocalShare", "Fraction of AI routes served locally in a time window.", []string{"what share of routes were local this week", "gateway local serving share last month", "how often did local AI routing serve the request"}, "share", "ratio", "{share} local-served route share ({window})"),
			compute: func(ctx context.Context, m Metrics, from, to time.Time) (string, string, error) {
				agg, err := m.Aggregate(ctx, from, to)
				return formatRate(agg.LocalServed, agg.Total), localShareQuery(from, to), err
			},
		},
	}
}

func formatRate(numerator, denominator int64) string {
	if denominator <= 0 {
		return "0"
	}
	return strconv.FormatFloat(float64(numerator)/float64(denominator), 'f', 4, 64)
}

func windowParams() map[string]measures.Param {
	return map[string]measures.Param{
		"window": {
			Name:    "window",
			Type:    measures.ParamTypeTimeWindow,
			Default: string(measures.TokenThisWeek),
		},
	}
}

func baseDecl(name, method, intent string, questions []string, result measures.Result) measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:        name,
		Scenario:    "ai-gateway",
		Domain:      "route_events",
		Intent:      intent,
		Questions:   questions,
		Params:      windowParams(),
		Result:      result,
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "MeasuresService",
		Method:      method,
	}
}

func countDecl(name, method, intent string, questions []string, unit, summary string) measures.MeasureDeclaration {
	return baseDecl(name, method, intent, questions, measures.Result{
		Kind:            measures.ResultScalar,
		ValueField:      "count",
		Unit:            unit,
		SummaryTemplate: summary,
	})
}

func rateDecl(name, method, intent string, questions []string, summary string) measures.MeasureDeclaration {
	return baseDecl(name, method, intent, questions, measures.Result{
		Kind:            measures.ResultScalar,
		ValueField:      "rate",
		Unit:            "ratio",
		SummaryTemplate: summary,
	})
}

func scalarDecl(name, method, intent string, questions []string, valueField, unit, summary string) measures.MeasureDeclaration {
	return baseDecl(name, method, intent, questions, measures.Result{Kind: measures.ResultScalar, ValueField: valueField, Unit: unit, SummaryTemplate: summary})
}

func latencyDecl(name, method, intent string, questions []string, summary string) measures.MeasureDeclaration {
	return baseDecl(name, method, intent, questions, measures.Result{
		Kind:            measures.ResultScalar,
		ValueField:      "latency_ms",
		Unit:            "ms",
		SummaryTemplate: summary,
	})
}

func countQuery(cond string, from, to time.Time) string {
	if cond == "*" {
		return fmt.Sprintf("SELECT COUNT(*) FROM route_events WHERE %s", windowClause(from, to))
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM route_events WHERE %s AND %s", cond, windowClause(from, to))
}

func rateQuery(cond string, from, to time.Time) string {
	return fmt.Sprintf(
		"SELECT SUM(CASE WHEN %s THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(*), 0) FROM route_events WHERE %s",
		cond, windowClause(from, to))
}

func latencyQuery(from, to time.Time) string {
	return fmt.Sprintf(
		"SELECT latency_ms FROM route_events WHERE %s ORDER BY latency_ms LIMIT 1 OFFSET nearest_rank(0.95)",
		windowClause(from, to))
}

func costQuery(from, to time.Time) string {
	return fmt.Sprintf("SELECT SUM(cost_estimate) FROM route_events WHERE cost_estimate > 0 AND %s", windowClause(from, to))
}

func tokenQuery(from, to time.Time) string {
	return fmt.Sprintf("SELECT SUM(input_tokens + output_tokens) FROM route_events WHERE %s", windowClause(from, to))
}

func localShareQuery(from, to time.Time) string {
	return fmt.Sprintf("SELECT SUM(CASE WHEN selected_locality = 'local' AND status = 'succeeded' THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(*), 0) FROM route_events WHERE %s", windowClause(from, to))
}

func windowClause(from, to time.Time) string {
	return fmt.Sprintf("substr(created_at,1,19) >= %q AND substr(created_at,1,19) < %q",
		from.UTC().Format("2006-01-02T15:04:05"), to.UTC().Format("2006-01-02T15:04:05"))
}

// Declarations returns the measure declarations (no compute) for harvesting.
func Declarations() []measures.MeasureDeclaration {
	ss := specs()
	out := make([]measures.MeasureDeclaration, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.decl)
	}
	return out
}

// NewRegistry builds the measures-go serve registry over the given metrics.
func NewRegistry(m Metrics, now func() time.Time) (*measures.Registry, error) {
	if now == nil {
		now = time.Now
	}
	reg := measures.NewRegistry(measures.WithClock(now))
	for _, s := range specs() {
		s := s
		err := reg.Register(s.decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
			rng, err := resolveToken(req.Params["window"], now())
			if err != nil {
				return measures.MeasureResult{}, err
			}
			val, query, err := s.compute(ctx, m, rng.From, rng.To)
			if err != nil {
				return measures.MeasureResult{}, err
			}
			return measures.MeasureResult{
				Value:      val,
				Provenance: measures.Provenance{ExecutedQuery: query},
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("measures: register %s: %w", s.decl.Name, err)
		}
	}
	return reg, nil
}

// MeasuresHandler returns the http.Handler serving GET /declarations and
// POST /execute (mounted under /measures by the caller).
func MeasuresHandler(m Metrics, now func() time.Time) (http.Handler, error) {
	reg, err := NewRegistry(m, now)
	if err != nil {
		return nil, err
	}
	return reg.Handler(), nil
}

func resolveToken(token string, now time.Time) (measures.Range, error) {
	t := measures.TimeWindowToken(token)
	if t == "" {
		t = measures.TokenThisWeek
	}
	return measures.ResolveToken(t, now, time.UTC)
}

func resolveProtoWindow(tw *measuresv1.TimeWindow, now time.Time) (measures.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measures.ResolveToken(measures.TokenThisWeek, now, time.UTC)
	}
	return measures.ResolveTimeWindow(tw, now, time.UTC)
}

// -----------------------------------------------------------------------------
// Connect-RPC MeasuresService — the typed surface, sharing the compute path.
// -----------------------------------------------------------------------------

// Handler implements vrooli.ai_gateway.v1.measures.MeasuresService.
type Handler struct {
	metrics Metrics
	now     func() time.Time
	byName  map[string]computeFn
}

// NewHandler constructs the Connect handler. now anchors window resolution; nil = time.Now.
func NewHandler(m Metrics, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	byName := map[string]computeFn{}
	for _, s := range specs() {
		byName[s.decl.Name] = s.compute
	}
	return &Handler{metrics: m, now: now, byName: byName}
}

func (h *Handler) value(ctx context.Context, name string, tw *measuresv1.TimeWindow) (string, error) {
	rng, err := resolveProtoWindow(tw, h.now())
	if err != nil {
		return "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	val, _, err := h.byName[name](ctx, h.metrics, rng.From, rng.To)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	}
	return val, nil
}

func (h *Handler) count(ctx context.Context, name string, tw *measuresv1.TimeWindow) (int64, error) {
	val, err := h.value(ctx, name, tw)
	if err != nil {
		return 0, err
	}
	n, _ := strconv.ParseInt(val, 10, 64)
	return n, nil
}

func (h *Handler) rate(ctx context.Context, name string, tw *measuresv1.TimeWindow) (float64, error) {
	val, err := h.value(ctx, name, tw)
	if err != nil {
		return 0, err
	}
	f, _ := strconv.ParseFloat(val, 64)
	return f, nil
}

// countResp / rateResp wrap a scalar compute into the typed Connect response so
// each RPC method stays a single delegating line (the Connect interface requires
// one method per RPC; the shared body lives here).
func (h *Handler) countResp(ctx context.Context, name string, tw *measuresv1.TimeWindow) (*connect.Response[aigwmeasuresv1.RouteCountResponse], error) {
	n, err := h.count(ctx, name, tw)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&aigwmeasuresv1.RouteCountResponse{Count: n}), nil
}

func (h *Handler) rateResp(ctx context.Context, name string, tw *measuresv1.TimeWindow) (*connect.Response[aigwmeasuresv1.RouteRateResponse], error) {
	r, err := h.rate(ctx, name, tw)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&aigwmeasuresv1.RouteRateResponse{Rate: r}), nil
}

func (h *Handler) CountRouteEvents(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteCountResponse], error) {
	return h.countResp(ctx, MeasureRouteTotal, req.Msg.GetWindow())
}

func (h *Handler) RouteSuccessRate(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteRateResponse], error) {
	return h.rateResp(ctx, MeasureRouteSuccessRate, req.Msg.GetWindow())
}

func (h *Handler) RouteFallbackRate(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteRateResponse], error) {
	return h.rateResp(ctx, MeasureRouteFallbackRate, req.Msg.GetWindow())
}

func (h *Handler) RouteFailureRate(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteRateResponse], error) {
	return h.rateResp(ctx, MeasureRouteFailureRate, req.Msg.GetWindow())
}

func (h *Handler) CountBreakerOpenRoutes(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteCountResponse], error) {
	return h.countResp(ctx, MeasureRouteBreakerOpen, req.Msg.GetWindow())
}

func (h *Handler) CountCapacityRejections(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteCountResponse], error) {
	return h.countResp(ctx, MeasureRouteCapacityRejected, req.Msg.GetWindow())
}

func (h *Handler) RouteLatencyP95(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteLatencyResponse], error) {
	n, err := h.count(ctx, MeasureRouteLatencyP95, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&aigwmeasuresv1.RouteLatencyResponse{LatencyMs: n}), nil
}

func (h *Handler) RouteCost(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteCostResponse], error) {
	val, err := h.value(ctx, MeasureRouteCost, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	cost, _ := strconv.ParseFloat(val, 64)
	return connect.NewResponse(&aigwmeasuresv1.RouteCostResponse{CostUsd: cost}), nil
}

func (h *Handler) RouteTokens(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteTokenResponse], error) {
	val, err := h.value(ctx, MeasureRouteTokens, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	tokens, _ := strconv.ParseInt(val, 10, 64)
	return connect.NewResponse(&aigwmeasuresv1.RouteTokenResponse{TotalTokens: tokens}), nil
}

func (h *Handler) RouteLocalShare(ctx context.Context, req *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[aigwmeasuresv1.RouteShareResponse], error) {
	val, err := h.value(ctx, MeasureRouteLocalShare, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	share, _ := strconv.ParseFloat(val, 64)
	return connect.NewResponse(&aigwmeasuresv1.RouteShareResponse{Share: share}), nil
}

// RegisterRoutes mounts the Connect MeasuresService on the given mux router.
func RegisterRoutes(router *mux.Router, m Metrics, now func() time.Time) {
	path, handler := aigwmeasuresconnect.NewMeasuresServiceHandler(NewHandler(m, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
