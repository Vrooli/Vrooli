package middleware

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// EntitlementMiddleware provides request-scoped entitlement checking.
type EntitlementMiddleware struct {
	service *entitlement.Service
	log     *logrus.Logger
	cfg     config.EntitlementConfig
}

// NewEntitlementMiddleware creates a new entitlement middleware.
func NewEntitlementMiddleware(service *entitlement.Service, log *logrus.Logger, cfg config.EntitlementConfig) *EntitlementMiddleware {
	return &EntitlementMiddleware{
		service: service,
		log:     log,
		cfg:     cfg,
	}
}

// InjectEntitlement is middleware that extracts user identity and injects entitlement into context.
// This runs on all requests to make entitlement info available to handlers.
func (m *EntitlementMiddleware) InjectEntitlement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user identity from request
		userIdentity := resolveUserIdentity(r)

		// Add to context even if empty (handlers can check)
		ctx := entitlement.WithUserIdentity(r.Context(), userIdentity)

		// If entitlements are enabled and we have a user, fetch and inject entitlement
		if m.cfg.Enabled && userIdentity != "" {
			ent, err := m.service.GetEntitlement(r.Context(), userIdentity)
			if err != nil {
				m.log.WithError(err).WithField("user", userIdentity).Debug("Failed to get entitlement")
				// Continue without entitlement - handlers will use defaults
			} else {
				ctx = entitlement.WithEntitlement(ctx, ent)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireActiveSubscription returns middleware that requires an active subscription.
// Returns 403 if subscription is not active.
func (m *EntitlementMiddleware) RequireActiveSubscription(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.cfg.Enabled {
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
			if !m.cfg.Enabled {
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
		if !m.cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		userIdentity := entitlement.UserIdentityFromContext(r.Context())
		if !m.service.CanUseAI(r.Context(), userIdentity) {
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
		if !m.cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		userIdentity := entitlement.UserIdentityFromContext(r.Context())
		if !m.service.CanUseRecording(r.Context(), userIdentity) {
			writeEntitlementError(w, http.StatusForbidden, "RECORDING_ACCESS_REQUIRED",
				"Recording features require Solo tier or higher")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// resolveUserIdentity extracts the user identity from the request.
// Checks in order: X-User-Email header, user query parameter, Authorization header.
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

// writeEntitlementError writes a standardized entitlement error response.
func writeEntitlementError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Simple JSON without importing encoding/json to keep middleware lightweight
	w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
