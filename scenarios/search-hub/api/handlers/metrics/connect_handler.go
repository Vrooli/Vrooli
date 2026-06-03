package metrics

import (
	"context"
	"errors"
	"log"
	"sort"

	"connectrpc.com/connect"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	internalmetrics "search-hub/internal/metrics"
	internalregistry "search-hub/internal/registry"
)

// InsightsReader is the telemetry-aggregation seam the handler depends on.
// Production wires the SQLite metrics store; tests wire a fake.
type InsightsReader interface {
	Insights(ctx context.Context, windowDays int) (*internalmetrics.Insights, error)
}

// ProviderLister is the registry read seam used to reconcile telemetry against
// the registered ACTIVE leaves so under-utilized (registered-but-never-routed)
// providers surface even though they have no telemetry rows. Declared at the
// consumer (seam-discovery); the registry Store satisfies it.
type ProviderLister interface {
	List(ctx context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error)
}

// Deps wires the seams the metrics Connect handler needs.
type Deps struct {
	Insights InsightsReader
	Lister   ProviderLister
	Logger   *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the MetricsService handler. Logger defaults to
// log.Default() when nil.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Compile-time guarantee the handler satisfies the generated interface. A new
// RPC added to metrics.proto that the handler hasn't implemented fails here.
var _ = func() any {
	type metricsServiceHandler interface {
		Insights(context.Context, *connect.Request[metricsv1.InsightsRequest]) (*connect.Response[metricsv1.InsightsResponse], error)
	}
	var _ metricsServiceHandler = (*connectHandler)(nil)
	return nil
}()

// Insights aggregates per-query telemetry into federation-health signals and
// reconciles per-provider utilization against the registered ACTIVE leaves so
// never-routed providers are reported as under-utilized.
func (h *connectHandler) Insights(ctx context.Context, req *connect.Request[metricsv1.InsightsRequest]) (*connect.Response[metricsv1.InsightsResponse], error) {
	window := int(req.Msg.GetWindowDays())

	agg, err := h.deps.Insights.Insights(ctx, window)
	if err != nil {
		h.deps.Logger.Printf("metrics.Insights(window=%d): %v", window, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}

	active, err := h.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		h.deps.Logger.Printf("metrics.Insights: list providers: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}

	resp := &metricsv1.InsightsResponse{
		TotalQueries:      agg.TotalQueries,
		ZeroResultQueries: agg.ZeroResultQueries,
		ZeroResultRate:    rate(agg.ZeroResultQueries, agg.TotalQueries),
		DegradedQueries:   agg.DegradedQueries,
		RerankedQueries:   agg.RerankedQueries,
		LatencyP50Ms:      agg.LatencyP50Ms,
		LatencyP95Ms:      agg.LatencyP95Ms,
		Providers:         reconcileUtilization(active, agg.ProviderUsage),
	}
	return connect.NewResponse(resp), nil
}

// reconcileUtilization joins the registry's ACTIVE leaves with telemetry usage:
// every active leaf appears (under_utilized when it was never routed-to), and
// the group/type come from the descriptor. Ordered by provider_id.
func reconcileUtilization(active []*registryv1.ProviderDescriptor, usage []internalmetrics.ProviderUsage) []*metricsv1.ProviderUtilization {
	byID := make(map[string]internalmetrics.ProviderUsage, len(usage))
	for _, u := range usage {
		byID[u.ProviderID] = u
	}

	out := make([]*metricsv1.ProviderUtilization, 0, len(active))
	for _, p := range active {
		u := byID[p.GetProviderId()]
		out = append(out, &metricsv1.ProviderUtilization{
			ProviderId:    p.GetProviderId(),
			ProviderGroup: p.GetProviderGroup(),
			Type:          p.GetType(),
			TimesRouted:   u.TimesRouted,
			TotalHits:     u.TotalHits,
			UnderUtilized: u.TimesRouted == 0,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetProviderId() < out[j].GetProviderId() })
	return out
}

// rate returns numerator/denominator as a float, or 0 when there is no data.
func rate(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
