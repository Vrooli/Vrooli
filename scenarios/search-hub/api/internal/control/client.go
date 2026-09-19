// Package control is search-hub's registry-side client for a provider's SHARED,
// token-gated control plane (search-hub.v1.control.SearchControlService). It is
// the counterpart to the registry/eval read path: where the eval client calls a
// provider's public search endpoint, this client calls a provider's MUTATING
// reindex + config-write verbs — the ones the Phase-6 sweep drives to run
// index-time experiments and persist a winning tuning.
//
// Every call:
//   - resolves the provider's live base URL at call-time via discovery (never a
//     client-computed URL), from the scenario_id on the descriptor's
//     reindex_endpoint / config_endpoint (a provider that declares neither is not
//     sweep-tunable — ErrNoControlPlane);
//   - presents the provider's control token (search-hub minted it at registration
//     and stores it; the caller looks it up via registry Store.Token);
//   - is bounded-retried on transient transport failures only — a permission /
//     argument / not-found error is returned immediately (retrying a rejected
//     token never changes the verdict).
package control

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/retry"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// ErrNoControlPlane is returned when a descriptor declares no usable control
// endpoint for the requested verb (the provider is routable + evaluable but not
// sweep-tunable).
var ErrNoControlPlane = errors.New("provider declares no control endpoint")

// URLResolver resolves a scenario's live base URL at call-time. The production
// implementation shells out to discovery per call (no caching) so a restarted,
// re-ported provider is always reached at its current address — the same
// backend cross-scenario resolution the routing + eval domains use.
type URLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioID string) (baseURL string, err error)
}

// ServiceClientFactory builds the generated Connect client for a base URL. The
// seam lets tests substitute a fake transport without an HTTP server.
type ServiceClientFactory func(baseURL string) controlconnect.SearchControlServiceClient

// Client calls a provider's SearchControlService.
type Client struct {
	resolver  URLResolver
	newClient ServiceClientFactory
	retry     retry.Config
}

// Option configures a Client.
type Option func(*Client)

// WithClientFactory overrides how the generated Connect client is built (tests).
func WithClientFactory(f ServiceClientFactory) Option {
	return func(c *Client) {
		if f != nil {
			c.newClient = f
		}
	}
}

// WithRetry overrides the bounded-retry policy (tests use a no-sleep config).
func WithRetry(cfg retry.Config) Option {
	return func(c *Client) { c.retry = cfg }
}

// NewClient builds a control client over the given resolver. The default
// transport is a 30s-timeout HTTP client and a short bounded-retry policy.
func NewClient(resolver URLResolver, opts ...Option) *Client {
	c := &Client{
		resolver:  resolver,
		newClient: defaultFactory,
		retry: retry.Config{
			MaxAttempts:    4,
			BaseDelay:      200 * time.Millisecond,
			MaxDelay:       2 * time.Second,
			JitterFraction: 0.2,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func defaultFactory(baseURL string) controlconnect.SearchControlServiceClient {
	return controlconnect.NewSearchControlServiceClient(&http.Client{Timeout: 30 * time.Second}, baseURL)
}

// Reindex starts an async reconcile on the provider behind d's reindex_endpoint.
func (c *Client) Reindex(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken, scope string, dryRun bool) (*controlv1.ReindexResponse, error) {
	return c.ReindexRequest(ctx, d, controlToken, &controlv1.ReindexRequest{Scope: scope, DryRun: dryRun})
}

// ReindexRequest sends an additive provider-owned action through the same
// declared reindex endpoint and token gate as ordinary reindexing. The registry
// service uses this server-side so operator clients never receive provider
// control tokens.
func (c *Client) ReindexRequest(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken string, request *controlv1.ReindexRequest) (*controlv1.ReindexResponse, error) {
	cl, err := c.clientFor(ctx, d.GetReindexEndpoint())
	if err != nil {
		return nil, err
	}
	var out *controlv1.ReindexResponse
	err = c.call(ctx, func() error {
		request.ControlToken = controlToken
		resp, err := cl.Reindex(ctx, connect.NewRequest(request))
		if err != nil {
			return err
		}
		out = resp.Msg
		return nil
	})
	return out, err
}

// ReindexStatus polls a reindex job on the provider behind d's reindex_endpoint.
func (c *Client) ReindexStatus(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken, jobID string) (*controlv1.ReindexStatusResponse, error) {
	cl, err := c.clientFor(ctx, d.GetReindexEndpoint())
	if err != nil {
		return nil, err
	}
	var out *controlv1.ReindexStatusResponse
	err = c.call(ctx, func() error {
		resp, err := cl.ReindexStatus(ctx, connect.NewRequest(&controlv1.ReindexStatusRequest{
			JobId: jobID, ControlToken: controlToken,
		}))
		if err != nil {
			return err
		}
		out = resp.Msg
		return nil
	})
	return out, err
}

// ReindexCancel cooperatively cancels a reindex job (idempotent).
func (c *Client) ReindexCancel(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken, jobID string) (*controlv1.ReindexCancelResponse, error) {
	cl, err := c.clientFor(ctx, d.GetReindexEndpoint())
	if err != nil {
		return nil, err
	}
	var out *controlv1.ReindexCancelResponse
	err = c.call(ctx, func() error {
		resp, err := cl.ReindexCancel(ctx, connect.NewRequest(&controlv1.ReindexCancelRequest{
			JobId: jobID, ControlToken: controlToken,
		}))
		if err != nil {
			return err
		}
		out = resp.Msg
		return nil
	})
	return out, err
}

// WriteConfig persists a new tuning block on the provider behind d's
// config_endpoint, reindexing automatically when an index-time factor changed.
func (c *Client) WriteConfig(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken string, tuning *registryv1.Tuning, dryRun bool) (*controlv1.WriteConfigResponse, error) {
	cl, err := c.clientFor(ctx, d.GetConfigEndpoint())
	if err != nil {
		return nil, err
	}
	var out *controlv1.WriteConfigResponse
	err = c.call(ctx, func() error {
		resp, err := cl.WriteConfig(ctx, connect.NewRequest(&controlv1.WriteConfigRequest{
			ProviderId: d.GetProviderId(), Tuning: tuning, ControlToken: controlToken, DryRun: dryRun,
		}))
		if err != nil {
			return err
		}
		out = resp.Msg
		return nil
	})
	return out, err
}

// WriteCorpus persists a new evaluation corpus on the provider behind d's
// config_endpoint — the SAME control plane WriteConfig uses (the corpus + tuning
// write-backs are sibling verbs on one SearchControlService). The corpus rides as
// the eval store/wire shape; the provider converts it to its file shape and
// rewrites only the tests block. A corpus write never triggers a reindex.
func (c *Client) WriteCorpus(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken string, corpus *evalv1.EvalSuite, dryRun bool) (*controlv1.WriteCorpusResponse, error) {
	cl, err := c.clientFor(ctx, d.GetConfigEndpoint())
	if err != nil {
		return nil, err
	}
	var out *controlv1.WriteCorpusResponse
	err = c.call(ctx, func() error {
		resp, err := cl.WriteCorpus(ctx, connect.NewRequest(&controlv1.WriteCorpusRequest{
			ProviderId: d.GetProviderId(), Corpus: corpus, ControlToken: controlToken, DryRun: dryRun,
		}))
		if err != nil {
			return err
		}
		out = resp.Msg
		return nil
	})
	return out, err
}

// clientFor resolves the live base URL for endpoint and builds a typed client.
func (c *Client) clientFor(ctx context.Context, endpoint *registryv1.Endpoint) (controlconnect.SearchControlServiceClient, error) {
	hj := endpoint.GetHttpJson()
	if hj == nil || hj.GetScenarioId() == "" {
		return nil, ErrNoControlPlane
	}
	base, err := c.resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", hj.GetScenarioId(), err)
	}
	return c.newClient(base), nil
}

// call runs op under bounded retry, retrying ONLY transient transport failures.
// A permanent error (permission/argument/not-found/unimplemented) is returned on
// the first attempt — retrying a rejected token or a bad request never changes
// the outcome.
func (c *Client) call(ctx context.Context, op func() error) error {
	var permErr error
	rerr := retry.Do(ctx, c.retry, func(int) error {
		if err := op(); err != nil {
			if isRetryable(err) {
				return err
			}
			permErr = err
			return nil // stop retrying; surface permErr below
		}
		return nil
	})
	if permErr != nil {
		return permErr
	}
	return rerr
}

// isRetryable reports whether a control-call error is a transient transport
// failure worth retrying. Only Unavailable and DeadlineExceeded qualify; every
// other connect code (and a nil-code non-connect error) is treated as permanent.
func isRetryable(err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded:
		return true
	default:
		return false
	}
}

// DiscoveryResolver is the production URLResolver: it resolves a scenario's base
// URL through api-core/discovery at call-time. Phase-6 wiring constructs one and
// passes it to NewClient.
type DiscoveryResolver struct{ r *discovery.Resolver }

// NewDiscoveryResolver builds the production resolver.
func NewDiscoveryResolver() *DiscoveryResolver {
	return &DiscoveryResolver{r: discovery.NewResolver(discovery.ResolverConfig{})}
}

func (d *DiscoveryResolver) ResolveScenarioURL(ctx context.Context, scenarioID string) (string, error) {
	return d.r.ResolveScenarioURLDefault(ctx, scenarioID)
}
