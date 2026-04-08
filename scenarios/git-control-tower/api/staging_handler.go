package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// operationResult is implemented by response types that have Success/Errors fields.
type operationResult interface {
	IsSuccess() bool
	ErrorMessages() []string
}

// logAndRespond handles the common pattern: audit log + error/success HTTP response.
func (s *Server) logAndRespond(hctx *HandlerContext, op AuditOperation, reqPaths []string, resultPaths []string, result operationResult, err error) {
	auditEntry := AuditEntry{
		Operation: op,
		RepoDir:   hctx.RepoDir,
		Paths:     reqPaths,
		Success:   result != nil && result.IsSuccess(),
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.IsSuccess() {
		auditEntry.Error = strings.Join(result.ErrorMessages(), "; ")
	}
	if result != nil && len(resultPaths) > 0 {
		auditEntry.Paths = resultPaths
	}
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	if !result.IsSuccess() {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// [REQ:GCT-OT-P0-004] Stage/unstage operations
func (s *Server) handleStage(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req StageRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if !ValidateStagingRequest(w, req) {
		return
	}

	result, err := StageFiles(hctx.Ctx, StagingDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	var resultPaths []string
	if result != nil {
		resultPaths = result.Staged
	}
	s.logAndRespond(hctx, AuditOpStage, req.Paths, resultPaths, result, err)
}

func (s *Server) handleUnstage(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req UnstageRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if !ValidateStagingRequest(w, req) {
		return
	}

	result, err := UnstageFiles(hctx.Ctx, StagingDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	var resultPaths []string
	if result != nil {
		resultPaths = result.Unstaged
	}
	s.logAndRespond(hctx, AuditOpUnstage, req.Paths, resultPaths, result, err)
}
