package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleRepoBranches(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	result, err := ListBranches(ctx, BranchDeps{Git: s.git, RepoDir: repoDir})
	if err != nil {
		resp.InternalError(err.Error())
		return
	}
	resp.OK(result)
}

func (s *Server) handleBranchCreate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req CreateBranchRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := CreateBranch(ctx, BranchDeps{Git: s.git, RepoDir: repoDir}, req)
	branchName := strings.TrimSpace(req.Name)
	logBranchAudit(s, repoDir, AuditOpBranchCreate, branchName, result != nil && result.Success, err)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}
	resp.OK(result)
}

func (s *Server) handleBranchSwitch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req SwitchBranchRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := SwitchBranch(ctx, BranchDeps{Git: s.git, RepoDir: repoDir}, req)
	branchName := strings.TrimSpace(req.Name)
	logBranchAudit(s, repoDir, AuditOpBranchSwitch, branchName, result != nil && result.Success, err)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}
	resp.OK(result)
}

func (s *Server) handleBranchPublish(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req PublishBranchRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PublishBranch(ctx, BranchDeps{Git: s.git, RepoDir: repoDir}, req)
	branchName := strings.TrimSpace(req.Branch)
	if branchName == "" && result != nil {
		branchName = result.Branch
	}
	logBranchAudit(s, repoDir, AuditOpBranchPublish, branchName, result != nil && result.Success, err)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}
	resp.OK(result)
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
