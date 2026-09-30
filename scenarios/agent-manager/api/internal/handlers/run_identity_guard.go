package handlers

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
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

// denyRunInitiatedLifecycleOperation closes the privilege escalation created
// by giving investigation runs shell access. A valid run identity may inspect
// the service, but it may not create, resume, or stop agent-manager runs.
// Operator requests carry no run identity token and continue unchanged.
// A presented but unverified credential is refused, never treated as absence.
func (h *Handler) denyRunInitiatedLifecycleOperation(w http.ResponseWriter, r *http.Request, operation string) bool {
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

	recordLifecycleRefusal(operation)
	claims := verified.Claims
	mintTime := ""
	if claims.IssuedAt != 0 {
		mintTime = time.Unix(claims.IssuedAt, 0).UTC().Format(time.RFC3339)
	}
	log.Printf("agent-manager lifecycle refusal caller=%s run_id=%s operation=%s", r.RemoteAddr, claims.RunID, operation)

	writeJSON(w, http.StatusForbidden, map[string]any{
		"error":           "run identity cannot perform lifecycle operation",
		"operation":       operation,
		"run_id":          claims.RunID.String(),
		"run_status":      string(verified.RunStatus),
		"token_mint_time": mintTime,
	})
	return true
}

// OrchestrateScope is the profile-declared ceiling that lets an orchestrator
// run manage its own children from inside the run. It never reaches beyond the
// caller's direct lineage.
const OrchestrateScope = "agent-manager:orchestrate"

// lineageLifecycleAllowed admits the lifecycle operations an orchestrated effort
// needs from inside a run, and nothing wider:
//   - any run may wake its own parked parent (a worker's friction wake);
//   - a run whose profile declares OrchestrateScope may wake or stop its own
//     direct child runs.
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
	if err != nil || targetRun == nil || targetRun.ParentRunID == nil || *targetRun.ParentRunID != caller.ID {
		return false
	}
	if caller.AgentProfileID == nil {
		return false
	}
	profile, err := h.svc.GetProfile(r.Context(), *caller.AgentProfileID)
	return err == nil && profile != nil && slices.Contains(profile.DeclaredScopes, OrchestrateScope)
}
