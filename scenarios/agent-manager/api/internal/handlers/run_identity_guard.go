package handlers

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

var lifecycleRefusalCounts = struct {
	sync.Mutex
	byOperation map[string]uint64
}{byOperation: make(map[string]uint64)}

const lifecycleRefusalFunctionalThreshold = 2

// LifecycleRefusalFunctionalStatus exposes the daemon-shaped failure to the
// scenario health contract without weakening the lifecycle guard.
func LifecycleRefusalFunctionalStatus() (healthy bool, reason string) {
	lifecycleRefusalCounts.Lock()
	defer lifecycleRefusalCounts.Unlock()
	var total uint64
	for _, count := range lifecycleRefusalCounts.byOperation {
		total += count
	}
	if total < lifecycleRefusalFunctionalThreshold {
		return true, ""
	}
	return false, "repeated run-identity lifecycle refusals detected"
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
		return false
	}

	lifecycleRefusalCounts.Lock()
	lifecycleRefusalCounts.byOperation[operation]++
	lifecycleRefusalCounts.Unlock()
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
