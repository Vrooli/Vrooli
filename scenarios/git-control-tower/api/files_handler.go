package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleFiles handles GET /api/v1/repo/files
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

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
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		req.Limit = limit
	}

	// Parse timeout
	if timeoutStr := query.Get("timeout"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil || timeout <= 0 {
			hctx.Resp.BadRequest("timeout must be a positive integer")
			return
		}
		req.Timeout = timeout
	}

	result, err := GetFileTree(hctx.Ctx, FileDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleDirectoryList handles GET /api/v1/repo/files/dir?path=<dir>
// path="" returns root contents, path="src/components" returns that folder's contents
func (s *Server) handleDirectoryList(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters - path is optional, empty means root
	dirPath := r.URL.Query().Get("path")

	result, err := GetDirectoryContents(hctx.Ctx, FileDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, dirPath)
	if err != nil {
		// Check if it's a "not found" type error
		if strings.Contains(err.Error(), "not found") {
			hctx.Resp.NotFound(err.Error())
			return
		}
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleRelatedFiles handles GET /api/v1/repo/related
func (s *Server) handleRelatedFiles(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters
	path := r.URL.Query().Get("path")
	if strings.TrimSpace(path) == "" {
		hctx.Resp.BadRequest("path query parameter is required")
		return
	}

	// Get related files using the related files service
	related, err := GetRelatedFiles(hctx.Ctx, FileDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, path)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(&RelatedFilesResponse{
		Path:      path,
		Related:   related,
		Timestamp: time.Now().UTC(),
	})
}

// handleDeletePath handles POST /api/v1/repo/files/delete
// Deletes a file or directory from the filesystem
func (s *Server) handleDeletePath(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req DeletePathRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := DeletePath(hctx.Ctx, FileDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handleSaveFileContent handles PUT /api/v1/repo/files/content
// Saves text content to an existing file with optimistic concurrency support.
func (s *Server) handleSaveFileContent(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req SaveFileContentRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		hctx.Resp.BadRequest("path is required")
		return
	}

	result, err := SaveFileContent(hctx.Ctx, FileContentDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
	}, req)
	if err != nil {
		var tooLarge *FileTooLargeError
		if errors.As(err, &tooLarge) {
			hctx.Resp.PayloadTooLarge(tooLarge.Error())
			return
		}
		var unsupported *UnsupportedBinaryError
		if errors.As(err, &unsupported) {
			hctx.Resp.UnsupportedMediaType(unsupported.Error())
			return
		}
		var conflict *FileContentConflictError
		if errors.As(err, &conflict) {
			hctx.Resp.JSON(http.StatusConflict, SaveFileContentConflictResponse{
				Error:       conflict.Error(),
				Path:        conflict.Path,
				CurrentHash: conflict.CurrentHash,
				Timestamp:   time.Now().UTC(),
			})
			return
		}
		if strings.Contains(err.Error(), "invalid path") {
			hctx.Resp.BadRequest(err.Error())
			return
		}
		if strings.Contains(err.Error(), "file not found") || strings.Contains(err.Error(), "directory") {
			hctx.Resp.NotFound(err.Error())
			return
		}
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleContentSearch handles GET /api/v1/repo/search/content
// Searches file contents using git grep
func (s *Server) handleContentSearch(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters
	query := r.URL.Query()
	req := ContentSearchRequest{
		Query:         query.Get("query"),
		CaseSensitive: query.Get("case_sensitive") == "true",
		WholeWord:     query.Get("whole_word") == "true",
		Regex:         query.Get("regex") == "true",
		Include:       query.Get("include"),
		Exclude:       query.Get("exclude"),
	}

	// Parse context_lines
	if contextStr := query.Get("context_lines"); contextStr != "" {
		contextLines, err := strconv.Atoi(contextStr)
		if err != nil || contextLines < 0 {
			hctx.Resp.BadRequest("context_lines must be a non-negative integer")
			return
		}
		req.ContextLines = contextLines
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		req.Limit = limit
	}

	// Parse timeout
	if timeoutStr := query.Get("timeout"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil || timeout <= 0 {
			hctx.Resp.BadRequest("timeout must be a positive integer")
			return
		}
		req.Timeout = timeout
	}

	result, err := SearchContent(hctx.Ctx, ContentSearchDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "query") || strings.Contains(err.Error(), "regex") {
			hctx.Resp.BadRequest(err.Error())
			return
		}
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}
