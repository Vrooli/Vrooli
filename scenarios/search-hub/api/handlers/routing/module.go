// Package routing is the search-hub routing domain's API surface: the generated
// RoutingService Connect-RPC handler that fans a query out across registered
// providers. It is the router core (Phase 4) sitting beside the registry
// domain.
//
// This package is the wiring edge: it composes the pure internal/routing.Router
// with the concrete cross-scenario URL resolver (api-core/discovery), the timed
// outbound HTTP client (internal/httpc), and the shared TEI-primary reranker
// chain (Phase 6 unified ranking).
// internal/routing itself stays dependency-light (interfaces only) so it is
// unit-testable without the network, a model, or the CLI.
package routing

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	internaleval "search-hub/internal/eval"
	"search-hub/internal/httpc"
	internalmetrics "search-hub/internal/metrics"
	"search-hub/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	internalregistry "search-hub/internal/registry"
	internalrouting "search-hub/internal/routing"
)

// CircuitOpenQuorumThreshold is the shared federation-health policy exposed
// for main's dependency checker without leaking internal router construction.
const CircuitOpenQuorumThreshold = internalrouting.CircuitOpenQuorumThreshold

// Module returns the routing domain's contribution to the API: the generated
// RoutingService Connect handler backed by a Router that reads the same SQLite
// provider registry the registry domain writes, resolves provider base URLs at
// call-time, and fans out over a timed HTTP client.
//
// recorder is the Phase-7 telemetry write seam (the metrics domain's bridge);
// it may be nil, in which case no telemetry is recorded. It is injected rather
// than constructed here so the routing handler stays free of any metrics-store
// import (the seam-discovery / wiring-edge convention).
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, recorder internalrouting.TelemetryRecorder) module.Module {
	return ModuleWithRouter(NewRouter(db, clk, logger, recorder), logger)
}

// NewRouter builds the production router once so the health domain and the
// RoutingService observe the same breaker state and registry snapshot.
func NewRouter(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, recorder internalrouting.TelemetryRecorder) *internalrouting.Router {
	activeStrategy, routerFactors, err := internalrouting.LoadActiveStrategy()
	if err != nil {
		// Strategy data is part of the executable's startup contract. Continuing
		// with guessed defaults would make measurements non-reproducible, so fail
		// the wiring edge loudly before serving a query.
		panic(err)
	}
	strategyCatalog, err := internalrouting.LoadStrategyCatalog()
	if err != nil {
		panic(err)
	}
	store := internalregistry.NewSQLiteStore(db, clk)
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:                    store,
		Resolver:                  newScenarioResolver(),
		Doer:                      httpc.NewDefault(),
		Reranker:                  internalrouting.NewDefaultRerankerChain(),
		Recorder:                  recorder,
		EvalQuality:               newEvalQualityReader(db, clk),
		DescriptionIndex:          newProviderDescriptionIndex(),
		DemotionStore:             internalmetrics.NewSQLiteDemotionStore(db, clk),
		Logger:                    logger,
		Strategy:                  &activeStrategy,
		StrategyCatalog:           strategyCatalog,
		RouterFactors:             &routerFactors,
		RerankTimeout:             durationEnv(logger, "SEARCH_HUB_RERANK_TIMEOUT", 10*time.Second, 100*time.Millisecond, 20*time.Second),
		CrossEncoderRerankTimeout: durationEnv(logger, "SEARCH_HUB_CROSS_ENCODER_RERANK_TIMEOUT", 500*time.Millisecond, 100*time.Millisecond, 20*time.Second),
		LLMRerankTimeout:          durationEnv(logger, "SEARCH_HUB_LLM_RERANK_TIMEOUT", 8*time.Second, time.Second, 20*time.Second),
		RerankBreaker: internalrouting.RerankBreakerConfig{
			FailureThreshold: intEnv(logger, "SEARCH_HUB_RERANK_BREAKER_FAILURES", 3, 1, 20),
			Cooldown:         durationEnv(logger, "SEARCH_HUB_RERANK_BREAKER_COOLDOWN", 60*time.Second, time.Second, 10*time.Minute),
		},
		ProviderBreaker: internalrouting.RerankBreakerConfig{
			FailureThreshold:       intEnv(logger, "SEARCH_HUB_PROVIDER_BREAKER_FAILURES", 3, 1, 20),
			Cooldown:               durationEnv(logger, "SEARCH_HUB_PROVIDER_BREAKER_COOLDOWN", 30*time.Second, time.Second, 10*time.Minute),
			ZeroYieldMinimumRoutes: routerFactors.ZeroYieldMinimumRoutes,
			DemotionWindow:         routerFactors.DemotionWindow,
		},
		AutoRouteExternal: autoRouteExternalEnabled(),
	})
	return router
}

func newProviderDescriptionIndex() internalrouting.ProviderDescriptionIndex {
	model := strings.TrimSpace(os.Getenv("SEARCH_HUB_DESCRIPTION_EMBED_MODEL"))
	if model == "" {
		model = aisearch.DefaultEmbedModel
	}
	cachePath := strings.TrimSpace(os.Getenv("SEARCH_HUB_DESCRIPTION_INDEX_CACHE"))
	if cachePath == "" {
		if dataDir := strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR")); dataDir != "" {
			cachePath = filepath.Join(dataDir, "provider-description-index.json")
		}
	}
	return internalrouting.NewPersistentEmbeddingDescriptionIndex(aisearch.NewEmbedderForConfig(aisearch.Config{
		EmbedModel:      model,
		EmbedTaskPrefix: true,
	}), cachePath)
}

// StartRecoveryProbes starts the low-rate, unattended zero-yield recovery
// loop. It is kept at the wiring edge so internal/routing remains usable in
// deterministic tests without starting a goroutine.
func StartRecoveryProbes(router *internalrouting.Router, logger *log.Logger) {
	if router == nil {
		return
	}
	interval := durationEnv(logger, "SEARCH_HUB_ZERO_YIELD_PROBE_INTERVAL", internalrouting.DefaultRecoveryProbeInterval, time.Second, time.Hour)
	go router.RunRecoveryProbes(context.Background(), interval, internalrouting.DefaultRecoveryProbeQuery)
}

// ModuleWithRouter mounts an already-wired router. Keeping construction
// separate lets main.go share the exact breaker state with health checks.
func ModuleWithRouter(router *internalrouting.Router, logger *log.Logger) module.Module {
	connectPath, connectHandler := routingconnect.NewRoutingServiceHandler(NewConnectHandler(Deps{
		Router: router,
		Logger: logger,
	}))
	return module.Module{
		Name: "routing",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

const evalQualityFreshnessWindow = 30 * 24 * time.Hour

type evalQualityReader struct {
	store internaleval.Store
	now   func() time.Time
}

func newEvalQualityReader(db *database.RoutedDB, clk schedule.Clock) internalrouting.EvalQualityReader {
	return &evalQualityReader{store: internaleval.NewSQLiteStore(db, clk), now: clk.Now}
}

func (r *evalQualityReader) LatestProviderEval(ctx context.Context, providerID string) (internalrouting.EvalQualityEvidence, error) {
	suites, err := r.store.ListSuites(ctx, internaleval.ListSuitesFilter{ProviderID: strings.TrimSpace(providerID)})
	if err != nil {
		return internalrouting.EvalQualityEvidence{}, err
	}
	if len(suites) == 0 || suites[0] == nil {
		return internalrouting.EvalQualityEvidence{EvidenceAvailable: true}, nil
	}
	evidence := internalrouting.EvalQualityEvidence{EvidenceAvailable: true, SuitePresent: true}
	for _, testCase := range suites[0].GetCases() {
		if testCase != nil && strings.EqualFold(strings.TrimSpace(testCase.GetStatus()), "candidate") {
			continue
		}
		if testCase != nil && len(testCase.GetExpectIds()) > 0 {
			evidence.LiveReviewedPositive = true
			break
		}
	}
	// Automatic eligibility is a provider-quality gate, so use provider-direct
	// evidence. The newest stored run is often a federated run and may be
	// unavailable because routing or a sibling provider was degraded; allowing
	// that run to erase a fresh direct pass would make the gate fail closed for
	// the wrong owner. Keep a bounded history so a single newer unavailable run
	// cannot hide a recent passing direct run.
	runs, err := r.store.ListRuns(ctx, internaleval.ListRunsFilter{
		SuiteID: suites[0].GetSuiteId(),
		Tier:    "provider_direct",
		Limit:   20,
	})
	if err != nil {
		return internalrouting.EvalQualityEvidence{}, err
	}
	if len(runs) == 0 || runs[0] == nil {
		return evidence, nil
	}
	run := runs[0]
	created, err := time.Parse(time.RFC3339Nano, run.GetCreatedAt())
	if err != nil {
		return evidence, nil
	}
	age := r.now().UTC().Sub(created)
	aggregate := run.GetAggregate()
	evidence = internalrouting.EvalQualityEvidence{
		EvidenceAvailable:    true,
		RunID:                run.GetRunId(),
		Fresh:                age >= 0 && age <= evalQualityFreshnessWindow,
		Degraded:             run.GetDegraded(),
		SuitePresent:         true,
		LiveReviewedPositive: evidence.LiveReviewedPositive,
	}
	if aggregate != nil {
		evidence.MeanStrongTop1 = aggregate.GetMeanStrongTop1()
		evidence.MaxGibberishScore = aggregate.GetMaxGibberishScore()
		evidence.GibberishLeak = internalrouting.QualityJunkLeak(evidence.MaxGibberishScore, evidence.MeanStrongTop1)
	}
	evidence.RecentPassingRun = hasRecentPassingDirectRun(runs, r.now())
	if validator, ok := r.store.(internaleval.CorpusValidationReader); ok {
		validation, validationErr := validator.LatestCorpusValidation(ctx, suites[0].GetSuiteId())
		if validationErr == nil && validation != nil && validation.Result != nil && validation.Result.GetRollup() != nil {
			rollup := validation.Result.GetRollup()
			evidence.CorpusAllStale = rollup.GetPositives() > 0 && rollup.GetStale() == rollup.GetPositives()
			if evidence.CorpusAllStale {
				evidence.RecentPassingRun = false
			}
		}
	}
	return evidence, nil
}

// hasRecentPassingDirectRun answers the evidence-gate question over the
// bounded provider-direct history. A newer unavailable or degraded run must
// not erase a still-fresh passing run: the gate measures whether the provider
// has recent direct quality evidence, not whether the latest federated attempt
// happened to reach every provider.
func hasRecentPassingDirectRun(runs []*evalv1.EvalRun, now time.Time) bool {
	for _, run := range runs {
		if run == nil || run.GetTier() != "provider_direct" || run.GetDegraded() {
			continue
		}
		created, err := time.Parse(time.RFC3339Nano, run.GetCreatedAt())
		if err != nil {
			continue
		}
		age := now.UTC().Sub(created)
		if age < 0 || age > evalQualityFreshnessWindow {
			continue
		}
		aggregate := run.GetAggregate()
		if aggregate == nil || aggregate.GetGradedCases() <= 0 || aggregate.GetPassRate() < 0.5 {
			continue
		}
		return true
	}
	return false
}

// autoRouteExternalEnabled reads the OT-P2-002 opt-in flag from the environment.
// DEFAULT FALSE: strategy-driven external auto-routing + fallback escalation
// only fire when an operator explicitly sets SEARCH_HUB_AUTO_ROUTE_EXTERNAL to a
// truthy value (1/true/yes/on). Keeps the thin-router default behavior — a plain
// federated query never reaches a rate-limited external corpus on its own.
func autoRouteExternalEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SEARCH_HUB_AUTO_ROUTE_EXTERNAL"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func durationEnv(logger *log.Logger, key string, fallback, min, max time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		logConfigFallback(logger, key, raw, fallback, "invalid duration")
		return fallback
	}
	if v < min || v > max {
		logConfigFallback(logger, key, raw, fallback, "outside supported range")
		return fallback
	}
	return v
}

func intEnv(logger *log.Logger, key string, fallback, min, max int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		logConfigFallback(logger, key, raw, fallback, "invalid integer")
		return fallback
	}
	if v < min || v > max {
		logConfigFallback(logger, key, raw, fallback, "outside supported range")
		return fallback
	}
	return v
}

func logConfigFallback(logger *log.Logger, key, raw string, fallback any, reason string) {
	if logger == nil {
		logger = log.Default()
	}
	logger.Printf("routing config: ignoring %s=%q (%s); using default %v", key, raw, reason, fallback)
}

// scenarioResolver adapts api-core/discovery's Resolver to the router's
// URLResolver seam. The discovery resolver uses a short-lived address cache
// (with failure invalidation), so repeated leaves from one scenario do not
// fork one CLI process per provider while a re-ported provider still converges
// quickly. This is backend resolution, never a client-computed URL.
type scenarioResolver struct {
	r *discovery.Resolver
}

func newScenarioResolver() *scenarioResolver {
	return &scenarioResolver{r: discovery.NewResolver(discovery.ResolverConfig{})}
}

func (s *scenarioResolver) ResolveScenarioURL(ctx context.Context, scenarioID string) (string, error) {
	return s.r.ResolveScenarioURLDefault(ctx, scenarioID)
}

func (s *scenarioResolver) CacheStats() (hits, misses int64) {
	return s.r.CacheStats()
}
