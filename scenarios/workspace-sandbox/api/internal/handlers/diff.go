package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/types"
)

// GetDiff handles getting sandbox diff.
func (h *Handlers) GetDiff(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	diff, err := h.Service.GetDiff(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, diff)
}

// Approve handles approving sandbox changes.
func (h *Handlers) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for approve-all
		req = types.ApprovalRequest{Mode: "all"}
	}
	req.SandboxID = id

	result, err := h.Service.Approve(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// Reject handles rejecting sandbox changes.
func (h *Handlers) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Actor string `json:"actor"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	sb, err := h.Service.Reject(r.Context(), id, req.Actor)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, sb)
}

// Discard handles discarding specific files from a sandbox.
// This allows rejecting individual files while keeping others pending.
func (h *Handlers) Discard(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.DiscardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.SandboxID = id

	// Must have at least one file to discard
	if len(req.FileIDs) == 0 && len(req.FilePaths) == 0 {
		h.JSONError(w, "fileIds or filePaths required", http.StatusBadRequest)
		return
	}

	result, err := h.Service.Discard(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// GetWorkspace handles getting the workspace path.
func (h *Handlers) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	path, err := h.Service.GetWorkspacePath(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, map[string]string{"path": path})
}

// ValidatePath handles path validation requests.
// This allows the UI to check if a path exists and is valid before creating a sandbox.
//
// Query parameters:
//   - path: Required. The absolute path to validate.
//   - projectRoot: Optional. The project root to check containment against.
//     If not provided, uses the server's configured PROJECT_ROOT.
func (h *Handlers) ValidatePath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		h.JSONError(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	projectRoot := r.URL.Query().Get("projectRoot")
	if projectRoot == "" {
		projectRoot = h.Config.Driver.ProjectRoot
	}

	// Delegate to service layer for all validation logic
	result, err := h.Service.ValidatePath(r.Context(), path, projectRoot)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}
