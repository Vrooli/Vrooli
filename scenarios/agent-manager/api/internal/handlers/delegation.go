package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

// MintDelegatedIdentity handles the child-token exchange. The parent token is
// accepted only from the standard identity header so it cannot be confused
// with an arbitrary body field or accidentally persisted in request data.
func (h *Handler) MintDelegatedIdentity(w http.ResponseWriter, r *http.Request) {
	if h.svc.DelegationService == nil {
		writeSimpleError(w, r, "identity", "delegation is not configured")
		return
	}
	var body struct {
		ChildRunID      string   `json:"child_run_id"`
		RequestedScopes []string `json:"scopes"`
		ExpiresAt       int64    `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON body")
		return
	}
	childID, err := uuid.Parse(strings.TrimSpace(body.ChildRunID))
	if err != nil {
		writeSimpleError(w, r, "child_run_id", "must be a UUID")
		return
	}
	parentToken := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if parentToken == "" {
		writeSimpleError(w, r, "identity", "parent identity header is required")
		return
	}
	result, err := h.svc.DelegationService.MintDelegatedIdentity(r.Context(), orchestration.MintDelegatedIdentityRequest{
		ParentToken: parentToken, ChildRunID: childID, RequestedScopes: body.RequestedScopes,
		ExpiresAt: unixTime(body.ExpiresAt),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func unixTime(value int64) (result time.Time) {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(value, 0)
}
