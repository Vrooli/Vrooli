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
