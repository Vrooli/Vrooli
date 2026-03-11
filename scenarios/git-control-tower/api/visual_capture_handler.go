package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleVisualCapture handles POST /api/v1/repo/visual-capture
func (s *Server) handleVisualCapture(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "browser-automation-studio") {
		hctx.Resp.ServiceUnavailable("browser-automation-studio is not available")
		return
	}

	var req VisualCaptureRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.ScenarioSlug) == "" {
		hctx.Resp.BadRequest("scenarioSlug is required")
		return
	}

	meta, err := CaptureScenario(hctx.Ctx, VisualCaptureDeps{
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

	hctx.Resp.OK(meta)
}

// handleVisualCaptureList handles GET /api/v1/repo/visual-captures?scenarioSlug=...
func (s *Server) handleVisualCaptureList(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	slug := strings.TrimSpace(r.URL.Query().Get("scenarioSlug"))
	if slug == "" {
		hctx.Resp.BadRequest("scenarioSlug query parameter is required")
		return
	}

	snapshots, err := s.visualCaptureStorage.ListSnapshotSets(hctx.RepoID, slug)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(map[string]interface{}{
		"snapshots": snapshots,
		"total":     len(snapshots),
	})
}

// handleVisualCaptureDetail handles GET /api/v1/repo/visual-captures/{id}?scenarioSlug=...
func (s *Server) handleVisualCaptureDetail(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
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

	detail, err := s.visualCaptureStorage.GetSnapshotSet(hctx.RepoID, slug, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(detail)
}

// handleVisualCaptureScreenshot handles GET /api/v1/repo/visual-captures/{id}/screenshot/{filename}
func (s *Server) handleVisualCaptureScreenshot(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
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

	data, err := s.visualCaptureStorage.GetScreenshotFile(hctx.RepoID, slug, id, filename)
	if err != nil {
		if strings.Contains(err.Error(), "path traversal") {
			hctx.Resp.BadRequest(err.Error())
			return
		}
		hctx.Resp.NotFound("screenshot not found")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := w.Write(data); err != nil {
		hctx.Resp.InternalError("failed to write screenshot response")
		return
	}
}

// handleVisualCaptureVideo handles GET /api/v1/repo/visual-captures/{id}/video/{filename}
func (s *Server) handleVisualCaptureVideo(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
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

	data, err := s.visualCaptureStorage.GetVideoFile(hctx.RepoID, slug, id, filename)
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

// handleVisualCaptureStorageStats handles GET /api/v1/repo/visual-capture-storage
func (s *Server) handleVisualCaptureStorageStats(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	stats, err := s.visualCaptureStorage.GetStorageStats(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(stats)
}

// handleVisualCaptureDelete handles DELETE /api/v1/repo/visual-captures/{id}?scenarioSlug=...
func (s *Server) handleVisualCaptureDelete(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
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

	if err := s.visualCaptureStorage.DeleteSnapshotSet(hctx.RepoID, slug, id); err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(map[string]interface{}{
		"deleted": true,
		"id":      id,
	})
}

// handleVisualCaptureClearAll handles DELETE /api/v1/repo/visual-capture-storage
func (s *Server) handleVisualCaptureClearAll(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if err := s.visualCaptureStorage.ClearAllSnapshots(hctx.RepoID); err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(map[string]interface{}{
		"cleared": true,
	})
}
