package auth

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// ctxKey is the unexported context key type for the request-scoped owner
// Identity. Unexported so only this package can set it — handlers read through
// OwnerFromContext / RequireOwner, never by reaching for the raw key.
type ctxKey struct{}

// WithIdentity returns a child context carrying id. Exported for tests that
// drive handlers directly without the HTTP middleware in front.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// OwnerFromContext returns the owner Identity injected by the middleware, and
// whether one was present (i.e. the request carried a valid owner token).
func OwnerFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

// RequireOwner is the handler-side gate for owner-authed RPCs. It returns the
// Identity or ErrUnauthenticated when none is present. Open RPCs (RedeemPairing
// Code, RequestPairing — called by not-yet-paired devices) simply never call
// this.
func RequireOwner(ctx context.Context) (Identity, error) {
	id, ok := OwnerFromContext(ctx)
	if !ok {
		return Identity{}, ErrUnauthenticated
	}
	return id, nil
}

// BearerToken extracts the raw token from an "Authorization: Bearer <token>"
// header value, or "" if absent/malformed.
func BearerToken(header string) string {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// Middleware validates the owner bearer token (when present) and injects the
// resolved Identity into the request context. It is intentionally
// best-effort-inject, not reject-on-failure: a few RPCs (pairing redeem /
// request) are reachable by devices that have no owner token yet, so global
// rejection would break the join handshake. Enforcement is per-handler via
// RequireOwner — owner-gated handlers reject when no Identity is present, which
// is fail-closed for both an invalid token and an unreachable authenticator
// (Validate injects nothing in either case).
func Middleware(v Validator, logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r.Header.Get("Authorization"))
			if token != "" {
				if id, err := v.Validate(r.Context(), token); err == nil {
					r = r.WithContext(WithIdentity(r.Context(), id))
				} else {
					// Logged, not surfaced: open RPCs proceed unauthenticated;
					// owner-gated RPCs reject downstream via RequireOwner.
					logger.Printf("auth: owner token rejected: %v", err)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
