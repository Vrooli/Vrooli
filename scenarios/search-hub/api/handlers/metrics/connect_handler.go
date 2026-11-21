package metrics

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	internalmetrics "search-hub/internal/metrics"
	internalregistry "search-hub/internal/registry"
)

// retirementRouteThreshold is deliberately high enough to distinguish a
// provider that has never been exercised from one that has repeatedly served
// no hits. The report is advisory; owners decide whether to retire it.
const retirementRouteThreshold int64 = 100

const concentratedGroupShare = 0.25

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
	Insights      InsightsReader
	RangeInsights RangeInsightsReader
	Lister        ProviderLister
	Logger        *log.Logger
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

	var agg *internalmetrics.Insights
	var err error
	if raw := strings.TrimSpace(req.Msg.GetWindow()); raw != "" && !isBareDays(raw) {
		if h.deps.RangeInsights == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("duration windows are unavailable"))
		}
		from, to, parseErr := resolveDurationWindow(raw, time.Now().UTC())
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}
		agg, err = h.deps.RangeInsights.InsightsRange(ctx, from, to)
	} else {
		if raw := strings.TrimSpace(req.Msg.GetWindow()); raw != "" {
			parsed, parseErr := strconv.Atoi(raw)
			if parseErr != nil || parsed < 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("window must be a non-negative day count or a duration such as 15m or 2h"))
			}
			window = parsed
		}
		agg, err = h.deps.Insights.Insights(ctx, window)
	}
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
		TotalQueries:                agg.TotalQueries,
		ZeroResultQueries:           agg.ZeroResultQueries,
		ZeroResultRate:              rate(agg.ZeroResultQueries, agg.TotalQueries),
		DegradedQueries:             agg.DegradedQueries,
		RerankedQueries:             agg.RerankedQueries,
		LatencyP50Ms:                agg.LatencyP50Ms,
		LatencyP95Ms:                agg.LatencyP95Ms,
		ResolverCacheHits:           agg.ResolverCacheHits,
		ResolverCacheMisses:         agg.ResolverCacheMisses,
		ResolverCacheHitRate:        agg.ResolverCacheHitRate,
		WindowFrom:                  formatWindowTime(agg.WindowFrom),
		WindowTo:                    formatWindowTime(agg.WindowTo),
		SampleCount:                 agg.SampleCount,
		MinimumSampleCount:          agg.MinimumSampleCount,
		SampleSufficient:            agg.SampleSufficient,
		RecentSampleCount:           agg.RecentSampleCount,
		RecentLatencyP50Ms:          agg.RecentLatencyP50Ms,
		RecentLatencyP95Ms:          agg.RecentLatencyP95Ms,
		SubstrateDegradationReasons: convertReasons(agg.SubstrateDegradationReasons),
		SubstrateDegradedLegs:       agg.SubstrateDegradedLegs,
		Providers:                   reconcileUtilization(active, agg.ProviderUsage),
	}
	resp.RetirementCandidates, resp.GroupAdvisories = HygieneReports(resp.Providers)
	return connect.NewResponse(resp), nil
}

func HygieneReports(providers []*metricsv1.ProviderUtilization) ([]*metricsv1.ProviderRetirementCandidate, []*metricsv1.ProviderGroupAdvisory) {
	retire := make([]*metricsv1.ProviderRetirementCandidate, 0)
	groups := make(map[string]int)
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		groups[provider.GetProviderGroup()]++
		if provider.GetTimesRouted() >= retirementRouteThreshold && provider.GetTotalHits() == 0 {
			retire = append(retire, &metricsv1.ProviderRetirementCandidate{
				ProviderId: provider.GetProviderId(), TimesRouted: provider.GetTimesRouted(), TotalHits: provider.GetTotalHits(),
				Reason: fmt.Sprintf("zero hits across %d window-routed calls", provider.GetTimesRouted()),
			})
		}
	}
	sort.Slice(retire, func(i, j int) bool { return retire[i].GetProviderId() < retire[j].GetProviderId() })
	advisories := make([]*metricsv1.ProviderGroupAdvisory, 0)
	for group, count := range groups {
		share := float64(count) / float64(len(providers))
		if share <= concentratedGroupShare {
			continue
		}
		advisories = append(advisories, &metricsv1.ProviderGroupAdvisory{
			ProviderGroup: group, ActiveLeaves: int32(count), Share: share,
			Reason: fmt.Sprintf("provider group holds %.1f%% of active leaves", share*100),
		})
	}
	sort.Slice(advisories, func(i, j int) bool { return advisories[i].GetProviderGroup() < advisories[j].GetProviderGroup() })
	return retire, advisories
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
			ProviderId:         p.GetProviderId(),
			ProviderGroup:      p.GetProviderGroup(),
			Type:               p.GetType(),
			TimesRouted:        u.TimesRouted,
			TotalHits:          u.TotalHits,
			UnderUtilized:      u.TimesRouted == 0,
			LatencyP50Ms:       u.LatencyP50Ms,
			LatencyP95Ms:       u.LatencyP95Ms,
			DegradedCount:      u.DegradedCount,
			DegradationRate:    u.DegradationRate,
			DegradationReasons: convertReasons(u.DegradationReasons),
			ActiveRerankerLeg:  u.ActiveRerankerLeg,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetProviderId() < out[j].GetProviderId() })
	return out
}

func isBareDays(raw string) bool {
	_, err := strconv.Atoi(raw)
	return err == nil
}

func resolveDurationWindow(raw string, now time.Time) (time.Time, time.Time, error) {
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 || d < time.Minute {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid window %q: use a positive duration such as 15m or 2h", raw)
	}
	return now.Add(-d), now, nil
}

func formatWindowTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func convertReasons(in []internalmetrics.ProviderDegradationReason) []*metricsv1.ProviderDegradationReason {
	out := make([]*metricsv1.ProviderDegradationReason, 0, len(in))
	for _, r := range in {
		out = append(out, &metricsv1.ProviderDegradationReason{
			Reason: r.Reason,
			Count:  r.Count,
		})
	}
	return out
}

// rate returns numerator/denominator as a float, or 0 when there is no data.
func rate(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
