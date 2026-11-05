package main

import (
	"context"
	"strings"
	"time"
)

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
