package heartbeat

import (
	"context"
	"prompt-manager/store"
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
	"sort"
	"strings"
	"testing"
	"time"
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

func promptHeadingIndex(prompt, heading string) int {
	if strings.HasPrefix(prompt, heading+"\n") {
		return 0
	}
	return strings.Index(prompt, "\n"+heading+"\n")
}

func promptSectionContent(sections []PromptSection, kind string) string {
	for _, section := range sections {
		if section.Kind == kind {
			return section.Content
		}
	}
	return ""
}

func joinedPromptSections(sections []PromptSection) string {
	var b strings.Builder
	for _, section := range sections {
		b.WriteString(section.Content)
		b.WriteString("\n")
	}
	return b.String()
}

func distinctSectionKinds(sections []PromptSection) []string {
	kinds := make([]string, 0, len(sections))
	seen := make(map[string]bool, len(sections))
	for _, section := range sections {
		if seen[section.Kind] {
			continue
		}
		seen[section.Kind] = true
		kinds = append(kinds, section.Kind)
	}
	return kinds
}

func sectionKindIndex(kinds []string, kind string) int {
	for i, candidate := range kinds {
		if candidate == kind {
			return i
		}
	}
	return -1
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

	team := newIndependentTestTeam("team-1", "Team One")
	team.Coordination.Pattern = teamconfig.CoordinationPatternPeer
	team.Coordination.ReportingMode = teamconfig.ReportingModeOrgChart
	team.Coordination.MessagingMode = teamconfig.MessagingModeAsyncInbox
	team.Coordination.Capabilities.ShowOrgContext = true
	team.Coordination.Capabilities.InjectInbox = true
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.WriteSharedFile(ctx, team.ID, "TEAM.md", "Operate as an initiative portfolio manager."); err != nil {
		t.Fatalf("write shared TEAM.md: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, team.ID, agent.ID, "Do the work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, team.ID, agent.ID, "Ship the update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}
	if err := teamStore.SetOrgChart(ctx, team.ID, &store.OrgChart{
		TeamID: team.ID,
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager-1", ReportAgentID: agent.ID},
		},
	}); err != nil {
		t.Fatalf("set org chart: %v", err)
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

	agentIndex := promptHeadingIndex(prompt, "# Agent Files (Markdown)")
	teamDocIndex := promptHeadingIndex(prompt, "# Team Charter (shared/TEAM.md)")
	briefIndex := promptHeadingIndex(prompt, promptHeadingActiveTaskBrief)
	contractIndex := promptHeadingIndex(prompt, "# Resolved Operating Contract")
	respIndex := promptHeadingIndex(prompt, "# Team Responsibilities (RESPONSIBILITIES.md)")
	orgIndex := promptHeadingIndex(prompt, "# Team Org Context")
	storageIndex := promptHeadingIndex(prompt, "# Storage Map")
	inboxIndex := promptHeadingIndex(prompt, "# Team Inbox")
	taskIndex := promptHeadingIndex(prompt, promptHeadingHeartbeatTask)
	reminderIndex := promptHeadingIndex(prompt, promptHeadingTaskReminder)

	if agentIndex == -1 || teamDocIndex == -1 || briefIndex == -1 || contractIndex == -1 || respIndex == -1 || orgIndex == -1 || storageIndex == -1 || inboxIndex == -1 || taskIndex == -1 || reminderIndex == -1 {
		t.Fatalf("expected all heartbeat sections in prompt")
	}

	if !(agentIndex < teamDocIndex && teamDocIndex < briefIndex && briefIndex < contractIndex && contractIndex < respIndex && respIndex < orgIndex && orgIndex < storageIndex && storageIndex < inboxIndex && inboxIndex < taskIndex && taskIndex < reminderIndex) {
		t.Fatalf("prompt sections are out of order")
	}

	// Verify coordination skill section is present between org context and storage map
	coordIndex := strings.Index(prompt, "# Team Coordination")
	if coordIndex == -1 {
		t.Fatalf("expected coordination skill section in prompt")
	}
	if !(orgIndex < coordIndex && coordIndex < storageIndex) {
		t.Fatalf("coordination skill section should be between org context and storage map")
	}

	for _, want := range []string{
		"# Storage Map",
		promptHeadingActiveTaskBrief,
		"The complete task source is included later in `# Heartbeat Task (HEARTBEAT.md)`.",
		"## Write Surface",
		"## Required Memory",
		"## Continue",
		"Use your final `## HANDOFF` for short-term continuity.",
		"## Observe",
		"If something expected was missing, broken, confusing, slow, undocumented, or harder than it should have been, capture it as friction.",
		"## Propose",
		"## Operate",
		"## Authority Order",
		"## Your Team Storage",
		"Primitive availability for this member:",
		"- decisions: `write-allowed`",
		"## Available Storage Commands",
		promptHeadingTaskReminder,
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing storage map content %q", want)
		}
	}
	if strings.Contains(prompt, "# Durable State") {
		t.Fatalf("prompt must not contain legacy durable state heading")
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
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-mp", "MP Team")); err != nil {
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

	if !strings.Contains(prompt, "team-coordination-independent") {
		t.Fatalf("expected independent coordination skill reference in prompt")
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
	if err := teamStore.Create(ctx, newLeaderLedSingleProcessTestTeam("team-sp", "SP Team", "agent-1")); err != nil {
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

	if !strings.Contains(prompt, "team-coordination-leader-led") {
		t.Fatalf("expected leader-led coordination skill reference in prompt")
	}
}

func TestBuildTeamLeadPromptIncludesApprovalConstraints(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	relationStore := fileStore.Relations()

	team := newLeaderLedSingleProcessTestTeam("team-sp", "SP Team", "director")
	team.DecisionMode = teamconfig.DecisionModeApproval
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	for _, agent := range []store.Agent{
		{ID: "director", DisplayName: "Director", Status: store.AgentStatusActive},
		{ID: "strategist", DisplayName: "Strategist", Status: store.AgentStatusActive},
	} {
		agentCopy := agent
		if err := agentStore.Create(ctx, &agentCopy); err != nil {
			t.Fatalf("create agent %s: %v", agent.ID, err)
		}
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID:  "team-sp",
			AgentID: agent.ID,
			Status:  store.MemberStatusActive,
		}); err != nil {
			t.Fatalf("set membership %s: %v", agent.ID, err)
		}
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	if err := teamStore.WriteSharedFile(ctx, "team-sp", "TEAM.md", "Focus on the initiative portfolio first."); err != nil {
		t.Fatalf("write shared TEAM.md: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-sp", "director", "Lead portfolio prioritization."); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, "team-sp", "director", "Review `swarm-manager overview --format json` before broad repo signals."); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}

	prompt, err := builder.BuildTeamLeadPrompt(ctx, "team-sp", "director", "/workdir")
	if err != nil {
		t.Fatalf("BuildTeamLeadPrompt: %v", err)
	}

	for _, want := range []string{
		"Lead Member Context",
		"Focus on the initiative portfolio first.",
		"Review `swarm-manager overview --format json` before broad repo signals.",
		"This team is running in `approval` decision mode.",
		"Do not create, import, or rename a team",
		"End your final response with a `## HANDOFF` section as the last section, then stop.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q", want)
		}
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
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
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
	if !strings.Contains(prompt, "The active heartbeat task is intentionally omitted from member context.") {
		t.Fatalf("BuildContext should explain that heartbeat task is omitted")
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

	team := newIndependentTestTeam("team-1", "Team One")
	team.Coordination.Pattern = teamconfig.CoordinationPatternPeer
	team.Coordination.ReportingMode = teamconfig.ReportingModeOrgChart
	team.Coordination.MessagingMode = teamconfig.MessagingModeAsyncInbox
	team.Coordination.Capabilities.ShowOrgContext = true
	team.Coordination.Capabilities.InjectInbox = true
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, team.ID, agent.ID, "Do the work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, team.ID, agent.ID, "Ship the update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}
	if err := teamStore.SetOrgChart(ctx, team.ID, &store.OrgChart{
		TeamID: team.ID,
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager-1", ReportAgentID: agent.ID},
		},
	}); err != nil {
		t.Fatalf("set org chart: %v", err)
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
		promptSectionKindActiveTaskBrief,
		"team-operating-contract",
		"team-responsibilities",
		"team-org-context",
		"team-coordination",
		"team-storage-map",
		"team-inbox",
		"heartbeat-task",
		promptSectionKindTaskReminder,
	}

	kindOrder := distinctSectionKinds(sections)

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

	team := newIndependentTestTeam("team-1", "Team One")
	team.Coordination.Pattern = teamconfig.CoordinationPatternPeer
	team.Coordination.ReportingMode = teamconfig.ReportingModeOrgChart
	team.Coordination.MessagingMode = teamconfig.MessagingModeAsyncInbox
	team.Coordination.Capabilities.ShowOrgContext = true
	team.Coordination.Capabilities.InjectInbox = true
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, team.ID, agent.ID, "Do the work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, team.ID, agent.ID, "Ship the update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}
	if err := teamStore.SetOrgChart(ctx, team.ID, &store.OrgChart{
		TeamID: team.ID,
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager-1", ReportAgentID: agent.ID},
		},
	}); err != nil {
		t.Fatalf("set org chart: %v", err)
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

func TestBundledVisionWalkPrepPromptUsesMemberAwareStorage(t *testing.T) {
	ctx := context.Background()
	fileStore := store.NewFileStore("../../store")
	builder := NewPromptBuilder(
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.Agents().(*store.FileAgentStore),
	)

	prompt, err := builder.Build(ctx, PromptBuildRequest{
		TeamID:  "director-swarm",
		AgentID: "vision-walk-prep",
	})
	if err != nil {
		t.Fatalf("build bundled vision-walk-prep prompt: %v", err)
	}

	for _, want := range []string{
		promptHeadingActiveTaskBrief,
		"Decision writes: not allowed for this member. Review decisions when useful; do not create them.",
		"Primitive availability for this member:",
		"- decisions: `review-only`",
		"- task board: `review-only`",
		"- Decision writes are not allowed for this member.",
		promptHeadingTaskReminder,
		"do not create decisions",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("bundled prompt missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"# Execution Brief",
		"execution-brief",
		"Always available:",
		"team decision-add director-swarm",
		"team task-add director-swarm",
		"team task-update director-swarm",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("bundled prompt contains forbidden text %q", forbidden)
		}
	}
}

func TestBundledPromptMatrixHardCutoverInvariants(t *testing.T) {
	ctx := context.Background()
	fileStore := store.NewFileStore("../../store")
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	builder := NewPromptBuilder(teamStore, agentStore)

	teams, err := teamStore.List(ctx)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].ID < teams[j].ID
	})

	for _, team := range teams {
		team := team
		if !team.Enabled || team.OperatingContract == nil {
			continue
		}
		t.Run(team.ID, func(t *testing.T) {
			members, err := teamStore.GetMembers(ctx, team.ID)
			if err != nil {
				t.Fatalf("list members: %v", err)
			}
			sort.Slice(members, func(i, j int) bool {
				return members[i].AgentID < members[j].AgentID
			})
			for _, relation := range members {
				relation := relation
				if relation.Status != "" && relation.Status != store.MemberStatusActive {
					continue
				}
				t.Run(relation.AgentID, func(t *testing.T) {
					member, ok := team.OperatingContract.Members[relation.AgentID]
					if !ok {
						t.Fatalf("missing operating contract member")
					}
					sections, err := builder.BuildStructured(ctx, PromptBuildRequest{
						TeamID:  team.ID,
						AgentID: relation.AgentID,
					})
					if err != nil {
						t.Fatalf("build structured prompt: %v", err)
					}

					kinds := distinctSectionKinds(sections)
					activeIndex := sectionKindIndex(kinds, promptSectionKindActiveTaskBrief)
					contractIndex := sectionKindIndex(kinds, "team-operating-contract")
					taskIndex := sectionKindIndex(kinds, "heartbeat-task")
					reminderIndex := sectionKindIndex(kinds, promptSectionKindTaskReminder)
					if activeIndex == -1 {
						t.Fatalf("missing %s section: %v", promptSectionKindActiveTaskBrief, kinds)
					}
					if contractIndex == -1 || activeIndex > contractIndex {
						t.Fatalf("active task brief must appear before operating contract: %v", kinds)
					}
					if taskIndex == -1 || reminderIndex == -1 || taskIndex > reminderIndex {
						t.Fatalf("heartbeat task must be followed by task reminder: %v", kinds)
					}
					if sections[len(sections)-1].Kind != promptSectionKindTaskReminder {
						t.Fatalf("final section = %q, want %q", sections[len(sections)-1].Kind, promptSectionKindTaskReminder)
					}

					prompt := joinedPromptSections(sections)
					for _, forbidden := range []string{
						"# Execution Brief",
						"execution-brief",
						"Always available:",
						"Exact paths: see",
						"Plan of record docs:",
					} {
						if strings.Contains(prompt, forbidden) {
							t.Fatalf("prompt contains legacy phrase %q", forbidden)
						}
					}
					if strings.Contains(prompt, "docs under `") {
						t.Fatalf("prompt contains grouped plan-of-record count wording")
					}
					if strings.Contains(prompt, "## Available Storage Commands") && !strings.Contains(prompt, "Primitive availability for this member:") {
						t.Fatalf("storage commands rendered without member primitive availability")
					}
					assertMemberWriteCommandsMatchContract(t, prompt, team.ID, member)
				})
			}
		})
	}
}

func assertMemberWriteCommandsMatchContract(t *testing.T, prompt, teamID string, member teamcontract.MemberContract) {
	t.Helper()
	if !memberCanWriteDecision(member) && strings.Contains(prompt, "team decision-add "+teamID) {
		t.Fatalf("decision-add rendered for a member that cannot create decisions")
	}
	if !memberCanWriteKindOrPath(member, "knowledge", "knowledge.jsonl") && strings.Contains(prompt, "team knowledge-add "+teamID) {
		t.Fatalf("knowledge-add rendered for a member that cannot write knowledge")
	}
	if !memberCanWriteKindOrPath(member, "task", "tasks.json") {
		for _, forbidden := range []string{"team task-add " + teamID, "team task-update " + teamID} {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("%s rendered for a member that cannot mutate the task board", forbidden)
			}
		}
	}
}

func memberCanWriteDecision(member teamcontract.MemberContract) bool {
	if member.NewDecisionCapPerHeartbeat != nil && *member.NewDecisionCapPerHeartbeat == 0 {
		return false
	}
	return memberCanWriteKindOrPath(member, "decision", "decisions.jsonl")
}

func memberCanWriteKindOrPath(member teamcontract.MemberContract, kind, pathSuffix string) bool {
	if writeRefsContainKind(member.ForbiddenWrites, kind) {
		return false
	}
	for _, ref := range member.AllowedWrites {
		if ref.Kind == kind {
			return true
		}
		if ref.Kind == "" && strings.HasSuffix(ref.Path, pathSuffix) {
			return true
		}
	}
	return false
}

func TestActiveTaskBriefUsesHumanReadableWriteSurface(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	team := newIndependentTestTeam("team-1", "Team One")
	optional := false
	team.OperatingContract.Documents.SharedState = append(team.OperatingContract.Documents.SharedState, teamcontract.SharedStateDocument{
		ID:             "ledger",
		Path:           teamcontract.PathRef{Base: teamcontract.BaseTeamShared, Path: "ledger.jsonl", Required: &optional, OptionalReason: "test fixture"},
		Kind:           teamcontract.TeamWorkingStateKindAppendOnlyEventLog,
		Required:       false,
		OptionalReason: "test fixture",
	})
	member := team.OperatingContract.Members["agent-1"]
	member.AllowedWrites = []teamcontract.WriteRef{
		{Base: teamcontract.BaseTeamShared, Path: "knowledge.jsonl"},
		{Base: teamcontract.BaseTeamShared, Path: "decisions.jsonl"},
		{Base: teamcontract.BaseTeamShared, Path: "ledger.jsonl"},
		{Kind: "handoff"},
	}
	team.OperatingContract.Members["agent-1"] = member
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, "team-1", "agent-1", "# Heartbeat: Test\n\nDo work."); err != nil {
		t.Fatalf("set heartbeat: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{AgentID: "agent-1", TeamID: "team-1"})
	if err != nil {
		t.Fatalf("BuildStructured: %v", err)
	}
	brief := promptSectionContent(sections, promptSectionKindActiveTaskBrief)
	if brief == "" {
		t.Fatalf("active task brief not found")
	}
	for _, want := range []string{
		"- decision proposals",
		"- knowledge observations and friction signals",
		"- team working state `scenarios/prompt-manager/store/teams/team-1/shared/ledger.jsonl` (append-only event log)",
		"- final `## HANDOFF` continuity",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("active task brief missing %q:\n%s", want, brief)
		}
	}
	for _, tooRaw := range []string{
		"`scenarios/prompt-manager/store/teams/team-1/shared/decisions.jsonl`",
		"`scenarios/prompt-manager/store/teams/team-1/shared/knowledge.jsonl`",
	} {
		if strings.Contains(brief, tooRaw) {
			t.Fatalf("active task brief should use semantic label instead of raw path %q:\n%s", tooRaw, brief)
		}
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
	team := newIndependentTestTeam("team-1", "Team")
	team.Coordination.Pattern = teamconfig.CoordinationPatternPeer
	team.Coordination.ReportingMode = teamconfig.ReportingModeOrgChart
	team.Coordination.MessagingMode = teamconfig.MessagingModeAsyncInbox
	team.Coordination.Capabilities.ShowOrgContext = true
	team.Coordination.Capabilities.InjectInbox = true
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-1", "agent-1", "Do work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, "team-1", "agent-1", "Ship update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}
	if err := teamStore.SetOrgChart(ctx, "team-1", &store.OrgChart{
		TeamID: "team-1",
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager", ReportAgentID: "agent-1"},
		},
	}); err != nil {
		t.Fatalf("set org chart: %v", err)
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
		promptHeadingActiveTaskBrief,
		"# Team Responsibilities (RESPONSIBILITIES.md)",
		"# Team Org Context",
		"# Team Coordination",
		"# Storage Map",
		"# Team Inbox",
	} {
		if !strings.Contains(prompt, section) {
			t.Fatalf("BuildContext should include %q section", section)
		}
	}
}
