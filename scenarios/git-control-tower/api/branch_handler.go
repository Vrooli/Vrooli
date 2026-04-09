package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleRepoBranches(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	result, err := ListBranches(hctx.Ctx, BranchDeps{Git: hctx.Git, RepoDir: hctx.RepoDir})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	hctx.Resp.OK(result)
}

func (s *Server) handleBranchCreate(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CreateBranchRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := CreateBranch(hctx.Ctx, BranchDeps{Git: hctx.Git, RepoDir: hctx.RepoDir}, req)
	branchName := strings.TrimSpace(req.Name)
	logBranchAudit(s, hctx.RepoDir, AuditOpBranchCreate, branchName, result != nil && result.Success, err)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	hctx.Resp.OK(result)
}

func (s *Server) handleBranchSwitch(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req SwitchBranchRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := SwitchBranch(hctx.Ctx, BranchDeps{Git: hctx.Git, RepoDir: hctx.RepoDir}, req)
	branchName := strings.TrimSpace(req.Name)
	logBranchAudit(s, hctx.RepoDir, AuditOpBranchSwitch, branchName, result != nil && result.Success, err)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	hctx.Resp.OK(result)
}

func (s *Server) handleBranchPublish(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req PublishBranchRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PublishBranch(hctx.Ctx, BranchDeps{Git: hctx.Git, RepoDir: hctx.RepoDir}, req)
	branchName := strings.TrimSpace(req.Branch)
	if branchName == "" && result != nil {
		branchName = result.Branch
	}
	logBranchAudit(s, hctx.RepoDir, AuditOpBranchPublish, branchName, result != nil && result.Success, err)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	hctx.Resp.OK(result)
}

func logBranchAudit(s *Server, repoDir string, op AuditOperation, branch string, success bool, err error) {
	auditEntry := AuditEntry{
		Operation: op,
		RepoDir:   repoDir,
		Branch:    branch,
		Success:   err == nil && success,
		Timestamp: time.Now().UTC(),
	}
	if err != nil {
		auditEntry.Error = err.Error()
	}
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		if logErr := s.audit.Log(logCtx, auditEntry); logErr != nil {
			log.Printf("audit log failed: %v", logErr)
		}
	}()
}
