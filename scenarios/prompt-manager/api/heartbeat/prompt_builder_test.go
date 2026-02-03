package heartbeat

import (
	"context"
	"strings"
	"testing"
	"time"

	"prompt-manager/store"
)

func TestPromptBuilderAgentOnly(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	agent := &store.Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
		Status:      store.AgentStatusActive,
		FileOrder:   []string{"SOUL.md", "NOTES.md"},
	}

	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := agentStore.CreateFile(ctx, agent.ID, "NOTES.md", "Notes content", false); err != nil {
		t.Fatalf("create notes file: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.Build(ctx, PromptBuildRequest{AgentID: agent.ID})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if !strings.Contains(prompt, "# Agent Files (Markdown)") {
		t.Fatalf("expected agent files section")
	}
	if !strings.Contains(prompt, "Notes content") {
		t.Fatalf("expected notes content in prompt")
	}
	if strings.Contains(prompt, "# Team Relationships") {
		t.Fatalf("did not expect team relationship section for agent-only prompt")
	}
	if strings.Contains(prompt, "# Heartbeat Task") {
		t.Fatalf("did not expect heartbeat task section for agent-only prompt")
	}
}

func TestPromptBuilderTeamContext(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	agent := &store.Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
		Status:      store.AgentStatusActive,
		FileOrder:   []string{"SOUL.md", "NOTES.md"},
	}

	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := agentStore.CreateFile(ctx, agent.ID, "NOTES.md", "Notes content", false); err != nil {
		t.Fatalf("create notes file: %v", err)
	}

	team := &store.Team{
		ID:          "team-1",
		DisplayName: "Team One",
		Enabled:     true,
		EnabledSet:  true,
	}
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, team.ID, agent.ID, "Do the work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, team.ID, agent.ID, "Ship the update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}

	inbox := &store.TeamInbox{
		Messages: []store.TeamMessage{
			{
				ID:          "msg-1",
				TeamID:      team.ID,
				FromAgentID: "manager-1",
				ToAgentID:   agent.ID,
				Content:     "Status update needed.",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := teamStore.SetInbox(ctx, team.ID, agent.ID, inbox); err != nil {
		t.Fatalf("set inbox: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.Build(ctx, PromptBuildRequest{AgentID: agent.ID, TeamID: team.ID})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	agentIndex := strings.Index(prompt, "# Agent Files (Markdown)")
	respIndex := strings.Index(prompt, "# Team Responsibilities (RESPONSIBILITIES.md)")
	relIndex := strings.Index(prompt, "# Team Relationships")
	inboxIndex := strings.Index(prompt, "# Team Inbox")
	taskIndex := strings.Index(prompt, "# Heartbeat Task (HEARTBEAT.md)")

	if agentIndex == -1 || respIndex == -1 || relIndex == -1 || inboxIndex == -1 || taskIndex == -1 {
		t.Fatalf("expected all heartbeat sections in prompt")
	}

	if !(agentIndex < respIndex && respIndex < relIndex && relIndex < inboxIndex && inboxIndex < taskIndex) {
		t.Fatalf("prompt sections are out of order")
	}
}
