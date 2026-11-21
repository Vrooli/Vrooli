package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
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
	"github.com/vrooli/api-core/provenance"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	permissionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions/permissionsv1connect"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	if isConnectRunCreationProcedure(r.URL.Path) {
		writeConnectApprovalPending(w, subject)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "approval_pending",
		"subject": subject,
		"detail":  "operator approval requested; the operation proceeds once approved",
	})
}

func isConnectRunCreationProcedure(path string) bool {
	return path == apiconnect.AgentManagerServiceCreateTaskProcedure || path == apiconnect.AgentManagerServiceCreateRunProcedure
}

// writeConnectApprovalPending preserves the REST 202 contract while giving
// generated Connect callers a definitive, typed error they can safely map to
// their existing approval-pending outcome.
func writeConnectApprovalPending(w http.ResponseWriter, subject string) {
	detail := &apipb.AuthorizationApprovalPending{
		Subject: subject,
		Detail:  "operator approval requested; the operation proceeds once approved",
	}
	encoded, err := proto.Marshal(detail)
	if err != nil {
		http.Error(w, "failed_precondition: approval pending", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    "failed_precondition",
		"message": "approval pending",
		"details": []map[string]string{{
			"type":  "type.googleapis.com/agent_manager.v1.AuthorizationApprovalPending",
			"value": base64.StdEncoding.EncodeToString(encoded),
		}},
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
		if r.Method == http.MethodPost && r.URL.Path == HookReceiptPath && ok && p.Kind == identity.ActorAgent && p.IsVerified() {
			verified := provenance.FromContext(r.Context())
			if verified.IsVerifiedAgent() && verified.RunID == strings.TrimSpace(p.Subject) && provenance.ForwardedIdentityToken(r.Context()) != "" {
				// This append-only receipt route is run self-evidence, not a global
				// management operation. The handler still requires the one-use
				// owner-held launch credential and exact run/session binding.
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "verified current run identity required", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == apiconnect.AgentManagerServiceDetachRunProcedure && ok && p.Kind == identity.ActorAgent {
			runID, valid := connectDetachRunID(r)
			if !p.IsVerified() || p.Subject == "" || strings.Contains(p.Subject, "/") || !valid || runID != p.Subject {
				http.Error(w, "current verified self identity required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		// The declarations plan handler always forces DryRun, regardless of the
		// request body. Permit a verified agent to use this exact read/validation
		// route without opening the mutating reconciliation endpoint or global
		// management. Other principals retain their existing admission path.
		if r.Method == http.MethodPost && r.URL.Path == apiconnect.AgentManagerServicePlanScenarioDeclarationsProcedure && ok && p.Kind == identity.ActorAgent && p.IsVerified() {
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
		// RequestOperationAuthority seam an agent grant uses. Run creation and two
		// narrowly named one-run evidence routes are the only service writes admitted
		// here; each route still checks its exact operation, target and owner evidence.
		// No grant logic lives here — this delegates.
		if ok && p.Kind == identity.ActorService {
			// Web Console may ask Agent Manager to resolve one browser pane for a
			// human-owned run. The handler independently verifies the dedicated
			// service role, asserted owner, exact Web Console session/thread, and
			// saved BrowserCapability before BAS is contacted.
			if r.Method == http.MethodPost && (r.URL.Path == "/api/v1/browser/pane/resolve" || r.URL.Path == "/api/v1/browser/pane/sessions") && p.IsVerified() && isBrowserPaneDelegator(p) {
				next.ServeHTTP(w, r)
				return
			}
			// Treasury may submit immutable run-bound research budget evidence.
			// The endpoint still requires the exact service identity, capability,
			// operation grant, and run target; this exception only lets it reach
			// those handler checks instead of granting a general service write.
			if isResearchBudgetIngestRoute(r) && p.IsVerified() && p.Subject == orchestration.TreasuryServiceSubject {
				next.ServeHTTP(w, r)
				return
			}
			// BAS may reach only the run-specific evidence authorization request.
			// The handler independently checks its signed run token, persisted browser
			// capability, opaque grant receipt, and exact one-run operation grant.
			if (isBrowserEvidenceAuthorizationRoute(r) || isBrowserEvidenceBrokerIngestRoute(r)) && p.IsVerified() && p.Subject == orchestration.BrowserEvidenceServiceSubject {
				next.ServeHTTP(w, r)
				return
			}
			// LPBS may resolve one exact-run BAS proof for its existing settlement
			// reservation through this dedicated route only. The handler verifies the
			// exact service subject and narrower settlement-attestation scope.
			if isBrowserSettlementAttestationRoute(r) && p.IsVerified() && p.Subject == browserSettlementAttestationSubject {
				next.ServeHTTP(w, r)
				return
			}
			// A standing grant authorizes the heartbeat's task+run as ONE unit: the
			// unattended heartbeat creates its task first, then the run it carries.
			// Both legs are admitted under the same run.create authority (a task is
			// an inert precursor with no authority of its own), evaluated through
			// the same seam — no grant logic here. The run admission still enforces
			// the grant's limits and attributes the owner.
			if r.Method == http.MethodPost && isRunCreationPath(r.URL.Path) {
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
		if ok && p.Kind == identity.ActorAgent && p.IsVerified() && r.Method == http.MethodPost && r.URL.Path == "/api/v1/browser/record-evidence" {
			// This route is explicitly run-agent self-report for transcript continuity.
			// It is not BAS evidence, proof of a live browser session or billing input;
			// the handler still requires the token subject and context principal match.
			next.ServeHTTP(w, r)
			return
		}
		// A run may attach an observed free comparator only to its own
		// previously retained settlement. The handler verifies this exact run
		// binding and the ledger's spend foreign key; it cannot write a budget or
		// settlement. This is run-result evidence, not spend authority.
		if r.Method == http.MethodPost && isResearchBudgetFreeAlternativeRoute(r) && ok && p.Kind == identity.ActorAgent && p.IsVerified() {
			runID := researchBudgetFreeAlternativeRunID(r)
			if runID != "" && p.Subject == runID {
				if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
					http.Error(w, "run write capability required", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "verified run identity must match the research budget run", http.StatusForbidden)
			return
		}
		startEligible, startReason := isGitControlTowerWorkflowStart(r)
		if r.URL.Path == apiconnect.AgentManagerServiceStartWorkflowExecutionProcedure {
			log.Printf("workflow-start admission principal_present=%t principal_kind=%s principal_verified=%t gct_fallback=%t reason=%s", ok, p.Kind, p.IsVerified(), startEligible, startReason)
		}
		if (!ok || !p.IsVerified() || p.Kind == identity.ActorConflict) && startEligible {
			// This message-only path deliberately fails open to programmatic when
			// identity verification is missing, invalid, or unavailable, including
			// when middleware supplied an unverified agent-shaped principal. Proxies
			// and container networking make peer addresses unreliable. Keep admission
			// bounded to the exact workflow pair, strip identity claims, and retain
			// trigger and input policy checks.
			r.Header.Del("X-Vrooli-Workflow-Initiator")
			r.Header.Del("X-Agent-Identity-Token")
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
				if r.Method == http.MethodPost && isRunCreationPath(r.URL.Path) && !owneridentity.CreateRunCallerEnforced() {
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
		http.Error(w, "agent authority does not include global management or approval; run review (run approve, run reject), an investigation's approved recommendations and the workspace sandbox's review result are operator decisions with no agent grant, answered in Agent Manager or the operator's decision inbox", http.StatusForbidden)
	})
}

func isBrowserEvidenceAuthorizationRoute(r *http.Request) bool {
	if r == nil || r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 6 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "browser" || parts[3] != "evidence" || parts[4] != "authorize" {
		return false
	}
	_, err := uuid.Parse(parts[5])
	return err == nil
}

func isBrowserEvidenceBrokerIngestRoute(r *http.Request) bool {
	if r == nil || r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "browser" || parts[3] != "bas-evidence" {
		return false
	}
	_, err := uuid.Parse(parts[4])
	return err == nil
}

func isBrowserSettlementAttestationRoute(r *http.Request) bool {
	if r == nil || r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "browser" || parts[3] != "settlement-attestation" {
		return false
	}
	_, err := uuid.Parse(parts[4])
	return err == nil
}

func isResearchBudgetIngestRoute(r *http.Request) bool {
	if r == nil || r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "research-budgets" && (parts[4] == "snapshot" || parts[4] == "settlements") {
		_, err := uuid.Parse(parts[3])
		return err == nil
	}
	if len(parts) != 7 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "research-budgets" || parts[4] != "settlements" || parts[5] == "" || parts[6] != "free-alternative" {
		return false
	}
	_, err := uuid.Parse(parts[3])
	return err == nil
}

func isResearchBudgetFreeAlternativeRoute(r *http.Request) bool {
	if r == nil || r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 7 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "research-budgets" || parts[4] != "settlements" || parts[5] == "" || parts[6] != "free-alternative" {
		return false
	}
	_, err := uuid.Parse(parts[3])
	return err == nil
}

func researchBudgetFreeAlternativeRunID(r *http.Request) string {
	if !isResearchBudgetFreeAlternativeRoute(r) {
		return ""
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	parsed, err := uuid.Parse(parts[3])
	if err != nil {
		return ""
	}
	return parsed.String()
}

// isRunCreationPath names the two legs of run creation (the task, then the
// run) on both transports; every admission rule for one applies to the other.
func isRunCreationPath(path string) bool {
	switch path {
	case "/api/v1/runs", "/api/v1/tasks", apiconnect.AgentManagerServiceCreateRunProcedure, apiconnect.AgentManagerServiceCreateTaskProcedure:
		return true
	}
	return false
}

const localWorkflowStartBodyLimit = 128 << 10

const workflowExecutionCancelByKeyPath = "/api/v1/workflow-executions/cancel-by-idempotency-key"

// isGitControlTowerWorkflowStart recognizes GCT's two message-only drafts.
// Source IP is intentionally excluded: proxies and container networking make
// it brittle. The fallback is limited to the exact owner/workflow pair and
// remains subject to trigger policy, budgets, self-trigger checks, and input
// validation in the workflow owner.
func isGitControlTowerWorkflowStart(r *http.Request) (bool, string) {
	if r == nil || r.Method != http.MethodPost || r.URL.Path != apiconnect.AgentManagerServiceStartWorkflowExecutionProcedure {
		return false, "wrong_method_or_path"
	}
	if r.Body == nil {
		return false, "missing_body"
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, localWorkflowStartBodyLimit+1))
	r.Body = io.NopCloser(bytes.NewReader(data))
	if err != nil {
		return false, "body_read_error"
	}
	if len(data) == 0 {
		return false, "empty_body"
	}
	if len(data) > localWorkflowStartBodyLimit {
		return false, "body_too_large"
	}
	var request apipb.StartWorkflowExecutionRequest
	if err := unmarshalConnectProtoRequest(r.Header.Get("Content-Type"), data, &request); err != nil {
		return false, "invalid_workflow_start_encoding"
	}
	if request.GetOwner() != "git-control-tower" {
		return false, "wrong_workflow_owner"
	}
	switch request.GetWorkflowKey() {
	case "git-control-tower/staged-message", "git-control-tower/staged-message-deep":
		return true, "eligible"
	default:
		return false, "wrong_workflow_key"
	}
}

// isGitControlTowerWorkflowCancelByKey keeps this internal lifecycle route
// scoped to the two workflows owned by Git Control Tower. The body is restored
// because the handler must perform its own strict decoding and validation.
func isGitControlTowerWorkflowCancelByKey(r *http.Request) bool {
	if r == nil || r.Method != http.MethodPost || r.URL.Path != workflowExecutionCancelByKeyPath || r.Body == nil {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, localWorkflowStartBodyLimit+1))
	r.Body = io.NopCloser(bytes.NewReader(data))
	if err != nil || len(data) == 0 || len(data) > localWorkflowStartBodyLimit {
		return false
	}
	var request struct {
		Owner       string `json:"owner"`
		WorkflowKey string `json:"workflowKey"`
	}
	if err := json.Unmarshal(data, &request); err != nil || request.Owner != "git-control-tower" {
		return false
	}
	switch request.WorkflowKey {
	case "git-control-tower/staged-message", "git-control-tower/staged-message-deep":
		return true
	default:
		return false
	}
}

func unmarshalConnectProtoRequest(contentType string, data []byte, request proto.Message) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		switch mediaType {
		case "application/proto", "application/grpc+proto", "application/x-protobuf":
			return proto.Unmarshal(data, request)
		case "application/json", "application/proto+json":
			return (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, request)
		}
	}
	// Connect peers may vary their supported media type. Decode only the two
	// protobuf wire formats; eligibility still requires the exact typed owner
	// and workflow fields below.
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, request); err == nil {
		return nil
	}
	return proto.Unmarshal(data, request)
}

func agentOperationRoute(r *http.Request, interactive bool) bool {
	if r.URL.Path == apiconnect.AgentManagerServiceContinueRunProcedure {
		// ContinueRun has a generated unary route as well as the legacy REST route.
		// Admission only lets a verified agent reach the handler; the existing
		// lineage/self-operation checks remain the authority for its target.
		return true
	}
	if r.Method == http.MethodDelete {
		// Run and task deletion each have target-specific policy checks in their
		// handlers. Other DELETE routes remain outside agent management.
		route := mux.CurrentRoute(r)
		if route == nil {
			return false
		}
		template, _ := route.GetPathTemplate()
		return template == "/api/v1/runs/{id}" || template == "/api/v1/tasks/{id}"
	}
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
	if strings.HasPrefix(r.URL.Path, "/api/v1/annotations/") {
		// Annotations are derived evidence, not run state or an operator
		// decision. Writing them does not approve, stop or re-task a run, so an
		// admitted agent may run a bounded annotation pass. The handler still
		// authenticates the caller and the annotation store is scoped to the run
		// ids the caller supplied or selected.
		return true
	}
	switch r.URL.Path {
	case apiconnect.AgentManagerServiceStartWorkflowExecutionProcedure:
		// The revision's trigger policy is the admission authority for starts.
		return true
	case workflowExecutionCancelByKeyPath:
		return isGitControlTowerWorkflowCancelByKey(r)
	case ExactChallengePath, ExactReadChallengePath, ExactReceiptPath, "/api/v1/runs", "/api/v1/runs/resume-from-failed", "/api/v1/investigations",
		apiconnect.AgentManagerServiceCreateRunProcedure:
		return true
	case "/api/v1/runs/stop-all":
		if _, ok := ownerScopedStopAll(r.Context()); ok {
			return true
		}
		// Global management stays closed to agents except under an approved
		// agent.manage grant covering every run (agent-manager:run/* or
		// agent-manager:*). The handler re-checks the same rule.
		_, err := authn.RequireOperation(r.Context(), "agent-manager:write", "agent.manage", "agent-manager:run/*")
		return err == nil
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
	case "/api/v1/usage/modes/ask":
		// Only raises an operator decision; a mode starts when a verified
		// human answers it, never on the caller's authority.
		return true
	case "/api/v1/runs/wake-by-key":
		// The handler applies the single-run wake lineage rule to every
		// parked run on the key (a worker waking its orchestrator).
		return true
	case "/api/v1/tasks", apiconnect.AgentManagerServiceCreateTaskProcedure:
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
	case "/api/v1/runs/{id}/stop", "/api/v1/runs/{id}/workload", "/api/v1/runs/tag/{tag}/stop", "/api/v1/runs/{id}/continue", "/api/v1/runs/{id}/wake", "/api/v1/runs/{id}/recover", "/api/v1/runs/{id}/park",
		"/api/v1/runs/{id}/messages/{event_id}/delete",
		"/api/v1/tasks/{id}/cancel",
		"/api/v1/investigations/{id}/result", "/api/v1/investigations/{id}/learning-retry":
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
	"/api/v1/runs/attach":                            true,
	apiconnect.AgentManagerServiceAttachRunProcedure: true,
	"/api/v1/runs/import-transcript":                 true,
	"/api/v1/runs/external/tombstone":                true,
}

// connectDetachRunID reads only enough of the generated request to apply the
// same self-target fence as the legacy REST path, then restores the body for
// Connect's decoder. The owner re-verifies the run-bound identity token.
func connectDetachRunID(r *http.Request) (string, bool) {
	if r == nil || r.Body == nil {
		return "", false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 16<<10))
	r.Body = io.NopCloser(bytes.NewReader(data))
	if err != nil || len(data) == 0 || len(data) >= 16<<10 {
		return "", false
	}
	var request apipb.DetachRunRequest
	if err := unmarshalConnectProtoRequest(r.Header.Get("Content-Type"), data, &request); err != nil {
		return "", false
	}
	runID := strings.TrimSpace(request.GetRunId())
	if _, err := uuid.Parse(runID); err != nil {
		return "", false
	}
	return runID, true
}

// Reviewed POST read routes from the current owner contract. Connect RPCs and
// standalone REST reads are listed by their exact route; unknown operations do
// not inherit read admission merely from their method or name. Read-only
// services are admitted whole from their generated descriptor (see
// readOnlyServiceRoutes) instead of being listed here.
var agentReadPOSTRoute = withReadOnlyServiceRoutes(map[string]bool{
	permissionsv1connect.PermissionProjectionServiceListPermissionProjectionProcedure: true,
	permissionsv1connect.PermissionProjectionServiceReadPermissionProjectionProcedure: true,
	apiconnect.AgentManagerServiceExplainRolePolicyProcedure:                          true,
	apiconnect.AgentManagerServiceDoctorPermissionPolicyProcedure:                     true,
	apiconnect.AgentManagerServicePlanPermissionPolicyProcedure:                       true,
	apiconnect.AgentManagerServiceValidatePermissionPolicyCatalogProcedure:            true,
	apiconnect.AgentManagerServiceValidateRolePolicyCatalogProcedure:                  true,
	"/agent_manager.v1.AgentManagerService/GetCohortWatch":                            true,
	"/agent_manager.v1.AgentManagerService/GetEffortBoard":                            true,
	"/agent_manager.v1.AgentManagerService/GetInvestigation":                          true,
	"/agent_manager.v1.AgentManagerService/GetPermissionPolicyCatalog":                true,
	"/agent_manager.v1.AgentManagerService/GetPermissionPolicyStatus":                 true,
	"/agent_manager.v1.AgentManagerService/GetProfile":                                true,
	"/agent_manager.v1.AgentManagerService/GetRolePolicyCatalog":                      true,
	"/agent_manager.v1.AgentManagerService/GetRolePolicyStatus":                       true,
	"/agent_manager.v1.AgentManagerService/GetRun":                                    true,
	"/agent_manager.v1.AgentManagerService/GetRunAccounting":                          true,
	"/agent_manager.v1.AgentManagerService/GetRunEvents":                              true,
	"/agent_manager.v1.AgentManagerService/GetRunReport":                              true,
	"/agent_manager.v1.AgentManagerService/GetRunTree":                                true,
	"/agent_manager.v1.AgentManagerService/GetSupervisionPolicy":                      true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowExecution":                      true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowExecutionResult":                true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowExecutionTrace":                 true,
	"/agent_manager.v1.AgentManagerService/GetWorkflowRevision":                       true,
	"/agent_manager.v1.AgentManagerService/Health":                                    true,
	"/agent_manager.v1.AgentManagerService/InspectCohortWatch":                        true,
	"/agent_manager.v1.AgentManagerService/ListCohortWatchActions":                    true,
	"/agent_manager.v1.AgentManagerService/ListCohortWatches":                         true,
	"/agent_manager.v1.AgentManagerService/ListEffortDecisions":                       true,
	"/agent_manager.v1.AgentManagerService/ListEffortSignals":                         true,
	"/agent_manager.v1.AgentManagerService/ListEfforts":                               true,
	"/agent_manager.v1.AgentManagerService/ListExecutionOptions":                      true,
	"/agent_manager.v1.AgentManagerService/ListInvestigations":                        true,
	"/agent_manager.v1.AgentManagerService/ListProfiles":                              true,
	"/agent_manager.v1.AgentManagerService/ListRuns":                                  true,
	apiconnect.AgentManagerServiceGetRuntimeActivityProcedure:                         true,
	"/agent_manager.v1.AgentManagerService/ListSupervisionOutcomes":                   true,
	"/agent_manager.v1.AgentManagerService/ListWorkflowExecutionRuns":                 true,
	"/agent_manager.v1.AgentManagerService/ListWorkflowExecutions":                    true,
	"/agent_manager.v1.AgentManagerService/ListWorkflowRevisions":                     true,
	"/agent_manager.v1.AgentManagerService/WaitCohortWatch":                           true,
	"/agent_manager.v1.AgentManagerService/WaitInvestigation":                         true,
	"/agent_manager.v1.AgentManagerService/WaitRun":                                   true,
	"/agent_manager.v1.AgentManagerService/WaitRunChildren":                           true,
	"/agent_manager.v1.AgentManagerService/WaitWorkflowExecution":                     true,
	"/agent_manager.v1.ConversationSearchService/GetConversationContext":              true,
	"/agent_manager.v1.ConversationSearchService/GetConversationIndexStatus":          true,
	"/agent_manager.v1.ConversationSearchService/SearchConversations":                 true,
	"/agent_manager.v1.EpisodesService/GetCrossScenarioLedger":                        true,
})

// readOnlyMeasureRoutes are the measure reads outside the generated service:
// the measures-go registry executes the same compute functions over HTTP.
var readOnlyMeasureRoutes = []string{"/measures/execute"}

// OperatorOnlyMeasureMethods are the MeasuresService procedures that change
// state. They are the only exceptions to the service's read-only contract: the
// middleware does not admit agents to them, and the handlers require a verified
// operator again.
var OperatorOnlyMeasureMethods = map[string]bool{"StartUsageMode": true, "EndUsageMode": true, "ExtendUsageMode": true, "RebaseUsageModeRuns": true, "StartUsageExperiment": true, "ProposeRoleDefault": true, "SetUsageSignals": true}

// withReadOnlyServiceRoutes admits every read procedure of Agent Manager's
// MeasuresService. The service is read-only by contract (measures.proto: "one
// typed, read-only analytical question per RPC"), so its generated descriptor
// is the declaration: a new measure is agent-readable without another list
// edit, and a non-measure RPC can never ride this rule. The usage mode writes
// in OperatorOnlyMeasureMethods are not admitted.
func withReadOnlyServiceRoutes(routes map[string]bool) map[string]bool {
	service := measurepb.File_agent_manager_v1_measures_measures_proto.Services().ByName("MeasuresService")
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		if OperatorOnlyMeasureMethods[string(methods.Get(i).Name())] {
			continue
		}
		routes["/"+string(service.FullName())+"/"+string(methods.Get(i).Name())] = true
	}
	for _, route := range readOnlyMeasureRoutes {
		routes[route] = true
	}
	return routes
}
