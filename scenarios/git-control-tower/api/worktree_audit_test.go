package main

import (
	"context"
	"testing"
	"time"

	worktreeH "git-control-tower/handlers/worktree"
)

type asyncAuditSink struct{ entries chan AuditEntry }

func (s asyncAuditSink) IsConfigured() bool { return true }
func (s asyncAuditSink) Log(_ context.Context, entry AuditEntry) error {
	s.entries <- entry
	return nil
}
func (s asyncAuditSink) Query(context.Context, AuditQueryRequest) (*AuditQueryResponse, error) {
	return &AuditQueryResponse{}, nil
}

func TestWorktreeMutationAuditUsesOwnerAuditLogger(t *testing.T) {
	sink := asyncAuditSink{entries: make(chan AuditEntry, 1)}
	server := &Server{audit: sink}
	server.auditWorktreeMutation(worktreeH.MutationAuditEvent{
		Operation: worktreeH.AuditWorktreeMoved,
		RepoPath:  "/repo",
		Paths:     []string{"/repo/.worktrees/old", "/repo/.worktrees/new"},
		Branch:    "feature",
		Reason:    "relocate retained worktree",
		DryRun:    true,
	})

	select {
	case entry := <-sink.entries:
		if entry.Operation != AuditOpWorktreeMoved || !entry.Success || entry.RepoDir != "/repo" || entry.Branch != "feature" || len(entry.Paths) != 2 || entry.Paths[0] != "/repo/.worktrees/old" || entry.Paths[1] != "/repo/.worktrees/new" || entry.Metadata["dry_run"] != true || entry.Metadata["reason"] != "relocate retained worktree" {
			t.Fatalf("worktree audit entry lost its typed event or dry-run boundary: %+v", entry)
		}
	case <-time.After(time.Second):
		t.Fatal("worktree audit event did not reach the GCT audit logger")
	}
}
