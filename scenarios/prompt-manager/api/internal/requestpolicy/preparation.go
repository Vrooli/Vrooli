// Package requestpolicy contains the Prompt Manager ingress boundary for
// reviewed operation grants. Domain owners still authorize the final effect.
package requestpolicy

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/actions/actions_v1connect"
	agentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents/agents_v1connect"
	aisearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch/aisearch_v1connect"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery/discovery_v1connect"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	memberflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow/memberflow_v1connect"
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

// RequirePreparationRead preserves ordinary Prompt Manager reads for unbounded
// callers and unrelated request-scoped proofs. Bounded team preparation may use
// either its explicit team.prepare grant or an exact team.manage grant, since
// reading the team is a prerequisite to that same team's authorized management.
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
	target := "prompt-manager:team/" + teamID
	if _, err := authn.RequireOperation(ctx, "prompt-manager:read", "team.prepare", target); err == nil {
		return nil
	}
	_, err := authn.RequireOperation(ctx, "prompt-manager:write", "team.manage", target)
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

// ErrNoBoundedRouteContract means the closed ingress allowlist has no owner
// contract for this method/path. An operation grant cannot make an undeclared
// route eligible; add a reviewed route contract instead.
var ErrNoBoundedRouteContract = stderrors.New("no bounded request contract is declared for this route; an operation grant cannot classify it")

type routeContract uint8

const (
	denyBoundedRoute routeContract = iota
	verifiedIdentityRoute
	readCapabilityRoute
	ownerAuthorizedRoute
	runCreateAuthorizedRoute
)

type routeKey struct {
	method string
	path   string
}

// boundedRouteContracts is the explicit ingress contract for requests carrying
// a grant or request-scoped operation proof. Only routes with an existing
// verified-identity, baseline-capability, exact owner authorization, or exact
// run.create contract are listed.
var boundedRouteContracts = map[routeKey]routeContract{
	{http.MethodGet, "/health"}:                                                           verifiedIdentityRoute,
	{http.MethodGet, "/api/v1/health"}:                                                    verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetHeartbeatProcedure}:             readCapabilityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetHeartbeatInstructionsProcedure}: readCapabilityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceListHeartbeatsProcedure}:           readCapabilityRoute,
	// GetTeam and GetRoles serve target-scoped preparation reads. Their handlers
	// check baseline read or exact team.prepare/team.manage authority before
	// returning data, so bounded callers must reach those owner checks.
	{http.MethodPost, teamsconnect.TeamsServiceGetTeamProcedure}:                         ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceGetRolesProcedure}:                        ownerAuthorizedRoute,
	{http.MethodPost, agentsconnect.AgentsServiceListAgentsProcedure}:                    readCapabilityRoute,
	{http.MethodPost, agentsconnect.AgentsServiceGetAgentProcedure}:                      readCapabilityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceListHeartbeatAttemptsProcedure}:   verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetTeamExecutionStatusProcedure}:  verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetTaskBoardProcedure}:            verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetTeamHeartbeatControlProcedure}: verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetHandoffHistoryProcedure}:       verifiedIdentityRoute,
	// Global heartbeat-control status is operational metadata. It exposes the
	// current pause/policy snapshot; mutations keep their existing owner checks.
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetHeartbeatControlProcedure}:  verifiedIdentityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceListLogsProcedure}:             readCapabilityRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceGetLogProcedure}:               readCapabilityRoute,
	{http.MethodPost, memberflowconnect.MemberflowServiceGetOperatingModelsProcedure}: verifiedIdentityRoute,
	// Action discovery is an ordinary verified read, independent of unrelated grants.
	{http.MethodPost, actionsconnect.ActionsServiceListActionsProcedure}:                 verifiedIdentityRoute,
	{http.MethodPost, actionsconnect.ActionsServiceGetActionProcedure}:                   verifiedIdentityRoute,
	{http.MethodPost, skillsconnect.SkillsServiceReadSkillsProcedure}:                    verifiedIdentityRoute,
	{http.MethodPost, skillsconnect.SkillsServiceListSkillsProcedure}:                    verifiedIdentityRoute,
	{http.MethodPost, skillsconnect.SkillsServiceGetSkillProcedure}:                      verifiedIdentityRoute,
	{http.MethodPost, skillsconnect.SkillsServiceSyncSkillsProcedure}:                    verifiedIdentityRoute,
	{http.MethodPost, skillsconnect.SkillsServiceRefreshProjectionProcedure}:             verifiedIdentityRoute,
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
	{http.MethodPost, heartbeatconnect.HeartbeatServiceCreateHeartbeatProcedure}:          ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceUpdateHeartbeatProcedure}:          ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceDeleteHeartbeatProcedure}:          ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceSetHeartbeatInstructionsProcedure}: ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceSetResponsibilitiesProcedure}:      ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceClearTeamQueueRunningProcedure}:    ownerAuthorizedRoute,
	// Selected-team scheduling control is exact team.manage; global control and
	// policy writes retain their existing human-only contracts.
	{http.MethodPost, heartbeatconnect.HeartbeatServicePauseTeamHeartbeatControlProcedure}:  ownerAuthorizedRoute,
	{http.MethodPost, heartbeatconnect.HeartbeatServiceResumeTeamHeartbeatControlProcedure}: ownerAuthorizedRoute,
	// Manual member dispatch is explicitly governed by the existing run.create
	// operation. The handler separately enforces CreateRun caller attribution.
	{http.MethodPost, heartbeatconnect.HeartbeatServiceTriggerHeartbeatProcedure}: runCreateAuthorizedRoute,
	// Team creation is routed to FileTeamStore.Create, which enforces its exact
	// team.manage operation and prompt-manager:team/<id> target when bounded.
	{http.MethodPost, teamsconnect.TeamsServiceCreateTeamProcedure}: ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceUpdateTeamProcedure}: ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceSetRolesProcedure}:   ownerAuthorizedRoute,
	// Agent creation is a distinct catalog operation, enforced by
	// FileAgentStore.Create against prompt-manager:agent/<id>.
	{http.MethodPost, agentsconnect.AgentsServiceCreateAgentProcedure}: ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceAddMemberProcedure}:     ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceUpdateMemberProcedure}:  ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceRemoveMemberProcedure}:  ownerAuthorizedRoute,
	{http.MethodPost, teamsconnect.TeamsServiceSendMessageProcedure}:   verifiedIdentityRoute,
	// Knowledge replacement has no reviewed operation-grant contract. Keep it
	// closed for bounded callers; ordinary caller authorization is unchanged.
}

func contractForBoundedRoute(method, path string) routeContract {
	if contract, ok := boundedRouteContracts[routeKey{method: method, path: path}]; ok {
		return contract
	}
	if isLegacyHeartbeatInstructionsPath(path) {
		switch method {
		case http.MethodGet:
			return readCapabilityRoute
		case http.MethodPut:
			return ownerAuthorizedRoute
		}
	}
	return denyBoundedRoute
}

func isLegacyHeartbeatInstructionsPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/api/v1/"):
		path = strings.TrimPrefix(path, "/api/v1")
	case strings.HasPrefix(path, "/teams/"):
	default:
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	return len(parts) == 5 && parts[0] == "teams" && parts[1] != "" &&
		parts[2] == "members" && parts[3] != "" && parts[4] == "heartbeat-instructions"
}

// PreparationBoundary is deliberately a closed allowlist. The preparation reads
// inspect a draft and its explicit amendment carrier; ReadSkills, ListKnowledge
// and prompt previews are ordinary verified reads, independent of unrelated
// grants. AddKnowledge and CloseKnowledge are exact attributed, append-only
// bookkeeping writes; their owners continue validating attribution, team,
// payload, closure reason and entry existence. Other knowledge mutations remain
// closed here, including UpdateKnowledge. Ordinary caller handling is unchanged;
// no reviewed operation grant currently authorizes corpus replacement.
// Heartbeat mutations enforce their exact team.manage operation/target before
// writes when bounded. TriggerHeartbeat requires the existing agent-manager
// run.create operation before the Prompt Manager handler may use
// its internal dispatch path. Unknown APIs, legacy paths and future RPCs are refused
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
		case runCreateAuthorizedRoute:
			_, err := authn.RequireOperation(r.Context(), "agent-manager:write", "run.create", "agent-manager:*")
			allowed = err == nil
		}
		if !allowed {
			denial := authn.ErrOperation
			if contract == denyBoundedRoute {
				denial = ErrNoBoundedRouteContract
			}
			err := connect.NewError(connect.CodePermissionDenied, denial)
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
