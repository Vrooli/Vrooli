// Package routing is the search-hub router core: it resolves which registered
// providers a query targets, fans out to each over HTTP+JSON with bounded
// concurrency and a per-provider timeout, maps every heterogeneous response
// through the generic providers.MapResults adapter, and groups the results by
// provider.
//
// It is the load-bearing realization of the thin-router invariant: this package
// holds NO corpus data and NO provider-specific code. A provider is reached
// entirely through its registered ProviderDescriptor (logical scenario_id +
// path + body_template + ResultMapping); the live base URL is resolved at
// call-time via the cross-scenario URLResolver seam (never client-computed).
//
// Routing is explicit (--type/--all/--group) or automatic: when no selector is
// given, the router uses the active provider-description strategy followed by
// the optional cross-encoder provider picker. Results are always grouped honestly
// by provider for provenance; when a Reranker is wired the fused
// shortlist is additionally reranked into one comparable cross-provider list,
// degrading back to grouping-only if the reranker is unavailable.
package routing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"search-hub/internal/httpc"
	"search-hub/internal/providers"
	internalregistry "search-hub/internal/registry"

	aisearch "github.com/vrooli/ai-go/search"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/shared"
	"google.golang.org/protobuf/proto"
)

const (
	// defaultPerProviderTimeout bounds a single provider's HTTP round-trip so a
	// slow or hung leaf never blocks the whole fan-out (Risk: latency/fan-out
	// cost — partial results return within the bound).
	defaultPerProviderTimeout = aisearch.DefaultRouterPerProviderTimeout
	// defaultConcurrency caps how many providers are queried at once.
	defaultConcurrency = aisearch.DefaultRouterConcurrency
	// defaultLimit is the per-provider result cap when the request omits one.
	defaultLimit = 10
	// defaultQueryTimeout keeps the server-side routing path comfortably below
	// the scenario CLI's default 30s HTTP timeout, so degraded responses can be
	// returned instead of surfacing as transport timeouts.
	defaultQueryTimeout = aisearch.DefaultRouterQueryBudget
	// defaultRerankTimeout bounds the reranker chain so a slow fallback model
	// degrades to honest grouping before the whole query times out. It remains
	// the compatibility fallback for injected rerankers; production uses the
	// active-leg bounds below.
	defaultRerankTimeout = 10 * time.Second
	// Cross-encoder requests are measured in tens of milliseconds on this host;
	// the bound leaves a small network margin without paying the 10s LLM tail.
	defaultCrossEncoderRerankTimeout = 500 * time.Millisecond
	// The LLM fallback is intentionally looser because its warm path is
	// model-dependent; cold residency is a host concern, not router code.
	defaultLLMRerankTimeout = 8 * time.Second
	// defaultResponseCushion reserves a small tail of the query budget for
	// response construction, telemetry stamping, and Connect header write-out.
	defaultResponseCushion = 500 * time.Millisecond
	// defaultCircuitOpenQuorum is the share of active leaves whose transport
	// circuit may be open before Search Hub itself reports federation degraded.
	// A majority threshold avoids declaring a fleet outage for one isolated leaf
	// while still making a substrate-wide failure visible to readiness consumers.
	defaultCircuitOpenQuorum          = 0.50
	defaultRerankBreakerFailures      = 3
	defaultRerankBreakerCooldown      = 60 * time.Second
	defaultProviderBreakerFailures    = 3
	defaultProviderBreakerCooldown    = 30 * time.Second
	defaultProviderBreakerMaxCooldown = 5 * time.Minute
	// maxResponseBytes caps how much of a provider response the router reads,
	// so a misbehaving leaf cannot exhaust memory.
	maxResponseBytes = 8 << 20 // 8 MiB
)

// QualityJunkLeakMargin prevents an exact floating-point tie between a
// strongest reviewed positive and a gibberish probe from withholding a
// provider. A real margin is required before quality evidence removes a
// production leaf from automatic routing.
const QualityJunkLeakMargin = 0.01

func QualityJunkLeak(maxGibberish, meanStrong float64) bool {
	return meanStrong > 0 && maxGibberish >= meanStrong+QualityJunkLeakMargin
}

// CircuitOpenQuorumThreshold exposes the generic federation-health policy to
// the health wiring without exposing provider-specific routing internals.
const CircuitOpenQuorumThreshold = defaultCircuitOpenQuorum

// ProviderLister is the registry read seam the router depends on. The
// SQLite-backed registry Store satisfies it in production; tests substitute a
// fake. Declared at the consumer (seam-discovery) so the router never imports
// a concrete store.
type ProviderLister interface {
	List(ctx context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error)
}

// fanoutWorstCase is the deterministic upper bound used by the budget test:
// every provider in each concurrency wave consumes its full per-provider
// timeout. Keeping this calculation beside the defaults makes an accidental
// timeout/concurrency change fail before it reaches production.
func fanoutWorstCase(activeProviders, concurrency int, perProviderTimeout time.Duration) time.Duration {
	if activeProviders <= 0 || concurrency <= 0 || perProviderTimeout <= 0 {
		return 0
	}
	waves := (activeProviders + concurrency - 1) / concurrency
	return time.Duration(waves) * perProviderTimeout
}

func nonNegativeDelta(after, before int64) int64 {
	if after < before {
		return 0
	}
	return after - before
}

// URLResolver turns a logical scenario_id into the scenario's live API base URL
// (scheme://host:port) at call-time. Production wraps api-core/discovery's
// Resolver (which shells out to `vrooli scenario port`); tests inject a static
// or recording resolver. This is the project rule "never compute proxy URLs
// client-side; resolve via the backend" made into a seam.
type URLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioID string) (baseURL string, err error)
}

// ResolutionCacheStats is an optional observability seam implemented by
// resolvers that cache scenario addresses. The router never requires caching
// for correctness; when available, it records before/after deltas in query
// telemetry so operators can verify that fan-out cost scales with scenarios.
type ResolutionCacheStats interface {
	CacheStats() (hits, misses int64)
}

func resolutionCacheStats(resolver URLResolver) (hits, misses int64) {
	if stats, ok := resolver.(ResolutionCacheStats); ok && stats != nil {
		return stats.CacheStats()
	}
	return 0, 0
}

// EvalQualityEvidence is the router's narrow read model for a provider's
// latest evaluation. The router consumes gate evidence; it does not own or
// grade the suite.
type EvalQualityEvidence struct {
	// EvidenceAvailable distinguishes an intentional empty evidence result from
	// a test double or an unavailable reader. Production wiring sets it true;
	// callers that cannot inspect eval state must not claim the gate passed.
	EvidenceAvailable    bool
	SuitePresent         bool
	LiveReviewedPositive bool
	RecentPassingRun     bool
	RunID                string
	Fresh                bool
	Degraded             bool
	MeanStrongTop1       float64
	MaxGibberishScore    float64
	GibberishLeak        bool
	CorpusAllStale       bool
}

func automaticEvidenceExclusion(evidence EvalQualityEvidence) string {
	if !evidence.SuitePresent {
		return "no suite"
	}
	if !evidence.LiveReviewedPositive {
		return "no live reviewed positive"
	}
	if evidence.CorpusAllStale {
		return "all reviewed positives are stale"
	}
	if !evidence.RecentPassingRun {
		return "no recent passing run"
	}
	return ""
}

type EvalQualityReader interface {
	LatestProviderEval(ctx context.Context, providerID string) (EvalQualityEvidence, error)
}

// ErrInvalidQuery is the typed sentinel for caller mistakes the router can
// detect before any fan-out (empty query text). The Connect handler translates it into
// InvalidArgument.
type ErrInvalidQuery struct{ Reason string }

func (e ErrInvalidQuery) Error() string { return e.Reason }

// Deps wires the router's seams. Lister/Resolver/Doer are required; the rest
// default in NewRouter. Reranker is optional: when nil, results
// stay grouped by provider; when set, the fused
// shortlist is reranked into one comparable list, degrading back to grouping if
// the reranker fails. Recorder is optional: when set, each completed
// query emits a TelemetrySample; when nil, no telemetry is
// recorded.
type Deps struct {
	Lister   ProviderLister
	Resolver URLResolver
	Doer     httpc.Doer
	// Classifier is a deprecated test-only seam. Production wiring leaves it
	// nil; automatic routing uses the strategy catalog below.
	Classifier                Classifier
	Reranker                  Reranker
	Recorder                  TelemetryRecorder
	EvalQuality               EvalQualityReader
	DescriptionIndex          ProviderDescriptionIndex
	Logger                    *log.Logger
	Concurrency               int
	PerProviderTimeout        time.Duration
	QueryTimeout              time.Duration
	RerankTimeout             time.Duration
	CrossEncoderRerankTimeout time.Duration
	LLMRerankTimeout          time.Duration
	RerankBreaker             RerankBreakerConfig
	// ProviderBreaker prevents repeated down providers from spending the query
	// deadline. It tracks availability only; no provider results are cached.
	ProviderBreaker RerankBreakerConfig
	DemotionStore   DemotionStore
	Now             func() time.Time
	// Strategy and RouterFactors are loaded from the scenario-owned strategy
	// record at the production wiring edge. Nil values retain the validated
	// current defaults for small unit-test fakes.
	Strategy        *RetrievalStrategy
	StrategyCatalog []RetrievalStrategy
	RouterFactors   *RouterFactorValues
	// AutoRouteExternal gates OT-P2-002: when true, the automatic (classifier)
	// path may fold SCOPE_EXTERNAL providers back into the fan-out — either
	// because the classifier judged the query web-shaped (above
	// autoExternalThreshold) or as a fallback escalation when the project corpus
	// returned no hits. DEFAULT FALSE: a plain federated query never auto-hits a
	// rate-limited/paid external corpus unless the operator opts in. Explicit
	// --all/--type always reach external providers regardless of this flag.
	AutoRouteExternal bool
}

// RerankBreakerConfig controls the generic circuit breaker guarding the
// reranker hot path.
type RerankBreakerConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
	// ZeroYieldMinimumRoutes controls how many successful empty automatic
	// responses constitute evidence of an unhelpful corpus. Transport failures
	// never use this counter.
	ZeroYieldMinimumRoutes int64
	// DemotionWindow is the maximum automatic-routing exclusion before a
	// probationary recovery probe is allowed.
	DemotionWindow time.Duration
}

// Router executes federated queries across registered providers.
type Router struct {
	deps             Deps
	strategy         RetrievalStrategy
	strategies       map[string]RetrievalStrategy
	factors          RouterFactorValues
	rerankBreaker    *rerankBreaker
	providerBreakers *providerBreakers
	statusMu         sync.Mutex
	statusCache      *routingv1.StatusResponse
	statusCacheAt    time.Time
	probeLatencies   map[string]time.Duration
	providerCallMu   sync.Mutex
	providerCalls    map[string]*providerCall
}

type providerCall struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	result  *routingv1.ProviderResultGroup
}

// NewRouter constructs a Router, applying defaults for the optional Deps
// fields. Logger defaults to log.Default(); concurrency and timeout fall back
// to the package defaults when non-positive.
func NewRouter(d Deps) *Router {
	strategy := RetrievalStrategy{
		Name:        "lexical-cross-encoder",
		Description: "built-in lexical fallback shortlist followed by cross-encoder provider selection",
		Stages: []RetrievalStage{
			{Kind: StageLexical, Params: map[string]interface{}{"shortlist_width": float64(6)}},
			{Kind: StageCrossEncoder, Params: map[string]interface{}{"selection": "provider_pick"}},
		},
	}
	factors := defaultRouterFactorValues()
	if d.Strategy != nil {
		strategy = *d.Strategy
	}
	if d.RouterFactors != nil {
		factors = *d.RouterFactors
	}
	strategies := make(map[string]RetrievalStrategy, len(d.StrategyCatalog)+1)
	for _, candidate := range d.StrategyCatalog {
		strategies[candidate.Name] = candidate
	}
	strategies[strategy.Name] = strategy
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Concurrency <= 0 {
		d.Concurrency = factors.Concurrency
	}
	if d.PerProviderTimeout <= 0 {
		d.PerProviderTimeout = factors.PerProviderTimeout
	}
	if d.QueryTimeout <= 0 {
		d.QueryTimeout = factors.QueryBudget
	}
	if d.RerankTimeout <= 0 {
		d.RerankTimeout = defaultRerankTimeout
	}
	if d.CrossEncoderRerankTimeout <= 0 {
		d.CrossEncoderRerankTimeout = defaultCrossEncoderRerankTimeout
	}
	if d.LLMRerankTimeout <= 0 {
		d.LLMRerankTimeout = defaultLLMRerankTimeout
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.RerankBreaker.FailureThreshold <= 0 {
		d.RerankBreaker.FailureThreshold = defaultRerankBreakerFailures
	}
	if d.RerankBreaker.Cooldown <= 0 {
		d.RerankBreaker.Cooldown = defaultRerankBreakerCooldown
	}
	if d.ProviderBreaker.FailureThreshold <= 0 {
		d.ProviderBreaker.FailureThreshold = defaultProviderBreakerFailures
	}
	if d.ProviderBreaker.Cooldown <= 0 {
		d.ProviderBreaker.Cooldown = defaultProviderBreakerCooldown
	}
	if d.ProviderBreaker.ZeroYieldMinimumRoutes <= 0 {
		d.ProviderBreaker.ZeroYieldMinimumRoutes = factors.ZeroYieldMinimumRoutes
	}
	if d.ProviderBreaker.DemotionWindow <= 0 {
		d.ProviderBreaker.DemotionWindow = factors.DemotionWindow
	}
	return &Router{
		deps:       d,
		strategy:   strategy,
		strategies: strategies,
		factors:    factors,
		rerankBreaker: newRerankBreaker(rerankBreakerConfig{
			FailureThreshold: d.RerankBreaker.FailureThreshold,
			Cooldown:         d.RerankBreaker.Cooldown,
		}),
		providerBreakers: newProviderBreakers(rerankBreakerConfig{
			FailureThreshold:       d.ProviderBreaker.FailureThreshold,
			Cooldown:               d.ProviderBreaker.Cooldown,
			MaxCooldown:            defaultProviderBreakerMaxCooldown,
			ZeroYieldMinimumRoutes: d.ProviderBreaker.ZeroYieldMinimumRoutes,
			DemotionWindow:         d.ProviderBreaker.DemotionWindow,
		}, d.DemotionStore),
		probeLatencies: make(map[string]time.Duration),
		providerCalls:  make(map[string]*providerCall),
	}
}

// RetrievalStrategy returns the immutable strategy record selected at router
// startup. Callers receive a copy so inspection cannot mutate hot-path state.
func (r *Router) RetrievalStrategy() RetrievalStrategy { return r.strategy }

// RouterFactors returns the immutable typed projection used by the router.
func (r *Router) RouterFactors() RouterFactorValues { return r.factors }

// Query fans out text to the providers selected by req (explicit types, --all,
// and/or --group), collects each provider's hits behind the generic adapter,
// and returns them grouped by provider. A provider that is unreachable, times
// out, returns a non-2xx, or yields an unmappable body is reported as a
// degraded group with a human note — it never fails the whole query (graceful
// degradation, partial results).
//
// It returns an error only for caller mistakes (ErrInvalidQuery) or a registry
// read failure — never for an individual provider's runtime failure.
func (r *Router) Query(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	r.providerBreakers.restore(ctx)
	start := r.deps.Now()
	cacheHitsBefore, cacheMissesBefore := resolutionCacheStats(r.deps.Resolver)
	qctx := ctx
	cancelQuery := func() {}
	if r.deps.QueryTimeout > 0 {
		qctx, cancelQuery = context.WithTimeout(ctx, r.deps.QueryTimeout)
	}
	defer cancelQuery()

	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, ErrInvalidQuery{Reason: "query text is required"}
	}
	backgroundEvaluation := isBackgroundEvaluation(ctx)
	routingEvaluation := isRoutingEvaluation(ctx)
	selectedStrategy := r.strategy
	if name := strings.TrimSpace(req.GetStrategyName()); name != "" {
		if !routingEvaluation {
			return nil, ErrInvalidQuery{Reason: "strategy_name is restricted to federated evaluation"}
		}
		candidate, ok := r.strategies[name]
		if !ok {
			return nil, ErrInvalidQuery{Reason: fmt.Sprintf("unknown retrieval strategy %q", name)}
		}
		selectedStrategy = candidate
		if strategyHasStage(selectedStrategy, StageLLM) {
			return nil, ErrInvalidQuery{Reason: fmt.Sprintf("retrieval strategy %q is retired and has no executable LLM classifier", name)}
		}
	}
	hasExplicit := backgroundEvaluation || req.GetAll() || len(nonEmpty(req.GetTypes())) > 0 || strings.TrimSpace(req.GetGroup()) != ""
	// Automatic routing has a model-free lexical fallback, so it remains
	// available even when both local inference resources are stopped.

	limit := req.GetLimit()
	if limit <= 0 {
		limit = defaultLimit
	}
	scope := strings.TrimSpace(req.GetScope())

	// Only ACTIVE leaves are callable; capability_gap stubs carry no endpoint
	// and are intentionally excluded from fan-out.
	active, err := r.deps.Lister.List(qctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	resolverLatency := r.deps.Now().Sub(start).Milliseconds()

	var (
		targets            []*registryv1.ProviderDescriptor
		autoExplain        []string
		selectionFallback  bool
		selectorLeg        = "none"
		selectionReason    string
		pendingExternal    []*registryv1.ProviderDescriptor
		autoRoutedExternal bool
		escalated          bool
		routingMode        string
		classifierLatency  int64
		selectedLeafCount  int
		widenedLeafCount   int
		fanoutBoundReached bool
		routingIndexReason string
		routingTrace       *sharedv1.RoutingTrace
		rerankLatency      int64
		partial            bool
		pendingProviders   int
	)
	if backgroundEvaluation {
		routingMode = "background_evaluation"
		providerID := backgroundEvaluationProvider(ctx)
		for _, provider := range active {
			if providerID == "" || provider.GetProviderId() == providerID {
				targets = append(targets, provider)
			}
		}
		selectedLeafCount = len(targets)
	} else if hasExplicit {
		if req.GetAll() {
			routingMode = "explicit_all"
		} else {
			routingMode = "explicit_scoped"
		}
		// Explicit selectors (--all / --type / --group) reach every active
		// provider, including SCOPE_EXTERNAL ones — the operator asked for them.
		targets = selectTargets(active, req)
		selectedLeafCount = len(targets)
	} else {
		routingMode = "automatic"
		// Automatic/classifier routing never auto-hits an external (e.g.
		// rate-limited / paid) corpus UNLESS the operator opted in
		// (AutoRouteExternal) and the classifier judged the query web-shaped.
		// External providers always stay reachable via the explicit path above.
		autoCandidates, withheldExternal := partitionByScope(active)
		var qualityExplain []string
		autoCandidates, qualityExplain = r.filterAutomatic(qctx, autoCandidates)
		autoCandidates = r.filterDemoted(autoCandidates)
		var webShaped bool
		targets, autoExplain, selectionFallback, selectorLeg, webShaped, selectedLeafCount, widenedLeafCount, fanoutBoundReached, routingIndexReason, selectionReason, routingTrace = r.autoSelect(qctx, autoCandidates, query, selectedStrategy)
		autoExplain = append(qualityExplain, autoExplain...)
		// The LLM classifier was removed from the interactive path. Preserve the
		// legacy telemetry field as an explicit zero rather than charging lexical
		// or cross-encoder selection to a retired model.
		classifierLatency = 0
		if selectionFallback {
			routingMode = "automatic_fallback"
		}
		pendingExternal = withheldExternal
		switch {
		case r.deps.AutoRouteExternal && webShaped && len(withheldExternal) > 0:
			// OT-P2-002: a confidently web-shaped query folds the withheld
			// external provider(s) back into the fan-out — driven purely off
			// descriptor scope + the generic web-shaped label (no per-provider
			// code). Rate-safety is the provider's own governor's job.
			targets = append(targets, withheldExternal...)
			autoRoutedExternal = true
			autoExplain = append(autoExplain, autoRoutedExternalLine(withheldExternal))
		case len(withheldExternal) > 0:
			autoExplain = append(autoExplain, withheldExternalLine(withheldExternal))
		}
	}

	fanoutStarted := r.deps.Now()
	groups := r.fanOut(qctx, targets, query, limit, hasExplicit, scope)
	groups = collapseDocumentHits(groups)
	if qctx.Err() != nil {
		usable := 0
		for _, group := range groups {
			if group.GetDegraded() {
				pendingProviders++
				continue
			}
			if len(group.GetHits()) > 0 {
				usable++
			}
		}
		partial = usable > 0 && pendingProviders > 0
	}

	// OT-P2-002 fallback escalation: if the project corpus returned nothing —
	// or only hits below the weakness threshold — and the operator opted in,
	// escalate to the withheld external provider(s) via the same
	// governor-protected path (only on the automatic path, and only when we
	// did not already auto-route external above).
	if !hasExplicit && r.deps.AutoRouteExternal && !autoRoutedExternal && len(pendingExternal) > 0 {
		if weakReason := resultsWeakness(groups, autoExternalThreshold()); weakReason != "" {
			escalationGroups := r.fanOut(qctx, pendingExternal, query, limit, false, scope)
			groups = append(groups, escalationGroups...)
			groups = collapseDocumentHits(groups)
			targets = append(targets, pendingExternal...)
			escalated = true
			autoExplain = append(autoExplain, escalationLine(pendingExternal, weakReason))
		}
	}
	fanoutLatency := r.deps.Now().Sub(fanoutStarted).Milliseconds()

	var ranked []*routingv1.SearchHit
	var reranked, rerankDegraded bool
	var rerankReason, rerankLeg string
	var rerankExplain []string
	if backgroundEvaluation || routingEvaluation {
		rerankExplain = []string{"background evaluation: classifier and reranker bypassed to protect interactive latency"}
		if routingEvaluation {
			rerankExplain = []string{"routing evaluation: reranker bypassed; provider selection remains automatic"}
		}
	} else {
		rerankStarted := r.deps.Now()
		ranked, reranked, rerankDegraded, rerankReason, rerankLeg, rerankExplain = r.maybeRerank(qctx, query, groups, targets)
		rerankLatency = r.deps.Now().Sub(rerankStarted).Milliseconds()
	}

	resp := &routingv1.QueryResponse{
		Ranked:           ranked,
		Groups:           groups,
		Reranked:         reranked,
		Degraded:         selectionFallback || rerankDegraded || routingIndexReason != "",
		Partial:          partial,
		PendingProviders: int32(pendingProviders),
		RerankerLeg:      rerankLeg,
		SelectorLeg:      selectorLeg,
		OrderedBy:        "score",
	}
	if reranked && len(ranked) > 0 {
		resp.OrderedBy = "rerank_score"
	}
	for _, g := range groups {
		resp.CorporaSearched = append(resp.CorporaSearched, g.GetProviderId())
		if g.GetDegraded() {
			resp.Degraded = true
		}
	}
	if routingEvaluation {
		if routingTrace == nil {
			routingTrace = &sharedv1.RoutingTrace{StrategyName: selectedStrategy.Name, IndexStatus: "unavailable", UnavailableReason: "automatic_selection_trace_unavailable"}
		}
		routingTrace.ReturnedEvidence = returnedEvidenceState(groups, partial, pendingProviders)
		resp.RoutingTrace = routingTrace
	}
	if req.GetExplain() {
		if backgroundEvaluation {
			resp.RoutingExplanation = []string{"background evaluation: explicit bounded fan-out; classifier and reranker bypassed"}
		} else if hasExplicit {
			resp.RoutingExplanation = explain(req, targets)
		} else {
			resp.RoutingExplanation = append(autoExplain, matchedProvidersLine(targets))
		}
		if partial {
			resp.RoutingExplanation = append(resp.RoutingExplanation,
				fmt.Sprintf("partial results: query deadline expired with %d provider(s) pending", pendingProviders))
		}
		resp.RoutingExplanation = append(resp.RoutingExplanation, rerankExplain...)
	}
	resp.RoutingDegradeReason = routingIndexReason
	resp.LatencyMs = r.deps.Now().Sub(start).Milliseconds()

	// Phase-7 telemetry: record the query's outcome (best-effort; the recorder
	// swallows its own errors so a telemetry failure never affects the query).
	// LatencyMs is already stamped, so recording time is not counted.
	if r.deps.Recorder != nil {
		sample := buildSample(query, targets, resp)
		sample.AutoRoutedExternal = autoRoutedExternal
		sample.Escalated = escalated
		sample.RoutingMode = routingMode
		sample.EligibleProviderCount = len(active)
		sample.SelectedProviderCount = len(targets)
		sample.SelectedLeafCount = selectedLeafCount
		sample.WidenedLeafCount = widenedLeafCount
		sample.FanoutWidthBoundReached = fanoutBoundReached
		sample.WithheldExternalCount = len(pendingExternal)
		sample.QueuedProviderCount = max(0, len(targets)-r.deps.Concurrency)
		sample.ClassifierLatencyMs = classifierLatency
		sample.ResolverLatencyMs = resolverLatency
		cacheHitsAfter, cacheMissesAfter := resolutionCacheStats(r.deps.Resolver)
		sample.ResolverCacheHits = nonNegativeDelta(cacheHitsAfter, cacheHitsBefore)
		sample.ResolverCacheMisses = nonNegativeDelta(cacheMissesAfter, cacheMissesBefore)
		sample.FanoutLatencyMs = fanoutLatency
		sample.RerankLatencyMs = rerankLatency
		sample.RerankCandidateCount = len(fuseGroups(groups))
		sample.ResponseDegradeReason = ResponseDegradeReasonWithSelection(selectionFallback, rerankDegraded, rerankReason, groups, sample.ResultCount)
		if selectionReason != "" {
			if sample.ResponseDegradeReason != "" {
				sample.ResponseDegradeReason += ","
			}
			sample.ResponseDegradeReason += selectionReason
		}
		if routingIndexReason != "" {
			if sample.ResponseDegradeReason != "" {
				sample.ResponseDegradeReason += ","
			}
			sample.ResponseDegradeReason += routingIndexReason
		}
		r.deps.Recorder.Record(qctx, sample)
	}
	return resp, nil
}

func returnedEvidenceState(groups []*routingv1.ProviderResultGroup, partial bool, pending int) string {
	if partial || pending > 0 {
		return "partial"
	}
	for _, group := range groups {
		if group.GetDegraded() {
			continue
		}
		if len(group.GetHits()) > 0 {
			return "hits"
		}
	}
	for _, group := range groups {
		if group.GetDegraded() {
			return "degraded"
		}
	}
	return "empty"
}

func (r *Router) filterDemoted(providers []*registryv1.ProviderDescriptor) []*registryv1.ProviderDescriptor {
	if r.providerBreakers == nil {
		return providers
	}
	out := make([]*registryv1.ProviderDescriptor, 0, len(providers))
	for _, p := range providers {
		if r.providerBreakers.eligibleAutomatic(p.GetProviderId(), r.deps.Now()) {
			out = append(out, p)
		}
	}
	return out
}

// ProbeProviderRecovery issues one unattended, provider-scoped probe when a
// demotion's decay window has elapsed. A successful graded hit clears the
// zero-yield demotion through the normal result recorder; an empty graded
// response restarts the decay window. Transport/degraded failures are released
// as probation failures and never become zero-yield evidence.
func (r *Router) ProbeProviderRecovery(ctx context.Context, providerID, query string) (bool, error) {
	return r.probeProviderRecovery(ctx, providerID, query, false)
}

func (r *Router) probeProviderRecovery(ctx context.Context, providerID, query string, failureClaimed bool) (bool, error) {
	providerID = strings.TrimSpace(providerID)
	query = strings.TrimSpace(query)
	if providerID == "" {
		return false, ErrInvalidQuery{Reason: "provider_id is required"}
	}
	if query == "" {
		query = DefaultRecoveryProbeQuery
	}
	if r.providerBreakers == nil {
		return false, nil
	}
	r.providerBreakers.restore(ctx)
	if !failureClaimed {
		if !r.providerBreakers.beginRecoveryProbe(providerID, r.deps.Now()) {
			failureClaimed = r.providerBreakers.beginFailureRecoveryProbe(providerID, r.deps.Now())
			if !failureClaimed {
				return false, nil
			}
		}
	}
	probeCtx := WithBackgroundEvaluationProvider(WithRecoveryProbe(ctx), providerID)
	if failureClaimed {
		probeCtx = WithBackgroundEvaluationProvider(WithFailureRecoveryProbe(ctx), providerID)
	}
	runQuery := r.Query
	probeResponse, err := runQuery(probeCtx, &routingv1.QueryRequest{Query: query, Limit: 1})
	if err != nil {
		if failureClaimed {
			r.providerBreakers.finishFailureRecoveryProbe(providerID, r.deps.Now(), err.Error())
		} else {
			r.providerBreakers.recoveryProbeFailed(providerID, r.deps.Now(), err.Error())
		}
		return false, err
	}
	if probeResponse == nil {
		if failureClaimed {
			r.providerBreakers.finishFailureRecoveryProbe(providerID, r.deps.Now(), "provider returned no response")
		} else {
			r.providerBreakers.recoveryProbeFailed(providerID, r.deps.Now(), "provider returned no response")
		}
		return false, nil
	}
	if probeResponse.GetDegraded() {
		if failureClaimed {
			r.providerBreakers.finishFailureRecoveryProbe(providerID, r.deps.Now(), "provider response degraded")
		} else {
			r.providerBreakers.recoveryProbeFailed(providerID, r.deps.Now(), "provider response degraded")
		}
		return false, nil
	}
	for _, group := range probeResponse.GetGroups() {
		if group.GetProviderId() == providerID && len(group.GetHits()) > 0 {
			if failureClaimed {
				r.providerBreakers.finishFailureRecoveryProbe(providerID, r.deps.Now(), "")
			}
			return true, nil
		}
	}
	if failureClaimed {
		r.providerBreakers.finishFailureRecoveryProbe(providerID, r.deps.Now(), "provider returned no recovery group")
	} else {
		r.providerBreakers.recoveryProbeFailed(providerID, r.deps.Now(), "provider returned no recovery group")
	}
	return false, nil
}

// RunRecoveryProbes owns the low-rate unattended recovery loop. It is
// deliberately separate from interactive Query traffic and from eval cadence:
// a demoted provider is checked shortly after its decay deadline without
// waiting for a user query or a seven-day suite run.
func (r *Router) RunRecoveryProbes(ctx context.Context, interval time.Duration, query string) {
	if interval <= 0 {
		interval = DefaultRecoveryProbeInterval
	}
	if strings.TrimSpace(query) == "" {
		query = DefaultRecoveryProbeQuery
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.recoveryProbeCycle(ctx, query)
		}
	}
}

func (r *Router) recoveryProbeCycle(ctx context.Context, query string) {
	if r.providerBreakers == nil || r.deps.Lister == nil {
		return
	}
	r.providerBreakers.restore(ctx)
	providers, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		if r.deps.Logger != nil {
			r.deps.Logger.Printf("recovery probe: list providers: %v", err)
		}
		return
	}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, r.deps.PerProviderTimeout+time.Second)
		failureClaimed := r.providerBreakers.beginFailureRecoveryProbe(provider.GetProviderId(), r.deps.Now())
		var err error
		if failureClaimed {
			_, err = r.probeProviderRecovery(probeCtx, provider.GetProviderId(), query, true)
		} else {
			_, err = r.ProbeProviderRecovery(probeCtx, provider.GetProviderId(), query)
		}
		cancel()
		if err != nil && r.deps.Logger != nil {
			r.deps.Logger.Printf("recovery probe: provider %q: %v", provider.GetProviderId(), err)
		}
	}
}

// CircuitOpenQuorum reports the share of ACTIVE leaves with an open transport
// circuit. A cooldown-elapsed recovery probe is not counted open: it is
// reachable and eligible to prove recovery on the next request.
func (r *Router) CircuitOpenQuorum(ctx context.Context) (share float64, breached bool, err error) {
	if r.providerBreakers == nil {
		return 0, false, nil
	}
	r.providerBreakers.restore(ctx)
	active, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		return 0, false, fmt.Errorf("list providers for circuit quorum: %w", err)
	}
	if len(active) == 0 {
		return 0, false, nil
	}
	open := 0
	now := r.deps.Now()
	for _, provider := range active {
		isOpen, note := r.providerBreakers.status(provider.GetProviderId(), now)
		if isOpen && !strings.Contains(note, "recovery probe is due") {
			open++
		}
	}
	share = float64(open) / float64(len(active))
	return share, share >= CircuitOpenQuorumThreshold, nil
}

// RepromoteProvider is the explicit operator escape hatch for a graded-empty
// demotion. Automatic decay/probation remains the primary recovery path; this
// command exists for an owner who has repaired or verified a corpus and wants
// to clear only its zero-yield evidence immediately.
func (r *Router) RepromoteProvider(ctx context.Context, providerID string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ErrInvalidQuery{Reason: "provider_id is required"}
	}
	active, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE)})
	if err != nil {
		return fmt.Errorf("list providers for repromotion: %w", err)
	}
	found := false
	for _, provider := range active {
		if provider.GetProviderId() == providerID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("active provider %q not found", providerID)
	}
	if !r.providerBreakers.repromote(providerID, r.deps.Now()) {
		return fmt.Errorf("provider %q has no graded-empty demotion state", providerID)
	}
	return nil
}

// collapseDocumentHits makes the fan-out document-oriented before any shared
// reranking or response construction. Providers commonly return several
// passages for one source document; treating those passages as independent
// response slots wastes the page and lets a later chunk overwrite an earlier
// reranker judgment. The highest provider-score passage is the representative
// and merged_count makes the folding observable to consumers.
func collapseDocumentHits(groups []*routingv1.ProviderResultGroup) []*routingv1.ProviderResultGroup {
	for _, group := range groups {
		if group == nil || len(group.GetHits()) < 2 {
			if group != nil && len(group.GetHits()) == 1 && group.GetHits()[0] != nil && group.GetHits()[0].GetMergedCount() == 0 {
				group.GetHits()[0].MergedCount = 1
			}
			continue
		}
		type documentHit struct {
			hit   *routingv1.SearchHit
			count int32
			order int
		}
		byDocument := make(map[string]documentHit, len(group.GetHits()))
		for i, hit := range group.GetHits() {
			if hit == nil {
				continue
			}
			key := documentKey(hit, i)
			entry, ok := byDocument[key]
			if !ok {
				byDocument[key] = documentHit{hit: hit, count: 1, order: i}
				continue
			}
			entry.count++
			if hit.GetScore() > entry.hit.GetScore() {
				entry.hit = hit
			}
			byDocument[key] = entry
		}
		collapsed := make([]*routingv1.SearchHit, 0, len(byDocument))
		entries := make([]documentHit, 0, len(byDocument))
		for _, entry := range byDocument {
			entry.hit.MergedCount = entry.count
			entries = append(entries, entry)
		}
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].order < entries[j].order })
		for _, entry := range entries {
			collapsed = append(collapsed, entry.hit)
		}
		group.Hits = collapsed
	}
	return groups
}

func documentKey(hit *routingv1.SearchHit, fallback int) string {
	if path := strings.TrimSpace(hit.GetPath()); path != "" {
		return "path:" + path
	}
	if id := strings.TrimSpace(hit.GetId()); id != "" {
		return "id:" + id
	}
	return fmt.Sprintf("anonymous:%d", fallback)
}

// maybeRerank fuses the per-provider groups into one shortlist and reranks it
// into a unified, cross-provider ordered list. It never fails the query:
//
//   - no reranker wired ⇒ keep honest by-provider grouping (Phase-4 behavior).
//   - nothing to rank (zero hits) ⇒ same — grouping only, not a degradation.
//   - reranker errors (model down, bad output) ⇒ keep grouping and flag the
//     response degraded (the mandatory Phase-6 graceful-degradation path,
//     mirroring the classifier's).
//
// On success it returns the ranked list (each hit carrying RerankScore),
// reranked=true, and a one-line --explain note.
func (r *Router) maybeRerank(ctx context.Context, query string, groups []*routingv1.ProviderResultGroup, targets []*registryv1.ProviderDescriptor) (ranked []*routingv1.SearchHit, reranked, degraded bool, reason, leg string, explain []string) {
	if r.deps.Reranker == nil {
		return nil, false, false, "", "none", nil
	}
	candidates := fuseGroups(groups)
	if len(candidates) == 0 {
		return nil, false, false, "", "none", nil
	}
	// Reranking is a cross-provider operation. A single healthy provider already
	// owns the ordering of its hits, so invoking an optional global reranker here
	// adds latency and can make an otherwise healthy scoped route look degraded
	// when the reranker substrate is unavailable.
	groupsWithHits := 0
	for _, group := range groups {
		if group != nil && len(group.GetHits()) > 0 {
			groupsWithHits++
		}
	}
	if groupsWithHits <= 1 {
		return nil, false, false, "", "none", []string{"reranker skipped (single provider group)"}
	}
	if len(candidates) == 1 {
		return nil, false, false, "", "none", []string{"reranker skipped (single candidate)"}
	}
	if ok, line := r.rerankBreaker.allow(r.deps.Now()); !ok {
		return nil, false, true, "reranker_circuit_open", "none", []string{line + " (reranker_circuit_open)"}
	}

	preference := rerankPreference(targets)
	timeout, ok := r.rerankBudgetForLeg(ctx, preference)
	if !ok {
		return nil, false, true, "reranker_budget_exhausted", "none", []string{
			"reranker skipped (query budget nearly exhausted; reranker_budget_exhausted) — showing honest by-provider grouping",
		}
	}
	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var err error
	if policy, ok := r.deps.Reranker.(interface {
		RerankWithPreference(context.Context, string, []*routingv1.SearchHit, string) ([]*routingv1.SearchHit, error)
	}); ok {
		ranked, err = policy.RerankWithPreference(rctx, query, candidates, preference)
	} else {
		ranked, err = r.deps.Reranker.Rerank(rctx, query, candidates)
	}
	if err != nil {
		r.rerankBreaker.recordFailure(r.deps.Now())
		r.deps.Logger.Printf("routing.maybeRerank: reranker failed, keeping by-provider grouping: %v", err)
		return nil, false, true, "reranker_down", "none", []string{
			fmt.Sprintf("reranker unavailable (%s; reranker_down) — showing honest by-provider grouping", oneLine(err.Error())),
		}
	}
	r.rerankBreaker.recordSuccess()
	leg = "reranker"
	rankReason := ""
	if named, ok := r.deps.Reranker.(interface{ ActiveName(context.Context) string }); ok {
		if active := strings.TrimSpace(named.ActiveName(ctx)); active != "" && active != "none" {
			leg = active
		}
	}
	if named, ok := r.deps.Reranker.(interface {
		ActiveNameWithPreference(context.Context, string) string
	}); ok {
		if active := strings.TrimSpace(named.ActiveNameWithPreference(ctx, preference)); active != "" && active != "none" {
			leg = active
		}
	}
	if strings.HasPrefix(leg, "llm:") {
		rankReason = "reranker_degraded_to_llm"
	}
	return ranked, true, false, rankReason, leg, []string{fmt.Sprintf("reranked %d candidate(s) into one unified cross-provider list via %s", len(ranked), leg)}
}

func rerankPreference(targets []*registryv1.ProviderDescriptor) string {
	for _, target := range targets {
		if target != nil && target.GetTuning() != nil && target.GetTuning().GetRerankPreference() == aisearch.RerankPreferenceCrossEncoderRequired {
			return aisearch.RerankPreferenceCrossEncoderRequired
		}
	}
	return aisearch.RerankPreferenceCrossEncoderPreferred
}

func (r *Router) rerankBudgetForLeg(ctx context.Context, preference string) (time.Duration, bool) {
	timeout := r.deps.RerankTimeout
	if named, ok := r.deps.Reranker.(interface {
		ActiveNameWithPreference(context.Context, string) string
	}); ok {
		active := strings.TrimSpace(named.ActiveNameWithPreference(ctx, preference))
		switch {
		case strings.HasPrefix(active, "cross") && r.deps.CrossEncoderRerankTimeout > 0:
			timeout = r.deps.CrossEncoderRerankTimeout
		case strings.HasPrefix(active, "llm:") && r.deps.LLMRerankTimeout > 0:
			timeout = r.deps.LLMRerankTimeout
		}
	}
	return r.rerankBudgetWithTimeout(ctx, timeout)
}

func (r *Router) rerankBudgetWithTimeout(ctx context.Context, timeout time.Duration) (time.Duration, bool) {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline) - defaultResponseCushion
		if remaining <= 0 {
			return 0, false
		}
		if timeout <= 0 || remaining < timeout {
			timeout = remaining
		}
	}
	return timeout, timeout > 0
}

// autoSelect executes the active provider-selection ladder. The lexical stage
// remains the measured incumbent, while the guarded semantic candidate ranks
// every eligible provider description before the cross-encoder chooses the
// bounded fan-out. A failed semantic index is visible and falls back to that
// lexical arm; it never silently changes the active strategy's evidence.
func (r *Router) autoSelect(ctx context.Context, active []*registryv1.ProviderDescriptor, query string, strategy RetrievalStrategy) (targets []*registryv1.ProviderDescriptor, explain []string, selectionFallback bool, selectorLeg string, webShaped bool, selectedLeafCount, widenedLeafCount int, boundReached bool, routingIndexReason, selectionReason string, trace *sharedv1.RoutingTrace) {
	profiles := buildProfiles(active)
	trace = &sharedv1.RoutingTrace{StrategyName: strategy.Name, IndexStatus: "not_used"}
	if len(profiles) == 0 {
		trace.UnavailableReason = "no_eligible_provider"
		return nil, []string{"automatic routing found no eligible provider leaves"}, true, selectorLegLexical, false, 0, 0, false, "no_eligible_provider", "", trace
	}
	if r.deps.Classifier != nil {
		result, err := r.deps.Classifier.Classify(ctx, query, profiles)
		if err != nil {
			shortlist := lexicalProviderShortlist(query, profiles, 1)
			trace.IndexStatus = "not_used"
			trace.SelectionReason = "classifier_compatibility_fallback"
			trace.SelectedProviderIds = profileIDs(shortlist)
			return providersByID(active, profileIDs(shortlist)), []string{"legacy classifier compatibility fallback; lexical top-1 selected"}, true, selectorLegLexical, false, len(shortlist), 0, len(profiles) > len(shortlist), "legacy_classifier_removed", "", trace
		}
		ids := result.ProviderIDs
		if len(ids) == 0 {
			ids = result.Types
		}
		chosen := profilesForIDs(profiles, ids)
		if len(chosen) == 0 {
			chosen = lexicalProviderShortlist(query, profiles, 1)
		}
		if len(chosen) > r.factors.MaxFanoutWidth {
			chosen = chosen[:r.factors.MaxFanoutWidth]
		}
		trace.SelectionReason = "classifier_compatibility"
		trace.SelectedProviderIds = profileIDs(chosen)
		return providersByID(active, profileIDs(chosen)), []string{"legacy classifier compatibility path"}, false, selectorLegLexical, result.WebShaped && result.Confidence >= autoExternalThreshold(), len(chosen), 0, false, "", "", trace
	}

	shortlistWidth := strategyIntParam(strategy, StageLexical, "shortlist_width", r.factors.MaxFanoutWidth)
	lexicalRanking := lexicalProviderShortlist(query, profiles, len(profiles))
	trace.LexicalTopProviderIds = topProviderIDs(lexicalRanking, routingTraceTopK)
	shortlist := lexicalRanking
	if len(shortlist) > shortlistWidth {
		shortlist = shortlist[:shortlistWidth]
	}
	var semanticRanking []ProviderProfile
	denseScores := map[string]float64{}
	if strategyHasStage(strategy, StageEmbedding) {
		trace.IndexStatus = "unavailable"
	}
	if strategyHasStage(strategy, StageEmbedding) {
		candidateLimit := len(profiles)
		if strategyStringParam(strategy, StageEmbedding, "candidate_scope", "") != "all" {
			candidateLimit = strategyIntParam(strategy, StageEmbedding, "shortlist_width", r.factors.MaxFanoutWidth)
		}
		if r.deps.DescriptionIndex == nil {
			routingIndexReason = "routing_index_unavailable"
			trace.IndexReason = routingIndexReason
			trace.UnavailableReason = routingIndexReason
			explain = append(explain, "semantic provider-description index unavailable; lexical fallback strategy selected")
		} else {
			semantic, result := r.deps.DescriptionIndex.Shortlist(ctx, query, profiles, candidateLimit)
			if result.Available && len(semantic) > 0 {
				semanticRanking = semantic
				denseScores = result.Scores
				trace.IndexStatus = "available"
				trace.DenseTopProviderIds = topProviderIDs(semanticRanking, routingTraceTopK)
				evidenceWidth := strategyIntParam(strategy, StageEmbedding, "evidence_width", r.factors.MaxFanoutWidth)
				if strategyStringParam(strategy, StageEmbedding, "evidence_scope", "") == "all" {
					evidenceWidth = len(profiles)
				}
				shortlist = boundedProviderEvidenceUnion(semantic, lexicalRanking, evidenceWidth)
				explain = append(explain, fmt.Sprintf("semantic provider-description index ranked %d eligible leaves", result.Total))
				explain = append(explain, fmt.Sprintf("semantic and lexical top-%d provider evidence windows fused with reciprocal rank evidence", evidenceWidth))
			} else {
				routingIndexReason = result.Reason
				if routingIndexReason == "" {
					routingIndexReason = "routing_index_unavailable"
				}
				trace.IndexReason = routingIndexReason
				trace.UnavailableReason = routingIndexReason
				explain = append(explain, fmt.Sprintf("semantic provider-description index unavailable (%s); lexical fallback strategy selected", routingIndexReason))
			}
		}
	}
	if !strategyHasStage(strategy, StageEmbedding) {
		selectionReason = "lexical_shortlist"
	}
	if strategyHasStage(strategy, StageEmbedding) && routingIndexReason == "" && strategyHasStage(strategy, StageCrossEncoder) {
		selectionWidth := strategyIntParam(strategy, StageEmbedding, "selection_width", len(shortlist))
		if len(shortlist) > selectionWidth {
			shortlist = shortlist[:selectionWidth]
			explain = append(explain, fmt.Sprintf("semantic selector narrowed the cross-encoder window to %d ranked leaves", selectionWidth))
		}
	}
	selectedLeafCount = len(shortlist)
	boundReached = len(profiles) > len(shortlist)
	if strategyHasStage(strategy, StageEmbedding) && routingIndexReason == "" {
		explain = append(explain, fmt.Sprintf("semantic provider-description shortlist selected %d of %d leaves", len(shortlist), len(profiles)))
	} else {
		explain = append(explain, fmt.Sprintf("lexical provider shortlist selected %d of %d leaves", len(shortlist), len(profiles)))
	}

	ordered := shortlist
	var picked []ProviderProfile
	var rerankScores map[string]float64
	selectorLeg = selectorLegLexical
	if strategyHasStage(strategy, StageCrossEncoder) && len(shortlist) > 1 {
		var leg string
		var err error
		picked, leg, rerankScores, err = rerankProviderCandidates(ctx, query, shortlist, r.deps.Reranker)
		if err != nil {
			selectionFallback = true
			selectionReason = rerankerDegradationReason(ctx, err)
			trace.UnavailableReason = selectionReason
			explain = append(explain, fmt.Sprintf("cross-encoder and LLM provider picks unavailable (%s); lexical provider pick selected (%s)", oneLine(err.Error()), selectionReason))
		} else {
			ordered = picked
			selectionReason = "cross_encoder"
			if strategyHasStage(strategy, StageEmbedding) && routingIndexReason == "" {
				// Keep a strong exact lexical signal in the final provider
				// decision. The cross-encoder remains the primary semantic
				// judge, while a bounded lexical evidence window protects
				// concrete implementation queries whose identifiers or
				// declaration words are more precise than a short provider
				// description. This is rank evidence fusion over registered
				// metadata, not a provider-specific exception.
				if strategyStringParam(strategy, StageEmbedding, "selection_policy", "fused") == "lexical_guarded" {
					ordered = guardedSemanticProviderSelection(query, picked, semanticRanking, lexicalRanking, r.factors.MaxFanoutWidth, rerankScores)
					selectionReason = "cross_encoder_guarded_lexical"
					explain = append(explain, "cross-encoder order applied with a guarded lexical safety floor")
				} else {
					lexicalEvidence := lexicalRanking
					if len(lexicalEvidence) > r.factors.MaxFanoutWidth {
						lexicalEvidence = lexicalEvidence[:r.factors.MaxFanoutWidth]
					}
					ordered = fuseProviderRankings(picked, lexicalEvidence)
					explain = append(explain, "cross-encoder order fused with full lexical evidence")
				}
			}
			selectorLeg = leg
			switch leg {
			case selectorLegLLM:
				explain = append(explain, "cross-encoder provider pick unavailable; LLM provider pick selected the leading lexical candidate")
			default:
				explain = append(explain, "cross-encoder provider pick selected the leading lexical candidate")
			}
		}
	}
	fanoutWidth := strategyIntParam(strategy, StageCrossEncoder, "fanout_width", r.factors.MaxFanoutWidth)
	if fanoutWidth > r.factors.MaxFanoutWidth {
		fanoutWidth = r.factors.MaxFanoutWidth
	}
	if len(ordered) > fanoutWidth {
		ordered = ordered[:fanoutWidth]
		boundReached = true
	}
	chosen := make([]string, 0, len(ordered))
	for _, profile := range ordered {
		chosen = append(chosen, profile.ProviderID)
	}
	if len(chosen) == 0 {
		trace.SelectionReason = "no_selection"
		trace.UnavailableReason = "reranker_down"
		return nil, append(explain, "lexical provider selection returned no candidate"), true, selectorLegLexical, false, selectedLeafCount, 0, boundReached, "no_selection", "reranker_down", trace
	}
	if strategyHasStage(strategy, StageEmbedding) && routingIndexReason == "" {
		explain = append(explain, "automatic routing via semantic provider-description strategy (no LLM classifier)")
	} else {
		explain = append(explain, "automatic routing via lexical retrieval fallback (no LLM classifier)")
	}
	explain = append(explain, fmt.Sprintf("bounded automatic fan-out width=%d", fanoutWidth))
	explain = append(explain, fmt.Sprintf("routed to provider leaves: %s", strings.Join(chosen, ", ")))
	trace.SelectedProviderIds = chosen
	trace.SelectionReason = selectionReason
	trace.Candidates = buildRoutingTraceCandidates(query, semanticRanking, lexicalRanking, shortlist, picked, ordered, denseScores, rerankScores)
	webShaped = queryLooksWebShaped(query)
	return providersByID(active, chosen), explain, selectionFallback, selectorLeg, webShaped, selectedLeafCount, widenedLeafCount, boundReached, routingIndexReason, selectionReason, trace
}

func rerankerDegradationReason(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	if ctx.Err() != nil {
		return "reranker_budget_exhausted"
	}
	if strings.Contains(strings.ToLower(err.Error()), "circuit") {
		return "reranker_circuit_open"
	}
	return "reranker_down"
}

func profileIDs(profiles []ProviderProfile) []string {
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ProviderID)
	}
	return ids
}

func profilesForIDs(profiles []ProviderProfile, ids []string) []ProviderProfile {
	byID := make(map[string]ProviderProfile, len(profiles))
	byType := make(map[string][]ProviderProfile, len(profiles))
	for _, profile := range profiles {
		byID[profile.ProviderID] = profile
		byType[profile.Type] = append(byType[profile.Type], profile)
	}
	chosen := make([]ProviderProfile, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if profile, ok := byID[id]; ok {
			if _, exists := seen[profile.ProviderID]; !exists {
				chosen = append(chosen, profile)
				seen[profile.ProviderID] = struct{}{}
			}
			continue
		}
		for _, profile := range byType[id] {
			if _, exists := seen[profile.ProviderID]; exists {
				continue
			}
			chosen = append(chosen, profile)
			seen[profile.ProviderID] = struct{}{}
		}
	}
	return chosen
}

// providersByID returns the active leaves whose provider ids were selected,
// preserving the registry's provider_id order.
func providersByID(active []*registryv1.ProviderDescriptor, ids []string) []*registryv1.ProviderDescriptor {
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make([]*registryv1.ProviderDescriptor, 0, len(active))
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue
		}
		if _, ok := want[p.GetProviderId()]; ok {
			out = append(out, p)
		}
	}
	return out
}

// matchedProvidersLine renders the final --explain line naming the leaves the
// router fanned out to (shared by the explicit and classifier paths' output).
func matchedProvidersLine(targets []*registryv1.ProviderDescriptor) string {
	if len(targets) == 0 {
		return "matched providers: none"
	}
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.GetProviderId())
	}
	return fmt.Sprintf("matched providers: %s", strings.Join(ids, ", "))
}

// selectTargets applies the explicit routing selectors to the active provider
// set, preserving the registry's provider_id order (List orders ascending).
// Precedence within the optional --group scope: --all > explicit types > group
// alone (group with no narrower selector means "every active leaf in the
// group"). Providers without a callable endpoint are skipped defensively.
func selectTargets(active []*registryv1.ProviderDescriptor, req *routingv1.QueryRequest) []*registryv1.ProviderDescriptor {
	group := strings.TrimSpace(req.GetGroup())
	typeSet := make(map[string]struct{})
	for _, t := range nonEmpty(req.GetTypes()) {
		typeSet[t] = struct{}{}
	}

	out := make([]*registryv1.ProviderDescriptor, 0, len(active))
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue
		}
		if group != "" && p.GetProviderGroup() != group {
			continue
		}
		switch {
		case req.GetAll():
			out = append(out, p)
		case len(typeSet) > 0:
			if providerTypeSelected(typeSet, p.GetType()) {
				out = append(out, p)
			}
		case group != "":
			out = append(out, p)
		}
	}
	return out
}

// providerTypeSelected keeps entity-specific result types while allowing the
// established record selector to address durable record-like entities. A run
// remains typed as run in results; record is only the broader query selector.
func providerTypeSelected(selected map[string]struct{}, providerType string) bool {
	if _, ok := selected[providerType]; ok {
		return true
	}
	_, recordSelected := selected["record"]
	return recordSelected && providerType == "run"
}

// partitionByScope splits the active providers into the project-scope candidate
// set the classifier/auto path may route to, and the SCOPE_EXTERNAL providers
// withheld from automatic routing. SCOPE_UNSPECIFIED is treated as project-scope
// (a provider that never declared a scope is assumed internal, not external) so
// a legacy provider keeps its default-routable behavior. Order is preserved.
func partitionByScope(active []*registryv1.ProviderDescriptor) (project, external []*registryv1.ProviderDescriptor) {
	for _, p := range active {
		if p.GetScope() == registryv1.Scope_SCOPE_EXTERNAL {
			external = append(external, p)
			continue
		}
		project = append(project, p)
	}
	return project, external
}

// withheldExternalLine is the --explain note emitted when external providers
// were kept out of the automatic candidate set.
func withheldExternalLine(external []*registryv1.ProviderDescriptor) string {
	ids := make([]string, 0, len(external))
	for _, p := range external {
		ids = append(ids, p.GetProviderId())
	}
	return fmt.Sprintf("withheld %d external provider(s) from auto-routing (reach with --all or --type <type>): %s",
		len(external), strings.Join(ids, ", "))
}

// autoRoutedExternalLine is the --explain note emitted when a web-shaped query
// (with the operator opt-in) folded the external provider(s) into auto-routing.
func autoRoutedExternalLine(external []*registryv1.ProviderDescriptor) string {
	return fmt.Sprintf("auto-routed %d external provider(s) — query judged web-shaped (opt-in enabled): %s",
		len(external), strings.Join(providerIDs(external), ", "))
}

// escalationLine is the --explain note emitted when the project corpus came
// back weak and the router escalated to the withheld external provider(s).
// reason distinguishes the empty round from the all-below-threshold round.
func escalationLine(external []*registryv1.ProviderDescriptor, reason string) string {
	return fmt.Sprintf("escalated to %d external provider(s) — %s (opt-in enabled): %s",
		len(external), reason, strings.Join(providerIDs(external), ", "))
}

// providerIDs projects descriptors to their provider_id list.
func providerIDs(ps []*registryv1.ProviderDescriptor) []string {
	ids := make([]string, 0, len(ps))
	for _, p := range ps {
		ids = append(ids, p.GetProviderId())
	}
	return ids
}

func (r *Router) filterAutomatic(ctx context.Context, providers []*registryv1.ProviderDescriptor) ([]*registryv1.ProviderDescriptor, []string) {
	filtered := make([]*registryv1.ProviderDescriptor, 0, len(providers))
	var explain []string
	for _, provider := range providers {
		if lifecycle := provider.GetLifecycle(); lifecycle != registryv1.Lifecycle_LIFECYCLE_UNSPECIFIED && lifecycle != registryv1.Lifecycle_LIFECYCLE_PRODUCTION {
			continue
		}
		if r.deps.EvalQuality != nil {
			evidence, err := r.deps.EvalQuality.LatestProviderEval(ctx, provider.GetProviderId())
			if err == nil && evidence.EvidenceAvailable {
				if reason := automaticEvidenceExclusion(evidence); reason != "" {
					explain = append(explain, fmt.Sprintf("withheld (evidence): %s — %s", provider.GetProviderId(), reason))
					continue
				}
			}
			if err == nil && evidence.Fresh && !evidence.Degraded && QualityJunkLeak(evidence.MaxGibberishScore, evidence.MeanStrongTop1) && strings.TrimSpace(provider.GetJunkLeakOptOutReason()) == "" {
				explain = append(explain, fmt.Sprintf("withheld (junk leak): %s run=%s gibberish=%.3f strong=%.3f margin=%.3f", provider.GetProviderId(), evidence.RunID, evidence.MaxGibberishScore, evidence.MeanStrongTop1, QualityJunkLeakMargin))
				continue
			}
			if err == nil && evidence.Fresh && !evidence.Degraded && QualityJunkLeak(evidence.MaxGibberishScore, evidence.MeanStrongTop1) && strings.TrimSpace(provider.GetJunkLeakOptOutReason()) != "" {
				explain = append(explain, fmt.Sprintf("quality gate opted out: %s reason=%s", provider.GetProviderId(), strings.TrimSpace(provider.GetJunkLeakOptOutReason())))
			}
		}
		filtered = append(filtered, provider)
	}
	return filtered, explain
}

// Weakness reasons surfaced in the escalation --explain line so an operator
// can tell an empty round from a low-scoring one.
const (
	weakReasonEmpty          = "project results were empty"
	weakReasonBelowThreshold = "all project results scored below the weakness threshold"
)

// resultsWeakness classifies the fanned-out project round for OT-P2-002
// fallback escalation. It returns "" (non-weak) as soon as any hit in a
// non-degraded group reaches threshold; otherwise weakReasonEmpty when the
// round produced no hits at all, or weakReasonBelowThreshold when hits exist
// but every one scored under threshold (normalized 0..1 scores — the same
// regime as SEARCH_HUB_AUTO_EXTERNAL_THRESHOLD; unscored providers report 0,
// which deliberately counts as weak: recall over precision behind the opt-in).
func resultsWeakness(groups []*routingv1.ProviderResultGroup, threshold float64) string {
	anyHit := false
	for _, g := range groups {
		if g.GetDegraded() {
			continue
		}
		for _, h := range g.GetHits() {
			anyHit = true
			if h.GetScore() >= threshold {
				return ""
			}
		}
	}
	if !anyHit {
		return weakReasonEmpty
	}
	return weakReasonBelowThreshold
}

// fanOut queries every target concurrently (bounded by Concurrency) and returns
// one group per target, ordered by provider_id for deterministic output.
func (r *Router) fanOut(ctx context.Context, targets []*registryv1.ProviderDescriptor, query string, limit int32, explicit bool, scope string) []*routingv1.ProviderResultGroup {
	if isRecoveryProbe(ctx) {
		// The provider must be selected explicitly so a demoted leaf can be
		// tested, but its result is automatic recovery evidence rather than an
		// operator override.
		explicit = false
	}
	groups := make([]*routingv1.ProviderResultGroup, len(targets))
	sem := make(chan struct{}, r.deps.Concurrency)
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t *registryv1.ProviderDescriptor) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if !isFailureRecoveryProbe(ctx) {
				if ok, note := r.providerBreakers.allow(t.GetProviderId(), r.deps.Now()); !ok {
					groups[i] = degrade(&routingv1.ProviderResultGroup{ProviderId: t.GetProviderId()}, note)
					return
				}
			}
			groups[i] = r.callProvider(ctx, t, query, limit, scope)
			r.providerBreakers.record(t.GetProviderId(), groups[i].GetDegraded(), r.deps.Now())
			r.providerBreakers.recordResult(t.GetProviderId(), len(groups[i].GetHits()), groups[i].GetDegraded(), explicit, r.deps.Now())
		}(i, t)
	}
	wg.Wait()

	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].GetProviderId() < groups[j].GetProviderId()
	})
	return groups
}

// callProvider performs one provider's round-trip and always returns a group:
// on any failure it returns a degraded group carrying a human-readable note
// rather than an error, so one bad provider never sinks the query.
func (r *Router) callProvider(ctx context.Context, d *registryv1.ProviderDescriptor, query string, limit int32, scope string) *routingv1.ProviderResultGroup {
	key := strings.Join([]string{d.GetProviderId(), query, fmt.Sprint(limit), scope}, "\x00")
	r.providerCallMu.Lock()
	if call := r.providerCalls[key]; call != nil {
		call.waiters++
		r.providerCallMu.Unlock()
		return r.waitForProviderCall(ctx, key, d.GetProviderId(), call)
	}
	callCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	call := &providerCall{done: make(chan struct{}), cancel: cancel, waiters: 1}
	r.providerCalls[key] = call
	r.providerCallMu.Unlock()
	go func() {
		defer cancel()
		call.result = r.callProviderDirect(callCtx, d, query, limit, scope)
		close(call.done)
		r.providerCallMu.Lock()
		if r.providerCalls[key] == call {
			delete(r.providerCalls, key)
		}
		r.providerCallMu.Unlock()
	}()
	return r.waitForProviderCall(ctx, key, d.GetProviderId(), call)
}

func (r *Router) waitForProviderCall(ctx context.Context, key, providerID string, call *providerCall) *routingv1.ProviderResultGroup {
	select {
	case <-call.done:
		if call.result == nil {
			return degrade(&routingv1.ProviderResultGroup{}, "provider call completed without a result")
		}
		return proto.Clone(call.result).(*routingv1.ProviderResultGroup)
	case <-ctx.Done():
		r.providerCallMu.Lock()
		call.waiters--
		if call.waiters == 0 {
			call.cancel()
			if r.providerCalls[key] == call {
				delete(r.providerCalls, key)
			}
		}
		r.providerCallMu.Unlock()
		return degrade(&routingv1.ProviderResultGroup{ProviderId: providerID}, fmt.Sprintf("request cancelled: %s", oneLine(ctx.Err().Error())))
	}
}

func (r *Router) callProviderDirect(ctx context.Context, d *registryv1.ProviderDescriptor, query string, limit int32, scope string) *routingv1.ProviderResultGroup {
	start := r.deps.Now()
	g := &routingv1.ProviderResultGroup{ProviderId: d.GetProviderId()}
	defer func() {
		g.LatencyMs = r.deps.Now().Sub(start).Milliseconds()
	}()

	hj := d.GetEndpoint().GetHttpJson()
	if hj == nil {
		return degrade(g, "unsupported endpoint kind (only http_json fan-out is implemented in Phase 4)")
	}

	base, err := r.deps.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		if isScenarioNotRunning(err) {
			r.providerBreakers.openImmediately(d.GetProviderId(), r.deps.Now())
		}
		return degrade(g, fmt.Sprintf("provider scenario %q unreachable: %s", hj.GetScenarioId(), oneLine(err.Error())))
	}

	body, err := providers.RenderBodyWithScope(hj.GetBodyTemplate(), query, limit, d.GetType(), scope)
	if err != nil {
		return degrade(g, fmt.Sprintf("request build failed: %s", err))
	}

	cctx, cancel := context.WithTimeout(ctx, r.deps.PerProviderTimeout)
	defer cancel()

	url := strings.TrimRight(base, "/") + hj.GetPath()
	httpReq, err := http.NewRequestWithContext(cctx, providers.HTTPMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return degrade(g, fmt.Sprintf("request build failed: %s", oneLine(err.Error())))
	}
	providers.ApplyHeaders(httpReq, hj.GetHeaders())

	resp, err := r.deps.Doer.Do(httpReq)
	if err != nil {
		return degrade(g, fmt.Sprintf("request failed: %s", oneLine(err.Error())))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return degrade(g, fmt.Sprintf("reading response failed: %s", oneLine(err.Error())))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return degrade(g, fmt.Sprintf("provider returned HTTP %d", resp.StatusCode))
	}

	mapped, err := providers.MapResponse(d, respBody)
	if err != nil {
		return degrade(g, fmt.Sprintf("result mapping failed: %s", oneLine(err.Error())))
	}
	hits := mapped.Hits
	if len(hits) > int(limit) {
		hits = hits[:limit]
	}
	g.Hits = hits
	g.Count = int32(len(hits))
	g.Coverage = mapped.Coverage
	g.Degradations = mapped.Degradations
	g.NextCursor = mapped.NextCursor
	if len(mapped.Degradations) > 0 {
		g.Degraded = true
		g.Note = mapped.Degradations[0].GetDetail()
	}
	return g
}

// degrade marks a group degraded with a note and returns it (for one-line use
// in callProvider's failure branches).
func degrade(g *routingv1.ProviderResultGroup, note string) *routingv1.ProviderResultGroup {
	g.Degraded = true
	g.Note = note
	return g
}

// explain renders the human-readable routing rationale for --explain on the
// explicit-selector path: the caller named --all/--type/--group, so the
// explanation states which selector chose the targets (the classifier path has
// its own explanation, built in autoSelect).
func explain(req *routingv1.QueryRequest, targets []*registryv1.ProviderDescriptor) []string {
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.GetProviderId())
	}
	var selector string
	switch {
	case req.GetAll():
		selector = "explicit --all → every active provider"
	case len(nonEmpty(req.GetTypes())) > 0:
		selector = fmt.Sprintf("explicit --type %s", strings.Join(nonEmpty(req.GetTypes()), ","))
	default:
		selector = fmt.Sprintf("explicit --group %s → every active leaf in the group", strings.TrimSpace(req.GetGroup()))
	}
	if g := strings.TrimSpace(req.GetGroup()); g != "" && !req.GetAll() && len(nonEmpty(req.GetTypes())) > 0 {
		selector += fmt.Sprintf(" scoped to --group %s", g)
	}
	out := []string{
		"explicit routing (selector provided; classifier not consulted)",
		fmt.Sprintf("selector: %s", selector),
	}
	if len(ids) == 0 {
		out = append(out, "matched providers: none")
	} else {
		out = append(out, fmt.Sprintf("matched providers: %s", strings.Join(ids, ", ")))
	}
	return out
}

// nonEmpty filters out blank entries (and trims) so a stray empty --type token
// is not treated as a selector.
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// oneLine collapses newlines so a multi-line transport error renders as a
// single clean note.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
