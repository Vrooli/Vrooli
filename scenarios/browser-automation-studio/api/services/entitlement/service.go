package entitlement

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/browser-automation-studio/config"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

var (
	ErrAccessTokenRequired     = errors.New("subscription access token is required")
	ErrEntitlementUnavailable  = errors.New("subscription entitlement service is unavailable")
	ErrEntitlementUnauthorized = errors.New("subscription access token was rejected")
)

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
		httpClient: apihttp.NewTestModeClient(&http.Client{
			Timeout: cfg.RequestTimeout,
		}),
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

// ClearSession removes the short-lived access token and any signed entitlement
// lease cache after the shared subscription session is deleted.
func (s *Service) ClearSession(context.Context) error {
	if s == nil {
		return nil
	}
	if s.sessionResolver != nil {
		s.sessionResolver.Clear()
	}
	if s.gate != nil && s.gate.Entitlements != nil {
		return s.gate.Entitlements.Clear()
	}
	return nil
}

// ResolveAccessToken returns the request-scoped or credential-session consumer
// token. It is intentionally short-lived and is never persisted by BAS.
func (s *Service) ResolveAccessToken(ctx context.Context, serviceURL string) (string, error) {
	if token := monetization.AccessTokenFromContext(ctx); token != "" {
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

// ResolveConsumerIdentity asks LPBS to bind a current access token to the
// account identity. Local identity hints are never sent as remote authority.
func (s *Service) ResolveConsumerIdentity(ctx context.Context) (string, error) {
	if s == nil {
		return "", ErrAccessTokenRequired
	}
	serviceURL := strings.TrimSpace(s.cfg.ServiceURL)
	token, err := s.ResolveAccessToken(ctx, serviceURL)
	if err != nil {
		return "", err
	}
	var identity string
	if s.sessionResolver != nil {
		identity, err = s.sessionResolver.ResolveIdentityAt(ctx, serviceURL, token)
	} else {
		identity, err = credentialclient.ResolveConsumerIdentity(ctx, serviceURL, token, s.httpClient)
	}
	if errors.Is(err, credentialclient.ErrSubscriptionUnauthorized) {
		return "", ErrEntitlementUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("%w: identity lookup: %v", ErrEntitlementUnavailable, err)
	}
	return identity, nil
}

// GetCachedEntitlement restores a verified signed lease without contacting
// LPBS. The identity selects a cache record only; the shared client verifies
// its issuer signature and signed expiry before returning it.
func (s *Service) GetCachedEntitlement(userIdentity string) (*Entitlement, error) {
	if s == nil || s.gate == nil || s.gate.Entitlements == nil {
		return nil, ErrEntitlementUnavailable
	}
	lease, err := s.gate.Entitlements.Cached(strings.TrimSpace(userIdentity))
	if err != nil {
		return nil, fmt.Errorf("%w: cached lease: %v", ErrEntitlementUnavailable, err)
	}
	snapshot := monetization.SnapshotFromLease(lease, time.Now().UTC())
	return &snapshot, nil
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

// ReserveWorkflowExecution creates one atomic Class B quota hold through LPBS.
// The caller commits it after persisting its accepted execution record, or
// releases it if that local write fails.
func (s *Service) ReserveWorkflowExecution(ctx context.Context, userIdentity string) (*monetization.QuotaHold, error) {
	userIdentity = strings.ToLower(strings.TrimSpace(userIdentity))
	if userIdentity == "" {
		return nil, fmt.Errorf("workflow execution requires an authenticated account")
	}
	lease, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		return nil, fmt.Errorf("workflow execution entitlement unavailable: %w", err)
	}
	if !lease.IsActive() || !lease.ExpiresAt.After(time.Now()) || !s.CanExecuteWorkflowAt(userIdentity, 0, time.Now()) {
		return nil, fmt.Errorf("workflow execution is not allowed by the signed lease")
	}
	token, err := s.ResolveAccessToken(ctx, strings.TrimSpace(s.cfg.ServiceURL))
	if err != nil {
		return nil, fmt.Errorf("workflow execution session unavailable: %w", err)
	}
	quota := monetization.QuotaReservationClient{BaseURL: s.cfg.ServiceURL, HTTPClient: s.httpClient}
	hold, err := quota.Begin(ctx, token, "workflow_executions", 1, 0)
	if err != nil {
		return nil, fmt.Errorf("workflow execution quota reservation failed: %w", err)
	}
	return hold, nil
}

// ProbeWorkflowExecution briefly reserves and releases one count unit. It is
// used by local conformance probes that must exercise LPBS admission without
// consuming a real execution allowance.
func (s *Service) ProbeWorkflowExecution(ctx context.Context, userIdentity string) error {
	hold, err := s.ReserveWorkflowExecution(ctx, userIdentity)
	if err != nil {
		return err
	}
	return hold.Release(context.WithoutCancel(ctx))
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
	if monetization.AccessTokenFromContext(ctx) == "" {
		accessToken, err := s.ResolveAccessToken(ctx, serviceURL)
		if err != nil {
			return nil, err
		}
		ctx = monetization.WithAccessToken(ctx, accessToken)
	}

	if s.gate == nil || s.gate.Entitlements == nil ||
		s.gate.Entitlements.BaseURL != strings.TrimRight(strings.TrimSpace(serviceURL), "/") ||
		s.gate.Entitlements.HTTPClient != s.httpClient {
		s.rebuildGateForURL(serviceURL)
	}
	var lease entitlementclient.Payload
	var err error
	if token := monetization.AccessTokenFromContext(ctx); token != "" {
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

	// The shared snapshot is a view over verified signed claims. The signed
	// not_after remains the hard offline boundary enforced by entitlementclient.
	snapshot := monetization.SnapshotFromLease(lease, time.Now().UTC())
	return &snapshot, nil
}

func (s *Service) rebuildGate() {
	s.rebuildGateForURL(s.cfg.ServiceURL)
}

func (s *Service) rebuildGateForURL(serviceURL string) {
	resolve := func(ctx context.Context, baseURL string) (string, error) {
		if token := monetization.AccessTokenFromContext(ctx); token != "" {
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
	client := entitlementclient.NewClient(serviceURL, resolve, s.httpClient)
	if store, err := newLeaseStore(); err != nil {
		if s.log != nil {
			s.log.WithError(err).Warn("monetization lease cache unavailable")
		}
	} else {
		client.SetStore(store)
	}
	if s.sessionResolver != nil && s.sessionResolver.Credentials != nil {
		client.SetIssuerKeyStore(monetization.NewCredentialIssuerKeyStore(s.sessionResolver.Credentials))
	}
	s.gate = monetization.NewGateForApp(client, s.sessionResolver, "business_suite", "browser-automation-studio")
}

func newLeaseStore() (*entitlementclient.FileLeaseStore, error) {
	base := strings.TrimSpace(os.Getenv("APP_DATA_DIR"))
	if base == "" {
		if vrooliHome := strings.TrimSpace(os.Getenv("VROOLI_HOME")); vrooliHome != "" {
			base = filepath.Join(vrooliHome, "browser-automation-studio")
		} else {
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				return nil, fmt.Errorf("resolve user config directory: %w", err)
			}
			base = filepath.Join(base, "Vrooli", "Browser Automation Studio")
		}
	}
	return entitlementclient.NewFileLeaseStore(filepath.Join(base, "monetization", "leases"))
}
