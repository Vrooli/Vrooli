package handlers

import (
	"errors"
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/owneridentity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

// defaultAskedStandingGrantConcurrency is the concurrency bound an operator is
// asked to approve for a service principal's standing grant raised via the
// "ask, don't block" path. It is stated in the notification so a one-tap approve
// is informed; the operator can later tune it by re-issuing the grant.
const defaultAskedStandingGrantConcurrency = 3

// serviceNeedsApproval handles the needs-approval outcome for a verified service
// caller: it raises an operator decision (idempotent, per-subject capped) and
// answers with a typed "approval pending" (202). A service principal is a
// recurring, subject-keyed caller, so the ask proposes a bounded STANDING grant.
// Without a configured raiser it degrades to the prior hard refuse (403).
func serviceNeedsApproval(w http.ResponseWriter, r *http.Request, subject string) {
	raiser, ok := ApprovalRaiserFromContext(r.Context())
	if !ok {
		writeServiceAdmissionRefusal(w, http.StatusForbidden, "approval_raiser_unavailable")
		return
	}
	raised, err := raiser.RaiseAuthorizationApproval(r.Context(), orchestration.AuthorizationApprovalRequest{
		Subject:           subject,
		Operation:         "run.create",
		Target:            RunCreationTarget,
		Kind:              domain.PendingAuthorizationKindStanding,
		MaxConcurrentRuns: defaultAskedStandingGrantConcurrency,
	})
	if errors.Is(err, orchestration.ErrAuthorizationFlood) {
		// Over the per-subject outstanding-ask cap: refuse WITHOUT raising another
		// operator notification, so a buggy or hostile caller cannot flood.
		writeServiceAdmissionRefusal(w, http.StatusTooManyRequests, "approval_raise_limited")
		return
	}
	if err != nil || !raised {
		// A raise failure (or no raiser) is a refusal, not an accidental allow.
		writeServiceAdmissionRefusal(w, http.StatusForbidden, "approval_raise_failed")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "approval_pending",
		"subject": subject,
		"detail":  "operator approval requested; the operation proceeds once approved",
	})
}

// writeServiceAdmissionRefusal adds only a stable category to the existing
// task/run refusal response. Keep this allowlist local and messages generic:
// the response must never expose grant, usage, or wrapped store details.
func writeServiceAdmissionRefusal(w http.ResponseWriter, status int, reason string) {
	switch reason {
	case "grant_limit_reached", "authority_state_unavailable", "authority_unavailable",
		"approval_raiser_unavailable", "approval_raise_failed", "approval_raise_limited",
		"operation_not_admitted":
	default:
		reason = "operation_not_admitted"
	}
	writeJSON(w, status, map[string]string{
		"error":            "service authority does not include this operation",
		"admission_reason": reason,
	})
}

// AgentOperationMiddleware prevents coarse write eligibility from opening
// global management to agents. Permitted mutations still pass their existing
// exact-target, profile, lineage and budget checks in the domain handlers.
func AgentOperationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (len(r.Header.Values(supervision.ExactProofHeader)) > 0 || len(r.Header.Values(supervision.ExactSignatureHeader)) > 0) && (r.Method != http.MethodPost || (r.URL.Path != apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure && r.URL.Path != ExactReceiptPath)) {
			http.Error(w, "exact proof offered on wrong operation channel", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && localLoopbackOnlyRoute[r.URL.Path] {
			// G2 contract: transcript CONTINUITY is a same-host, zero-authority
			// operation, mounted LoopbackOnly exactly like /api/v1/runs/attach.
			//   - /api/v1/runs/attach: local attachment establishes a zero-scope
			//     identity, never consent or delegated authority. Launcher
			//     attachment must remain usable before the owner independently
			//     approves a session grant.
			//   - /api/v1/runs/import-transcript: web-console hands agent-manager a
			//     LOCAL transcript file path for its own session (a filesystem path
			//     only meaningful on the shared host); agent-manager reads the native
			//     transcript and owns the resulting search projection. No Web Console
			//     catalog or raw event text crosses, and it creates NO launch or
			//     attribution dependency — only transcript continuity.
			//   - /api/v1/runs/external/tombstone: the idempotent delete of that same
			//     projection by (sourceHarness, sourceSessionId).
			// web-console holds no agent-manager service credential (verified: it
			// sends only Content-Type today), so forwarding one is not an option; and
			// these operations are not a costly write a standing grant governs. The
			// loopback fence is the honest authority: a non-loopback caller is
			// refused. If agent-manager is remote/unavailable the publish fails
			// gracefully at the caller, which is acceptable for continuity.
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
		// The declarations plan handler always forces DryRun, regardless of the
		// request body. Permit a verified agent to use this exact read/validation
		// route without opening the mutating reconciliation endpoint or global
		// management. Other principals retain their existing admission path.
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/declarations/plan" && ok && p.Kind == identity.ActorAgent && p.IsVerified() {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.URL.Path == "/api/v1/identity/verify" || (r.Method == http.MethodPost && agentReadPOSTRoute[r.URL.Path]) {
			next.ServeHTTP(w, r)
			return
		}
		// A verified service principal carries no capability of its own for a
		// costly write; its sole admission is an operator-approved standing grant,
		// sourced durably by StandingGrantMiddleware and evaluated through the same
		// RequestOperationAuthority seam an agent grant uses. run.create is the only
		// operation a standing grant admits today; every other service write falls
		// through to the refusal below. No grant logic lives here — this delegates.
		if ok && p.Kind == identity.ActorService {
			// A standing grant authorizes the heartbeat's task+run as ONE unit: the
			// unattended heartbeat creates its task first, then the run it carries.
			// Both legs are admitted under the same run.create authority (a task is
			// an inert precursor with no authority of its own), evaluated through
			// the same seam — no grant logic here. The run admission still enforces
			// the grant's limits and attributes the owner.
			if r.Method == http.MethodPost && (r.URL.Path == "/api/v1/runs" || r.URL.Path == "/api/v1/tasks") {
				switch _, err := authn.RequireOperation(r.Context(), "agent-manager:write", "run.create", RunCreationTarget); {
				case err == nil:
					next.ServeHTTP(w, r)
					return
				case errors.Is(err, errStandingGrantStateUnavailable):
					writeServiceAdmissionRefusal(w, http.StatusForbidden, "authority_state_unavailable")
					return
				case errors.Is(err, authn.ErrNeedsApproval):
					// Ask, don't block: a verified known caller with no covering grant.
					// Raise an operator decision (idempotent; capped per subject) and
					// return a typed "approval pending" (202), never a hard block. The
					// caller's next attempt proceeds once the operator approves.
					serviceNeedsApproval(w, r, p.Subject)
					return
				case errors.Is(err, authn.ErrStandingGrantLimit):
					writeServiceAdmissionRefusal(w, http.StatusForbidden, "grant_limit_reached")
					return
				case errors.Is(err, authn.ErrOperation):
					if _, present := authn.RequestOperationAuthorityFromContext(r.Context()); !present {
						writeServiceAdmissionRefusal(w, http.StatusForbidden, "authority_unavailable")
						return
					}
					writeServiceAdmissionRefusal(w, http.StatusForbidden, "operation_not_admitted")
					return
				default:
					writeServiceAdmissionRefusal(w, http.StatusForbidden, "operation_not_admitted")
					return
				}
			}
			http.Error(w, "service authority does not include this operation", http.StatusForbidden)
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
				// ONE authority decision for create-run absence. This middleware and
				// the CreateRun handler (runs_create.go) must never disagree about an
				// anonymous unattended create-run: both read the SAME AUTH-01 switch.
				// With the switch OFF (operator decision P-18) an anonymous create-run
				// is admitted to the handler, which applies the pre-AUTH-01 anonymous
				// path; with it ON, both refuse. Previously this gate 401'd
				// unconditionally, so the switch never took effect for this path (the
				// P1 root cause). The heartbeat's task leg rides the same decision:
				// task+run is one unit. Every OTHER anonymous operation is still
				// refused below. Attributed callers (human/agent/service-standing) are
				// handled in their own branches above and never reach this.
				if r.Method == http.MethodPost && (r.URL.Path == "/api/v1/runs" || r.URL.Path == "/api/v1/tasks") && !owneridentity.CreateRunCallerEnforced() {
					next.ServeHTTP(w, r)
					return
				}
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
	if r.Method == http.MethodPut {
		// The task handler admits only the verified creator orchestrator's exact
		// queued, undispatched brief update. It does not expose task management.
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
		_, err := uuid.Parse(id)
		return id != r.URL.Path && err == nil
	}
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case ExactChallengePath, ExactReadChallengePath, ExactReceiptPath, "/api/v1/runs", "/api/v1/investigations", "/api/v1/runs/episode-cohort/compare":
		return true
	case "/api/v1/runs/quiesce":
		// The default drain only observes active runs and waits; it does not
		// change run state. The handler verifies any supplied run credential and
		// retains the root-orchestrator fence for Force=true.
		return true
	case "/api/v1/workflow-executions":
		// Starting a workflow has its own owner policy: the workflow definition's
		// trigger policy is checked by StartWorkflowExecution before admission.
		// Workflow child runs are dispatched through the admitted workflow
		// lifecycle, not as public CreateRun calls. Keep every other workflow
		// lifecycle route outside this exact start exception.
		return true
	case "/api/v1/storage/evidence/prune":
		// A dry run only previews. A real prune from an agent never removes on
		// its own authority: the handler raises an operator decision and Agent
		// Manager applies exactly the previewed prune once the operator approves.
		return true
	case "/api/v1/runs/wake-by-key":
		// The handler applies the single-run wake lineage rule to every
		// parked run on the key (a worker waking its orchestrator).
		return true
	case "/api/v1/tasks":
		// Task creation is the inert first leg of the run.create unit. The
		// handler decides whether the caller is a live profile orchestrator or
		// has the exact canonical run.create authority. Do not classify a
		// principal's mere possession of an unrelated grant as a refusal here.
		return true
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
	case apiconnect.AgentManagerServiceEnrollEffortProcedure:
		// The handler checks exact delivery authority or the current team owner
		// for a team-backed enrollment. No coarse write scope admits it.
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
	case "/api/v1/workflow-executions/{id}/wait":
		// Waiting observes durable workflow state and never advances, signals, or
		// cancels the execution. It is the REST counterpart of the read-only
		// WaitWorkflowExecution Connect procedure. Keep adjacent POST actions gated.
		return true
	case "/api/v1/runs/{id}/stop", "/api/v1/runs/{id}/continue", "/api/v1/runs/{id}/wake", "/api/v1/runs/{id}/recover", "/api/v1/runs/{id}/park",
		"/api/v1/tasks/{id}/cancel",
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

// localLoopbackOnlyRoute lists the same-host, zero-authority routes a co-located
// caller reaches. They are mounted LoopbackOnly: the loopback fence is their
// authority, so a non-loopback caller is refused. /api/v1/runs/attach is local
// launcher attachment; import-transcript and external/tombstone are transcript
// continuity (see AgentOperationMiddleware for the full G2 contract). None carry
// launch or attribution coupling.
var localLoopbackOnlyRoute = map[string]bool{
	"/api/v1/runs/attach":             true,
	"/api/v1/runs/import-transcript":  true,
	"/api/v1/runs/external/tombstone": true,
}

// Reviewed POST read routes from the current owner contract. Connect RPCs and
// standalone REST reads are listed by their exact route; unknown operations do
// not inherit read admission merely from their method or name.
var agentReadPOSTRoute = map[string]bool{
	"/api/v1/permission-policy/doctor":                                       true,
	"/api/v1/permission-policy/plan":                                         true,
	"/api/v1/permission-policy/validate":                                     true,
	"/api/v1/role-policy/explain":                                            true,
	"/api/v1/role-policy/validate":                                           true,
	apiconnect.AgentManagerServiceExplainRolePolicyProcedure:                 true,
	apiconnect.AgentManagerServiceDoctorPermissionPolicyProcedure:            true,
	apiconnect.AgentManagerServicePlanPermissionPolicyProcedure:              true,
	apiconnect.AgentManagerServiceValidatePermissionPolicyCatalogProcedure:   true,
	apiconnect.AgentManagerServiceValidateRolePolicyCatalogProcedure:         true,
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
	"/agent_manager.v1.ConversationSearchService/SearchConversations":        true,
	"/agent_manager.v1.EpisodesService/GetCrossScenarioLedger":               true,
	"/agent_manager.v1.EpisodesService/GetEpisodes":                          true,
	"/agent_manager.v1.EpisodesService/GetSelfReportSpans":                   true,
}
