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
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
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

// EvalClient is the narrow seam over search-hub's EvalService.RegisterSuite — the
// only eval RPC corpus self-registration needs. The generated Connect client
// satisfies it; unit tests inject a fake so they exercise the mirror/degrade logic
// without a live hub.
type EvalClient interface {
	RegisterSuite(
		ctx context.Context,
		req *connect.Request[evalv1.RegisterSuiteRequest],
	) (*connect.Response[evalv1.RegisterSuiteResponse], error)
}

// BaseURLResolver resolves search-hub's live base URL. Production resolves it
// through api-core discovery (lifecycle-allocated ports are dynamic); tests pass
// a static httptest URL.
type BaseURLResolver func(ctx context.Context) (string, error)

// ClientFactory builds a RegistryClient for a resolved base URL.
type ClientFactory func(baseURL string) RegistryClient

// EvalClientFactory builds an EvalClient for a resolved base URL.
type EvalClientFactory func(baseURL string) EvalClient

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
	// NewEvalClient builds the EvalService client used to mirror each provider's
	// tests corpus into search-hub's eval store (defaults to a Connect client).
	// Overridden in tests.
	NewEvalClient EvalClientFactory
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

	// ControlToken, when set, supplies the control token the scenario currently
	// holds for a provider so Register can ECHO it on re-registration — the
	// ownership proof that stops a different actor from hijacking an existing
	// provider_id (registry.RegisterProviderRequest.control_token). It is the read
	// side of OnControlToken's write side: a scenario wires its in-memory token
	// holder's getter here. Returning "" (the holder is empty — first boot, or a
	// hub that predates token minting) is fine; search-hub treats an empty
	// presented token as first-contact and simply echoes the stored one back. The
	// getter is consulted once per RegisterProvider attempt; it must be safe to
	// call from the registration goroutine.
	ControlToken func(providerID string) string
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
	// CorpusRegistered is true when the provider's tests corpus was mirrored into
	// search-hub's eval store at boot. False when the provider declares no corpus
	// or the eval upsert failed (see CorpusErr).
	CorpusRegistered bool
	// CorpusErr is non-nil when corpus self-registration failed after retries
	// (logged, never fatal — the corpus re-registers on the next boot).
	CorpusErr error
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
	newEvalClient := cfg.NewEvalClient
	if newEvalClient == nil {
		newEvalClient = defaultEvalClientFactory
	}

	deps := registerDeps{
		scenarioID:     cfg.ScenarioID,
		logger:         logger,
		resolve:        resolve,
		newClient:      newClient,
		newEvalClient:  newEvalClient,
		retryCfg:       bootRetry(cfg.Retry),
		onControlToken: cfg.OnControlToken,
		presentedToken: cfg.ControlToken,
	}

	// descriptors[i] is built from file.Providers[i] (Descriptors preserves order),
	// so the descriptor (transport shape, tests dropped) and the parsed provider
	// (which still carries the tests corpus) line up.
	results := make([]Result, 0, len(descriptors))
	for i, d := range descriptors {
		res := registerOne(ctx, deps, d)
		// Mirror the provider's corpus into the eval store so the store is a cache
		// of the file SSOT (corpusStoreMirrorsFile). Only after the descriptor
		// registered (the suite FKs the provider) and only when a corpus exists.
		if res.Err == nil && len(file.Providers[i].Tests.Cases) > 0 {
			registerCorpus(ctx, deps, file.Providers[i], &res)
		}
		results = append(results, res)
	}
	return results
}

type registerDeps struct {
	scenarioID     string
	logger         *log.Logger
	resolve        BaseURLResolver
	newClient      ClientFactory
	newEvalClient  EvalClientFactory
	retryCfg       retry.Config
	onControlToken func(providerID, controlToken string)
	presentedToken func(providerID string) string
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
		// Echo the cached control token (if the scenario holds one) so search-hub
		// can verify ownership on an update. Empty is the normal first-boot case;
		// the hub then treats it as first-contact and mints/echoes the token.
		var presented string
		if deps.presentedToken != nil {
			presented = deps.presentedToken(d.GetProviderId())
		}
		resp, cerr := deps.newClient(baseURL).RegisterProvider(ctx, connect.NewRequest(
			&registryv1.RegisterProviderRequest{Descriptor_: d, ControlToken: presented},
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

// INVARIANT: corpusStoreMirrorsFile
//
//	The eval store's suite for a provider is a MIRROR of that provider's tests
//	block in search.json (the file SSOT). registerCorpus re-registers it — an
//	idempotent upsert keyed by suite_id — at every boot, so a manual file edit
//	self-heals on the scenario's next restart and the store never becomes an
//	independent authority. The corpus is converted by SuiteToProto (lossless; see
//	corpusRoundTripsLossless). search-hub is OPTIONAL: a failed mirror is logged
//	and the scenario keeps serving (it re-registers next boot). Annotates res in
//	place rather than returning, so the provider's descriptor + corpus outcome
//	live on one Result.
func registerCorpus(ctx context.Context, deps registerDeps, p aisearch.ProviderConfig, res *Result) {
	suite := SuiteToProto(p.ProviderID, p.Tests)
	err := retry.Do(ctx, deps.retryCfg, func(int) error {
		baseURL, rerr := deps.resolve(ctx)
		if rerr != nil {
			return fmt.Errorf("resolve %s: %w", hubScenarioID, rerr)
		}
		if _, cerr := deps.newEvalClient(baseURL).RegisterSuite(ctx, connect.NewRequest(
			&evalv1.RegisterSuiteRequest{Suite: suite},
		)); cerr != nil {
			return fmt.Errorf("register corpus %q: %w", suite.GetSuiteId(), cerr)
		}
		return nil
	})
	if err != nil {
		res.CorpusErr = err
		deps.logger.Printf(
			"[%s] corpus self-registration of %q degraded (search-hub optional, continuing): %v",
			deps.scenarioID, suite.GetSuiteId(), err,
		)
		return
	}
	res.CorpusRegistered = true
	deps.logger.Printf("[%s] search corpus %q (%d cases) mirrored to search-hub",
		deps.scenarioID, suite.GetSuiteId(), len(suite.GetCases()))
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

func defaultEvalClientFactory(baseURL string) EvalClient {
	return evalconnect.NewEvalServiceClient(
		&http.Client{Timeout: 15 * time.Second},
		baseURL,
	)
}
