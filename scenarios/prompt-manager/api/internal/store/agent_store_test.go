package store

import (
	"context"
	"strings"
	"testing"
)

func TestAgentUpdateReplacesEveryMutableFieldAndPreservesIdentity(t *testing.T) {
	ctx := context.Background()
	agentStore := NewFileAgentStore(t.TempDir())
	original := &Agent{
		ID:                "agent-1",
		DisplayName:       "Agent One",
		Description:       "old description",
		Status:            AgentStatusActive,
		Capabilities:      &AgentCapabilities{Provides: []AgentCapability{{CapabilityID: "old"}}},
		Connectors:        []AgentConnector{{Type: "old", ID: "old", Enabled: true}},
		DefaultProfileRef: "old-profile",
		Heartbeat:         &AgentHeartbeat{IntervalSeconds: 10},
		Tags:              []string{"old"},
		FileOrder:         []string{"AGENT.md"},
	}
	if err := agentStore.Create(ctx, original); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	replacement := &Agent{
		DisplayName: "Renamed",
		Description: "",
		Status:      AgentStatusInactive,
		Tags:        []string{},
		FileOrder:   []string{},
	}
	if err := agentStore.Update(ctx, original.ID, replacement); err != nil {
		t.Fatalf("update agent: %v", err)
	}
	updated, err := agentStore.Get(ctx, original.ID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if updated.ID != original.ID || updated.DisplayName != "Renamed" || updated.Description != "" || updated.Status != AgentStatusInactive {
		t.Fatalf("unexpected scalar fields after replacement: %#v", updated)
	}
	if updated.Capabilities != nil || updated.Connectors != nil || updated.DefaultProfileRef != "" || updated.Heartbeat != nil {
		t.Fatalf("optional fields were not cleared: %#v", updated)
	}
	if len(updated.Tags) != 0 || len(updated.FileOrder) != 0 {
		t.Fatalf("explicit empty lists were not preserved: tags=%#v fileOrder=%#v", updated.Tags, updated.FileOrder)
	}
}

func TestRenameFileUpdatesFileOrder(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	agentStore := NewFileAgentStore(storeDir)

	agent := &Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
		Status:      AgentStatusActive,
		FileOrder:   []string{"SOUL.md", "NOTES.md"},
	}

	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := agentStore.CreateFile(ctx, agent.ID, "NOTES.md", "Notes content", false); err != nil {
		t.Fatalf("create notes file: %v", err)
	}

	if err := agentStore.RenameFile(ctx, agent.ID, "NOTES.md", "README.md"); err != nil {
		t.Fatalf("rename file: %v", err)
	}

	updated, err := agentStore.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}

	if strings.Contains(strings.Join(updated.FileOrder, ","), "NOTES.md") {
		t.Fatalf("expected NOTES.md to be removed from file order")
	}
	found := false
	for _, entry := range updated.FileOrder {
		if entry == "README.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected README.md to be added to file order")
	}
}

// TestListFilesIsScopedToTheRequestedAgentAcrossAgentBoundary guards R27's
// "another agent or team's files never appear" acceptance: the listing for one
// agent must never surface files owned by a sibling agent, and path traversal
// beyond the agent folder must be rejected.
func TestListFilesIsScopedToTheRequestedAgentAcrossAgentBoundary(t *testing.T) {
	ctx := context.Background()
	agentStore := NewFileAgentStore(t.TempDir())

	for _, id := range []string{"agent-a", "agent-b"} {
		if err := agentStore.Create(ctx, &Agent{ID: id, DisplayName: id, Status: AgentStatusActive}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	if err := agentStore.CreateFile(ctx, "agent-a", "a-only.md", "a", false); err != nil {
		t.Fatalf("create agent-a file: %v", err)
	}
	if err := agentStore.CreateFile(ctx, "agent-b", "b-only.md", "b", false); err != nil {
		t.Fatalf("create agent-b file: %v", err)
	}

	entries, err := agentStore.ListFiles(ctx, "agent-a")
	if err != nil {
		t.Fatalf("list agent-a files: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Path, "b-only") {
			t.Fatalf("agent-a listing leaked agent-b file: %#v", entries)
		}
	}
	found := false
	for _, entry := range entries {
		if entry.Path == "a-only.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("agent-a listing did not include its own file: %#v", entries)
	}

	if _, err := agentStore.ReadFile(ctx, "agent-a", "../agent-b/b-only.md"); err == nil {
		t.Fatalf("expected cross-agent traversal to be rejected")
	} else if !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("expected invalid path error, got: %v", err)
	}

	if _, err := agentStore.ListFiles(ctx, "missing-agent"); err == nil {
		t.Fatalf("expected missing agent to return an error")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got: %v", err)
	}
}

func TestRenameFileRemovesNonMarkdownFromFileOrder(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	agentStore := NewFileAgentStore(storeDir)

	agent := &Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
		Status:      AgentStatusActive,
		FileOrder:   []string{"SOUL.md", "NOTES.md"},
	}

	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := agentStore.CreateFile(ctx, agent.ID, "NOTES.md", "Notes content", false); err != nil {
		t.Fatalf("create notes file: %v", err)
	}

	if err := agentStore.RenameFile(ctx, agent.ID, "NOTES.md", "NOTES.txt"); err != nil {
		t.Fatalf("rename file: %v", err)
	}

	updated, err := agentStore.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}

	for _, entry := range updated.FileOrder {
		if entry == "NOTES.txt" {
			t.Fatalf("expected non-markdown file to be excluded from file order")
		}
	}
}
