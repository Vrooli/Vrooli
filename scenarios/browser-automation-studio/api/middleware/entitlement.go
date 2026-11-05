package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// EntitlementMiddleware provides request-scoped entitlement checking.
type EntitlementMiddleware struct {
	service      *entitlement.Service
	log          *logrus.Logger
	cfg          config.EntitlementConfig
	settingsRepo EntitlementSettingsRepository
}

// EntitlementSettingsRepository defines the interface for user settings storage.
type EntitlementSettingsRepository interface {
	GetSetting(ctx context.Context, key string) (string, error)
}

// NewEntitlementMiddleware creates a new entitlement middleware.
func NewEntitlementMiddleware(
	service *entitlement.Service,
	log *logrus.Logger,
	cfg config.EntitlementConfig,
	settingsRepo EntitlementSettingsRepository,
) *EntitlementMiddleware {
	return &EntitlementMiddleware{
		service:      service,
		log:          log,
		cfg:          cfg,
		settingsRepo: settingsRepo,
	}
}

// InjectEntitlement is middleware that extracts user identity and injects entitlement into context.
// This runs on all requests to make entitlement info available to handlers.
func (m *EntitlementMiddleware) InjectEntitlement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request headers, query parameters, and bodies never select an
		// entitlement lease. The stored identity is only a lookup hint for the
		// shared consumer session; LPBS overwrites it with the verified lease
		// identity before the request is allowed to use paid state.
		userIdentity := m.resolveStoredUserIdentity(r.Context())

		// Carry the consumer access token through the request context. The
		// entitlement authority derives the authoritative identity from this
		// token; request headers are only a lookup hint for the local cache.
		ctx := r.Context()
		if token := bearerToken(r.Header.Get("Authorization")); token != "" {
			ctx = entitlement.WithAccessToken(ctx, token)
		}
		ctx = entitlement.WithUserIdentity(ctx, userIdentity)

		// Fetch and inject entitlement for authenticated users
		if userIdentity != "" {
			ent, err := m.service.GetEntitlement(ctx, userIdentity)
			if err != nil {
				m.log.WithError(err).WithField("user", userIdentity).Debug("Failed to get entitlement")
				// Continue without entitlement - handlers will use defaults
			} else {
				ctx = entitlement.WithEntitlement(ctx, ent)
				ctx = entitlement.WithUserIdentity(ctx, ent.UserIdentity)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireActiveSubscription returns middleware that requires an active subscription.
// Returns 403 if subscription is not active.
func (m *EntitlementMiddleware) RequireActiveSubscription(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.entitlementsEnabled(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		decision := m.featureDecision(r.Context(), "", 0)
		if !decision.Allowed {
			monetization.WriteError(w, http.StatusForbidden, monetization.ErrorSubscriptionRequired, decision)
			return
		}
		if decision.Warning {
			w.Header().Set("X-Entitlement-Warning", monetization.ReasonPastDue)
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAIAccess returns middleware that requires AI feature access.
func (m *EntitlementMiddleware) RequireAIAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.entitlementsEnabled(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		decision := m.featureDecision(r.Context(), entitlement.FeatureAI, 0)
		if !decision.Allowed {
			monetization.WriteError(w, http.StatusForbidden, monetization.ErrorRankRequired, decision)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRecordingAccess returns middleware that requires recording feature access.
func (m *EntitlementMiddleware) RequireRecordingAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.entitlementsEnabled(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		decision := m.featureDecision(r.Context(), entitlement.FeatureRecording, 0)
		if !decision.Allowed {
			monetization.WriteError(w, http.StatusForbidden, monetization.ErrorRankRequired, decision)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *EntitlementMiddleware) featureDecision(ctx context.Context, feature string, minPlanRank int32) monetization.Decision {
	const upgradePath = "/settings/subscription"
	ent := entitlement.FromContext(ctx)
	if ent == nil {
		return monetization.Decision{Reason: monetization.ReasonLeaseUnavailable, UpgradePath: upgradePath}
	}
	payload := entitlementclient.Payload{Status: string(ent.Status), PlanRank: ent.PlanRank, Features: ent.Features, Limits: ent.Limits}
	decision := monetization.StatusDecision(payload, upgradePath)
	return monetization.FeatureDecision(payload, feature, minPlanRank, decision)
}

func bearerToken(authorization string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
}

func (m *EntitlementMiddleware) entitlementsEnabled(_ context.Context) bool {
	// Entitlement enforcement is fail-closed. A missing lease is not an
	// entitlement, and must not turn a protected route into an unprotected one.
	return true
}

func (m *EntitlementMiddleware) resolveStoredUserIdentity(ctx context.Context) string {
	if m.settingsRepo == nil {
		return ""
	}
	value, err := m.settingsRepo.GetSetting(ctx, "user_identity")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(value))
}
