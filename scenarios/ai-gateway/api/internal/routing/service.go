package routing

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/gateway"
	"ai-gateway/internal/providers"
)

type Service struct {
	validator     *gateway.Service
	adapters      map[string]providers.Adapter
	order         []string
	repo          Repository
	health        HealthRepository
	breaker       Breaker
	capacity      CapacityAdapter
	resolver      AttachmentResolver
	clock         func() time.Time
	rightsContext RightsContextSource
	consent       RemoteConsentVerifier
}

const (
	failureReasonAdapterMissing = "adapter.not_configured"
	failureReasonRoleResolve    = "provider.role_resolution_failed"
	failureReasonProfile        = "policy.profile_rejected"
	failureReasonRights         = "policy.local_rights_rejected"
	failureReasonSampling       = "policy.sampling_unsupported"
	failureReasonCapacity       = "capacity.admission_unverified"
	failureReasonExecute        = "provider.execution_failed"
	failureReasonNoProvider     = "route.no_eligible_provider"
	failureReasonUnknown        = "provider.failure"
)

type RemoteConsentVerifier interface {
	Verify(context.Context, string, string) error
}

// Option configures optional Service collaborators without breaking callers
// that only need routing/evidence.
type Option func(*Service)

// AttachmentResolver resolves opaque application-owned references into
// ephemeral inline bytes. It must never accept provider URLs or credentials.
type AttachmentResolver interface {
	Resolve(context.Context, *sharedv1.Attachment) (*sharedv1.Attachment, error)
}

func WithAttachmentResolver(resolver AttachmentResolver) Option {
	return func(s *Service) { s.resolver = resolver }
}

// WithHealth enables provider-health/circuit-breaker tracking. A nil repository
// leaves breaker tracking disabled.
func WithHealth(health HealthRepository) Option {
	return func(s *Service) { s.health = health }
}

// WithBreakerPolicy overrides the deterministic breaker thresholds.
func WithBreakerPolicy(policy BreakerPolicy) Option {
	return func(s *Service) { s.breaker = NewBreaker(policy) }
}

// WithCapacity enables capacity-aware local route eligibility. A nil adapter
// leaves capacity gating disabled (local routes proceed unconditionally).
func WithCapacity(capacity CapacityAdapter) Option {
	return func(s *Service) { s.capacity = capacity }
}

// WithClock injects the time source used for breaker transitions and evidence
// timestamps so tests can drive deterministic cooldown behavior.
func WithClock(clock func() time.Time) Option {
	return func(s *Service) {
		if clock != nil {
			s.clock = clock
		}
	}
}

// WithRightsContextSource injects operator-controlled organization facts.
// Nil or unreadable sources fail closed for local model candidates.
func WithRightsContextSource(source RightsContextSource) Option {
	return func(s *Service) { s.rightsContext = source }
}

func WithRemoteConsentVerifier(verifier RemoteConsentVerifier) Option {
	return func(s *Service) { s.consent = verifier }
}

func NewService(adapters []providers.Adapter, repo Repository, opts ...Option) *Service {
	s := &Service{
		validator: gateway.New(),
		adapters:  map[string]providers.Adapter{},
		repo:      repo,
		breaker:   NewBreaker(DefaultBreakerPolicy()),
		clock:     time.Now,
	}
	for _, adapter := range adapters {
		name := strings.TrimSpace(strings.ToLower(adapter.Provider))
		if name == "" {
			continue
		}
		s.adapters[name] = adapter
		s.order = append(s.order, name)
	}
	sort.Strings(s.order)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func NewSQLService(db *sql.DB, adapters []providers.Adapter, opts ...Option) *Service {
	base := []Option{
		WithHealth(NewSQLHealthRepository(db)),
		WithCapacity(&CLICapacityAdapter{}),
	}
	return NewService(adapters, NewSQLRepository(db), append(base, opts...)...)
}

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now()
	}
	return s.clock()
}

// effectiveBreaker returns the breaker state routing should act on for a
// candidate. It reports an empty state when health tracking is disabled so
// callers can distinguish "no tracking" from a healthy closed breaker.
func (s *Service) effectiveBreaker(ctx context.Context, provider, role string, kind sharedv1.RequestKind) BreakerState {
	if s.health == nil {
		return BreakerState("")
	}
	h, found, err := s.health.Get(ctx, HealthKey{Provider: provider, Role: role, Kind: kind})
	if err != nil || !found {
		return BreakerClosed
	}
	return s.breaker.Effective(h, s.now())
}

// recordOutcome updates persisted breaker state after an execution attempt.
// It is best-effort: a health-store write failure must not fail the caller's
// request, since routing already succeeded or failed on its own merits. A
// capacity refusal and a caller's cancellation are not health observations
// and never reach the breaker: a client that gives up on slow requests must
// not open the breaker for the next caller.
func (s *Service) recordOutcome(ctx context.Context, provider, role string, kind sharedv1.RequestKind, failure ProviderFailure, success bool) {
	if s.health == nil || failure.Class == FailureCapacityRefused || failure.Class == FailureCancellation {
		return
	}
	key := HealthKey{Provider: provider, Role: role, Kind: kind}
	h, found, err := s.health.Get(ctx, key)
	if err != nil {
		return
	}
	h.Provider, h.Role, h.Kind = provider, role, kind
	if !found {
		h = ProviderHealth{Provider: provider, Role: role, Kind: kind, State: BreakerClosed}
	}
	now := s.now()
	if success {
		h, err = s.breaker.OnSuccess(h, now)
	} else {
		h, err = s.breaker.OnProviderFailure(h, failure, now)
	}
	if err != nil {
		return
	}
	_ = s.health.Upsert(ctx, h)
}

// capacityOwner builds an op-scoped owner id for a capacity claim so the broker
// ledger attributes the reservation to this gateway request.
func capacityOwner(req *sharedv1.GatewayRequest) string {
	return "ai-gateway:" + firstNonEmpty(req.GetScenario(), "unknown") + ":" + firstNonEmpty(req.GetOperation(), "route")
}

func allowReclaim(req *sharedv1.GatewayRequest) bool {
	return strings.EqualFold(strings.TrimSpace(req.GetMetadata()["capacity_allow_reclaim"]), "true")
}

// ListProviderHealth returns persisted breaker records with the effective state
// computed at read time. Returns an empty slice when health tracking is disabled.
func (s *Service) ListProviderHealth(ctx context.Context) ([]*routingv1.ProviderHealth, error) {
	if s.health == nil {
		return nil, nil
	}
	records, err := s.health.List(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]*routingv1.ProviderHealth, 0, len(records))
	for _, h := range records {
		out = append(out, &routingv1.ProviderHealth{
			Provider:            h.Provider,
			Role:                h.Role,
			Kind:                h.Kind,
			State:               string(stateOrClosed(h.State)),
			EffectiveState:      string(s.breaker.Effective(h, now)),
			ConsecutiveFailures: int64(h.ConsecutiveFailures),
			LastFailureClass:    string(h.LastFailureClass),
			LastSuccessAt:       formatTime(h.LastSuccessAt),
			LastFailureAt:       formatTime(h.LastFailureAt),
			CooldownUntil:       formatTime(h.CooldownUntil),
			OpenedAt:            formatTime(h.OpenedAt),
			Generation:          h.Generation,
			UpdatedAt:           formatTime(h.UpdatedAt),
		})
	}
	return out, nil
}
