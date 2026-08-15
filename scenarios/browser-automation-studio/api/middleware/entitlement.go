package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
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
		// Extract user identity from request
		userIdentity := resolveUserIdentity(r)
		if userIdentity == "" {
			userIdentity = m.resolveStoredUserIdentity(r.Context())
		}

		// Carry the consumer access token through the request context. The
		// entitlement authority derives the authoritative identity from this
		// token; request headers are only a lookup hint for the local cache.
		ctx := r.Context()
		if token := bearerToken(r.Header.Get("Authorization")); token != "" {
			ctx = entitlement.WithAccessToken(ctx, token)
		}
		ctx = entitlement.WithUserIdentity(ctx, userIdentity)

		overrideTier := m.resolveOverrideTier(r.Context())
		if overrideTier != "" {
			ent := m.service.BuildOverrideEntitlement(userIdentity, overrideTier)
			ctx = entitlement.WithEntitlement(ctx, ent)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

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

		ent := entitlement.FromContext(r.Context())
		if ent == nil || !ent.IsActive() {
			writeEntitlementError(w, http.StatusForbidden, "SUBSCRIPTION_REQUIRED",
				"An active subscription is required for this feature")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireTier returns middleware that requires at least the specified tier.
func (m *EntitlementMiddleware) RequireTier(minTier entitlement.Tier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.entitlementsEnabled(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			ent := entitlement.FromContext(r.Context())
			if ent == nil || !ent.Tier.AtLeast(minTier) {
				writeEntitlementError(w, http.StatusForbidden, "TIER_REQUIRED",
					"This feature requires "+string(minTier)+" tier or higher")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAIAccess returns middleware that requires AI feature access.
func (m *EntitlementMiddleware) RequireAIAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.entitlementsEnabled(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		userIdentity := entitlement.UserIdentityFromContext(r.Context())
		if !m.canUseAI(r.Context(), userIdentity) {
			writeEntitlementError(w, http.StatusForbidden, "AI_ACCESS_REQUIRED",
				"AI features require Pro tier or higher")
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

		userIdentity := entitlement.UserIdentityFromContext(r.Context())
		if !m.canUseRecording(r.Context(), userIdentity) {
			writeEntitlementError(w, http.StatusForbidden, "RECORDING_ACCESS_REQUIRED",
				"Recording features require Solo tier or higher")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// resolveUserIdentity extracts a local lookup hint from the request. The hint
// is never trusted as billing identity: a fetched signed lease replaces it in
// the request context, and LPBS independently derives identity from the bearer
// token for Class A operations.
func resolveUserIdentity(r *http.Request) string {
	// 1. Check X-User-Email header (standard for our apps)
	if email := strings.TrimSpace(r.Header.Get("X-User-Email")); email != "" {
		return strings.ToLower(email)
	}

	// 2. Check user query parameter (for simple GET requests)
	if email := strings.TrimSpace(r.URL.Query().Get("user")); email != "" {
		return strings.ToLower(email)
	}

	// 3. Check X-User-Identity header (alternative)
	if identity := strings.TrimSpace(r.Header.Get("X-User-Identity")); identity != "" {
		return strings.ToLower(identity)
	}

	return ""
}

func bearerToken(authorization string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
}

func (m *EntitlementMiddleware) entitlementsEnabled(ctx context.Context) bool {
	// Entitlement enforcement is fail-closed. A missing lease is not an
	// entitlement, and must not turn a protected route into an unprotected one.
	return true
}

func (m *EntitlementMiddleware) canUseAI(ctx context.Context, userIdentity string) bool {
	if ent := entitlement.FromContext(ctx); ent != nil {
		return m.service.CanUseAIWithEntitlement(ent)
	}
	return m.service.CanUseAI(ctx, userIdentity)
}

func (m *EntitlementMiddleware) canUseRecording(ctx context.Context, userIdentity string) bool {
	if ent := entitlement.FromContext(ctx); ent != nil {
		return m.service.CanUseRecordingWithEntitlement(ent)
	}
	return m.service.CanUseRecording(ctx, userIdentity)
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

func (m *EntitlementMiddleware) resolveOverrideTier(ctx context.Context) entitlement.Tier {
	if m.settingsRepo == nil {
		return ""
	}
	value, err := m.settingsRepo.GetSetting(ctx, entitlement.OverrideTierSettingKey)
	if err != nil || value == "" {
		return ""
	}
	tier, ok := entitlement.ParseTier(value)
	if !ok {
		return ""
	}
	return tier
}

// writeEntitlementError writes a standardized entitlement error response.
func writeEntitlementError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Simple JSON without importing encoding/json to keep middleware lightweight
	if _, err := w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`)); err != nil {
		return
	}
}
