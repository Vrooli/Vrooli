package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// EntitlementMiddleware adapts the shared consumer token and LPBS-verified
// lease to BAS's existing request context. Route enforcement belongs to the
// shared monetization kit, not scenario-local middleware methods.
type EntitlementMiddleware struct {
	service      *entitlement.Service
	log          *logrus.Logger
	settingsRepo EntitlementSettingsRepository
}

// EntitlementSettingsRepository supplies the local identity lookup hint used
// to ask the shared credential authority for the current consumer lease.
type EntitlementSettingsRepository interface {
	GetSetting(ctx context.Context, key string) (string, error)
}

type entitlementSettingsWriter interface {
	SetSetting(ctx context.Context, key, value string) error
}

// NewEntitlementMiddleware creates the BAS request-context adapter.
func NewEntitlementMiddleware(service *entitlement.Service, log *logrus.Logger, settingsRepo EntitlementSettingsRepository) *EntitlementMiddleware {
	return &EntitlementMiddleware{service: service, log: log, settingsRepo: settingsRepo}
}

// InjectEntitlement attaches the shared short-lived consumer token, then adds
// the verified lease to BAS's legacy handler context when LPBS is available.
// Missing/offline authority remains a denied paid decision; the request itself
// continues so local/free functionality stays available.
func (m *EntitlementMiddleware) InjectEntitlement(next http.Handler) http.Handler {
	return monetization.InjectEntitlement(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userIdentity := m.resolveStoredUserIdentity(ctx)
		remoteIdentity := false
		if m.service != nil {
			identity, err := m.service.ResolveConsumerIdentity(ctx)
			if err == nil {
				userIdentity, remoteIdentity = identity, true
				if writer, ok := m.settingsRepo.(entitlementSettingsWriter); ok {
					_ = writer.SetSetting(ctx, "user_identity", identity)
				}
			} else if m.log != nil {
				m.log.WithError(err).Debug("Failed to resolve consumer identity; using only a verified offline lease if available")
			}
		}
		ctx = monetization.WithUserIdentity(ctx, userIdentity)
		if m.service != nil && userIdentity != "" {
			var ent *entitlement.Entitlement
			var err error
			if remoteIdentity {
				ent, err = m.service.GetEntitlement(ctx, userIdentity)
			} else {
				ent, err = m.service.GetCachedEntitlement(userIdentity)
			}
			if err != nil {
				if m.log != nil {
					m.log.WithError(err).WithField("user", userIdentity).Debug("Failed to get entitlement")
				}
			} else {
				ctx = monetization.WithSnapshot(ctx, ent)
				ctx = monetization.WithUserIdentity(ctx, ent.UserIdentity)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
}

func (m *EntitlementMiddleware) resolveStoredUserIdentity(ctx context.Context) string {
	if m == nil || m.settingsRepo == nil {
		return ""
	}
	value, err := m.settingsRepo.GetSetting(ctx, "user_identity")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(value))
}
