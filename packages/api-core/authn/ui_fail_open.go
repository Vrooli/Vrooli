package authn

import (
	"net/http"
	"strings"

	"github.com/vrooli/api-core/identity"
)

// UIAvailabilityFallback keeps an app's interface usable when its optional
// identity provider or scope configuration fails. Every request that reaches
// an opted-in scenario API receives a synthetic operator principal with
// wildcard scopes. This is an explicit availability tradeoff, not proof that a
// request came from a human; anyone who can reach the API receives that access.
func UIAvailabilityFallback(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		previous, _ := identity.StatusFromContext(r.Context())
		principal, _ := identity.PrincipalFromContext(r.Context())
		subject := strings.TrimSpace(principal.Subject)
		if subject == "" {
			subject = "ui-fail-open"
		}
		sources := append([]identity.AuthSource(nil), principal.Sources...)
		if principal.Source != "" {
			sources = appendUnique(sources, principal.Source)
		}
		sources = appendUnique(sources, identity.SourceUIFailOpen)
		fallback := identity.Principal{
			Kind:     identity.ActorHuman,
			Subject:  subject,
			Scopes:   []string{"*"},
			Verified: true,
			Source:   identity.SourceUIFailOpen,
			Sources:  sources,
		}
		status := identity.Status{
			State:           identity.StateVerified,
			Authenticated:   true,
			Source:          identity.SourceUIFailOpen,
			Principal:       fallback,
			FailureClass:    previous.FailureClass,
			ProviderSources: append([]identity.AuthSource(nil), previous.ProviderSources...),
		}
		ctx := identity.WithStatus(identity.WithPrincipal(r.Context(), fallback), status)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
