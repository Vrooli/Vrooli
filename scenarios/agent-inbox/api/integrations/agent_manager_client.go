package integrations

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/resilience"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

var protoMarshalOpts = protojson.MarshalOptions{UseProtoNames: true}

// AgentManagerClient provides generated Connect access to agent-manager.
// It serves chat actions, run reads, and restart reconciliation.
//
// Peer addressing is delegated to api-core discovery, which briefly coalesces
// lookups while keeping restart and port changes visible within a bounded time.
type AgentManagerClient struct {
	baseURL    string // Explicit test/injected base; production uses resolver.
	resolver   *discovery.Resolver
	httpClient *http.Client
	timeout    time.Duration
	retryCfg   resilience.RetryConfig
	cb         *resilience.CircuitBreaker
}

// NewAgentManagerClient creates a new agent-manager client. Peer resolution is
// lazy, so an optional agent-manager may be unavailable during startup.
func NewAgentManagerClient() (*AgentManagerClient, error) {
	cfg := config.Default()
	return NewAgentManagerClientWithConfig(cfg.Integration.AgentManagerTimeout)
}

// NewAgentManagerClientWithConfig creates a new agent-manager client with explicit timeout.
// This enables testing and custom configuration injection.
//
// The initial URL resolution is best-effort: if agent-manager is not yet
// available, the client is still created and will attempt re-resolution
// on the first request.
func NewAgentManagerClientWithConfig(timeout time.Duration) (*AgentManagerClient, error) {
	cfg := config.Default()
	retryCfg := resilience.RetryConfig{
		MaxAttempts: cfg.Resilience.RetryAttempts,
		BaseDelay:   cfg.Resilience.RetryBaseDelay,
		MaxDelay:    cfg.Resilience.RetryMaxDelay,
		Jitter:      0.1,
	}
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		FailureThreshold: cfg.Resilience.CircuitBreakerThreshold,
		Cooldown:         cfg.Resilience.CircuitBreakerCooldown,
	})

	return &AgentManagerClient{
		resolver: discovery.NewResolver(discovery.ResolverConfig{
			StaticBaseURL: os.Getenv("AGENT_MANAGER_API_URL"),
			CacheTTL:      500 * time.Millisecond,
		}),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:  timeout,
		retryCfg: retryCfg,
		cb:       cb,
	}, nil
}

// getBaseURL delegates peer resolution and short-lived caching to api-core.
func (c *AgentManagerClient) getBaseURL(ctx context.Context) (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	if c.resolver == nil {
		return "", fmt.Errorf("agent-manager resolver is not configured")
	}
	return c.resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
}

func (c *AgentManagerClient) connectClient(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	baseURL, err := c.getBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	return apiconnect.NewAgentManagerServiceClient(c.httpClient, strings.TrimRight(baseURL, "/")), nil
}

// callAgentManager retries bounded read-only Connect failures. Mutating calls
// opt out until their contract provides an idempotency key.
func callAgentManager[T any](ctx context.Context, c *AgentManagerClient, retryRead bool, call func(context.Context, apiconnect.AgentManagerServiceClient) (*T, error)) (*T, error) {
	var lastErr error
	var result *T
	retryCfg := c.retryCfg
	// RetryConfig.MaxAttempts counts the initial call. Treat zero as the
	// supported no-retry setting while still making that call once.
	if retryCfg.MaxAttempts < 1 {
		retryCfg.MaxAttempts = 1
	}
	if !retryRead {
		retryCfg.MaxAttempts = 1
	}
	err := resilience.Retry(ctx, retryCfg, func(attemptCtx context.Context, _ int) error {
		client, err := c.connectClient(attemptCtx)
		if err != nil {
			lastErr = err
			return err
		}
		var callErr error
		invoke := func(invokeCtx context.Context) error {
			result, callErr = call(invokeCtx, client)
			// The old breaker observed the HTTP transport, not application-level
			// 4xx responses. Keep that distinction with Connect error codes.
			code := connect.CodeOf(callErr)
			if callErr != nil && code != connect.CodeUnavailable && code != connect.CodeDeadlineExceeded {
				return nil
			}
			return callErr
		}
		if c.cb != nil {
			err = c.cb.Execute(ctx, invoke)
		} else {
			err = invoke(ctx)
		}
		if err == nil {
			err = callErr
		}
		if err != nil {
			lastErr = err
			if code := connect.CodeOf(err); code != connect.CodeUnavailable && code != connect.CodeDeadlineExceeded {
				return resilience.Permanent(err)
			}
		}
		return err
	})
	if err != nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, err
	}
	return result, nil
}
