package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

// GetDiff is the single front-door for both live and archived sandbox
// diffs.
//
// Resolution by sandbox status:
//   - Active or Stopped → serve from the live overlay via Service.GetDiff.
//   - Approved, Rejected, or Deleted → serve from the durable archive
//     via Service.GetArchive. The response carries ArchiveState so the
//     UI can render an explicit "no diff captured" state for archives
//     that were intentionally skipped (Error → Deleted).
//   - Creating → 200 with empty diff and ArchiveState="not_captured"
//     (the sandbox has no upper dir yet; this is not an error).
//   - Error → fall through to live path; live GetDiff handles the
//     missing-upper case by returning an empty diff. After Error has
//     transitioned to Deleted, the archive path serves the row.
//
// Query parameters:
//   - mode: View mode — "diff" (default), "full_diff", or "source".
//     full_diff and source require live overlay paths and are rejected
//     with 400 for archived sandboxes (no merged dir to read from).
func (h *Handlers) GetDiff(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	mode := types.ViewMode(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = types.ViewModeDiff
	}
	switch mode {
	case types.ViewModeDiff, types.ViewModeFullDiff, types.ViewModeSource:
	default:
		h.JSONError(w, "invalid mode parameter: must be 'diff', 'full_diff', or 'source'", http.StatusBadRequest)
		return
	}

	sandbox, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Archive-bearing terminal states: serve from the durable archive.
	switch sandbox.Status {
	case types.StatusApproved, types.StatusRejected, types.StatusDeleted:
		if mode != types.ViewModeDiff {
			h.JSONError(w,
				"view modes 'full_diff' and 'source' require a live overlay; this sandbox has been archived",
				http.StatusBadRequest)
			return
		}
		archived, err := h.Service.GetArchive(r.Context(), id)
		if h.HandleDomainError(w, err) {
			return
		}
		if archived == nil {
			// Status is terminal but no archive row exists. This is
			// only possible for sandboxes that crossed terminal before
			// the archive seam was wired (legacy data). Return a
			// not_captured marker rather than 404 so the UI renders
			// "no diff captured" instead of a hard error.
			archived = &types.DiffResult{
				SandboxID:    id,
				Files:        []*types.FileChange{},
				Generated:    sandbox.UpdatedAt,
				ArchiveState: types.ArchiveStateNotCaptured,
			}
		}
		archived.Mode = mode
		h.JSONSuccess(w, archived)
		return
	}

	// Pre-overlay states: nothing to diff yet. Live response with an
	// empty Files list; ArchiveState stays empty (this is not an archive
	// — there's just no data to show yet).
	if sandbox.Status == types.StatusCreating {
		h.JSONSuccess(w, &types.DiffResult{
			SandboxID: id,
			Files:     []*types.FileChange{},
			Generated: sandbox.CreatedAt,
			Mode:      mode,
		})
		return
	}

	// Active / Stopped / Error: live path. ArchiveState stays empty
	// (zero value) to signal "live overlay" to consumers. The
	// Error-with-missing-upper case yields an empty diff, also with
	// ArchiveState empty — clients render it as "no diff" rather than
	// "archived no_capture", which would be misleading.
	diffResult, err := h.Service.GetDiff(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}
	diffResult.Mode = mode

	if mode == types.ViewModeFullDiff || mode == types.ViewModeSource {
		diffResult.FileContents = make(map[string]types.FileViewData)
		for _, file := range diffResult.Files {
			content, err := diff.GetFileContent(sandbox.UpperDir, sandbox.LowerDir, file.FilePath, file.ChangeType)
			if err != nil {
				continue
			}
			if content == "" {
				continue
			}
			fileData := types.FileViewData{FullContent: content}
			if mode == types.ViewModeFullDiff {
				fileData.AnnotatedLines = diff.BuildAnnotatedLines(content, diffResult.UnifiedDiff, file.FilePath)
			}
			diffResult.FileContents[file.FilePath] = fileData
		}
	}

	h.JSONSuccess(w, diffResult)
}

// GetDiffFile serves the raw content of one file in an archive,
// addressed by its path within the archive's index.
//
// Used by DiffViewer for on-demand per-file expansion: the list
// response from GetDiff returns metadata only; this endpoint streams
// the underlying blob bytes when the user expands a file. Live-overlay
// diffs do not use this endpoint — full_diff/source mode embeds content
// in the GetDiff response.
//
// Returns 404 when the sandbox has no archive, the path doesn't match
// any archived entry, or the entry has no associated blob (e.g. a
// directory entry, or a file whose content was not captured).
func (h *Handlers) GetDiffFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		h.JSONError(w, "path query parameter is required", http.StatusBadRequest)
		return
	}

	content, err := h.Service.FetchArchiveFile(r.Context(), id, path)
	if errors.Is(err, blobstore.ErrNotFound) {
		h.JSONError(w, "no archived blob for this path", http.StatusNotFound)
		return
	}
	if err != nil {
		h.JSONError(w, "failed to fetch archive blob: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := w.Write(content); err != nil {
		// Best-effort; the response is already started so JSONError
		// would just produce malformed output.
		return
	}
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
		// Allow empty body for default approval
		req = types.ApprovalRequest{Mode: "all"}
	}
	req.SandboxID = id

	result, err := h.Service.Approve(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// ApplyAtRunEnd handles the agent-manager run-end apply call. It carries
// agent-manager run-context metadata (agent_manager_run_id, conversation_id,
// cost, runOutcome, source) onto the apply path defined by the auditability
// contract. See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md and
// scenarios/swarm-manager/execute/agent-manager-sandbox-auto-apply-defaults/plan.md
// (Decision D6) for the full contract.
func (h *Handlers) ApplyAtRunEnd(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.ApplyAtRunEndRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.SandboxID = id

	result, err := h.Service.ApplyAtRunEnd(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// TurnCheckpoint handles the agent-manager post-turn checkpoint call.
func (h *Handlers) TurnCheckpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req types.TurnCheckpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.SandboxID = id

	result, err := h.Service.TurnCheckpoint(r.Context(), &req)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

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

// PostCommitPreview returns a preview of what would be committed for a subset of files.
// Accepts JSON body matching CommitPreviewRequest with optional filePaths.
func (h *Handlers) PostCommitPreview(w http.ResponseWriter, r *http.Request) {
	var req types.CommitPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectRoot == "" {
		req.ProjectRoot = h.Config.Driver.ProjectRoot
	}

	result, err := h.Service.GetCommitPreview(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// MarkCommitted marks pending changes as committed for files committed by external tools.
//
// Request body: MarkCommittedRequest (projectRoot, filePaths, commitHash, commitMessage).
func (h *Handlers) MarkCommitted(w http.ResponseWriter, r *http.Request) {
	var req types.MarkCommittedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectRoot == "" {
		req.ProjectRoot = h.Config.Driver.ProjectRoot
	}

	result, err := h.Service.MarkCommitted(r.Context(), &req)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, result)
}

// GetProvenanceByRun returns pending applied changes grouped by agent-manager run ID.
//
// Query parameters:
//   - projectRoot: Optional. Filter by project root.
func (h *Handlers) GetProvenanceByRun(w http.ResponseWriter, r *http.Request) {
	projectRoot := r.URL.Query().Get("projectRoot")
	if projectRoot == "" {
		projectRoot = h.Config.Driver.ProjectRoot
	}

	groups, err := h.Service.GetProvenanceByRun(r.Context(), projectRoot)
	if h.HandleDomainError(w, err) {
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"runGroups": groups,
	})
}

// parseInt is a helper for parsing integer query parameters.
func parseInt(s string) (int, error) {
	var i int
	err := json.Unmarshal([]byte(s), &i)
	return i, err
}
