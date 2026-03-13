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

	// Verify coordination skill section is present between relationships and inbox
	coordIndex := strings.Index(prompt, "# Team Coordination")
	if coordIndex == -1 {
		t.Fatalf("expected coordination skill section in prompt")
	}
	if !(relIndex < coordIndex && coordIndex < inboxIndex) {
		t.Fatalf("coordination skill section should be between relationships and inbox")
	}
}

func TestBuildIncludesCoordinationSkillMultiProcess(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, &store.Team{ID: "team-mp", DisplayName: "MP Team", Enabled: true, SpawnMode: "multi-process"}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-mp", "agent-1", "Do work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.Build(ctx, PromptBuildRequest{AgentID: "agent-1", TeamID: "team-mp"})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if !strings.Contains(prompt, "team-coordination-multi-process") {
		t.Fatalf("expected multi-process coordination skill reference in prompt")
	}
}

func TestBuildIncludesCoordinationSkillSingleProcess(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, &store.Team{ID: "team-sp", DisplayName: "SP Team", Enabled: true, SpawnMode: "single-process"}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-sp", "agent-1", "Do work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.Build(ctx, PromptBuildRequest{AgentID: "agent-1", TeamID: "team-sp"})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if !strings.Contains(prompt, "team-coordination-single-process") {
		t.Fatalf("expected single-process coordination skill reference in prompt")
	}
}

func TestBuildContextOmitsHeartbeatSection(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, &store.Team{ID: "team-1", DisplayName: "Team", Enabled: true}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-1", "agent-1", "Do work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, "team-1", "agent-1", "Ship update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.BuildContext(ctx, PromptBuildRequest{AgentID: "agent-1", TeamID: "team-1"})
	if err != nil {
		t.Fatalf("build context: %v", err)
	}

	if strings.Contains(prompt, "# Heartbeat Task") {
		t.Fatalf("BuildContext should not include heartbeat task section")
	}
	if strings.Contains(prompt, "Ship update") {
		t.Fatalf("BuildContext should not include heartbeat instructions content")
	}
}

func TestBuildStructuredAgentOnly(t *testing.T) {
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
	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{AgentID: agent.ID})
	if err != nil {
		t.Fatalf("build structured: %v", err)
	}

	if len(sections) == 0 {
		t.Fatalf("expected at least one section")
	}

	for _, s := range sections {
		if s.Kind != "agent-file" {
			t.Fatalf("expected only agent-file sections for agent-only prompt, got %q", s.Kind)
		}
		if s.Label == "" {
			t.Fatalf("expected non-empty label on agent-file section")
		}
		if s.SourcePath == "" {
			t.Fatalf("expected non-empty source path on agent-file section")
		}
		if !strings.HasPrefix(s.SourcePath, "agents/agent-1/") {
			t.Fatalf("expected source path to start with agents/agent-1/, got %q", s.SourcePath)
		}
	}

	// Should not contain any team sections
	for _, s := range sections {
		if strings.HasPrefix(s.Kind, "team-") || s.Kind == "heartbeat-task" {
			t.Fatalf("did not expect %q section in agent-only structured prompt", s.Kind)
		}
	}
}

func TestBuildStructuredTeamContext(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	agent := &store.Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
		Status:      store.AgentStatusActive,
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
	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{AgentID: agent.ID, TeamID: team.ID})
	if err != nil {
		t.Fatalf("build structured: %v", err)
	}

	// Verify all expected kinds in correct order
	expectedKinds := []string{
		"agent-file",
		"team-responsibilities",
		"team-relationships",
		"team-coordination",
		"team-inbox",
		"heartbeat-task",
	}

	kindOrder := make([]string, 0, len(sections))
	seen := make(map[string]bool)
	for _, s := range sections {
		if !seen[s.Kind] {
			kindOrder = append(kindOrder, s.Kind)
			seen[s.Kind] = true
		}
	}

	if len(kindOrder) != len(expectedKinds) {
		t.Fatalf("expected %d distinct kinds, got %d: %v", len(expectedKinds), len(kindOrder), kindOrder)
	}
	for i, kind := range expectedKinds {
		if kindOrder[i] != kind {
			t.Fatalf("expected kind[%d] = %q, got %q", i, kind, kindOrder[i])
		}
	}
}

func TestBuildStructuredMatchesBuild(t *testing.T) {
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
	// SOUL.md is auto-created by the agent store, so only create NOTES.md
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

	// Get flat Build() output
	flatPrompt, err := builder.Build(ctx, PromptBuildRequest{AgentID: agent.ID, TeamID: team.ID})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Get structured sections and reassemble
	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{AgentID: agent.ID, TeamID: team.ID})
	if err != nil {
		t.Fatalf("BuildStructured: %v", err)
	}

	// Reassemble: group adjacent agent-file sections with header, join with separator
	var parts []string
	for i := 0; i < len(sections); i++ {
		if sections[i].Kind == "agent-file" {
			block := "# Agent Files (Markdown)\n\n"
			for i < len(sections) && sections[i].Kind == "agent-file" {
				block += sections[i].Content
				i++
			}
			i--
			parts = append(parts, block)
		} else {
			parts = append(parts, sections[i].Content)
		}
	}
	reassembled := strings.Join(parts, "\n\n---\n\n")

	if flatPrompt != reassembled {
		t.Fatalf("Build() output does not match reassembled structured sections.\n\n--- Build() ---\n%s\n\n--- Reassembled ---\n%s", flatPrompt, reassembled)
	}
}

func TestBuildContextIncludesAllOtherSections(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := agentStore.CreateFile(ctx, "agent-1", "NOTES.md", "My notes", false); err != nil {
		t.Fatalf("create notes file: %v", err)
	}
	if err := teamStore.Create(ctx, &store.Team{ID: "team-1", DisplayName: "Team", Enabled: true}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-1", "agent-1", "Do work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, "team-1", "agent-1", "Ship update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}

	inbox := &store.TeamInbox{
		Messages: []store.TeamMessage{
			{
				ID:          "msg-1",
				TeamID:      "team-1",
				FromAgentID: "manager",
				ToAgentID:   "agent-1",
				Content:     "Check status",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := teamStore.SetInbox(ctx, "team-1", "agent-1", inbox); err != nil {
		t.Fatalf("set inbox: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.BuildContext(ctx, PromptBuildRequest{AgentID: "agent-1", TeamID: "team-1"})
	if err != nil {
		t.Fatalf("build context: %v", err)
	}

	// All sections except heartbeat should be present
	for _, section := range []string{
		"# Agent Files (Markdown)",
		"# Team Responsibilities (RESPONSIBILITIES.md)",
		"# Team Relationships",
		"# Team Coordination",
		"# Team Inbox",
	} {
		if !strings.Contains(prompt, section) {
			t.Fatalf("BuildContext should include %q section", section)
		}
	}
}
