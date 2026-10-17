package backlog

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// InteractiveGrantMiddleware bounds the new coarse agent eligibility to the
// reviewed operation surface. Object checks remain in the owning handlers.
// Unknown methods default to refusal, including future Connect mutations.
// Retained operator mutations require verified human authority. Coarse agent
// write eligibility alone never opens unrelated legacy mutations.
func InteractiveGrantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := identity.PrincipalFromContext(r.Context())
		if !ok && r.Header.Get("X-Agent-Identity-Token") != "" && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			http.Error(w, "verified agent caller required", http.StatusUnauthorized)
			return
		}
		if interactiveGrantRoute(r) {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := authn.RequireIndependentHuman(r.Context()); err == nil {
			if _, err := authn.RequireCapability(r.Context(), "swarm-manager:write"); err == nil {
				next.ServeHTTP(w, r)
				return
			}
		}
		// A caller cannot drop its agent header and fall back to an anonymous
		// operator on a retained human mutation.
		if !ok {
			http.Error(w, "verified human operator required", http.StatusUnauthorized)
			return
		}

		http.Error(w, "operation is outside the reviewed swarm bookkeeping policy", http.StatusForbidden)
	})
}

func interactiveGrantRoute(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	if r.Method == http.MethodPost {
		switch r.URL.Path {
		case apiconnect.BacklogServiceListItemsProcedure, apiconnect.BacklogServiceGetItemProcedure,
			apiconnect.BacklogServiceUpdateItemProcedure,
			apiconnect.BacklogServiceArchiveTerminalItemsProcedure,
			apiconnect.GoalServiceListGoalsProcedure, apiconnect.GoalServiceListUnwiredGoalsProcedure,
			apiconnect.GoalServiceGetGoalProcedure, apiconnect.GoalServiceGetScopeProcedure,
			apiconnect.GoalServiceListPendingGoalWorkflowsProcedure,
			apiconnect.GoalServiceUpdateGoalProcedure, apiconnect.GoalServiceArchiveGoalProcedure,
			apiconnect.GoalServiceUnarchiveGoalProcedure, apiconnect.GoalServiceAddTargetsProcedure,
			apiconnect.GoalServiceRemoveTargetsProcedure, apiconnect.GoalServiceCreateMilestoneProcedure,
			apiconnect.GoalServiceUpdateMilestoneProcedure, apiconnect.GoalServiceArchiveMilestoneProcedure,
			apiconnect.GoalServiceAssignMilestoneItemsProcedure, apiconnect.GoalServiceUnassignMilestoneItemsProcedure,
			apiconnect.RelatedServiceGetRelatedProcedure, apiconnect.EvidenceServiceListRunProcedure,
			apiconnect.EvidenceServiceListEntityProcedure, apiconnect.TransitionServiceListTransitionsProcedure,
			apiconnect.DevelopmentServiceGetDevelopmentProcedure, apiconnect.DevelopmentServiceGetDevelopmentArtifactProcedure,
			"/api/v1/backlog/next-actions", "/api/v1/backlog/validate-globs", "/api/v1/backlog/export":
			return true
		}
	}
	route := mux.CurrentRoute(r)
	if route == nil {
		return false
	}
	template, err := route.GetPathTemplate()
	if err != nil {
		return false
	}
	switch template {
	case "/api/v1/backlog/{kind}/{name}":
		return r.Method == http.MethodPatch
	case "/api/v1/backlog/{kind}/{name}/archive-item":
		return r.Method == http.MethodPatch || r.Method == http.MethodDelete
	case "/api/v1/goals/{name}":
		return r.Method == http.MethodPut
	case "/api/v1/goals/{name}/archive-item":
		return r.Method == http.MethodPatch
	case "/api/v1/goals/{name}/targets", "/api/v1/goals/{name}/milestones/{milestone}/items":
		return r.Method == http.MethodPost || r.Method == http.MethodDelete
	case "/api/v1/goals/{name}/milestones":
		return r.Method == http.MethodPost
	case "/api/v1/goals/{name}/milestones/{milestone}":
		return r.Method == http.MethodPut || r.Method == http.MethodDelete
	}
	return false
}
