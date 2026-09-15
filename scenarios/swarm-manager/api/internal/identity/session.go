package identity

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/provenance"
)

// SessionReference is the session ownership data resolved from a verified
// Agent Manager run ID.
type SessionReference struct {
	SessionID   string
	SessionKind string
	Source      string
}

// SessionResolver maps Agent Manager run IDs to Swarm Manager sessions.
type SessionResolver interface {
	ResolveSessionForRun(ctx context.Context, runID string) (SessionReference, bool, error)
}

// SessionMiddleware enriches verified agent provenance with session ownership
// when the request came from a run spawned by an agent session.
func SessionMiddleware(resolver SessionResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			prov := FromContext(r.Context())
			if resolver == nil || !prov.IsAgent() || strings.TrimSpace(prov.RunID) == "" {
				next.ServeHTTP(w, r)
				return
			}
			ref, ok, err := resolver.ResolveSessionForRun(r.Context(), prov.RunID)
			if err != nil {
				slog.Warn("session provenance lookup failed", "run_id", prov.RunID, "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			ctx := provenance.NewContext(r.Context(), prov.WithSession(ref.SessionID, ref.SessionKind, ref.Source))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
