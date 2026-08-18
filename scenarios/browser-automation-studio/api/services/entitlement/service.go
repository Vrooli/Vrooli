package entitlement

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

const accessTokenContextKey contextKey = "subscription_access_token"

var (
	ErrAccessTokenRequired     = errors.New("subscription access token is required")
	ErrEntitlementUnavailable  = errors.New("subscription entitlement service is unavailable")
	ErrEntitlementUnauthorized = errors.New("subscription access token was rejected")
)

// WithAccessToken attaches the short-lived consumer access token to a
// request-scoped entitlement lookup. It is never persisted or cached.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenContextKey, strings.TrimSpace(token))
}

func accessTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(accessTokenContextKey).(string)
	return strings.TrimSpace(token)
}

// AccessTokenFromContext exposes the request-scoped consumer token to other
// BAS adapters that must call the trusted authority. It never returns a
// refresh credential or a persisted secret.
func AccessTokenFromContext(ctx context.Context) string {
	return accessTokenFromContext(ctx)
}

// Service provides entitlement checking and feature gating.
type Service struct {
	cfg        config.EntitlementConfig
	log        *logrus.Logger
	httpClient *http.Client

	sessionResolver *credentialclient.ConsumerSessionResolver
	gate            *monetization.Gate
}

// NewService creates a new entitlement service.
func NewService(cfg config.EntitlementConfig, log *logrus.Logger) *Service {
	svc := &Service{
		cfg: cfg,
		log: log,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
	svc.rebuildGate()
	return svc
}

// SetSessionResolver wires the platform-owned shared subscription session.
// The resolver keeps only the short-lived access token in memory.
func (s *Service) SetSessionResolver(resolver *credentialclient.ConsumerSessionResolver) {
	s.sessionResolver = resolver
	s.rebuildGate()
}

// ResolveAccessToken returns the request-scoped or credential-session consumer
// token. It is intentionally short-lived and is never persisted by BAS.
func (s *Service) ResolveAccessToken(ctx context.Context, serviceURL string) (string, error) {
	if token := accessTokenFromContext(ctx); token != "" {
		return token, nil
	}
	if s.sessionResolver == nil {
		return "", ErrAccessTokenRequired
	}
	access, err := s.sessionResolver.ResolveAt(ctx, serviceURL)
	if err != nil {
		return "", err
	}
	return access.AccessToken, nil
}

// GetEntitlement retrieves the entitlement for a user, using cache when available.
func (s *Service) GetEntitlement(ctx context.Context, userIdentity string) (*Entitlement, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))

	// Empty identity is never a valid paid lookup. The token's verified claims
	// are the only identity accepted by LPBS.
	if userIdentity == "" {
		return nil, ErrAccessTokenRequired
	}

	// Fetch from service
	ent, err := s.fetchEntitlement(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).WithField("user", userIdentity).Warn("Failed to fetch entitlement; denying until subscription is verified")
		return nil, err
	}

	return ent, nil
}

// CanExecuteWorkflow checks if the user can execute a workflow based on their tier limits.
// Returns true if execution is allowed, false if limit reached.
func (s *Service) CanExecuteWorkflow(ctx context.Context, userIdentity string, currentMonthCount int) bool {
	decision := s.gate.Meter(ctx, userIdentity, "workflow_executions")
	if !decision.Allowed || !decision.LimitFound {
		return false
	}
	if decision.Limit < 0 {
		// Unlimited
		return true
	}

	return int64(currentMonthCount) < decision.Limit
}

// CanExecuteWorkflowOffline evaluates the cached, signed Class B lease. It
// never refreshes the authority, so a transient outage cannot block local
// work while an unexpired lease remains valid.
func (s *Service) CanExecuteWorkflowOffline(userIdentity string, currentMonthCount int) bool {
	decision := s.gate.CachedMeter(userIdentity, "workflow_executions")
	return decisionAllowsCount(decision, currentMonthCount)
}

// CanUseFeatureOffline evaluates a feature from the verified lease cache and
// never contacts LPBS. It is intended for local Class B capability checks.
func (s *Service) CanUseFeatureOffline(userIdentity, feature string, minPlanRank int32) bool {
	return s.gate.CachedFeature(userIdentity, feature, minPlanRank).Allowed
}

// CanExecuteWorkflowAt evaluates the cached lease at an explicit instant.
// The method makes the signed expiry boundary testable without changing the
// process clock or mutating production state.
func (s *Service) CanExecuteWorkflowAt(userIdentity string, currentMonthCount int, now time.Time) bool {
	decision := s.gate.CachedMeterAt(userIdentity, "workflow_executions", now)
	return decisionAllowsCount(decision, currentMonthCount)
}

func decisionAllowsCount(decision monetization.Decision, currentMonthCount int) bool {
	if !decision.Allowed || !decision.LimitFound {
		return false
	}
	return decision.Limit < 0 || int64(currentMonthCount) < decision.Limit
}

// GetRemainingExecutions returns how many executions the user has remaining this month.
// Returns -1 for unlimited.
func (s *Service) GetRemainingExecutions(ctx context.Context, userIdentity string, currentMonthCount int) int {
	decision := s.gate.Meter(ctx, userIdentity, "workflow_executions")
	if !decision.Allowed || !decision.LimitFound {
		return 0
	}
	if decision.Limit < 0 {
		return -1
	}

	remaining := int(decision.Limit) - currentMonthCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RequiresWatermark returns true if exports for this user should be watermarked.
func (s *Service) RequiresWatermark(ctx context.Context, userIdentity string) bool {
	return !s.gate.Feature(ctx, userIdentity, FeatureWatermarkFree, 0).Allowed
}

// CanUseAI returns true if the user has access to AI-powered features.
func (s *Service) CanUseAI(ctx context.Context, userIdentity string) bool {
	return s.gate.Feature(ctx, userIdentity, FeatureAI, 0).Allowed
}

// CanUseRecording returns true if the user has access to live recording features.
func (s *Service) CanUseRecording(ctx context.Context, userIdentity string) bool {
	return s.gate.Feature(ctx, userIdentity, FeatureRecording, 0).Allowed
}

// InvalidateCache removes a user's cached entitlement, forcing a refresh on next check.
func (s *Service) InvalidateCache(_ string) {
	// entitlementclient owns the verified lease cache. Invalidation is
	// intentionally not a local billing decision; forcing a fresh lease is
	// represented by rebuilding the shared client.
	s.rebuildGate()
}

// fetchEntitlement calls the remote entitlement service.
func (s *Service) fetchEntitlement(ctx context.Context, userIdentity string) (*Entitlement, error) {
	serviceURL := s.cfg.ServiceURL
	if serviceURL == "" {
		return nil, fmt.Errorf("%w: service URL is not configured", ErrEntitlementUnavailable)
	}
	if accessTokenFromContext(ctx) == "" {
		accessToken, err := s.ResolveAccessToken(ctx, serviceURL)
		if err != nil {
			return nil, err
		}
		ctx = WithAccessToken(ctx, accessToken)
	}

	if s.gate == nil || s.gate.Entitlements == nil ||
		s.gate.Entitlements.BaseURL != strings.TrimRight(strings.TrimSpace(serviceURL), "/") ||
		s.gate.Entitlements.HTTPClient != s.httpClient {
		s.rebuildGateForURL(serviceURL)
	}
	var lease entitlementclient.Payload
	var err error
	if token := accessTokenFromContext(ctx); token != "" {
		// Bypass the identity-keyed cache whenever the caller supplied a
		// bearer. LPBS then checks that the requested identity matches the
		// verified token before this process accepts the lease.
		lease, err = s.gate.Entitlements.GetWithAccess(ctx, userIdentity, token)
	} else {
		lease, err = s.gate.Lease(ctx, userIdentity)
	}
	if err != nil {
		if errors.Is(err, entitlementclient.ErrLeaseUnauthorized) {
			return nil, fmt.Errorf("%w: %v", ErrEntitlementUnauthorized, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrEntitlementUnavailable, err)
	}

	// Convert the verified, signed lease to the local entitlement type. The
	// signed not_after is the hard offline boundary; cache TTL only controls
	// how often BAS refreshes while the lease remains valid.
	now := time.Now()
	ent := &Entitlement{
		UserIdentity:      lease.UserIdentity,
		Status:            Status(lease.Status),
		Tier:              strings.ToLower(lease.PlanTier),
		PlanRank:          lease.PlanRank,
		PriceID:           lease.PriceID,
		Features:          lease.Features,
		Limits:            lease.Limits,
		BillingCycleStart: lease.BillingCycleStart,
		FetchedAt:         now,
		ExpiresAt:         lease.NotAfter,
	}

	if balance, ok := lease.Credits.(map[string]interface{}); ok {
		if value, ok := balance["balance_credits"].(float64); ok {
			ent.Credits = int64(value)
		}
	}

	// Normalize tier
	if ent.Tier == "" {
		ent.Tier = TierFree
	}

	// Normalize status
	if ent.Status == "" {
		ent.Status = StatusInactive
	}

	return ent, nil
}

func (s *Service) rebuildGate() {
	s.rebuildGateForURL(s.cfg.ServiceURL)
}

func (s *Service) rebuildGateForURL(serviceURL string) {
	resolve := func(ctx context.Context, baseURL string) (string, error) {
		if token := accessTokenFromContext(ctx); token != "" {
			return token, nil
		}
		if s.sessionResolver == nil {
			return "", ErrAccessTokenRequired
		}
		access, err := s.sessionResolver.ResolveAt(ctx, baseURL)
		if err != nil {
			return "", err
		}
		return access.AccessToken, nil
	}
	s.gate = monetization.NewGate(entitlementclient.NewClient(serviceURL, resolve, s.httpClient), s.sessionResolver, "business_suite")
}

// CanUseAIWithEntitlement checks AI access using features array first, then tier fallback.
// If the Features array is present (non-nil and non-empty), it is authoritative.
// If Features is nil or empty, falls back to tier-based config for backwards compatibility.
func (s *Service) CanUseAIWithEntitlement(ent *Entitlement) bool {
	if ent == nil {
		return false
	}
	return monetization.HasFeature(entitlementclient.Payload{Features: ent.Features}, FeatureAI)
}

// CanUseRecordingWithEntitlement checks recording access using features array first.
// If the Features array is present (non-nil and non-empty), it is authoritative.
// If Features is nil or empty, falls back to tier-based config for backwards compatibility.
func (s *Service) CanUseRecordingWithEntitlement(ent *Entitlement) bool {
	if ent == nil {
		return false
	}
	return monetization.HasFeature(entitlementclient.Payload{Features: ent.Features}, FeatureRecording)
}

// RequiresWatermarkWithEntitlement checks if watermark is required using features array first.
// If the Features array is present (non-nil and non-empty), it is authoritative.
// If Features is nil or empty, falls back to tier-based config for backwards compatibility.
func (s *Service) RequiresWatermarkWithEntitlement(ent *Entitlement) bool {
	if ent == nil {
		return true // Fail safe: require watermark
	}
	return !monetization.HasFeature(entitlementclient.Payload{Features: ent.Features}, FeatureWatermarkFree)
}
