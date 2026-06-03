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
		Lister:     store,
		Resolver:   newScenarioResolver(),
		Doer:       httpc.NewDefault(),
		Classifier: internalrouting.NewOllamaClassifier(),
		Reranker:   internalrouting.NewOllamaReranker(),
		Recorder:   recorder,
		Logger:     logger,
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
