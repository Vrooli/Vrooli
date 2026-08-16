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
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"search-hub/internal/evalsched"
	"search-hub/internal/httpc"
	"search-hub/internal/module"
	internalrouting "search-hub/internal/routing"

	"github.com/vrooli/api-core/schedule"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	"search-hub/internal/control"
	internaleval "search-hub/internal/eval"
	internalregistry "search-hub/internal/registry"
	"search-hub/internal/sweep"
)

// RoutabilityReader supplies the router's live automatic-eligibility view for
// the second, end-to-end denominator in strategy comparisons.
type RoutabilityReader interface {
	Status(context.Context) (*routingv1.StatusResponse, error)
}

// Module returns the eval domain's contribution to the API: the generated
// EvalService Connect handler backed by the SQLite eval store and a Runner that
// reaches each provider through its registry descriptor (resolving the live base
// URL at call-time and reusing the shared providers.MapResults adapter).
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithRoutability(db, clk, logger, nil)
}

// ModuleWithRoutability wires the optional shared router status seam. Keeping
// Module's original signature preserves lightweight consumers and tests.
func ModuleWithRoutability(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, routability RoutabilityReader) module.Module {
	store := internaleval.NewSQLiteStore(db, clk)
	// The registry store doubles as the runner's provider resolver (its Get
	// returns the descriptor whose endpoint the runner reuses) and the sweep's
	// provider reader (Get + Token, to present the control token on the secured
	// reindex/config-write verbs).
	resolver := internalregistry.NewSQLiteStore(db, clk)
	activeStrategy, _, strategyErr := internalrouting.LoadActiveStrategy()
	if strategyErr != nil {
		panic(strategyErr)
	}
	routingClient := newRoutingQueryClient()
	client := &providerClientWithSubstrate{
		ProviderClient: newHTTPProviderClient(newScenarioResolver(), httpc.NewDefault()),
		substrate:      routingClient,
	}
	runner := internaleval.NewRunner(resolver, client, clk, uuid.NewString)
	federated := internaleval.NewFederatedRunnerWithRoutability(resolver, routingClient, clk, uuid.NewString, routability)
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
		Store:          store,
		Registry:       resolver,
		Providers:      resolver,
		Runner:         runner,
		Federated:      federated,
		Validator:      validator,
		Sweeper:        sweeper,
		ActiveStrategy: activeStrategy.Name,
		Routability:    routability,
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

func newRoutingQueryClient() *routingQueryClient {
	return &routingQueryClient{resolver: newScenarioResolver(), http: &http.Client{Timeout: 30 * time.Second}}
}

// providerClientWithSubstrate keeps provider-direct runs honest about the
// shared substrate. Provider status remains authoritative when it exposes a
// field; routing status fills the otherwise-unknown reranker/embed/index
// fields so every stored run explains the configuration it was evaluated
// under, even for providers whose status endpoint is intentionally minimal.
type providerClientWithSubstrate struct {
	internaleval.ProviderClient
	substrate *routingQueryClient
}

func (c *providerClientWithSubstrate) Snapshot(ctx context.Context, descriptor *registryv1.ProviderDescriptor) *evalv1.ConfigSnapshot {
	snapshot := c.ProviderClient.Snapshot(ctx, descriptor)
	if snapshot == nil {
		snapshot = &evalv1.ConfigSnapshot{}
	}
	if strings.TrimSpace(snapshot.SelectorLeg) == "" {
		snapshot.SelectorLeg = "provider_direct"
	}
	live := c.substrate.Snapshot(ctx)
	if snapshot.RerankerLeg == "" || snapshot.RerankerLeg == "unknown" || snapshot.RerankerLeg == "none" {
		snapshot.RerankerLeg = live.GetRerankerLeg()
		snapshot.RerankEnabled = live.GetRerankEnabled()
	}
	if snapshot.EmbedModel == "" || snapshot.EmbedModel == "unknown" {
		snapshot.EmbedModel = live.GetEmbedModel()
	}
	if snapshot.IndexedCount <= 0 && live.GetIndexedCount() > 0 {
		snapshot.IndexedCount = live.GetIndexedCount()
	}
	if strings.TrimSpace(snapshot.RerankerLeg) == "" {
		snapshot.RerankerLeg = "unknown"
	}
	if strings.TrimSpace(snapshot.EmbedModel) == "" {
		snapshot.EmbedModel = "unknown"
	}
	return snapshot
}

func (c *routingQueryClient) Query(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	// FederatedRunner marks router.routing contexts with WithRoutingEvaluation
	// (and provider-owned suites with a provider-scoped background marker).
	// Preserve that marker: replacing it here would make strategy overrides look
	// like public queries and would erase the evaluation-only authorization.
	base, err := c.resolver.ResolveScenarioURL(ctx, "search-hub")
	if err != nil {
		return nil, err
	}
	runQuery := routingconnect.NewRoutingServiceClient(c.http, base).Query
	rpcRequest := connect.NewRequest(req)
	// The federated runner and router are separate HTTP handlers, so Go context
	// values cannot cross this Connect hop. Carry the evaluation authorization as
	// an internal metadata marker; the routing handler turns it back into its
	// request context before invoking the router.
	rpcRequest.Header().Set("X-Vrooli-Search-Hub-Evaluation", "1")
	rpcResponse, err := runQuery(ctx, rpcRequest)
	if err != nil {
		return nil, err
	}
	return rpcResponse.Msg, nil
}

// Snapshot reads the same typed routing status surface used by operators. It
// records the active reranker and the aggregate indexed population, while the
// provider health rows expose each provider's declared embedding model. A
// mixed model fleet is recorded explicitly instead of pretending one model
// governed the run.
func (c *routingQueryClient) Snapshot(ctx context.Context) *evalv1.ConfigSnapshot {
	snapshot := &evalv1.ConfigSnapshot{
		SelectorLeg: "unknown",
		RerankerLeg: "unknown",
		EmbedModel:  "unknown",
	}
	base, err := c.resolver.ResolveScenarioURL(ctx, "search-hub")
	if err != nil {
		return snapshot
	}
	status, err := routingconnect.NewRoutingServiceClient(c.http, base).Status(ctx, connect.NewRequest(&routingv1.StatusRequest{}))
	if err != nil || status == nil || status.Msg == nil {
		return snapshot
	}
	snapshot.RerankerLeg = strings.TrimSpace(status.Msg.GetRerankerLeg())
	if snapshot.RerankerLeg == "" {
		snapshot.RerankerLeg = "none"
	}
	snapshot.RerankEnabled = status.Msg.GetRerankerAvailable()
	models := make(map[string]struct{})
	var indexed int64
	for _, provider := range status.Msg.GetProviders() {
		indexed += provider.GetPointCount()
		if model := strings.TrimSpace(provider.GetEmbeddingModel()); model != "" {
			models[model] = struct{}{}
		}
	}
	if indexed > int64(1<<31-1) {
		indexed = int64(1<<31 - 1)
	}
	snapshot.IndexedCount = int32(indexed)
	if len(models) > 0 {
		values := make([]string, 0, len(models))
		for model := range models {
			values = append(values, model)
		}
		sort.Strings(values)
		if len(values) == 1 {
			snapshot.EmbedModel = values[0]
		} else {
			snapshot.EmbedModel = "mixed:" + strings.Join(values, ",")
		}
	}
	return snapshot
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
