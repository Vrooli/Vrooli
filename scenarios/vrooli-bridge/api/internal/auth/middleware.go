package auth

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// ctxKey is the unexported context key for the request-scoped owner Identity.
type ctxKey struct{}

// WithIdentity returns a child context carrying id. Exported for tests that
// drive handlers directly without the HTTP middleware in front.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// OwnerFromContext returns the owner Identity injected by the middleware, and
// whether one was present.
func OwnerFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

// RequireOwner is the handler-side gate for owner-authed RPCs. It returns the
// Identity or ErrUnauthenticated when none is present. The pairing redeem path
// (Phase 2), reachable by a not-yet-trusted node, simply never calls this.
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
// best-effort-inject, not reject-on-failure: a few RPCs (pairing redeem) are
// reachable by nodes that have no owner token yet, so global rejection would
// break the join handshake. Enforcement is per-handler via RequireOwner —
// owner-gated handlers reject when no Identity is present, which is fail-closed
// for both an invalid token and an unreachable authenticator.
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
					logger.Printf("auth: owner token rejected: %v", err)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// FakeValidator is the test double for Validator. Used by middleware and
// handler tests so they never touch the network.
type FakeValidator struct {
	Identity Identity
	Err      error
}

var _ Validator = (*FakeValidator)(nil)

// Validate returns the configured Identity/Err.
func (f *FakeValidator) Validate(_ context.Context, _ string) (Identity, error) {
	return f.Identity, f.Err
}
