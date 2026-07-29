package handlers

import (
	"net/http"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

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

	writeJSON(w, http.StatusForbidden, map[string]any{
		"error":     "run identity cannot perform lifecycle operation",
		"operation": operation,
	})
	return true
}
