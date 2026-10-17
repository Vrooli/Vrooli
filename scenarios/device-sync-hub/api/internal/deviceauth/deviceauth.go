// Package deviceauth is the device-token trust gate for the transfer and
// realtime surfaces. It is the device-side mirror of internal/auth (which gates
// owner-identity RPCs): where auth answers "is this a valid owner session",
// deviceauth answers "is this a TRUSTED device of the owner".
//
// A device presents the opaque hub token it received at pairing in the
// `X-Device-Token` header (or, for an EventSource SSE connection that cannot set
// headers, a `?token=` query parameter). The middleware resolves it via
// devices.Authenticator and injects the resolved Device into the request
// context; handlers read it through RequireDevice. Resolution is best-effort at
// the middleware (so a malformed token doesn't 500 the whole request); the
// per-handler RequireDevice is what fails closed.
package deviceauth

import (
	"context"
	"log"
	"net/http"
	"strings"

	"device-sync-hub/internal/devices"
)

// HeaderName is the request header carrying the raw hub device token.
const HeaderName = "X-Device-Token"

// queryParam is the EventSource fallback: the browser EventSource API cannot set
// request headers, so an SSE client passes its token as `?token=...`.
const queryParam = "token"

type ctxKey struct{}

// WithDevice returns a child context carrying d. Exported for tests that drive
// handlers directly without the HTTP middleware in front.
func WithDevice(ctx context.Context, d devices.Device) context.Context {
	return context.WithValue(ctx, ctxKey{}, d)
}

// DeviceFromContext returns the trusted device injected by the middleware, and
// whether one was present.
func DeviceFromContext(ctx context.Context) (devices.Device, bool) {
	d, ok := ctx.Value(ctxKey{}).(devices.Device)
	return d, ok
}

// RequireDevice is the handler-side gate for device-token-authed calls. It
// returns the trusted Device or devices.ErrUntrustedDevice when none is present
// (an absent, malformed, unknown, or non-TRUSTED token all land here).
func RequireDevice(ctx context.Context) (devices.Device, error) {
	d, ok := DeviceFromContext(ctx)
	if !ok {
		return devices.Device{}, devices.ErrUntrustedDevice
	}
	return d, nil
}

// Token extracts the presented device token from a request: the X-Device-Token
// header first, then the `?token=` query fallback for EventSource.
func Token(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get(HeaderName)); v != "" {
		return v
	}
	return strings.TrimSpace(r.URL.Query().Get(queryParam))
}

// Middleware resolves the presented device token (when present) and injects the
// trusted Device into the request context. Best-effort-inject, not
// reject-on-failure: enforcement is per-handler via RequireDevice, so an
// unauthenticated transfer call fails closed there while leaving owner-authed
// routes (which carry no device token) untouched.
func Middleware(a devices.Authenticator, logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := Token(r); token != "" {
				if d, err := a.Authenticate(r.Context(), token); err == nil {
					r = r.WithContext(WithDevice(r.Context(), d))
				} else {
					logger.Printf("deviceauth: device token rejected: %v", err)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
