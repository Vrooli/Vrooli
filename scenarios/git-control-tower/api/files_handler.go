package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleFiles handles GET /api/v1/repo/files
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	// Use request context for cancellation when client disconnects
	ctx := r.Context()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	req := FileTreeRequest{
		Pattern: query.Get("pattern"),
		Deep:    query.Get("deep") == "true",
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			resp.BadRequest("limit must be a positive integer")
			return
		}
		req.Limit = limit
	}

	// Parse timeout
	if timeoutStr := query.Get("timeout"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil || timeout <= 0 {
			resp.BadRequest("timeout must be a positive integer")
			return
		}
		req.Timeout = timeout
	}

	result, err := GetFileTree(ctx, FileDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(result)
}

// handleDirectoryList handles GET /api/v1/repo/files/dir?path=<dir>
// path="" returns root contents, path="src/components" returns that folder's contents
func (s *Server) handleDirectoryList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	// Parse query parameters - path is optional, empty means root
	dirPath := r.URL.Query().Get("path")

	result, err := GetDirectoryContents(ctx, FileDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, dirPath)
	if err != nil {
		// Check if it's a "not found" type error
		if strings.Contains(err.Error(), "not found") {
			resp.NotFound(err.Error())
			return
		}
		resp.InternalError(err.Error())
		return
	}

	resp.OK(result)
}

// handleRelatedFiles handles GET /api/v1/repo/related
func (s *Server) handleRelatedFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	// Parse query parameters
	path := r.URL.Query().Get("path")
	if strings.TrimSpace(path) == "" {
		resp.BadRequest("path query parameter is required")
		return
	}

	// Get related files using the related files service
	related, err := GetRelatedFiles(ctx, FileDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, path)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(&RelatedFilesResponse{
		Path:      path,
		Related:   related,
		Timestamp: time.Now().UTC(),
	})
}

// handleDeletePath handles POST /api/v1/repo/files/delete
// Deletes a file or directory from the filesystem
func (s *Server) handleDeletePath(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req DeletePathRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := DeletePath(ctx, FileDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}
