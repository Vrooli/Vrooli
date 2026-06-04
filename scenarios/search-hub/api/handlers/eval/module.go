// Package eval is the search-hub eval domain's API surface: the generated
// EvalService Connect-RPC handler that registers provider-owned golden suites,
// runs them against the provider's registered endpoint, and stores immutable,
// tagged runs for the history/trend/compare views.
//
// This package is the wiring edge: it composes the pure internal/eval store +
// runner with the concrete cross-scenario URL resolver (api-core/discovery), the
// timed outbound HTTP client (internal/httpc), and the registry store (the
// runner's provider resolver). internal/eval stays dependency-light (seams only)
// so it is unit-testable without the network.
package eval

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"search-hub/internal/clock"
	"search-hub/internal/httpc"
	"search-hub/internal/module"

	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"

	internaleval "search-hub/internal/eval"
	internalregistry "search-hub/internal/registry"
)

// Module returns the eval domain's contribution to the API: the generated
// EvalService Connect handler backed by the SQLite eval store and a Runner that
// reaches each provider through its registry descriptor (resolving the live base
// URL at call-time and reusing the shared providers.MapResults adapter).
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	store := internaleval.NewSQLiteStore(db, clk)
	// The registry store doubles as the runner's provider resolver (its Get
	// returns the descriptor whose endpoint the runner reuses).
	resolver := internalregistry.NewSQLiteStore(db, clk)
	client := newHTTPProviderClient(newScenarioResolver(), httpc.NewDefault())
	runner := internaleval.NewRunner(resolver, client, clk, uuid.NewString)

	connectPath, connectHandler := evalconnect.NewEvalServiceHandler(NewConnectHandler(Deps{
		Store:  store,
		Runner: runner,
		Logger: logger,
	}))
	return module.Module{
		Name: "eval",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internaleval.Schema so the modules registry collects both
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaleval.Schema() }

// scenarioResolver adapts api-core/discovery's Resolver to the eval client's
// URLResolver seam — the same backend cross-scenario resolution the routing
// domain uses (shells out to `vrooli scenario port` per call, no caching, so a
// restarted provider is always reached at its current address).
type scenarioResolver struct {
	r *discovery.Resolver
}

func newScenarioResolver() *scenarioResolver {
	return &scenarioResolver{r: discovery.NewResolver(discovery.ResolverConfig{})}
}

func (s *scenarioResolver) ResolveScenarioURL(ctx context.Context, scenarioID string) (string, error) {
	return s.r.ResolveScenarioURLDefault(ctx, scenarioID)
}
