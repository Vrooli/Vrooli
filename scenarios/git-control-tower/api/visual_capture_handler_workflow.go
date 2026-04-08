package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ============================================================================
// Workflow Capture Handlers
// ============================================================================

// validateWorkflowCaptureRequest validates the workflow capture request fields.
// Returns an error message string if invalid, or "" if valid.
func validateWorkflowCaptureRequest(req *WorkflowCaptureRequest) string {
	if strings.TrimSpace(req.ScenarioSlug) == "" {
		return "scenarioSlug is required"
	}
	if req.Mode != "" && req.Mode != CaptureModeBaseline && req.Mode != CaptureModeCapture {
		return "mode must be \"baseline\" or \"capture\""
	}
	validModes := map[string]bool{"observer": true, "mutating": true, "destructive": true}
	for _, m := range req.ExecutionModes {
		if !validModes[m] {
			return "invalid executionMode: " + m + "; must be observer, mutating, or destructive"
		}
	}
	return ""
}

// handleWorkflowCapture handles POST /api/v1/repo/workflow-capture
func (s *Server) handleWorkflowCapture(w http.ResponseWriter, r *http.Request) {
	// Pass nil for repoLock — workflow captures don't touch git, and holding the
	// lock for minutes while polling BAS would block all other repo operations.
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 300*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "browser-automation-studio") {
		hctx.Resp.ServiceUnavailable("browser-automation-studio is not available")
		return
	}

	var req WorkflowCaptureRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if msg := validateWorkflowCaptureRequest(&req); msg != "" {
		hctx.Resp.BadRequest(msg)
		return
	}

	result, err := CaptureWorkflows(hctx.Ctx, WorkflowCaptureDeps{
		BAS:     s.basClient,
		Storage: s.visualCaptureStorage,
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
		RepoID:  hctx.RepoID,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleWorkflowCaptureList handles GET /api/v1/repo/workflow-captures?scenarioSlug=...
func (s *Server) handleWorkflowCaptureList(w http.ResponseWriter, r *http.Request) {
	// nil repoLock — file I/O only, no git operations
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	slug := strings.TrimSpace(r.URL.Query().Get("scenarioSlug"))
	if slug == "" {
		hctx.Resp.BadRequest("scenarioSlug query parameter is required")
		return
	}

	captures, err := s.visualCaptureStorage.ListWorkflowCaptures(hctx.RepoID, slug)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	resp := map[string]interface{}{
		"captures": captures,
		"total":    len(captures),
	}

	// Compute staleness for the most recent capture-role workflow result
	for _, c := range captures {
		if c.Role == "capture" {
			staleness := CheckCaptureStaleness(hctx.RepoDir, slug, c.CreatedAt)
			resp["staleness"] = staleness
			break
		}
	}

	hctx.Resp.OK(resp)
}

// handleWorkflowCaptureDetail handles GET /api/v1/repo/workflow-captures/{id}?scenarioSlug=...
func (s *Server) handleWorkflowCaptureDetail(w http.ResponseWriter, r *http.Request) {
	// nil repoLock — file I/O only, no git operations
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	vars := mux.Vars(r)
	id := vars["id"]
	slug := strings.TrimSpace(r.URL.Query().Get("scenarioSlug"))
	if slug == "" {
		hctx.Resp.BadRequest("scenarioSlug query parameter is required")
		return
	}

	result, videos, err := s.visualCaptureStorage.GetWorkflowCapture(hctx.RepoID, slug, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(map[string]interface{}{
		"capture": result,
		"videos":  videos,
	})
}

// handleWorkflowCaptureVideo handles GET /api/v1/repo/workflow-captures/{id}/video/{filename}?scenarioSlug=...
func (s *Server) handleWorkflowCaptureVideo(w http.ResponseWriter, r *http.Request) {
	// nil repoLock — file I/O only, no git operations
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	vars := mux.Vars(r)
	id := vars["id"]
	filename := vars["filename"]
	slug := strings.TrimSpace(r.URL.Query().Get("scenarioSlug"))
	if slug == "" {
		hctx.Resp.BadRequest("scenarioSlug query parameter is required")
		return
	}

	data, err := s.visualCaptureStorage.GetWorkflowVideo(hctx.RepoID, slug, id, filename)
	if err != nil {
		if strings.Contains(err.Error(), "path traversal") {
			hctx.Resp.BadRequest(err.Error())
			return
		}
		hctx.Resp.NotFound("video not found")
		return
	}

	w.Header().Set("Content-Type", "video/webm")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := w.Write(data); err != nil {
		hctx.Resp.InternalError("failed to write video response")
		return
	}
}

// handleWorkflowCaptureDelete handles DELETE /api/v1/repo/workflow-captures/{id}?scenarioSlug=...
func (s *Server) handleWorkflowCaptureDelete(w http.ResponseWriter, r *http.Request) {
	// nil repoLock — file I/O only, no git operations
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	vars := mux.Vars(r)
	id := vars["id"]
	slug := strings.TrimSpace(r.URL.Query().Get("scenarioSlug"))
	if slug == "" {
		hctx.Resp.BadRequest("scenarioSlug query parameter is required")
		return
	}

	if err := s.visualCaptureStorage.DeleteWorkflowCapture(hctx.RepoID, slug, id); err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(map[string]interface{}{
		"deleted": true,
		"id":      id,
	})
}
