package handlers

import (
	"agent-manager/internal/supervision"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lifecycle"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

// AgentOperationMiddleware prevents coarse write eligibility from opening
// global management to agents. Permitted mutations still pass their existing
// exact-target, profile, lineage and budget checks in the domain handlers.
func AgentOperationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (len(r.Header.Values(supervision.ExactProofHeader)) > 0 || len(r.Header.Values(supervision.ExactSignatureHeader)) > 0) && (r.Method != http.MethodPost || (r.URL.Path != apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure && r.URL.Path != ExactReceiptPath)) {
			http.Error(w, "exact proof offered on wrong operation channel", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/api/v1/runs/attach" && r.Method == http.MethodPost {
			// Local attachment establishes a zero-scope identity, never consent
			// or delegated authority. Launcher attachment must remain usable
			// before the owner independently approves a session grant.
			lifecycle.LoopbackOnly(next).ServeHTTP(w, r)
			return
		}
		// Vrooli Events webhooks carry no caller identity; each handler
		// verifies the HMAC signature of the delivery itself.
		if r.Method == http.MethodPost && signedEventWebhookRoute[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		// The local control plane owns this separate transport contract. Its
		// mounted handler retains LoopbackOnly and the maintenance fence.
		switch r.URL.Path {
		case commonconnect.LifecycleMaintenanceServicePrepareProcedure, commonconnect.LifecycleMaintenanceServiceStatusProcedure,
			commonconnect.LifecycleMaintenanceServiceDrainProcedure, commonconnect.LifecycleMaintenanceServiceResumeProcedure:
			next.ServeHTTP(w, r)
			return
		}
		p, ok := identity.PrincipalFromContext(r.Context())
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.URL.Path == "/api/v1/identity/verify" || (r.Method == http.MethodPost && agentReadRPC[r.URL.Path]) {
			next.ServeHTTP(w, r)
			return
		}
		if !ok || p.Kind != identity.ActorAgent {
			if !ok && r.Header.Get("X-Agent-Identity-Token") != "" {
				http.Error(w, "verified agent caller required", http.StatusUnauthorized)
				return
			}
			// Consent preparation remains a local, non-authoritative operation.
			if r.URL.Path == "/api/v1/authorizations/requests" {
				next.ServeHTTP(w, r)
				return
			}
			if !ok {
				http.Error(w, "verified operator required", http.StatusUnauthorized)
				return
			}
			if _, err := authn.RequireIndependentHuman(r.Context()); err != nil {
				http.Error(w, "verified human operator required", http.StatusForbidden)
				return
			}
			if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		// Self cleanup is intrinsic only to this current verified identity. Domain
		// re-verifies the retained leaf and refuses non-attached targets before effects.
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs/"+p.Subject+"/detach" {
			if !p.IsVerified() || p.Subject == "" || strings.Contains(p.Subject, "/") {
				http.Error(w, "current verified self identity required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if agentOperationRoute(r, p.Grant != nil) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "agent authority does not include global management or approval", http.StatusForbidden)
	})
}

func agentOperationRoute(r *http.Request, interactive bool) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case ExactChallengePath, ExactReadChallengePath, ExactReceiptPath, "/api/v1/runs", "/api/v1/investigations", "/api/v1/runs/episode-cohort/compare":
		return true
	case "/api/v1/runs/wake-by-key":
		// The handler applies the single-run wake lineage rule to every
		// parked run on the key (a worker waking its orchestrator).
		return true
	case "/api/v1/tasks":
		// An orchestrator creates each epoch's task before its child run.
		// The handler admits only a live run whose profile declares
		// agent-manager:orchestrate (orchestratorTaskCreateAllowed).
		return !interactive
	case apiconnect.AgentManagerServiceStartInvestigationProcedure, apiconnect.AgentManagerServiceCancelInvestigationProcedure,
		apiconnect.AgentManagerServiceGetInvestigationProcedure, apiconnect.AgentManagerServiceListInvestigationsProcedure,
		apiconnect.AgentManagerServiceWaitInvestigationProcedure, apiconnect.AgentManagerServiceListRunsProcedure,
		apiconnect.AgentManagerServiceListExecutionOptionsProcedure, apiconnect.AgentManagerServiceListEffortsProcedure,
		apiconnect.AgentManagerServiceGetEffortBoardProcedure, apiconnect.AgentManagerServiceHealthProcedure,
		// Effort signal and decision bookkeeping: a signal only wakes a
		// check-in, and marking a decision recorded is fenced on its answer.
		apiconnect.AgentManagerServiceRecordEffortSignalProcedure, apiconnect.AgentManagerServiceMarkEffortDecisionRecordedProcedure:
		return true
	case apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure:
		// An interactive grant reaches the handler, which requires the
		// effort.reconcile operation on that exact effort (effortGrantActor).
		return true
	}
	// Existing native typed workflows retain their separate owner contracts.
	// An interactive grant cannot acquire them through the coarse write flag.
	if !interactive {
		switch r.URL.Path {
		case apiconnect.AgentManagerServiceRequestCohortWatchActionProcedure,
			apiconnect.AgentManagerServiceRaiseEffortDecisionProcedure:
			return true
		}
	}
	route := mux.CurrentRoute(r)
	if route == nil {
		return false
	}
	template, _ := route.GetPathTemplate()
	switch template {
	case "/api/v1/runs/{id}/stop", "/api/v1/runs/{id}/continue", "/api/v1/runs/{id}/wake", "/api/v1/runs/{id}/recover", "/api/v1/runs/{id}/park",
		"/api/v1/investigations/{id}/wait", "/api/v1/investigations/{id}/result", "/api/v1/investigations/{id}/cancel", "/api/v1/investigations/{id}/learning-retry":
		return true
	}
	return false
}

// signedEventWebhookRoute lists the Vrooli Events webhook targets. They are
// authenticated by the delivery signature, not by a caller principal.
var signedEventWebhookRoute = map[string]bool{
	ProgramEventWebhookPath:  true,
	DecisionEventWebhookPath: true,
}

// Reviewed read procedures from the current owner contract. Unknown RPCs
// never inherit read authority merely from their method spelling.
var agentReadRPC = map[string]bool{
	"/agent_manager.v1.AgentManagerService/GetCohortWatch":                   true,
	"/agent_manager.v1.AgentManagerService/GetEffortBoard":                   true,
	"/agent_manager.v1.AgentManagerService/GetInvestigation":                 true,
	"/agent_manager.v1.AgentManagerService/GetPermissionPolicyCatalog":       true,
	"/agent_manager.v1.AgentManagerService/GetPermissionPolicyStatus":        true,
	"/agent_manager.v1.AgentManagerService/GetProfile":                       true,
	"/agent_manager.v1.AgentManagerService/GetRolePolicyCatalog":             true,
	"/agent_manager.v1.AgentManagerService/GetRolePolicyStatus":              true,
	"/agent_manager.v1.AgentManagerService/GetRun":                           true,
	"/agent_manager.v1.AgentManagerService/GetRunAccounting":                 true,
	"/agent_manager.v1.AgentManagerService/GetRunByTag":                      true,
	"/agent_manager.v1.AgentManagerService/GetRunDiff":                       true,
	"/agent_manager.v1.AgentManagerService/GetRunEvents":                     true,
	"/agent_manager.v1.AgentManagerService/GetRunReport":                     true,
	"/agent_manager.v1.AgentManagerService/GetRunnerStatus":                  true,
	"/agent_manager.v1.AgentManagerService/GetSupervisionPolicy":             true,
	"/agent_manager.v1.AgentManagerService/GetTask":                          true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowExecution":             true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowExecutionResult":       true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowExecutionTrace":        true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowRevision":              true,
	"/agent_manager.v1.AgentManagerService/Health":                           true,
	"/agent_manager.v1.AgentManagerService/InspectCohortWatch":               true,
	"/agent_manager.v1.AgentManagerService/ListCohortWatchActions":           true,
	"/agent_manager.v1.AgentManagerService/ListCohortWatches":                true,
	"/agent_manager.v1.AgentManagerService/ListEffortDecisions":              true,
	"/agent_manager.v1.AgentManagerService/ListEffortSignals":                true,
	"/agent_manager.v1.AgentManagerService/ListEfforts":                      true,
	"/agent_manager.v1.AgentManagerService/ListExecutionOptions":             true,
	"/agent_manager.v1.AgentManagerService/ListInvestigations":               true,
	"/agent_manager.v1.AgentManagerService/ListProfiles":                     true,
	"/agent_manager.v1.AgentManagerService/ListRuns":                         true,
	"/agent_manager.v1.AgentManagerService/ListSupervisionOutcomes":          true,
	"/agent_manager.v1.AgentManagerService/ListTasks":                        true,
	"/agent_manager.v1.AgentManagerService/ListWorkflowExecutionRuns":        true,
	"/agent_manager.v1.AgentManagerService/ListWorkflowExecutions":           true,
	"/agent_manager.v1.AgentManagerService/ListWorkflowRevisions":            true,
	"/agent_manager.v1.AgentManagerService/WaitCohortWatch":                  true,
	"/agent_manager.v1.AgentManagerService/WaitInvestigation":                true,
	"/agent_manager.v1.AgentManagerService/WaitWorkflowExecution":            true,
	"/agent_manager.v1.ConversationSearchService/GetConversationContext":     true,
	"/agent_manager.v1.ConversationSearchService/GetConversationIndexStatus": true,
	"/agent_manager.v1.EpisodesService/GetCrossScenarioLedger":               true,
	"/agent_manager.v1.EpisodesService/GetEpisodes":                          true,
	"/agent_manager.v1.EpisodesService/GetSelfReportSpans":                   true,
}
