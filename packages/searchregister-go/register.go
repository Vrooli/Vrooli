package searchregister

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"

	aisearch "github.com/vrooli/aisearch-go"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/retry"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
)

// hubScenarioID is the scenario that hosts the RegistryService.
const hubScenarioID = "search-hub"

// RegistryClient is the narrow seam over search-hub's RegistryService — the only
// RPC self-registration needs. The generated Connect client satisfies it;
// unit tests inject a fake so they exercise the retry/degrade logic without a
// live hub.
type RegistryClient interface {
	RegisterProvider(
		ctx context.Context,
		req *connect.Request[registryv1.RegisterProviderRequest],
	) (*connect.Response[registryv1.RegisterProviderResponse], error)
}

// BaseURLResolver resolves search-hub's live base URL. Production resolves it
// through api-core discovery (lifecycle-allocated ports are dynamic); tests pass
// a static httptest URL.
type BaseURLResolver func(ctx context.Context) (string, error)

// ClientFactory builds a RegistryClient for a resolved base URL.
type ClientFactory func(baseURL string) RegistryClient

// Config drives Register. Only ScenarioID and SearchFilePath are required; the
// seams (ResolveBaseURL, NewClient, Retry) default to the production wiring and
// are overridden in tests.
type Config struct {
	// ScenarioID is the owning scenario, used only for log attribution.
	ScenarioID string
	// SearchFilePath is the absolute path to the scenario's .vrooli/search.json.
	SearchFilePath string
	// Logger receives degrade/success lines; defaults to log.Default().
	Logger *log.Logger

	// ResolveBaseURL resolves search-hub's base URL (defaults to discovery).
	ResolveBaseURL BaseURLResolver
	// NewClient builds the RegistryService client (defaults to a Connect client).
	NewClient ClientFactory
	// Retry tunes the bounded retry; zero-value fields fall back to a short,
	// boot-friendly policy (search-hub is an OPTIONAL dependency — registration
	// must never block or fail the scenario's own startup).
	Retry retry.Config

	// OnControlToken, when set, is invoked after a successful registration with
	// the provider id and the non-empty control token search-hub minted (or
	// echoed) for it. The scenario caches the token IN MEMORY so its own request
	// handlers can validate token-gated override / reindex / config-write calls
	// against it (search-hub presents the same token when it calls those verbs).
	// Invoked at most once per provider per Register, only on success, never with
	// an empty token. Must be safe to call from the registration goroutine.
	OnControlToken func(providerID, controlToken string)
}

// Result reports the outcome of registering one provider.
type Result struct {
	ProviderID string
	// Created is true when a new leaf was inserted, false on an update upsert.
	Created bool
	// ControlToken is the token search-hub minted/echoed for this provider on a
	// successful registration (empty on failure). Mirrors the OnControlToken
	// callback for callers that consume the returned slice directly.
	ControlToken string
	// Err is non-nil when registration failed after exhausting retries. A failed
	// Result is logged and returned, never fatal: the scenario keeps serving.
	Err error
}

// Register reads the scenario's search.json, maps each provider to a registry
// descriptor, and upserts it to search-hub with bounded retry. It degrades
// gracefully — search-hub being down (or absent) yields logged, error-bearing
// Results, never a panic or a boot failure. Call it from a background goroutine
// at boot so a slow or unreachable hub never delays the scenario's own listener.
//
// A malformed/missing search.json IS returned as a single error Result (the file
// is the scenario's own committed SSOT; the scenario's primary boot path already
// fails loudly on it, so here we only attribute and degrade).
func Register(ctx context.Context, cfg Config) []Result {
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	file, err := aisearch.LoadSearchFile(cfg.SearchFilePath)
	if err != nil {
		logger.Printf("[%s] search self-registration skipped: %v", cfg.ScenarioID, err)
		return []Result{{Err: fmt.Errorf("load search.json: %w", err)}}
	}
	descriptors, err := Descriptors(file)
	if err != nil {
		logger.Printf("[%s] search self-registration skipped: %v", cfg.ScenarioID, err)
		return []Result{{Err: err}}
	}

	resolve := cfg.ResolveBaseURL
	if resolve == nil {
		resolve = defaultResolveBaseURL
	}
	newClient := cfg.NewClient
	if newClient == nil {
		newClient = defaultClientFactory
	}

	results := make([]Result, 0, len(descriptors))
	for _, d := range descriptors {
		results = append(results, registerOne(ctx, registerDeps{
			scenarioID:     cfg.ScenarioID,
			logger:         logger,
			resolve:        resolve,
			newClient:      newClient,
			retryCfg:       bootRetry(cfg.Retry),
			onControlToken: cfg.OnControlToken,
		}, d))
	}
	return results
}

type registerDeps struct {
	scenarioID     string
	logger         *log.Logger
	resolve        BaseURLResolver
	newClient      ClientFactory
	retryCfg       retry.Config
	onControlToken func(providerID, controlToken string)
}

// registerOne upserts a single descriptor. The base URL is re-resolved on every
// attempt (the hub may come up, or its port may change, mid-retry), so a
// transient "scenario not running" is recovered without restarting the caller.
func registerOne(ctx context.Context, deps registerDeps, d *registryv1.ProviderDescriptor) Result {
	res := Result{ProviderID: d.GetProviderId()}
	err := retry.Do(ctx, deps.retryCfg, func(int) error {
		baseURL, rerr := deps.resolve(ctx)
		if rerr != nil {
			return fmt.Errorf("resolve %s: %w", hubScenarioID, rerr)
		}
		resp, cerr := deps.newClient(baseURL).RegisterProvider(ctx, connect.NewRequest(
			&registryv1.RegisterProviderRequest{Descriptor_: d},
		))
		if cerr != nil {
			return fmt.Errorf("register %q: %w", d.GetProviderId(), cerr)
		}
		res.Created = resp.Msg.GetCreated()
		res.ControlToken = resp.Msg.GetControlToken()
		return nil
	})
	if err != nil {
		res.Err = err
		deps.logger.Printf(
			"[%s] search self-registration of %q degraded (search-hub optional, continuing): %v",
			deps.scenarioID, d.GetProviderId(), err,
		)
		return res
	}
	// Surface the minted/echoed control token so the scenario can cache it for
	// validating its own token-gated verbs. Only on success and only when the hub
	// actually returned one (a hub that predates token minting returns empty).
	if res.ControlToken != "" && deps.onControlToken != nil {
		deps.onControlToken(d.GetProviderId(), res.ControlToken)
	}
	verb := "updated"
	if res.Created {
		verb = "registered"
	}
	deps.logger.Printf("[%s] search provider %q %s with search-hub", deps.scenarioID, d.GetProviderId(), verb)
	return res
}

// bootRetry fills a boot-friendly retry policy over any caller overrides: a few
// quick attempts, then give up (search-hub is optional; the scenario re-registers
// on its next boot, and the registry upsert is idempotent).
func bootRetry(cfg retry.Config) retry.Config {
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = 1 * time.Second
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = 10 * time.Second
	}
	return cfg
}

func defaultResolveBaseURL(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, hubScenarioID)
}

func defaultClientFactory(baseURL string) RegistryClient {
	return registryconnect.NewRegistryServiceClient(
		&http.Client{Timeout: 15 * time.Second},
		baseURL,
	)
}
