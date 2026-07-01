package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	gomeasures "github.com/vrooli/measures-go"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	shmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/measures"

	internalmetrics "search-hub/internal/metrics"
)

const (
	MeasureFederatedLatency        = "query_telemetry.federated-latency"
	MeasureDegradedQueryRate       = "query_telemetry.degraded-query-rate"
	MeasureProviderDegradationRate = "query_telemetry.provider-degradation-rate"
)

// RangeInsightsReader is the exact-window telemetry seam used by declared
// measures after resolving the canonical time_window param.
type RangeInsightsReader interface {
	InsightsRange(ctx context.Context, from, to time.Time) (*internalmetrics.Insights, error)
}

type measureSpec struct {
	decl    gomeasures.MeasureDeclaration
	compute func(context.Context, RangeInsightsReader, gomeasures.Range, map[string]string) (gomeasures.MeasureResult, error)
}

func measureSpecs() []measureSpec {
	windowParam := gomeasures.Param{
		Name:    "window",
		Type:    gomeasures.ParamTypeTimeWindow,
		Default: string(gomeasures.TokenThisWeek),
	}
	return []measureSpec{
		{
			decl: measureDecl(
				MeasureFederatedLatency,
				"Federated query latency percentiles for search-hub in a time window.",
				[]string{
					"what is search hub federated query p95 latency this week",
					"show federated search p50 and p95 latency in the last 30 days",
					"how slow are search-hub federated queries this month",
				},
				"p95_ms", "milliseconds", "{p95_ms}ms p95 federated latency ({window})",
				map[string]gomeasures.Param{"window": windowParam},
			),
			compute: computeFederatedLatency,
		},
		{
			decl: measureDecl(
				MeasureDegradedQueryRate,
				"Fraction of federated search queries that degraded in a time window.",
				[]string{
					"what fraction of search hub queries degraded this week",
					"degraded federated query rate last month",
					"how often did search-hub federation degrade in the last 7 days",
				},
				"rate", "ratio", "{rate} degraded query rate ({window})",
				map[string]gomeasures.Param{"window": windowParam},
			),
			compute: computeDegradedQueryRate,
		},
		{
			decl: measureDecl(
				MeasureProviderDegradationRate,
				"Provider-leg degradation rate, optionally scoped to one provider id.",
				[]string{
					"how often does provider measures-health.measures degrade this week",
					"provider degradation rate for swarm-manager.records in the last 30 days",
					"which provider degradation rate is search hub seeing this month",
				},
				"rate", "ratio", "{rate} provider degradation rate ({window})",
				map[string]gomeasures.Param{
					"window":      windowParam,
					"provider_id": {Name: "provider_id", Type: "string"},
				},
			),
			compute: computeProviderDegradationRate,
		},
	}
}

func measureDecl(name, intent string, questions []string, valueField, unit, summary string, params map[string]gomeasures.Param) gomeasures.MeasureDeclaration {
	return gomeasures.MeasureDeclaration{
		Name:        name,
		Scenario:    "search-hub",
		Domain:      "query_telemetry",
		Intent:      intent,
		Questions:   questions,
		Params:      params,
		Result:      gomeasures.Result{Kind: gomeasures.ResultScalar, ValueField: valueField, Unit: unit, SummaryTemplate: summary},
		Effect:      gomeasures.EffectRead,
		RunEligible: true,
		Service:     "MetricsService",
		Method:      "Insights",
	}
}

// MeasuresHandler returns the packages/measures-go serve registry mounted by
// the metrics module at /measures.
func MeasuresHandler(reader RangeInsightsReader, now func() time.Time) (http.Handler, error) {
	reg, err := NewMeasureRegistry(reader, now)
	if err != nil {
		return nil, err
	}
	return reg.Handler(), nil
}

func NewMeasureRegistry(reader RangeInsightsReader, now func() time.Time) (*gomeasures.Registry, error) {
	if now == nil {
		now = time.Now
	}
	reg := gomeasures.NewRegistry(gomeasures.WithClock(now))
	for _, spec := range measureSpecs() {
		spec := spec
		if err := reg.Register(spec.decl, func(ctx context.Context, req gomeasures.MeasureRequest) (gomeasures.MeasureResult, error) {
			rng, err := resolveMeasureWindow(req.Params["window"], now())
			if err != nil {
				return gomeasures.MeasureResult{}, err
			}
			return spec.compute(ctx, reader, rng, req.Params)
		}); err != nil {
			return nil, fmt.Errorf("metrics measures: register %s: %w", spec.decl.Name, err)
		}
	}
	return reg, nil
}

func resolveMeasureWindow(token string, now time.Time) (gomeasures.Range, error) {
	t := gomeasures.TimeWindowToken(token)
	if t == "" {
		t = gomeasures.TokenThisWeek
	}
	return gomeasures.ResolveToken(t, now, time.UTC)
}

func resolveProtoWindow(window *measuresv1.TimeWindow, now time.Time) (gomeasures.Range, error) {
	if window == nil || window.GetWindow() == nil {
		return gomeasures.ResolveToken(gomeasures.TokenThisWeek, now, time.UTC)
	}
	return gomeasures.ResolveTimeWindow(window, now, time.UTC)
}

func computeFederatedLatency(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	insights, err := reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return gomeasures.MeasureResult{
		Value: strconv.FormatInt(insights.LatencyP95Ms, 10),
		Fields: []map[string]string{{
			"p50_ms": strconv.FormatInt(insights.LatencyP50Ms, 10),
			"p95_ms": strconv.FormatInt(insights.LatencyP95Ms, 10),
		}},
		Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery("query_telemetry latency percentiles", rng)},
	}, nil
}

// MeasuresConnectHandler implements search-hub's typed MeasuresService. It is
// backed by the same Store.InsightsRange compute path as the measures-go JSON
// registry so typed RPCs and /measures/execute cannot disagree.
type MeasuresConnectHandler struct {
	reader RangeInsightsReader
	now    func() time.Time
}

func NewMeasuresConnectHandler(reader RangeInsightsReader, now func() time.Time) *MeasuresConnectHandler {
	if now == nil {
		now = time.Now
	}
	return &MeasuresConnectHandler{reader: reader, now: now}
}

func (h *MeasuresConnectHandler) FederatedLatency(ctx context.Context, req *connect.Request[shmeasuresv1.FederatedLatencyRequest]) (*connect.Response[shmeasuresv1.FederatedLatencyResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	insights, err := h.reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&shmeasuresv1.FederatedLatencyResponse{
		P95Ms: insights.LatencyP95Ms,
		P50Ms: insights.LatencyP50Ms,
	}), nil
}

func (h *MeasuresConnectHandler) DegradedQueryRate(ctx context.Context, req *connect.Request[shmeasuresv1.DegradedQueryRateRequest]) (*connect.Response[shmeasuresv1.DegradedQueryRateResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	insights, err := h.reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&shmeasuresv1.DegradedQueryRateResponse{
		Rate:            rate(insights.DegradedQueries, insights.TotalQueries),
		DegradedQueries: insights.DegradedQueries,
		TotalQueries:    insights.TotalQueries,
	}), nil
}

func (h *MeasuresConnectHandler) ProviderDegradationRate(ctx context.Context, req *connect.Request[shmeasuresv1.ProviderDegradationRateRequest]) (*connect.Response[shmeasuresv1.ProviderDegradationRateResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	insights, err := h.reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var degraded, routed int64
	for _, provider := range insights.ProviderUsage {
		if req.Msg.GetProviderId() != "" && provider.ProviderID != req.Msg.GetProviderId() {
			continue
		}
		degraded += provider.DegradedCount
		routed += provider.TimesRouted
	}
	return connect.NewResponse(&shmeasuresv1.ProviderDegradationRateResponse{
		Rate:          rate(degraded, routed),
		DegradedCount: degraded,
		TimesRouted:   routed,
	}), nil
}

func computeDegradedQueryRate(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	insights, err := reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return gomeasures.MeasureResult{
		Value:      formatRate(rate(insights.DegradedQueries, insights.TotalQueries)),
		Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery("SUM(degraded) / COUNT(*) FROM query_telemetry", rng)},
	}, nil
}

func computeProviderDegradationRate(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, params map[string]string) (gomeasures.MeasureResult, error) {
	insights, err := reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	providerID := params["provider_id"]
	var degraded, routed int64
	for _, provider := range insights.ProviderUsage {
		if providerID != "" && provider.ProviderID != providerID {
			continue
		}
		degraded += provider.DegradedCount
		routed += provider.TimesRouted
	}
	query := "SUM(provider.degraded) / COUNT(provider rows) FROM query_telemetry_provider"
	if providerID != "" {
		query += " WHERE provider_id=" + strconv.Quote(providerID)
	}
	return gomeasures.MeasureResult{
		Value:      formatRate(rate(degraded, routed)),
		Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery(query, rng)},
	}, nil
}

func formatRate(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}

func rangeQuery(base string, rng gomeasures.Range) string {
	return fmt.Sprintf("%s WHERE created_at >= %q AND created_at < %q",
		base, rng.From.UTC().Format(time.RFC3339Nano), rng.To.UTC().Format(time.RFC3339Nano))
}
