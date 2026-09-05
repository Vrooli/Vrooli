package auth

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// ctxKey is the unexported context key for the request-scoped owner Identity.
type ctxKey struct{}

// BreakGlassValidator is the explicit offline validation seam. A normal
// bearer failure never selects this path implicitly.
type BreakGlassValidator interface {
	ValidateBreakGlass(ctx context.Context, token string) (Identity, error)
}

// LocalSessionValidator is the offline enrollment path. It is separate from
// bearer validation so a malformed local credential can never fall through to
// a different authorization scheme.
type LocalSessionValidator interface {
	ValidateLocal(ctx context.Context, token string) (Identity, error)
}

// BreakGlassAuditor records accepted break-glass uses as typed audit events.
// Audit failure is refusal: emergency access remains accountable.
type BreakGlassAuditor interface {
	AuditBreakGlass(ctx context.Context, id Identity) error
}

// BreakGlassAuditFunc adapts a function to BreakGlassAuditor.
type BreakGlassAuditFunc func(context.Context, Identity) error

func (f BreakGlassAuditFunc) AuditBreakGlass(ctx context.Context, id Identity) error {
	return f(ctx, id)
}

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
	return MiddlewareWithAudit(v, logger, nil)
}

// MiddlewareWithAudit adds the explicit BreakGlass authorization scheme while
// preserving the pairing endpoint's best-effort identity injection.
func MiddlewareWithAudit(v Validator, logger *log.Logger, auditor BreakGlassAuditor) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "BreakGlass") {
				if bg, ok := v.(BreakGlassValidator); ok {
					if id, err := bg.ValidateBreakGlass(r.Context(), strings.TrimSpace(parts[1])); err == nil {
						if auditor == nil {
							logger.Printf("auth: break-glass rejected: no audit sink")
						} else if err := auditor.AuditBreakGlass(r.Context(), id); err != nil {
							logger.Printf("auth: break-glass rejected: audit failed: %v", err)
						} else {
							r = r.WithContext(WithIdentity(r.Context(), id))
						}
					} else {
						logger.Printf("auth: break-glass rejected: %v", err)
					}
				} else {
					logger.Printf("auth: break-glass rejected: verifier unavailable")
				}
			} else if parts := strings.SplitN(header, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "LocalSession") {
				if local, ok := v.(LocalSessionValidator); ok {
					if id, err := local.ValidateLocal(r.Context(), strings.TrimSpace(parts[1])); err == nil {
						r = r.WithContext(WithIdentity(r.Context(), id))
					} else {
						logger.Printf("auth: local owner session rejected: %v", err)
					}
				} else {
					logger.Printf("auth: local owner session rejected: verifier unavailable")
				}
			} else if token := BearerToken(header); token != "" {
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
