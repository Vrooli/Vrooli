package store

import (
	"context"
	"strings"
	"testing"
)

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
