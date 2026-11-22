package main

import worktreeH "git-control-tower/handlers/worktree"

// auditWorktreeMutation adapts the Worktree Connect owner's bounded event
// projection to the GCT audit log. The handler calls it only after a
// successful mutation or validation-only dry run.
func (s *Server) auditWorktreeMutation(event worktreeH.MutationAuditEvent) {
	metadata := map[string]interface{}{"dry_run": event.DryRun}
	if event.Mode != "" {
		metadata["mode"] = event.Mode
	}
	if event.Reason != "" {
		metadata["reason"] = event.Reason
	}
	if event.Force {
		metadata["force"] = true
	}
	s.auditLogAsync(AuditEntry{
		Operation: AuditOperation(event.Operation),
		RepoDir:   event.RepoPath,
		Paths:     append([]string(nil), event.Paths...),
		Branch:    event.Branch,
		Success:   true,
		Metadata:  metadata,
	})
}
