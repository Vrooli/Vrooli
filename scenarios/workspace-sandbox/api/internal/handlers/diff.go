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

// --- Provenance Tracking Handlers ---

// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
//
// Query parameters:
//   - projectRoot: Optional. Filter by project root.
//   - limit: Optional. Maximum results to return (default 100).
//   - offset: Optional. Pagination offset.
func (h *Handlers) GetPendingChanges(w http.ResponseWriter, r *http.Request) {
	projectRoot := r.URL.Query().Get("projectRoot")
	if projectRoot == "" {
		projectRoot = h.Config.Driver.ProjectRoot
	}

	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	result, err := h.Service.GetPendingChanges(r.Context(), projectRoot, limit, offset)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// GetFileProvenance returns the history of changes for a specific file.
//
// Query parameters:
//   - path: Required. The file path to query.
//   - projectRoot: Optional. Filter by project root.
//   - limit: Optional. Maximum history entries to return (default 50).
func (h *Handlers) GetFileProvenance(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		h.JSONError(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	projectRoot := r.URL.Query().Get("projectRoot")
	if projectRoot == "" {
		projectRoot = h.Config.Driver.ProjectRoot
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	changes, err := h.Service.GetFileProvenance(r.Context(), filePath, projectRoot, limit)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"filePath": filePath,
		"changes":  changes,
	})
}

// CommitPending commits pending changes to git and updates provenance records.
func (h *Handlers) CommitPending(w http.ResponseWriter, r *http.Request) {
	var req types.CommitPendingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body with defaults
		req = types.CommitPendingRequest{}
	}

	if req.ProjectRoot == "" {
		req.ProjectRoot = h.Config.Driver.ProjectRoot
	}

	result, err := h.Service.CommitPending(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// GetCommitPreview returns a preview of what would be committed.
// This includes reconciliation with git status to detect externally-committed files.
//
// Query parameters:
//   - projectRoot: Optional. The project root to check.
//     If not provided, uses the server's configured PROJECT_ROOT.
//
// Response includes:
//   - List of files with their status (pending or already_committed)
//   - Suggested commit message
//   - Summary grouped by sandbox
func (h *Handlers) GetCommitPreview(w http.ResponseWriter, r *http.Request) {
	projectRoot := r.URL.Query().Get("projectRoot")
	if projectRoot == "" {
		projectRoot = h.Config.Driver.ProjectRoot
	}

	result, err := h.Service.GetCommitPreview(r.Context(), &types.CommitPreviewRequest{
		ProjectRoot: projectRoot,
	})
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// parseInt is a helper for parsing integer query parameters.
func parseInt(s string) (int, error) {
	var i int
	err := json.Unmarshal([]byte(s), &i)
	return i, err
}
