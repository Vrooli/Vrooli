package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// [REQ:GCT-OT-P0-005] Commit composition API
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CommitRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	// Capture staged files before commit (needed for workspace-sandbox notification)
	stagedFiles, _ := hctx.Git.ListStagedFiles(hctx.Ctx, hctx.RepoDir)

	result, err := CreateCommit(hctx.Ctx, CommitDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for commit operation
	s.logCommitAudit(hctx.RepoDir, req, result, err)

	// Notify workspace-sandbox that files have been committed (fire-and-forget)
	s.notifyCommitToSandbox(hctx.RepoDir, stagedFiles, result)

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

// logCommitAudit builds an AuditEntry from commit results and logs it asynchronously.
func (s *Server) logCommitAudit(repoDir string, req CommitRequest, result *CommitResponse, err error) {
	commitMessage := req.Message
	if result != nil && strings.TrimSpace(result.Message) != "" {
		commitMessage = result.Message
	}
	auditEntry := AuditEntry{
		Operation:     AuditOpCommit,
		RepoDir:       repoDir,
		CommitMessage: commitMessage,
		Success:       result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil {
		auditEntry.Error = commitAuditError(result)
		if result.Success {
			auditEntry.CommitHash = result.Hash
		}
	}
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()
}

// commitAuditError returns the error string for a failed commit result.
func commitAuditError(result *CommitResponse) string {
	if result.Success {
		return ""
	}
	if len(result.ValidationErrors) > 0 {
		return strings.Join(result.ValidationErrors, "; ")
	}
	return result.Error
}

// notifyCommitToSandbox fires a background notification to workspace-sandbox about committed files.
func (s *Server) notifyCommitToSandbox(repoDir string, stagedFiles []string, result *CommitResponse) {
	if result == nil || !result.Success || len(stagedFiles) == 0 {
		return
	}
	go func() {
		notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer notifyCancel()
		_ = s.sandbox.MarkCommitted(notifyCtx, repoDir, stagedFiles, result.Hash, result.Message)
	}()
}
