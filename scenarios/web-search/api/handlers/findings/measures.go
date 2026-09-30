package findings

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"web-search/internal/capture"
	internalfindings "web-search/internal/findings"
	internallivesearch "web-search/internal/livesearch"
	internalresearch "web-search/internal/research"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/database"
	measures "github.com/vrooli/measures-go"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// MeasureName is the canonical "<domain>.<command>" id of the findings measure.
// It must match the manifest group + command ("findings" + "count") so the
// measures-health behavioral probe finds the registered compute func.
const (
	MeasureName               = "findings.count"
	MeasureUsageRate          = "findings.used-rate"
	MeasureNeverSurfaced      = "findings.never-surfaced"
	MeasureCaptureCoverage    = "research.capture-coverage"
	MeasureCacheHitRate       = "research.cache-hit-rate"
	MeasureGovernorRefusals   = "research.governor-refusals"
	MeasureSupportedClaimRate = "research.supported-claim-rate"
	MeasureQuestionCoverage   = "research.question-coverage"
	MeasureFetchSuccessRate   = "research.fetch-success-rate"
	MeasureAttemptLatency     = "research.attempt-latency-ms"
	MeasureCallsPerAttempt    = "research.calls-per-attempt"
)

func findingsCountDeclaration() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:   MeasureName,
		Domain: "findings",
		Intent: "How many findings were captured in a given time window.",
		Questions: []string{
			"how many findings were captured this week",
			"findings added last month",
			"how many findings did we record in the last 7 days",
		},
		Params: map[string]measures.Param{
			"window": {
				Name:    "window",
				Type:    measures.ParamTypeTimeWindow,
				Default: string(measures.TokenThisWeek),
			},
		},
		Result: measures.Result{
			Kind:            measures.ResultScalar,
			ValueField:      "count",
			Unit:            "findings",
			SummaryTemplate: "{count} findings captured ({window})",
		},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "FindingsService",
		Method:      "CountFindings",
	}
}

func usageDeclaration(name, intent, unit, field, summary string, questions []string) measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name: name, Domain: "findings", Intent: intent, Questions: questions,
		Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}},
		Result: measures.Result{Kind: measures.ResultScalar, ValueField: field, Unit: unit, SummaryTemplate: summary}, Effect: measures.EffectRead, RunEligible: true, Service: "FindingsService", Method: "ListEffectiveness",
	}
}

// MeasuresHandler builds the measures-go serve registry for the findings domain
// and returns it as an http.Handler to mount at /measures (see api/main.go).
func MeasuresHandler(db *database.RoutedDB, clk schedule.Clock, liveMetrics ...*internallivesearch.Metrics) (http.Handler, error) {
	var live *internallivesearch.Metrics
	if len(liveMetrics) > 0 {
		live = liveMetrics[0]
	}
	return measuresHandler(db, clk, live, nil)
}

// MeasuresHandlerWithMetrics wires both live-search and research outcome
// readings while retaining the original two-argument constructor for callers
// that only need findings measures.
func MeasuresHandlerWithMetrics(db *database.RoutedDB, clk schedule.Clock, live *internallivesearch.Metrics, outcome *internalresearch.OutcomeMetrics) (http.Handler, error) {
	return measuresHandler(db, clk, live, outcome)
}

func measuresHandler(db *database.RoutedDB, clk schedule.Clock, liveMetrics *internallivesearch.Metrics, outcomeMetrics *internalresearch.OutcomeMetrics) (http.Handler, error) {
	svc := internalfindings.NewService(internalfindings.NewSQLiteRepository(db, clk))
	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	if err := registerFindingsCount(reg, svc, clk); err != nil {
		return nil, err
	}
	if err := registerUsageMeasures(reg, svc, clk); err != nil {
		return nil, err
	}
	if err := registerCaptureMeasure(reg, capture.NewRepository(db, clk.Now), clk); err != nil {
		return nil, err
	}
	if liveMetrics != nil {
		if err := registerLiveMeasures(reg, liveMetrics, clk); err != nil {
			return nil, err
		}
	}
	if outcomeMetrics != nil {
		if err := registerOutcomeMeasures(reg, outcomeMetrics, clk); err != nil {
			return nil, err
		}
	}
	return reg.Handler(), nil
}

func registerOutcomeMeasures(reg *measures.Registry, metrics *internalresearch.OutcomeMetrics, clk schedule.Clock) error {
	declarations := []measures.MeasureDeclaration{
		{Name: MeasureSupportedClaimRate, Domain: "research", Intent: "Supported assessed claims divided by all assessed claims in a fixed window.", Questions: []string{"what is the supported research claim rate"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "rate", Unit: "ratio", SummaryTemplate: "{rate} supported research claim rate ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "ResearchService", Method: "SupportedClaimRate"},
		{Name: MeasureQuestionCoverage, Domain: "research", Intent: "Supported required questions divided by original required questions in a fixed window.", Questions: []string{"what is research question coverage"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "coverage", Unit: "ratio", SummaryTemplate: "{coverage} required research question coverage ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "ResearchService", Method: "QuestionCoverage"},
		{Name: MeasureFetchSuccessRate, Domain: "research", Intent: "Usable fetched sources divided by attempted sources in a fixed window.", Questions: []string{"what is research source fetch success"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "rate", Unit: "ratio", SummaryTemplate: "{rate} research fetch success ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "ResearchService", Method: "FetchSuccessRate"},
		{Name: MeasureAttemptLatency, Domain: "research", Intent: "Mean observed research attempt duration in milliseconds for a fixed window.", Questions: []string{"how long does research take"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "milliseconds", Unit: "milliseconds", SummaryTemplate: "{milliseconds} mean research attempt milliseconds ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "ResearchService", Method: "AttemptLatency"},
		{Name: MeasureCallsPerAttempt, Domain: "research", Intent: "Observed owner calls divided by admitted research attempts in a fixed window.", Questions: []string{"how many calls does research use"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "calls", Unit: "calls", SummaryTemplate: "{calls} mean research calls per attempt ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "ResearchService", Method: "CallsPerAttempt"},
	}
	for _, decl := range declarations {
		decl := decl
		if err := reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
			_ = ctx
			rng, err := resolveMeasureRange(req.Params["window"], clk.Now())
			if err != nil {
				return measures.MeasureResult{}, err
			}
			totals, attempts, complete := metrics.Snapshot(rng.From, rng.To)
			if !complete || attempts == 0 {
				return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "research_metrics:unknown:empty_or_truncated_window"}}, nil
			}
			var value string
			switch decl.Name {
			case MeasureSupportedClaimRate:
				if totals.AssessedClaims == 0 {
					return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "research_metrics:unknown:no_assessed_claims"}}, nil
				}
				value = strconv.FormatFloat(float64(totals.SupportedClaims)/float64(totals.AssessedClaims), 'f', 4, 64)
			case MeasureQuestionCoverage:
				if totals.RequiredQuestions == 0 {
					return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "research_metrics:unknown:no_required_questions"}}, nil
				}
				value = strconv.FormatFloat(float64(totals.SupportedQuestions)/float64(totals.RequiredQuestions), 'f', 4, 64)
			case MeasureFetchSuccessRate:
				if totals.FetchAttempts == 0 {
					return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "research_metrics:unknown:no_fetch_attempts"}}, nil
				}
				value = strconv.FormatFloat(float64(totals.FetchSuccesses)/float64(totals.FetchAttempts), 'f', 4, 64)
			case MeasureAttemptLatency:
				value = strconv.FormatInt(totals.Duration.Milliseconds()/int64(attempts), 10)
			case MeasureCallsPerAttempt:
				value = strconv.FormatFloat(float64(totals.Calls)/float64(attempts), 'f', 4, 64)
			}
			return measures.MeasureResult{Value: value, Provenance: measures.Provenance{ExecutedQuery: "research_metrics:fixed_window_owner_events"}}, nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func registerLiveMeasures(reg *measures.Registry, metrics *internallivesearch.Metrics, clk schedule.Clock) error {
	declarations := []measures.MeasureDeclaration{
		{Name: MeasureCacheHitRate, Domain: "research", Intent: "Ratio of live-search cache hits to observed cache lookups in a fixed window.", Questions: []string{"what is the web search cache hit rate", "how often does web search use cache"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "rate", Unit: "ratio", SummaryTemplate: "{rate} live-search cache hit rate ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "LiveSearchService", Method: "CacheHitRate"},
		{Name: MeasureGovernorRefusals, Domain: "research", Intent: "Count of live-search requests refused by the owner budget governor in a fixed window.", Questions: []string{"how many web searches were rate limited", "live search governor refusals"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "requests", SummaryTemplate: "{count} live-search governor refusals ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "LiveSearchService", Method: "GovernorRefusals"},
	}
	for _, decl := range declarations {
		decl := decl
		if err := reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
			_ = ctx
			rng, err := resolveMeasureRange(req.Params["window"], clk.Now())
			if err != nil {
				return measures.MeasureResult{}, err
			}
			counts, complete := metrics.Snapshot(rng.From, rng.To)
			if !complete {
				return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "live_search_metrics:unknown:bounded_history_truncated"}}, nil
			}
			hits, misses := counts[internallivesearch.MetricCacheHit], counts[internallivesearch.MetricCacheMiss]
			if decl.Name == MeasureCacheHitRate {
				if hits+misses == 0 {
					return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "live_search_metrics:unknown:no_cache_observations"}}, nil
				}
				return measures.MeasureResult{Value: strconv.FormatFloat(float64(hits)/float64(hits+misses), 'f', 4, 64), Provenance: measures.Provenance{ExecutedQuery: "live_search_metrics:cache_hits/(cache_hits+cache_misses)"}}, nil
			}
			refusals := counts[internallivesearch.MetricGovernorRefusal]
			if hits+misses == 0 && refusals == 0 {
				return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: "live_search_metrics:unknown:no_live_search_observations"}}, nil
			}
			return measures.MeasureResult{Value: strconv.Itoa(refusals), Provenance: measures.Provenance{ExecutedQuery: "live_search_metrics:governor_refusals"}}, nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func registerCaptureMeasure(reg *measures.Registry, repo *capture.Repository, clk schedule.Clock) error {
	decl := measures.MeasureDeclaration{Name: MeasureCaptureCoverage, Domain: "research", Intent: "Fraction of admitted research attempts delivered to Memory in the observed outbox.", Questions: []string{"research capture delivery coverage", "how many research attempts reached memory"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "coverage", Unit: "ratio", SummaryTemplate: "{coverage} research capture coverage ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "ResearchService", Method: "CaptureCoverage"}
	return reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
		rng, err := resolveMeasureRange(req.Params["window"], clk.Now())
		if err != nil {
			return measures.MeasureResult{}, err
		}
		counts, err := repo.CountsInWindow(ctx, rng.From, rng.To)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		total := counts.Pending + counts.Delivered + counts.Failed
		if total == 0 {
			return measures.MeasureResult{Provenance: measures.Provenance{ExecutedQuery: fmt.Sprintf("research_attempt_outbox:unknown:no_admitted_attempts:%s..%s", rng.From.UTC().Format(time.RFC3339), rng.To.UTC().Format(time.RFC3339))}}, nil
		}
		return measures.MeasureResult{Value: strconv.FormatFloat(float64(counts.Delivered)/float64(total), 'f', 4, 64), Provenance: measures.Provenance{ExecutedQuery: fmt.Sprintf("research_attempt_outbox:delivered/(pending+delivered+failed):%s..%s", rng.From.UTC().Format(time.RFC3339), rng.To.UTC().Format(time.RFC3339))}}, nil
	})
}

func registerUsageMeasures(reg *measures.Registry, svc internalfindings.Service, clk schedule.Clock) error {
	decls := []measures.MeasureDeclaration{
		usageDeclaration(MeasureUsageRate, "Ratio of explicitly used findings to surfaced findings. The usage table has no independent creation timestamp, so the window is anchored to finding creation and the latest surfacing timestamp.", "ratio", "rate", "{rate} finding usage rate ({window})", []string{"what fraction of surfaced findings were used", "finding utilization this week", "web search finding used rate"}),
		usageDeclaration(MeasureNeverSurfaced, "Count of findings with no explicit use record in the finding-creation window; finding_usage has no independent creation timestamp.", "findings", "count", "{count} never-used findings ({window})", []string{"how many findings were never used", "never surfaced findings this week", "unused web findings"}),
	}
	for _, decl := range decls {
		decl := decl
		if err := reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
			rng, err := resolveMeasureRange(req.Params["window"], clk.Now())
			if err != nil {
				return measures.MeasureResult{}, err
			}
			agg, err := svc.UsageAggregate(ctx, rng.From, rng.To)
			if err != nil {
				return measures.MeasureResult{}, err
			}
			value := "0"
			if decl.Name == MeasureUsageRate && agg.Surfaced > 0 {
				value = strconv.FormatFloat(float64(agg.Used)/float64(agg.Surfaced), 'f', 4, 64)
			}
			if decl.Name == MeasureNeverSurfaced {
				value = strconv.FormatInt(agg.Never, 10)
			}
			return measures.MeasureResult{Value: value, Provenance: measures.Provenance{ExecutedQuery: internalfindings.UsageAggregateQuery}}, nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func resolveMeasureRange(token string, now time.Time) (measures.Range, error) {
	if token == "" {
		token = string(measures.TokenThisWeek)
	}
	return measures.ResolveToken(measures.TimeWindowToken(token), now, time.UTC)
}

func registerFindingsCount(reg *measures.Registry, svc internalfindings.Service, clk schedule.Clock) error {
	decl := findingsCountDeclaration()
	return reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
		token := measures.TimeWindowToken(req.Params["window"])
		if token == "" {
			token = measures.TokenThisWeek
		}
		rng, err := measures.ResolveToken(token, clk.Now(), time.UTC)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		n, err := svc.CountInWindow(ctx, rng.From, rng.To)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		return measures.MeasureResult{
			Value: strconv.Itoa(n),
			Provenance: measures.Provenance{
				ExecutedQuery: fmt.Sprintf(
					"findings.created_at_count:%s..%s",
					rng.From.UTC().Format(time.RFC3339), rng.To.UTC().Format(time.RFC3339),
				),
			},
		}, nil
	})
}

// resolveCountWindow maps a request's canonical TimeWindow to a concrete
// [from, to) range, defaulting to this_week when unset. Shared by the
// CountFindings RPC handler and the measure compute path so both resolve dates
// identically.
func resolveCountWindow(tw *measuresv1.TimeWindow, now time.Time) (measures.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measures.ResolveToken(measures.TokenThisWeek, now, time.UTC)
	}
	return measures.ResolveTimeWindow(tw, now, time.UTC)
}
