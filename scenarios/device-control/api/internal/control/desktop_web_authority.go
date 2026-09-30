package control

import (
	"context"
	"strings"
	"time"

	"device-control/internal/sessions"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/owneridentity"
)

type desktopWebIdentityKey struct{}
type desktopIdentityResolver struct{}

func (desktopIdentityResolver) ResolveScenarioURLDefault(ctx context.Context, name string) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, name)
}

func desktopIdentityAllows(identity owneridentity.Identity, control bool, now time.Time) bool {
	if identity.Subject == "" || !now.Before(identity.ExpiresAt) {
		return false
	}
	read, write := false, false
	for _, scope := range identity.Scopes {
		if scope == "device-control:read" {
			read = true
		}
		if scope == "device-control:write" {
			write = true
		}
	}
	return read && (!control || write)
}

// An optional forwarded bearer never falls back to the local Unix principal.
// The locally configured operator pin authorizes an account to this desktop;
// possession of some valid account token is not destination authorization.
func (a *LocalDesktopAdmission) authenticateForwarded(ctx context.Context, authorization string) (context.Context, error) {
	if authorization == "" {
		return ctx, nil
	}
	token, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || len(token) > 16*1024 || a.webSubject == "" || a.webIdentity == nil {
		return nil, sessions.ErrDesktopAdmission
	}
	identity, err := a.webIdentity.Validate(ctx, token)
	if err != nil || identity.Subject != a.webSubject || !desktopIdentityAllows(identity, false, time.Now()) {
		return nil, sessions.ErrDesktopAdmission
	}
	return context.WithValue(ctx, desktopWebIdentityKey{}, identity), nil
}

func (a *LocalDesktopAdmission) sessionActorAllowed(ctx context.Context, actor string, control bool) bool {
	if identity, ok := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); ok {
		return identity.Subject == actor && identity.Subject == a.webSubject && desktopIdentityAllows(identity, control, time.Now())
	}
	return actor == a.principal.String()
}
