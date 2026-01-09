package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/types"
)

// --- Retry/Rebase Workflow Endpoints [OT-P2-003] ---

// CheckConflicts handles checking for conflicts between sandbox and canonical repo.
// [OT-P2-003] Retry/Rebase Workflow
//
// This endpoint checks if the canonical repo has changed since the sandbox was created
// and identifies any files that have been modified in both the sandbox and the repo.
// Use this before approving changes to determine if a rebase is needed.
func (h *Handlers) CheckConflicts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	result, err := h.Service.CheckConflicts(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// Rebase handles rebasing a sandbox against the current repo state.
// [OT-P2-003] Retry/Rebase Workflow
//
// This endpoint updates the sandbox's BaseCommitHash to the current repo state.
// After rebasing:
//   - Future conflict checks will compare against the new base commit
//   - The diff will be regenerated against the current canonical repo state
//   - If there were conflicts before, they may be resolved (or new ones detected)
//
// Request body:
//   - strategy: "regenerate" (only option currently; updates baseline without merging)
//   - actor: optional identifier for who initiated the rebase
func (h *Handlers) Rebase(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.RebaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use default strategy
		req = types.RebaseRequest{Strategy: types.RebaseStrategyRegenerate}
	}
	req.SandboxID = id

	result, err := h.Service.Rebase(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}
