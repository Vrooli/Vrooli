package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// [REQ:GCT-OT-P0-006] Push/pull status
func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters
	query := r.URL.Query()
	req := SyncStatusRequest{
		Fetch:  query.Get("fetch") == "true",
		Remote: query.Get("remote"),
	}

	result, err := GetSyncStatus(hctx.Ctx, SyncStatusDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleDiscard handles POST /api/v1/repo/discard
func (s *Server) handleDiscard(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req DiscardRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if len(req.Paths) == 0 {
		hctx.Resp.BadRequest("paths are required")
		return
	}

	result, err := DiscardFiles(hctx.Ctx, DiscardDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	s.auditDiscard(hctx.RepoDir, req.Paths, result, err)

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

// handleIgnore handles POST /api/v1/repo/ignore
func (s *Server) handleIgnore(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req IgnoreRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.Path) == "" {
		hctx.Resp.BadRequest("path is required")
		return
	}

	result, err := IgnorePath(hctx.Ctx, IgnoreDeps{
		Git:     hctx.Git,
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
	}, req)

	s.auditIgnore(hctx.RepoDir, req.Path, result, err)

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

// handlePush handles POST /api/v1/repo/push
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req PushRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PushToRemote(hctx.Ctx, PushPullDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)

	// Audit logging for push operation
	auditEntry := AuditEntry{
		Operation: AuditOpPush,
		RepoDir:   hctx.RepoDir,
		Success:   result != nil && result.Success,
		Metadata: map[string]interface{}{
			"remote": result.Remote,
			"branch": result.Branch,
		},
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = result.Error
	}
	// Log asynchronously
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

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

// handlePull handles POST /api/v1/repo/pull
func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req PullRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PullFromRemote(hctx.Ctx, PushPullDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)

	// Audit logging for pull operation
	auditEntry := AuditEntry{
		Operation: AuditOpPull,
		RepoDir:   hctx.RepoDir,
		Success:   result != nil && result.Success,
		Metadata: map[string]interface{}{
			"remote":        result.Remote,
			"branch":        result.Branch,
			"has_conflicts": result.HasConflicts,
		},
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = result.Error
	}
	// Log asynchronously
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

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

func (s *Server) auditDiscard(repoDir string, reqPaths []string, result *DiscardResponse, err error) {
	auditEntry := AuditEntry{
		Operation: AuditOpDiscard,
		RepoDir:   repoDir,
		Paths:     reqPaths,
		Success:   result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = strings.Join(result.Errors, "; ")
	}
	if result != nil {
		auditEntry.Paths = result.Discarded
	}
	s.auditLogAsync(auditEntry)
}

func (s *Server) auditIgnore(repoDir, reqPath string, result *IgnoreResponse, err error) {
	auditEntry := AuditEntry{
		Operation: AuditOpIgnore,
		RepoDir:   repoDir,
		Paths:     []string{reqPath},
		Success:   result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = strings.Join(result.Errors, "; ")
	}
	if result != nil && len(result.Ignored) > 0 {
		auditEntry.Paths = result.Ignored
	}
	s.auditLogAsync(auditEntry)
}

func (s *Server) auditLogAsync(entry AuditEntry) {
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, entry)
	}()
}

// handleUpstreamAction handles POST /api/v1/repo/upstream-action
func (s *Server) handleUpstreamAction(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req UpstreamActionRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := RunUpstreamAction(hctx.Ctx, PushPullDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
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
