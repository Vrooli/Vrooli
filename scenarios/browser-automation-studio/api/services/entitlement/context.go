package entitlement

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	entitlementKey  contextKey = "entitlement"
	userIdentityKey contextKey = "user_identity"
)

// WithEntitlement adds an entitlement to the context.
func WithEntitlement(ctx context.Context, ent *Entitlement) context.Context {
	return context.WithValue(ctx, entitlementKey, ent)
}

// FromContext retrieves the entitlement from the context.
// Returns nil if not present.
func FromContext(ctx context.Context) *Entitlement {
	if ent, ok := ctx.Value(entitlementKey).(*Entitlement); ok {
		return ent
	}
	return nil
}

// WithUserIdentity adds a user identity to the context.
func WithUserIdentity(ctx context.Context, userIdentity string) context.Context {
	return context.WithValue(ctx, userIdentityKey, userIdentity)
}

// UserIdentityFromContext retrieves the user identity from the context.
// Returns empty string if not present.
func UserIdentityFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIdentityKey).(string); ok {
		return id
	}
	return ""
}
