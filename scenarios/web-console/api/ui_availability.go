package main

import (
	"net/http"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// Proven callers must retain their verified actor and scopes: reserved child
// creation and native lifecycle operations authorize Agent Manager itself,
// and bounded agent requests must not become unrestricted human requests.
// The browser availability policy remains owned by the shared UI fallback.
func webConsoleUIAvailabilityFallback(next http.Handler) http.Handler {
	fallback := authn.UIAvailabilityFallback(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if principal, ok := identity.PrincipalFromContext(r.Context()); ok && principal.IsVerified() {
			next.ServeHTTP(w, r)
			return
		}
		fallback.ServeHTTP(w, r)
	})
}
