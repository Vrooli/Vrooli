// Package routing is the search-hub routing domain's API surface: the generated
// RoutingService Connect-RPC handler that fans a query out across registered
// providers. It is the router core (Phase 4) sitting beside the registry
// domain.
//
// This package is the wiring edge: it composes the pure internal/routing.Router
// with the concrete cross-scenario URL resolver (api-core/discovery), the timed
// outbound HTTP client (internal/httpc), the local-Ollama classifier (Phase 5
// automatic routing), and the shared TEI-primary reranker chain (Phase 6 unified ranking).
// internal/routing itself stays dependency-light (interfaces only) so it is
// unit-testable without the network, a model, or the CLI.
package routing

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"search-hub/internal/clock"
	internaleval "search-hub/internal/eval"
	"search-hub/internal/httpc"
	internalmetrics "search-hub/internal/metrics"
	"search-hub/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

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
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger, recorder internalrouting.TelemetryRecorder) module.Module {
	return ModuleWithRouter(NewRouter(db, clk, logger, recorder), logger)
}

// NewRouter builds the production router once so the health domain and the
// RoutingService observe the same breaker state and registry snapshot.
func NewRouter(db *database.RoutedDB, clk clock.Clock, logger *log.Logger, recorder internalrouting.TelemetryRecorder) *internalrouting.Router {
	store := internalregistry.NewSQLiteStore(db, clk)
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:             store,
		Resolver:           newScenarioResolver(),
		Doer:               httpc.NewDefault(),
		Classifier:         internalrouting.NewOllamaClassifier(),
		Reranker:           internalrouting.NewDefaultRerankerChain(),
		Recorder:           recorder,
		EvalQuality:        newEvalQualityReader(db, clk),
		DemotionStore:      internalmetrics.NewSQLiteDemotionStore(db, clk),
		Logger:             logger,
		Concurrency:        intEnv(logger, "SEARCH_HUB_ROUTING_CONCURRENCY", 8, 1, 128),
		PerProviderTimeout: durationEnv(logger, "SEARCH_HUB_PROVIDER_TIMEOUT", 4*time.Second, 100*time.Millisecond, 20*time.Second),
		QueryTimeout:       durationEnv(logger, "SEARCH_HUB_QUERY_TIMEOUT", 25*time.Second, time.Second, 29*time.Second),
		RerankTimeout:      durationEnv(logger, "SEARCH_HUB_RERANK_TIMEOUT", 10*time.Second, 100*time.Millisecond, 20*time.Second),
		RerankBreaker: internalrouting.RerankBreakerConfig{
			FailureThreshold: intEnv(logger, "SEARCH_HUB_RERANK_BREAKER_FAILURES", 3, 1, 20),
			Cooldown:         durationEnv(logger, "SEARCH_HUB_RERANK_BREAKER_COOLDOWN", 60*time.Second, time.Second, 10*time.Minute),
		},
		ProviderBreaker: internalrouting.RerankBreakerConfig{
			FailureThreshold:       intEnv(logger, "SEARCH_HUB_PROVIDER_BREAKER_FAILURES", 3, 1, 20),
			Cooldown:               durationEnv(logger, "SEARCH_HUB_PROVIDER_BREAKER_COOLDOWN", 30*time.Second, time.Second, 10*time.Minute),
			ZeroYieldMinimumRoutes: int64(intEnv(logger, "SEARCH_HUB_ZERO_YIELD_ROUTES", 5, 1, 1000)),
			DemotionWindow:         durationEnv(logger, "SEARCH_HUB_ZERO_YIELD_WINDOW", 15*time.Minute, time.Minute, 24*time.Hour),
		},
		AutoRouteExternal: autoRouteExternalEnabled(),
	})
	return router
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

func newEvalQualityReader(db *database.RoutedDB, clk clock.Clock) internalrouting.EvalQualityReader {
	return &evalQualityReader{store: internaleval.NewSQLiteStore(db, clk), now: clk.Now}
}

func (r *evalQualityReader) LatestProviderEval(ctx context.Context, providerID string) (internalrouting.EvalQualityEvidence, error) {
	suites, err := r.store.ListSuites(ctx, internaleval.ListSuitesFilter{ProviderID: strings.TrimSpace(providerID)})
	if err != nil {
		return internalrouting.EvalQualityEvidence{}, err
	}
	if len(suites) == 0 || suites[0] == nil {
		return internalrouting.EvalQualityEvidence{}, nil
	}
	runs, err := r.store.ListRuns(ctx, internaleval.ListRunsFilter{SuiteID: suites[0].GetSuiteId(), Limit: 1})
	if err != nil {
		return internalrouting.EvalQualityEvidence{}, err
	}
	if len(runs) == 0 || runs[0] == nil {
		return internalrouting.EvalQualityEvidence{}, nil
	}
	run := runs[0]
	created, err := time.Parse(time.RFC3339Nano, run.GetCreatedAt())
	if err != nil {
		return internalrouting.EvalQualityEvidence{}, nil
	}
	age := r.now().UTC().Sub(created)
	aggregate := run.GetAggregate()
	evidence := internalrouting.EvalQualityEvidence{
		RunID:    run.GetRunId(),
		Fresh:    age >= 0 && age <= evalQualityFreshnessWindow,
		Degraded: run.GetDegraded(),
	}
	if aggregate != nil {
		evidence.MeanStrongTop1 = aggregate.GetMeanStrongTop1()
		evidence.MaxGibberishScore = aggregate.GetMaxGibberishScore()
		evidence.GibberishLeak = evidence.MeanStrongTop1 > 0 && evidence.MaxGibberishScore >= evidence.MeanStrongTop1
	}
	return evidence, nil
}

// autoRouteExternalEnabled reads the OT-P2-002 opt-in flag from the environment.
// DEFAULT FALSE: classifier-driven external auto-routing + fallback escalation
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
