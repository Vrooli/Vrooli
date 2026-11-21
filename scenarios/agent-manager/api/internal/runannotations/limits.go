package runannotations

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory/inventory_v1connect"
)

// ModelLimits are what a decision model declares it can take: how much text and
// how many questions. Zero means undeclared. The engine sizes digests and
// question batches from them instead of finding the bound by refusal.
type ModelLimits struct {
	ContextTokens int `json:"context_tokens,omitempty"`
	MaxOptions    int `json:"max_options,omitempty"`
	ScoreLevels   int `json:"score_levels,omitempty"`
	MaxQuestions  int `json:"max_questions,omitempty"`
}

// LimitsSource supplies the declared limits of the resource role that serves a
// model. Provider may be empty to match the role under any provider. It reports
// false when the role declares none, in which case the pack's values stand.
type LimitsSource interface {
	ResourceLimits(ctx context.Context, provider, resourceRole string) (ModelLimits, bool)
}

const (
	// limitsTTL is how long the gateway's role inventory is trusted. Limits move
	// only when a model policy changes, so a few minutes is both fresh and cheap.
	limitsTTL = 5 * time.Minute
	// limitsFailureTTL keeps a failed read from being retried on every call.
	limitsFailureTTL = 30 * time.Second
	limitsTimeout    = 10 * time.Second
)

// GatewayLimitsSource reads declared limits from ai-gateway's role inventory. It
// resolves the gateway lazily and degrades to "no limits" when it is absent: an
// annotation pass must never fail because limits could not be read.
type GatewayLimitsSource struct {
	ResolveURL func(context.Context) (string, error)
	HTTPClient connect.HTTPClient
	Now        func() time.Time

	mu      sync.Mutex
	client  inventoryconnect.InventoryServiceClient
	limits  map[string]ModelLimits
	fetched time.Time
	failed  bool
}

// NewGatewayLimitsSource returns the default discovery-backed source.
func NewGatewayLimitsSource() *GatewayLimitsSource {
	return &GatewayLimitsSource{
		HTTPClient: &http.Client{Timeout: limitsTimeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
	}
}

// ResourceLimits implements LimitsSource.
func (g *GatewayLimitsSource) ResourceLimits(ctx context.Context, provider, resourceRole string) (ModelLimits, bool) {
	if g == nil || strings.TrimSpace(resourceRole) == "" {
		return ModelLimits{}, false
	}
	table := g.table(ctx)
	if limits, ok := table[limitsKey(provider, resourceRole)]; ok {
		return limits, true
	}
	limits, ok := table[limitsKey("", resourceRole)]
	return limits, ok
}

func limitsKey(provider, role string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + "\x00" + strings.TrimSpace(role)
}

// table returns the cached role limits, refreshing them when stale. Roles that
// declare no limits are absent from it.
func (g *GatewayLimitsSource) table(ctx context.Context) map[string]ModelLimits {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}
	ttl := limitsTTL
	if g.failed {
		ttl = limitsFailureTTL
	}
	if g.limits != nil && now().Sub(g.fetched) < ttl {
		return g.limits
	}
	table, err := g.fetch(ctx)
	g.fetched = now()
	g.failed = err != nil
	if err == nil || g.limits == nil {
		g.limits = table
	}
	return g.limits
}

func (g *GatewayLimitsSource) fetch(ctx context.Context) (map[string]ModelLimits, error) {
	table := map[string]ModelLimits{}
	if g.client == nil {
		if g.ResolveURL == nil {
			return table, errNoGateway
		}
		base, err := g.ResolveURL(ctx)
		if err != nil {
			return table, err
		}
		base = strings.TrimRight(strings.TrimSpace(base), "/")
		if base == "" {
			return table, errNoGateway
		}
		httpClient := g.HTTPClient
		if httpClient == nil {
			httpClient = &http.Client{Timeout: limitsTimeout}
		}
		g.client = inventoryconnect.NewInventoryServiceClient(httpClient, base)
	}
	callCtx, cancel := context.WithTimeout(ctx, limitsTimeout)
	defer cancel()
	response, err := g.client.ListProviderRoles(callCtx, connect.NewRequest(&inventoryv1.ListProviderRolesRequest{}))
	if err != nil {
		return table, err
	}
	for _, role := range response.Msg.GetRoles() {
		limits := role.GetLimits()
		if limits == nil {
			continue
		}
		declared := ModelLimits{
			ContextTokens: int(limits.GetContextTokens()), MaxOptions: int(limits.GetMaxOptions()),
			ScoreLevels: int(limits.GetScoreLevels()), MaxQuestions: int(limits.GetMaxQuestions()),
		}
		if declared == (ModelLimits{}) {
			continue
		}
		table[limitsKey(role.GetProvider(), role.GetRole())] = declared
		// A role is also reachable by name alone when a pinned pass does not
		// know which provider serves it; the first declaration wins.
		if _, exists := table[limitsKey("", role.GetRole())]; !exists {
			table[limitsKey("", role.GetRole())] = declared
		}
	}
	return table, nil
}

var errNoGateway = &limitsError{"ai-gateway URL is empty"}

type limitsError struct{ message string }

func (e *limitsError) Error() string { return e.message }
