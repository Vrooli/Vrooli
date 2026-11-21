package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestGrantSelectorsSnapshotDynamicAndRawReadSeparation(t *testing.T) {
	manager := newPasswordManager(nil)
	ctx := context.Background()
	now := time.Now().UTC()
	snapshot, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", PrincipalType: "agent", PrincipalID: "agent-a", SelectorMode: "current_snapshot",
		Members: []string{"agent-a"}, Operations: []string{"use"}, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.grantAllows(ctx, snapshot, "agent-b", "item-a", "use", now) {
		t.Fatal("current-snapshot grant inherited access for a new agent")
	}
	if manager.grantAllows(ctx, snapshot, "agent-a", "item-a", "reveal", now) || manager.grantAllows(ctx, snapshot, "agent-a", "item-a", "export", now) {
		t.Fatal("use-only grant authorized raw read")
	}
	dynamic, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", PrincipalType: "workspace-agent", PrincipalID: "workspace-agent:*", SelectorMode: "dynamic",
		Operations: []string{"use"}, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !manager.grantAllows(ctx, dynamic, "agent-new", "item-a", "use", now) {
		t.Fatal("dynamic workspace-agent grant did not admit a future matching agent")
	}
	manager.mu.Lock()
	manager.members[memberKey("workspace-a", "agent-new")] = workspaceMember{WorkspaceID: "workspace-a", PrincipalID: "agent-new", Status: "revoked"}
	manager.mu.Unlock()
	if manager.grantAllows(ctx, dynamic, "agent-new", "item-a", "use", now) {
		t.Fatal("dynamic grant authorized a revoked workspace member")
	}
	if _, err := manager.getGrant(ctx, "workspace-b", dynamic.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("foreign workspace lookup unexpectedly succeeded")
	}
}

func TestChildGrantCannotWidenParent(t *testing.T) {
	manager := newPasswordManager(nil)
	ctx := context.Background()
	parent, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use", "sign"}, Target: "https://login.example.test", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", ParentGrantID: parent.ID, PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use"}, Target: "https://login.example.test", ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	if err != nil || child.ParentGrantID != parent.ID {
		t.Fatalf("narrow child grant = %+v, err=%v", child, err)
	}
	if _, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", ParentGrantID: parent.ID, PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"reveal"}, Target: "https://login.example.test", ExpiresAt: time.Now().Add(30 * time.Minute),
	}); err == nil {
		t.Fatal("child grant widened parent operations")
	}
	if _, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", ParentGrantID: parent.ID, PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use"}, Target: "https://other.example.test", ExpiresAt: time.Now().Add(30 * time.Minute),
	}); err == nil {
		t.Fatal("child grant widened parent target")
	}
}
