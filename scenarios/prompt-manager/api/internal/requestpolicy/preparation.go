// Package requestpolicy contains the Prompt Manager ingress boundary for
// reviewed operation grants. Domain owners still authorize the final effect.
package requestpolicy

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	aisearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch/aisearch_v1connect"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery/discovery_v1connect"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
)

// IsBounded includes both interactive grants and verified ingress authorities.
// A presented authority cannot fall back to ambient legacy CRUD permissions.
func IsBounded(ctx context.Context) bool {
	_, bounded := authn.RequestOperationAuthorityFromContext(ctx)
	p, present := identity.PrincipalFromContext(ctx)
	return bounded || (present && p.Grant != nil)
}

// RequirePreparationRead preserves a caller's ordinary Prompt Manager read
// capability when an unrelated request-scoped operation proof accompanies it.
// A caller relying on a preparation grant or preparation operation still needs
// the exact reviewed team.prepare target.
func RequirePreparationRead(ctx context.Context, teamID string) error {
	if !IsBounded(ctx) {
		return nil
	}
	if _, hasRequestAuthority := authn.RequestOperationAuthorityFromContext(ctx); hasRequestAuthority {
		if p, present := identity.PrincipalFromContext(ctx); present && p.IsVerified() && p.Grant == nil {
			if _, err := authn.RequireCapability(ctx, "prompt-manager:read"); err == nil {
				return nil
			}
		}
	}
	_, err := authn.RequireOperation(ctx, "prompt-manager:read", "team.prepare", "prompt-manager:team/"+teamID)
	return err
}

// IsVerifiedIdentity confirms that a caller is verified before it reaches a
// narrowly enumerated ordinary route, even if unrelated authority bounds the
// request. These operations do not exercise the attached grant. An unverified
// principal never qualifies through scopes or identity alone.
func IsVerifiedIdentity(ctx context.Context) bool {
	p, present := identity.PrincipalFromContext(ctx)
	return present && p.IsVerified()
}

type routeContract uint8

const (
	denyBoundedRoute routeContract = iota
	verifiedIdentityRoute
	readCapabilityRoute
	ownerAuthorizedRoute
)

type routeKey struct {
	method string
	path   string
}

// boundedRouteContracts is the explicit ingress contract for requests carrying
// a grant or request-scoped operation proof. Only routes with an existing
// verified-identity or exact owner authorization contract are listed.
var boundedRouteContracts = map[routeKey]routeContract{
	{http.MethodGet, "/health"}:                                                          verifiedIdentityRoute,
	{http.MethodGet, "/api/v1/health"}:                                                   verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetHeartbeatProcedure}:            readCapabilityRoute,
	{http.MethodPost, teamsconnect.TeamsServiceGetTeamProcedure}:                         readCapabilityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceListHeartbeatAttemptsProcedure}:   verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetTeamExecutionStatusProcedure}:  verifiedIdentityRoute,
	{http.MethodPost, skillsconnect.SkillsServiceReadSkillsProcedure}:                    verifiedIdentityRoute,
	{http.MethodPost, discoveryconnect.DiscoveryServiceDiscoverProcedure}:                verifiedIdentityRoute,
	{http.MethodPost, aisearchconnect.AISearchServiceSearchSkillsProcedure}:              verifiedIdentityRoute,
	{http.MethodPost, teamsconnect.TeamsServiceListKnowledgeProcedure}:                   verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServicePreviewPromptProcedure}:           verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServicePreviewPromptStructuredProcedure}: verifiedIdentityRoute,
	{http.MethodPost, teamsconnect.TeamsServiceAddKnowledgeProcedure}:                    verifiedIdentityRoute,
	{http.MethodPost, teamsconnect.TeamsServiceCloseKnowledgeProcedure}:                  verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceCaptureBugProcedure}:              verifiedIdentityRoute,

	// These heartbeat mutations already enforce exact operation/target rules in
	// their domain owner. No authority is inferred by this classifier.
	{http.MethodPost, heartbeatconnect.HeartbeatServiceCreateHeartbeatProcedure}: ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceUpdateHeartbeatProcedure}: ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceDeleteHeartbeatProcedure}: ownerAuthorizedRoute,
}

func contractForBoundedRoute(method, path string) routeContract {
	if contract, ok := boundedRouteContracts[routeKey{method: method, path: path}]; ok {
		return contract
	}
	return denyBoundedRoute
}

// PreparationBoundary is deliberately a closed allowlist. The preparation reads
// inspect a draft and its explicit amendment carrier; ReadSkills, ListKnowledge
// and prompt previews are ordinary verified reads, independent of unrelated
// grants. AddKnowledge and CloseKnowledge are exact attributed, append-only
// bookkeeping writes; their owners continue validating attribution, team,
// payload, closure reason and entry existence. Other knowledge mutations remain
// closed here.
// UpdateHeartbeat enforces its exact operation/target and rejects unreviewed
// heartbeat mutations when bounded. Unknown APIs, legacy paths and future RPCs are refused
// before handlers can mutate relations, schedule or dispatch work. The two exact
// GET health routes also permit the CLI's read-only bootstrap. This does not
// expand or certify the authentication of ordinary legacy callers.
func PreparationBoundary(next http.Handler) http.Handler {
	errors := connect.NewErrorWriter()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsBounded(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		contract := contractForBoundedRoute(r.Method, r.URL.Path)
		allowed := false
		switch contract {
		case verifiedIdentityRoute:
			allowed = IsVerifiedIdentity(r.Context())
		case readCapabilityRoute:
			_, err := authn.RequireCapability(r.Context(), "prompt-manager:read")
			allowed = err == nil
		case ownerAuthorizedRoute:
			allowed = true
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
