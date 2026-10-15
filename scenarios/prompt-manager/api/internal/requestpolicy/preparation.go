// Package requestpolicy contains the Prompt Manager ingress boundary for
// reviewed operation grants. Domain owners still authorize the final effect.
package requestpolicy

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
)

// IsBounded includes both interactive grants and verified ingress authorities.
// A presented authority cannot fall back to ambient legacy CRUD permissions.
func IsBounded(ctx context.Context) bool {
	_, bounded := authn.RequestOperationAuthorityFromContext(ctx)
	p, present := identity.PrincipalFromContext(ctx)
	return bounded || (present && p.Grant != nil)
}

// RequirePreparationRead narrows the allowlisted reads to the approved object.
func RequirePreparationRead(ctx context.Context, teamID string) error {
	if !IsBounded(ctx) {
		return nil
	}
	_, err := authn.RequireOperation(ctx, "prompt-manager:read", "team.prepare", "prompt-manager:team/"+teamID)
	return err
}

// PreparationBoundary is deliberately a closed allowlist. These are the two
// read-only owner procedures used to inspect a draft and its explicit amendment
// carrier. UpdateHeartbeat enforces the exact operation/target and rejects all
// ordinary mutations when bounded. Unknown APIs, legacy paths and future RPCs
// are refused before handlers can mutate relations, schedule or dispatch work.
// The two exact GET health routes also permit the CLI's read-only bootstrap.
// This does not expand or certify the authentication of ordinary legacy callers.
func PreparationBoundary(next http.Handler) http.Handler {
	errors := connect.NewErrorWriter()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsBounded(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		allowed := false
		if r.Method == http.MethodGet && (r.URL.Path == "/health" || r.URL.Path == "/api/v1/health") {
			_, err := authn.RequireCapability(r.Context(), "prompt-manager:read")
			allowed = err == nil
		}
		if r.Method == http.MethodPost {
			switch r.URL.Path {
			case heartbeatconnect.HeartbeatServiceGetHeartbeatProcedure,
				teamsconnect.TeamsServiceGetTeamProcedure:
				_, err := authn.RequireCapability(r.Context(), "prompt-manager:read")
				allowed = err == nil
			case heartbeatconnect.HeartbeatServiceUpdateHeartbeatProcedure:
				// Authorization and strict body exclusivity live at the owner.
				allowed = true
			}
		}
		if !allowed {
			err := connect.NewError(connect.CodePermissionDenied, authn.ErrOperation)
			if errors.IsSupported(r) {
				_ = errors.Write(w, r, err)
			} else {
				http.Error(w, err.Error(), http.StatusForbidden)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}
