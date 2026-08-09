package interop

import (
	"strings"
	"testing"

	"prompt-manager/store"
)

func makeSnapshot(members []PMTeamMember) *PMTeamSnapshot {
	return &PMTeamSnapshot{
		Team: store.Team{
			BaseEntity: store.BaseEntity{Kind: store.KindTeam, SchemaVersion: store.CurrentSchemaVersion},
			ID:         "alpha-team",
			Mission:    "Build the widget",
			Enabled:    true,
		},
		Members: members,
		Roles:   []store.Role{{ID: "dev", Name: "Developer"}},
		OrgEdges: []store.OrgEdge{
			{ManagerAgentID: "lead", ReportAgentID: "worker"},
		},
	}
}

func makeAgent(id string, tags []string) store.Agent {
	return store.Agent{
		BaseEntity: store.BaseEntity{Kind: store.KindAgent, SchemaVersion: store.CurrentSchemaVersion},
		ID:         id,
		Status:     store.AgentStatusActive,
		Tags:       tags,
	}
}

func TestFromPMTeam_Basic(t *testing.T) {
	snap := makeSnapshot([]PMTeamMember{
		{
			Agent:    makeAgent("lead", []string{"planning"}),
			Relation: store.TeamMemberRelation{TeamID: "alpha-team", AgentID: "lead", Status: store.MemberStatusActive},
		},
		{
			Agent:    makeAgent("worker", []string{"coding"}),
			Relation: store.TeamMemberRelation{TeamID: "alpha-team", AgentID: "worker", Status: store.MemberStatusActive},
		},
	})

	conv := &ClaudeCodeConverter{}
	cfg, err := conv.FromPMTeam(snap)
	if err != nil {
		t.Fatalf("FromPMTeam returned error: %v", err)
	}

	if cfg.TeamName != "alpha-team" {
		t.Errorf("TeamName = %q, want %q", cfg.TeamName, "alpha-team")
	}
	if cfg.Description != "Build the widget" {
		t.Errorf("Description = %q, want %q", cfg.Description, "Build the widget")
	}
	if len(cfg.Members) != 2 {
		t.Fatalf("len(Members) = %d, want 2", len(cfg.Members))
	}
	if cfg.Members[0].Name != "lead" {
		t.Errorf("Members[0].Name = %q, want %q", cfg.Members[0].Name, "lead")
	}
	if cfg.Members[1].Name != "worker" {
		t.Errorf("Members[1].Name = %q, want %q", cfg.Members[1].Name, "worker")
	}
}

func TestFromPMTeam_NilSnapshot(t *testing.T) {
	conv := &ClaudeCodeConverter{}
	_, err := conv.FromPMTeam(nil)
	if err == nil {
		t.Fatal("expected error for nil snapshot")
	}
}

func TestToPMTeam_Basic(t *testing.T) {
	cfg := &ToolTeamConfig{
		TeamName:    "My Cool Team",
		Description: "Do great things",
		Members: []ToolMember{
			{Name: "Alice", AgentType: "general-purpose"},
			{Name: "Bob", AgentType: "Explore"},
		},
	}

	conv := &ClaudeCodeConverter{}
	imp, err := conv.ToPMTeam(cfg)
	if err != nil {
		t.Fatalf("ToPMTeam returned error: %v", err)
	}

	// Team
	if imp.Team.ID != "my-cool-team" {
		t.Errorf("Team.ID = %q, want %q", imp.Team.ID, "my-cool-team")
	}
	if imp.Team.DisplayName != "My Cool Team" {
		t.Errorf("Team.DisplayName = %q, want %q", imp.Team.DisplayName, "My Cool Team")
	}
	if imp.Team.Mission != "Do great things" {
		t.Errorf("Team.Mission = %q, want %q", imp.Team.Mission, "Do great things")
	}
	if !imp.Team.Enabled {
		t.Error("Team.Enabled = false, want true")
	}

	// Agents
	if len(imp.Agents) != 2 {
		t.Fatalf("len(Agents) = %d, want 2", len(imp.Agents))
	}
	if imp.Agents[0].ID != "alice" {
		t.Errorf("Agents[0].ID = %q, want %q", imp.Agents[0].ID, "alice")
	}
	if imp.Agents[0].DisplayName != "Alice" {
		t.Errorf("Agents[0].DisplayName = %q, want %q", imp.Agents[0].DisplayName, "Alice")
	}
	if imp.Agents[0].Status != store.AgentStatusActive {
		t.Errorf("Agents[0].Status = %q, want %q", imp.Agents[0].Status, store.AgentStatusActive)
	}

	// Members
	if len(imp.Members) != 2 {
		t.Fatalf("len(Members) = %d, want 2", len(imp.Members))
	}
	if imp.Members[0].TeamID != "my-cool-team" {
		t.Errorf("Members[0].TeamID = %q, want %q", imp.Members[0].TeamID, "my-cool-team")
	}
	if imp.Members[0].Status != store.MemberStatusActive {
		t.Errorf("Members[0].Status = %q, want %q", imp.Members[0].Status, store.MemberStatusActive)
	}

	// OrgEdges: first member is lead, second reports to first
	if len(imp.OrgEdges) != 1 {
		t.Fatalf("len(OrgEdges) = %d, want 1", len(imp.OrgEdges))
	}
	if imp.OrgEdges[0].ManagerAgentID != "alice" {
		t.Errorf("OrgEdges[0].ManagerAgentID = %q, want %q", imp.OrgEdges[0].ManagerAgentID, "alice")
	}
	if imp.OrgEdges[0].ReportAgentID != "bob" {
		t.Errorf("OrgEdges[0].ReportAgentID = %q, want %q", imp.OrgEdges[0].ReportAgentID, "bob")
	}
}

func TestToPMTeam_NilConfig(t *testing.T) {
	conv := &ClaudeCodeConverter{}
	_, err := conv.ToPMTeam(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestToPMTeam_EmptyTeamName(t *testing.T) {
	conv := &ClaudeCodeConverter{}
	_, err := conv.ToPMTeam(&ToolTeamConfig{TeamName: "!!!"})
	if err == nil {
		t.Fatal("expected error for team name that produces empty slug")
	}
}

func TestRoundTrip(t *testing.T) {
	snap := makeSnapshot([]PMTeamMember{
		{
			Agent:    makeAgent("lead-agent", nil),
			Relation: store.TeamMemberRelation{TeamID: "alpha-team", AgentID: "lead-agent", Status: store.MemberStatusActive},
		},
		{
			Agent:    makeAgent("dev-agent", []string{"explore"}),
			Relation: store.TeamMemberRelation{TeamID: "alpha-team", AgentID: "dev-agent", Status: store.MemberStatusActive},
		},
	})

	conv := &ClaudeCodeConverter{}

	// PM -> CC
	cfg, err := conv.FromPMTeam(snap)
	if err != nil {
		t.Fatalf("FromPMTeam: %v", err)
	}

	// CC -> PM
	imp, err := conv.ToPMTeam(cfg)
	if err != nil {
		t.Fatalf("ToPMTeam: %v", err)
	}

	// Verify essential data preserved.
	if imp.Team.ID != snap.Team.ID {
		t.Errorf("round-trip Team.ID = %q, want %q", imp.Team.ID, snap.Team.ID)
	}
	if imp.Team.Mission != snap.Team.Mission {
		t.Errorf("round-trip Team.Mission = %q, want %q", imp.Team.Mission, snap.Team.Mission)
	}
	if len(imp.Agents) != len(snap.Members) {
		t.Errorf("round-trip agent count = %d, want %d", len(imp.Agents), len(snap.Members))
	}
}

func TestAgentTypeMapping(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		caps     *store.AgentCapabilities
		conns    []store.AgentConnector
		wantType string
	}{
		{
			name:     "explore tag",
			tags:     []string{"explore", "research"},
			wantType: "Explore",
		},
		{
			name:     "bash tag",
			tags:     []string{"bash-only"},
			wantType: "Bash",
		},
		{
			name:     "no tags defaults to general-purpose",
			wantType: "general-purpose",
		},
		{
			name: "explore capability",
			caps: &store.AgentCapabilities{
				Provides: []store.AgentCapability{
					{CapabilityID: "code-explore"},
				},
			},
			wantType: "Explore",
		},
		{
			name: "bash connector",
			conns: []store.AgentConnector{
				{Type: "bash-runner", ID: "b1", Enabled: true},
			},
			wantType: "Bash",
		},
		{
			name:     "explore tag takes priority over bash capability",
			tags:     []string{"explore"},
			caps:     &store.AgentCapabilities{Provides: []store.AgentCapability{{CapabilityID: "bash"}}},
			wantType: "Explore",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agent := store.Agent{
				Tags:         tc.tags,
				Capabilities: tc.caps,
				Connectors:   tc.conns,
			}
			got := resolveAgentType(agent)
			if got != tc.wantType {
				t.Errorf("resolveAgentType() = %q, want %q", got, tc.wantType)
			}
		})
	}
}

func TestFormatSpawnPrompt(t *testing.T) {
	cfg := &ToolTeamConfig{
		TeamName:    "widget-builders",
		Description: "Build widgets fast",
		Members: []ToolMember{
			{Name: "leader", AgentType: "general-purpose"},
			{Name: "researcher", AgentType: "Explore"},
			{Name: "coder", AgentType: "Bash"},
		},
	}

	ctx := SpawnContext{
		WorkingDir:    "/home/user/project",
		VrooliRoot:    "/opt/vrooli",
		TeamID:        "widget-builders",
		AdditionalCtx: "Focus on performance.",
	}

	conv := &ClaudeCodeConverter{}
	prompt, err := conv.FormatSpawnPrompt(cfg, ctx)
	if err != nil {
		t.Fatalf("FormatSpawnPrompt returned error: %v", err)
	}

	requiredSections := []string{
		"Leader-Led Team Heartbeat Instructions",
		"Existing Team",
		"Team Roster",
		"Spawn Direct Reports",
		"Coordination",
		"Operating Loop",
		"Org Chart",
		"Context",
	}
	for _, section := range requiredSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt missing section %q", section)
		}
	}

	// Verify member names appear.
	for _, m := range cfg.Members {
		if !strings.Contains(prompt, m.Name) {
			t.Errorf("prompt missing member name %q", m.Name)
		}
	}

	// Verify context values appear.
	if !strings.Contains(prompt, ctx.WorkingDir) {
		t.Errorf("prompt missing WorkingDir %q", ctx.WorkingDir)
	}
	if !strings.Contains(prompt, ctx.AdditionalCtx) {
		t.Errorf("prompt missing AdditionalCtx %q", ctx.AdditionalCtx)
	}
	if !strings.Contains(prompt, "## HANDOFF") {
		t.Errorf("prompt missing handoff contract")
	}
	if !strings.Contains(prompt, "Do not create, import, or rename a team") {
		t.Errorf("prompt missing existing-team constraint")
	}
	for _, want := range []string{
		"team-specific planning surface named in your lead context",
		"current priorities, blockers, and recommended next moves",
		"### Lead Context",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing updated guidance %q", want)
		}
	}
}

func TestFormatSpawnPrompt_NilConfig(t *testing.T) {
	conv := &ClaudeCodeConverter{}
	_, err := conv.FormatSpawnPrompt(nil, SpawnContext{})
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestToolID(t *testing.T) {
	conv := &ClaudeCodeConverter{}
	if id := conv.ToolID(); id != "claude-code" {
		t.Errorf("ToolID() = %q, want %q", id, "claude-code")
	}
}

// ============== ParseCCConfig Tests ==============

func TestParseCCConfig_Basic(t *testing.T) {
	data := []byte(`{
		"team_name": "test-team",
		"description": "A test team",
		"members": [
			{"name": "lead", "agentType": "general-purpose", "model": "opus", "mode": "plan"},
			{"name": "worker", "agentType": "Explore"}
		]
	}`)

	cfg, err := ParseCCConfig(data, "fallback")
	if err != nil {
		t.Fatalf("ParseCCConfig returned error: %v", err)
	}
	if cfg.TeamName != "test-team" {
		t.Errorf("TeamName = %q, want %q", cfg.TeamName, "test-team")
	}
	if cfg.Description != "A test team" {
		t.Errorf("Description = %q, want %q", cfg.Description, "A test team")
	}
	if len(cfg.Members) != 2 {
		t.Fatalf("len(Members) = %d, want 2", len(cfg.Members))
	}
	if cfg.Members[0].Model != "opus" {
		t.Errorf("Members[0].Model = %q, want %q", cfg.Members[0].Model, "opus")
	}
	if cfg.Members[0].Mode != "plan" {
		t.Errorf("Members[0].Mode = %q, want %q", cfg.Members[0].Mode, "plan")
	}
}

func TestParseCCConfig_FallbackName(t *testing.T) {
	data := []byte(`{"members": [{"name": "a", "agentType": "general-purpose"}]}`)

	cfg, err := ParseCCConfig(data, "my-fallback")
	if err != nil {
		t.Fatalf("ParseCCConfig returned error: %v", err)
	}
	if cfg.TeamName != "my-fallback" {
		t.Errorf("TeamName = %q, want fallback %q", cfg.TeamName, "my-fallback")
	}
}

func TestParseCCConfig_InvalidJSON(t *testing.T) {
	_, err := ParseCCConfig([]byte(`{invalid`), "fallback")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseCCConfig_EmptyMembers(t *testing.T) {
	data := []byte(`{"team_name": "empty", "members": []}`)

	cfg, err := ParseCCConfig(data, "")
	if err != nil {
		t.Fatalf("ParseCCConfig returned error: %v", err)
	}
	if len(cfg.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(cfg.Members))
	}
}
