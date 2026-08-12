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
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"search-hub/internal/clock"
	"search-hub/internal/evalsched"
	"search-hub/internal/httpc"
	"search-hub/internal/module"
	internalrouting "search-hub/internal/routing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	"search-hub/internal/control"
	internaleval "search-hub/internal/eval"
	internalregistry "search-hub/internal/registry"
	"search-hub/internal/sweep"
)

// Module returns the eval domain's contribution to the API: the generated
// EvalService Connect handler backed by the SQLite eval store and a Runner that
// reaches each provider through its registry descriptor (resolving the live base
// URL at call-time and reusing the shared providers.MapResults adapter).
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	store := internaleval.NewSQLiteStore(db, clk)
	// The registry store doubles as the runner's provider resolver (its Get
	// returns the descriptor whose endpoint the runner reuses) and the sweep's
	// provider reader (Get + Token, to present the control token on the secured
	// reindex/config-write verbs).
	resolver := internalregistry.NewSQLiteStore(db, clk)
	client := newHTTPProviderClient(newScenarioResolver(), httpc.NewDefault())
	runner := internaleval.NewRunner(resolver, client, clk, uuid.NewString)
	federated := internaleval.NewFederatedRunner(resolver, newRoutingQueryClient(), clk, uuid.NewString)
	validator := internaleval.NewValidator(resolver, client)

	// One registry-side control client drives BOTH the sweep's index-time tier +
	// tuning write-back AND the eval handler's corpus write-back (generate --apply).
	controlClient := control.NewClient(control.NewDiscoveryResolver())

	// The sweep orchestrator drives the SAME runner (with per-arm overrides) and
	// the control client. Its arm runner adapts the runner's pure RunWith into the
	// persist-each-arm seam.
	sweeper := sweep.New(sweep.Deps{
		Suites:    store,
		Providers: resolver,
		Runner:    armRunner{runner: runner, store: store},
		Control:   controlClient,
		// The registry store doubles as the tuning cache the write-back refreshes.
		Cache: resolver,
		Clock: clk,
	}, sweep.Options{})

	// Evaluation freshness is maintained by the service lifecycle, not by an
	// operator remembering to run a command. The scheduler discovers suites
	// from the same store on every cycle and runs both evaluation tiers through
	// the already-wired production seams.
	schedulerOpts := evalsched.OptionsFromEnv(logger)
	schedulerOpts.Validation = validator
	scheduler := evalsched.New(clk, store, runner, federated, store, schedulerOpts)
	go scheduler.Run(context.Background())

	connectPath, connectHandler := evalconnect.NewEvalServiceHandler(NewConnectHandler(Deps{
		Store:     store,
		Registry:  resolver,
		Providers: resolver,
		Runner:    runner,
		Federated: federated,
		Validator: validator,
		Sweeper:   sweeper,
		// The corpus generator samples the provider through the SAME client the
		// runner/sweep use (its index is reached only via its search endpoint) and
		// inverts items with the local Ollama gateway.
		Generator: newLiveCorpusGenerator(client),
		// generate --apply writes the grown corpus back to the provider's
		// search.json via the control client (authorized with the registry-minted
		// token), then re-registers it into the store.
		Control: controlClient,
		Tokens:  resolver,
		Logger:  logger,
	}))
	return module.Module{
		Name: "eval",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

type routingQueryClient struct {
	resolver *scenarioResolver
	http     connect.HTTPClient
}

func newRoutingQueryClient() internaleval.QueryClient {
	return &routingQueryClient{resolver: newScenarioResolver(), http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *routingQueryClient) Query(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	ctx = internalrouting.WithBackgroundEvaluation(ctx)
	base, err := c.resolver.ResolveScenarioURL(ctx, "search-hub")
	if err != nil {
		return nil, err
	}
	resp, err := routingconnect.NewRoutingServiceClient(c.http, base).Query(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// Schema re-exports internaleval.Schema so the modules registry collects both
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaleval.Schema() }

// armRunner adapts the pure eval Runner (which builds but does not persist a
// run) into the sweep's ArmRunner seam (run an arm THEN store it — the sweep's
// contract is one immutable, tagged run per arm). It threads the per-arm
// query-time overrides + control token through RunWith.
type armRunner struct {
	runner *internaleval.Runner
	store  internaleval.Store
}

func (a armRunner) Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, overrides *aisearch.SearchOverrides, controlToken string, limit int32) (*evalv1.EvalRun, error) {
	run, err := a.runner.RunWith(ctx, suite, tag, limit, internaleval.SearchCallOptions{Overrides: overrides, ControlToken: controlToken})
	if err != nil {
		return nil, err
	}
	if err := a.store.AppendRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

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
