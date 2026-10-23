package handlers

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliutil"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

var lifecycleRefusalCounts = struct {
	sync.Mutex
	byOperation map[string]uint64
	// recent holds the newest refusal times, oldest first, capped at the burst
	// threshold. Health reads only this window, so expected in-run refusals
	// age out instead of latching the scenario unhealthy.
	recent []time.Time
}{byOperation: make(map[string]uint64)}

const (
	lifecycleRefusalBurstThreshold = 10
	lifecycleRefusalBurstWindow    = 5 * time.Minute
)

var lifecycleRefusalNow = time.Now

// LifecycleRefusalFunctionalStatus reports a daemon-shaped burst of refusals
// (a loop retrying a forbidden lifecycle operation) without weakening the
// guard. It recovers once the burst leaves the window.
func LifecycleRefusalFunctionalStatus() (healthy bool, reason string) {
	lifecycleRefusalCounts.Lock()
	defer lifecycleRefusalCounts.Unlock()
	recent := lifecycleRefusalCounts.recent
	if len(recent) < lifecycleRefusalBurstThreshold || lifecycleRefusalNow().Sub(recent[0]) > lifecycleRefusalBurstWindow {
		return true, ""
	}
	return false, fmt.Sprintf("%d run-identity lifecycle refusals within %s", len(recent), lifecycleRefusalBurstWindow)
}

func recordLifecycleRefusal(operation string) {
	lifecycleRefusalCounts.Lock()
	defer lifecycleRefusalCounts.Unlock()
	lifecycleRefusalCounts.byOperation[operation]++
	recent := append(lifecycleRefusalCounts.recent, lifecycleRefusalNow())
	if len(recent) > lifecycleRefusalBurstThreshold {
		recent = recent[len(recent)-lifecycleRefusalBurstThreshold:]
	}
	lifecycleRefusalCounts.recent = recent
}

// LifecycleRefusalCount returns the number of valid run-identity refusals for
// an operation. It is intentionally small and process-local; the structured
// refusal log is the durable handoff to the platform metrics collector.
func LifecycleRefusalCount(operation string) uint64 {
	lifecycleRefusalCounts.Lock()
	defer lifecycleRefusalCounts.Unlock()
	return lifecycleRefusalCounts.byOperation[operation]
}

// denyRunInitiatedLifecycleOperation refuses lifecycle work that a verified
// run identity alone does not authorize. A current covering run.create grant
// permits creation; a covering agent.manage grant permits the named lifecycle
// operations below. Separately checked lineage routes remain bounded to their
// caller and target. Identity alone grants none of those operations.
// Operator requests carry no run identity token and continue unchanged.
// A presented but unverified credential is refused, never treated as absence.
func (h *Handler) denyRunInitiatedLifecycleOperation(w http.ResponseWriter, r *http.Request, operation string) bool {
	return h.denyRunInitiatedLifecycleOperationWithGuidance(w, r, operation, "")
}

func (h *Handler) denyRunInitiatedLifecycleOperationWithGuidance(w http.ResponseWriter, r *http.Request, operation, guidance string) bool {
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" {
		return false
	}

	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil {
		writeError(w, r, err)
		return true
	}
	if verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "run identity is invalid or unavailable",
		})
		return true
	}
	if operation == "create-run" {
		// A top-level run.create grant is distinct from agent.manage lifecycle
		// authority. The orchestration admission path re-verifies the same proof
		// and records grant-bounded owner attribution before effects.
		if _, err := authn.RequireOperation(r.Context(), "agent-manager:write", "run.create", RunCreationTarget); err == nil {
			return false
		}
	}
	target := mux.Vars(r)["id"]
	if target != "" && (operation == "stop-run" || operation == "continue-run" || operation == "wake-run" || operation == "recover-run") {
		if _, err := authn.RequireOperation(r.Context(), "agent-manager:write", "agent.manage", "agent-manager:run/"+target); err == nil {
			return false
		}
	}
	recordLifecycleRefusal(operation)
	claims := verified.Claims
	mintTime := ""
	if claims.IssuedAt != 0 {
		mintTime = time.Unix(claims.IssuedAt, 0).UTC().Format(time.RFC3339)
	}
	log.Printf("agent-manager lifecycle refusal caller=%s run_id=%s operation=%s", r.RemoteAddr, claims.RunID, operation)

	message := "run identity cannot perform lifecycle operation"
	if operation == "create-run" && strings.TrimSpace(guidance) != "" {
		message += ": " + strings.TrimSpace(guidance)
	}
	writeJSON(w, http.StatusForbidden, map[string]any{
		"error":           message,
		"operation":       operation,
		"run_id":          claims.RunID.String(),
		"run_status":      string(verified.RunStatus),
		"token_mint_time": mintTime,
	})
	return true
}

// runLifecycleGrantAllowsTarget checks the caller's exact, already-resolved run
// target. Keyed wake endpoints use this because no run ID appears in the URL.
// Keep the run-identity verification here: a context grant must not override a
// presented invalid run identity.
func (h *Handler) runLifecycleGrantAllowsTarget(r *http.Request, operation string, target uuid.UUID) bool {
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" || target == uuid.Nil {
		return false
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		return false
	}
	if operation != "wake-run" {
		return false
	}
	_, err = authn.RequireOperation(r.Context(), "agent-manager:write", "agent.manage", "agent-manager:run/"+target.String())
	return err == nil
}

// OrchestrateScope is the profile-declared ceiling that lets an orchestrator
// run manage its own children from inside the run. It never reaches beyond the
// caller's direct lineage.
const OrchestrateScope = "agent-manager:orchestrate"

// lineageLifecycleAllowed admits the lifecycle operations an orchestrated effort
// needs from inside a run, and nothing wider:
//   - any run may wake its own parked parent (a worker's friction wake);
//   - a run whose profile declares OrchestrateScope may wake or stop its own
//     direct child runs;
//   - lineage is also effort-level (DL-2): a running root run whose profile
//     declares OrchestrateScope and that holds effort E as orchestrator may
//     wake, continue or stop a run holding E as worker whose parent held E as
//     orchestrator. A check-in may therefore steer workers an earlier check-in
//     started; ParentRunID stays the immutable provenance link.
//
// Every other run-initiated lifecycle request still goes through
// denyRunInitiatedLifecycleOperation.
func (h *Handler) lineageLifecycleAllowed(r *http.Request, target uuid.UUID, allowParentWake bool) bool {
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" || h.svc.IdentityService == nil || h.svc.RunService == nil || h.svc.ProfileService == nil {
		return false
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		return false
	}
	caller, err := h.svc.GetRun(r.Context(), verified.Claims.RunID)
	if err != nil || caller == nil {
		return false
	}
	if allowParentWake && caller.ParentRunID != nil && *caller.ParentRunID == target {
		return true
	}
	targetRun, err := h.svc.GetRun(r.Context(), target)
	if err != nil || targetRun == nil || targetRun.ParentRunID == nil {
		return false
	}
	if caller.AgentProfileID == nil {
		return false
	}
	profile, err := h.svc.GetProfile(r.Context(), *caller.AgentProfileID)
	if err != nil || profile == nil || !slices.Contains(profile.DeclaredScopes, OrchestrateScope) {
		return false
	}
	if *targetRun.ParentRunID == caller.ID {
		return true
	}
	return h.effortLineageAllowed(r, caller, targetRun)
}

// orchestratorTaskCaller returns the verified, running profile orchestrator
// whose identity is carried by the request. Task writes use this as their
// existing same-run owner proof; it does not grant global task management.
func (h *Handler) orchestratorTaskCaller(r *http.Request) *domain.Run {
	caller, allowed := h.taskCreateCaller(r)
	if !allowed {
		return nil
	}
	return caller
}

// taskCreateCaller returns a verified active run caller when one is present,
// along with whether its profile has intrinsic orchestration authority. The
// caller run is safe attribution for an agent-created task even when creation
// itself is admitted by an explicit run.create grant.
func (h *Handler) taskCreateCaller(r *http.Request) (*domain.Run, bool) {
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" || h.svc.IdentityService == nil || h.svc.RunService == nil {
		return nil, false
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		return nil, false
	}
	caller, err := h.svc.GetRun(r.Context(), verified.Claims.RunID)
	if err != nil || caller == nil || caller.Status != domain.RunStatusRunning {
		return nil, false
	}
	if caller.AgentProfileID == nil || h.svc.ProfileService == nil {
		return caller, false
	}
	profile, err := h.svc.GetProfile(r.Context(), *caller.AgentProfileID)
	if err != nil || profile == nil || !slices.Contains(profile.DeclaredScopes, OrchestrateScope) {
		return caller, false
	}
	return caller, true
}

// orchestratorTaskCreateAllowed admits task creation from a running profile
// orchestrator. A task is an inert record; the child run it is for still passes
// run-creation admission.
func (h *Handler) orchestratorTaskCreateAllowed(r *http.Request) bool {
	return h.orchestratorTaskCaller(r) != nil
}

// orchestratorTaskUpdateTarget limits the intrinsic task-write path to the
// exact queued, undispatched task created by this running profile orchestrator.
func (h *Handler) orchestratorTaskUpdateTarget(r *http.Request, taskID uuid.UUID) (*domain.Task, bool) {
	caller := h.orchestratorTaskCaller(r)
	if caller == nil {
		return nil, false
	}
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil || task == nil || task.CreatedBy != caller.ID.String() ||
		task.Status != domain.TaskStatusQueued || task.CurrentRunID != nil {
		return nil, false
	}
	return task, true
}

// effortLineageAllowed is the effort-level rule of lineageLifecycleAllowed.
// The caller's profile has already been checked for OrchestrateScope.
func (h *Handler) effortLineageAllowed(r *http.Request, caller, target *domain.Run) bool {
	if caller.Status != domain.RunStatusRunning || caller.ParentRunID != nil {
		return false
	}
	efforts := orchestratorEffortRefs(caller)
	if len(efforts) == 0 {
		return false
	}
	var shared []string
	for _, ref := range effortRefsWithRelationship(target, "worker") {
		if slices.Contains(efforts, ref) {
			shared = append(shared, ref)
		}
	}
	if len(shared) == 0 {
		return false
	}
	parent, err := h.svc.GetRun(r.Context(), *target.ParentRunID)
	if err != nil || parent == nil {
		return false
	}
	for _, ref := range orchestratorEffortRefs(parent) {
		if slices.Contains(shared, ref) {
			return true
		}
	}
	return false
}

// effortRefsWithRelationship lists the efforts a run holds as an active,
// verified reference with the given relationship.
func effortRefsWithRelationship(run *domain.Run, relationship string) []string {
	var refs []string
	for _, ref := range run.WorkReferences {
		if ref.GetKind() == "effort" && ref.GetRelationship() == relationship && ref.GetVerified() &&
			ref.GetState() == eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE && ref.GetId() != "" &&
			!slices.Contains(refs, ref.GetId()) {
			refs = append(refs, ref.GetId())
		}
	}
	return refs
}
