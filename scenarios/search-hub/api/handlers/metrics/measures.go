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
	MeasureProviderRerankerLeg     = "query_telemetry.provider-reranker-leg"
	MeasureStuckProviderCount      = "federation.stuck-provider-count"
	MeasureIncubatingProviderCount = "federation.incubating-provider-count"
	MeasureOldestIncubatingAge     = "federation.oldest-incubating-age"
	MeasureCorpusValidationLive    = "federation.corpus-validation-live-count"
	MeasureCorpusValidationHard    = "federation.corpus-validation-hard-count"
	MeasureCorpusValidationStale   = "federation.corpus-validation-stale-count"
	MeasureOldestStaleCorpusAge    = "federation.oldest-stale-corpus-age"
	MeasureZeroYieldRoutableIDs    = "federation.zero-yield-routable-id-count"
)

// RangeInsightsReader is the exact-window telemetry seam used by declared
// measures after resolving the canonical time_window param.
type RangeInsightsReader interface {
	InsightsRange(ctx context.Context, from, to time.Time) (*internalmetrics.Insights, error)
}

type stuckProviderReader interface {
	StuckProviderCount(context.Context, time.Time) (int64, error)
}

type incubatingProviderReader interface {
	IncubatingProviderStats(context.Context, time.Time) (count int64, oldestAgeSeconds int64, err error)
}

type corpusValidationReader interface {
	CorpusValidationStats(context.Context, time.Time) (internalmetrics.CorpusValidationStats, error)
}

type zeroYieldReader interface {
	ZeroYieldRoutableIDCount(context.Context, int64) (int64, error)
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
		{
			decl: measureDecl(
				MeasureProviderRerankerLeg,
				"The active reranker leg most recently observed while serving a provider in a time window.",
				[]string{
					"which reranker leg served provider measures-health.measures this week",
					"is search hub using the cross encoder for provider swarm-manager.records",
					"show the active reranker leg for a search hub provider",
				},
				"active_reranker_leg", "leg", "{active_reranker_leg} active reranker leg for {provider_id} ({window})",
				map[string]gomeasures.Param{
					"window":      windowParam,
					"provider_id": {Name: "provider_id", Type: "string", Required: true},
				},
			),
			compute: computeProviderRerankerLeg,
		},
		{
			decl: measureDecl(
				MeasureStuckProviderCount,
				"Number of providers stranded in an elapsed recovery probation state.",
				[]string{
					"how many search hub providers are stuck in recovery",
					"show the search hub stuck provider count",
					"are any search hub providers stuck",
				},
				"count", "providers", "{count} stuck provider(s)",
				map[string]gomeasures.Param{"window": windowParam},
			),
			compute: computeStuckProviderCount,
		},
		{
			decl: measureDecl(MeasureIncubatingProviderCount, "Number of registered providers in the experimental adoption lane.", []string{
				"how many search hub providers are incubating", "show the experimental provider count", "how many providers still need adoption evidence",
			}, "count", "providers", "{count} incubating provider(s)", map[string]gomeasures.Param{"window": windowParam}),
			compute: computeIncubatingProviderCount,
		},
		{
			decl: measureDecl(MeasureOldestIncubatingAge, "Age of the oldest registered experimental provider.", []string{
				"how old is the oldest incubating search provider", "show the oldest experimental provider age", "how long has the oldest provider been incubating",
			}, "age_seconds", "seconds", "oldest incubating provider age is {age_seconds}s", map[string]gomeasures.Param{"window": windowParam}),
			compute: computeOldestIncubatingAge,
		},
		corpusMeasureSpec(MeasureCorpusValidationLive, "Number of reviewed positives currently live in the newest validation per suite.", "live", "count", computeCorpusLive),
		corpusMeasureSpec(MeasureCorpusValidationHard, "Number of reviewed positives currently hard in the newest validation per suite.", "hard", "count", computeCorpusHard),
		corpusMeasureSpec(MeasureCorpusValidationStale, "Number of reviewed positives currently stale in the newest validation per suite.", "stale", "count", computeCorpusStale),
		corpusMeasureSpec(MeasureOldestStaleCorpusAge, "Age in seconds of the oldest suite with stale reviewed positives.", "age_seconds", "seconds", computeOldestStaleCorpusAge),
		{
			decl: measureDecl(MeasureZeroYieldRoutableIDs, "Number of routable accounting ids that have routed repeatedly without a hit.", []string{
				"how many search hub providers have zero yield", "show routable providers with no hits", "which search hub ids route but return nothing",
			}, "count", "ids", "{count} zero-yield routable id(s)", map[string]gomeasures.Param{"window": windowParam}),
			compute: computeZeroYieldRoutableIDs,
		},
	}
}

func corpusMeasureSpec(name, intent, field, unit string, compute func(context.Context, RangeInsightsReader, gomeasures.Range, map[string]string) (gomeasures.MeasureResult, error)) measureSpec {
	return measureSpec{decl: measureDecl(name, intent, []string{intent}, field, unit, "{"+field+"} corpus validation", map[string]gomeasures.Param{"window": {Name: "window", Type: gomeasures.ParamTypeTimeWindow, Default: string(gomeasures.TokenThisWeek)}}), compute: compute}
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

func computeProviderRerankerLeg(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, params map[string]string) (gomeasures.MeasureResult, error) {
	providerID := params["provider_id"]
	insights, err := reader.InsightsRange(ctx, rng.From, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	leg := "none"
	for _, provider := range insights.ProviderUsage {
		if provider.ProviderID == providerID {
			if provider.ActiveRerankerLeg != "" {
				leg = provider.ActiveRerankerLeg
			}
			break
		}
	}
	return gomeasures.MeasureResult{
		Value: leg,
		Fields: []map[string]string{{
			"provider_id":         providerID,
			"active_reranker_leg": leg,
		}},
		Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery("latest query_telemetry_provider.reranker_leg by provider_id", rng)},
	}, nil
}

func computeStuckProviderCount(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	stuckReader, ok := reader.(stuckProviderReader)
	if !ok {
		return gomeasures.MeasureResult{}, fmt.Errorf("stuck provider count is unavailable")
	}
	count, err := stuckReader.StuckProviderCount(ctx, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return gomeasures.MeasureResult{
		Value:      strconv.FormatInt(count, 10),
		Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery("COUNT(*) FROM provider_demotion_state WHERE demoted=1 AND probation=1 AND decay_deadline<=", rng)},
	}, nil
}

func incubatingStats(ctx context.Context, reader RangeInsightsReader, at time.Time) (int64, int64, error) {
	provider, ok := reader.(incubatingProviderReader)
	if !ok {
		return 0, 0, fmt.Errorf("incubating provider stats are unavailable")
	}
	return provider.IncubatingProviderStats(ctx, at)
}

func computeIncubatingProviderCount(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	count, _, err := incubatingStats(ctx, reader, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return gomeasures.MeasureResult{Value: strconv.FormatInt(count, 10), Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery("COUNT(*) FROM providers WHERE lifecycle=experimental", rng)}}, nil
}

func computeOldestIncubatingAge(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	_, age, err := incubatingStats(ctx, reader, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return gomeasures.MeasureResult{Value: strconv.FormatInt(age, 10), Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery("MIN(declared_at) FROM providers WHERE lifecycle=experimental", rng)}}, nil
}

func corpusStats(ctx context.Context, reader RangeInsightsReader, at time.Time) (internalmetrics.CorpusValidationStats, error) {
	validation, ok := reader.(corpusValidationReader)
	if !ok {
		return internalmetrics.CorpusValidationStats{}, fmt.Errorf("corpus validation stats are unavailable")
	}
	return validation.CorpusValidationStats(ctx, at)
}

func corpusMeasureResult(value int64, name string, rng gomeasures.Range) gomeasures.MeasureResult {
	return gomeasures.MeasureResult{Value: strconv.FormatInt(value, 10), Provenance: gomeasures.Provenance{ExecutedQuery: rangeQuery(name, rng)}}
}

func computeCorpusLive(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	stats, err := corpusStats(ctx, reader, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return corpusMeasureResult(stats.Live, "SUM(latest_validation.rollup.live)", rng), nil
}

func computeCorpusHard(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	stats, err := corpusStats(ctx, reader, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return corpusMeasureResult(stats.Hard, "SUM(latest_validation.rollup.hard)", rng), nil
}

func computeCorpusStale(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	stats, err := corpusStats(ctx, reader, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return corpusMeasureResult(stats.Stale, "SUM(latest_validation.rollup.stale)", rng), nil
}

func computeOldestStaleCorpusAge(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	stats, err := corpusStats(ctx, reader, rng.To)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return corpusMeasureResult(stats.OldestStaleAgeSec, "MAX(age(latest_validation) WHERE stale>0)", rng), nil
}

func computeZeroYieldRoutableIDs(ctx context.Context, reader RangeInsightsReader, rng gomeasures.Range, _ map[string]string) (gomeasures.MeasureResult, error) {
	zeroYield, ok := reader.(zeroYieldReader)
	if !ok {
		return gomeasures.MeasureResult{}, fmt.Errorf("zero-yield routable id stats are unavailable")
	}
	count, err := zeroYield.ZeroYieldRoutableIDCount(ctx, 5)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return corpusMeasureResult(count, "COUNT(provider_demotion_state WHERE routed>=5 AND hits=0)", rng), nil
}

func formatRate(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}

func rangeQuery(base string, rng gomeasures.Range) string {
	return fmt.Sprintf("%s WHERE created_at >= %q AND created_at < %q",
		base, rng.From.UTC().Format(time.RFC3339Nano), rng.To.UTC().Format(time.RFC3339Nano))
}
