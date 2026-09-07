package main

import (
	"context"
	"log"
	"time"
)

// claimedBranchesFn is the seam through which enrichBranchesWithWorktreeClaims
// resolves the branch -> worktree mapping. Tests overwrite this variable
// with a fake; production uses the real worktree.Inspector wired in
// connect_wiring.go. Empty map (no error) means no branches are claimed.
var claimedBranchesFn = func(ctx context.Context, repoDir string) (map[string]string, error) {
	return newWorktreeInspector().ClaimedBranches(ctx, repoDir)
}

// enrichBranchesWithWorktreeClaims annotates each local branch with
// the worktree path (if any) that has the branch checked out. Errors
// are intentionally swallowed: the branch list must continue to load
// even when worktree inspection fails (e.g. a corrupted .git/worktrees
// directory). Empty CheckedOutInWorktree is the unclaimed sentinel.
func enrichBranchesWithWorktreeClaims(ctx context.Context, result *RepoBranchesResponse, repoDir string) {
	if result == nil || repoDir == "" || len(result.Locals) == 0 {
		return
	}
	claims, err := claimedBranchesFn(ctx, repoDir)
	if err != nil || len(claims) == 0 {
		return
	}
	for i, b := range result.Locals {
		if path, ok := claims[b.Name]; ok {
			result.Locals[i].CheckedOutInWorktree = path
		}
	}
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
