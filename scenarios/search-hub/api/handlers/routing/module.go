// Package routing is the search-hub routing domain's API surface: the generated
// RoutingService Connect-RPC handler that fans a query out across registered
// providers. It is the router core (Phase 4) sitting beside the registry
// domain.
//
// This package is the wiring edge: it composes the pure internal/routing.Router
// with the concrete cross-scenario URL resolver (api-core/discovery), the timed
// outbound HTTP client (internal/httpc), the local-Ollama classifier (Phase 5
// automatic routing), and the local-Ollama reranker (Phase 6 unified ranking).
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
	"search-hub/internal/httpc"
	"search-hub/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	internalregistry "search-hub/internal/registry"
	internalrouting "search-hub/internal/routing"
)

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
	store := internalregistry.NewSQLiteStore(db, clk)
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:        store,
		Resolver:      newScenarioResolver(),
		Doer:          httpc.NewDefault(),
		Classifier:    internalrouting.NewOllamaClassifier(),
		Reranker:      internalrouting.NewOllamaReranker(),
		Recorder:      recorder,
		Logger:        logger,
		QueryTimeout:  durationEnv(logger, "SEARCH_HUB_QUERY_TIMEOUT", 25*time.Second, time.Second, 29*time.Second),
		RerankTimeout: durationEnv(logger, "SEARCH_HUB_RERANK_TIMEOUT", 5*time.Second, 100*time.Millisecond, 20*time.Second),
		RerankBreaker: internalrouting.RerankBreakerConfig{
			FailureThreshold: intEnv(logger, "SEARCH_HUB_RERANK_BREAKER_FAILURES", 3, 1, 20),
			Cooldown:         durationEnv(logger, "SEARCH_HUB_RERANK_BREAKER_COOLDOWN", 60*time.Second, time.Second, 10*time.Minute),
		},
		AutoRouteExternal: autoRouteExternalEnabled(),
	})
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
// URLResolver seam. The discovery resolver shells out to `vrooli scenario port`
// per call (no caching) so a restarted, re-ported provider is always reached at
// its current address — this is the backend cross-scenario resolution the
// project mandates over client-computed URLs.
type scenarioResolver struct {
	r *discovery.Resolver
}

func newScenarioResolver() *scenarioResolver {
	return &scenarioResolver{r: discovery.NewResolver(discovery.ResolverConfig{})}
}

func (s *scenarioResolver) ResolveScenarioURL(ctx context.Context, scenarioID string) (string, error) {
	return s.r.ResolveScenarioURLDefault(ctx, scenarioID)
}
