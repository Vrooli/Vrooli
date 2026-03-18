package interop

import (
	"encoding/json"
	"fmt"
	"strings"

	"prompt-manager/store"
	"prompt-manager/validation"
)

// CCTeamConfig mirrors the structure of ~/.claude/teams/{name}/config.json.
type CCTeamConfig struct {
	TeamName    string     `json:"team_name"`
	Description string     `json:"description"`
	Members     []CCMember `json:"members"`
}

// CCMember represents a member in a Claude Code team config.
type CCMember struct {
	Name      string `json:"name"`
	AgentType string `json:"agentType"`
	Model     string `json:"model,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

// ParseCCConfig parses a Claude Code team config JSON into a ToolTeamConfig.
// teamNameFallback is used if the config's team_name field is empty.
func ParseCCConfig(data []byte, teamNameFallback string) (*ToolTeamConfig, error) {
	var ccConfig CCTeamConfig
	if err := json.Unmarshal(data, &ccConfig); err != nil {
		return nil, fmt.Errorf("invalid CC team config: %w", err)
	}

	cfg := &ToolTeamConfig{
		TeamName:    ccConfig.TeamName,
		Description: ccConfig.Description,
		Members:     make([]ToolMember, 0, len(ccConfig.Members)),
	}
	if cfg.TeamName == "" {
		cfg.TeamName = teamNameFallback
	}
	for _, m := range ccConfig.Members {
		cfg.Members = append(cfg.Members, ToolMember{
			Name:      m.Name,
			AgentType: m.AgentType,
			Model:     m.Model,
			Mode:      m.Mode,
		})
	}
	return cfg, nil
}

// ClaudeCodeConverter implements Converter for the Claude Code teams system.
type ClaudeCodeConverter struct{}

var _ Converter = (*ClaudeCodeConverter)(nil)

// ToolID returns "claude-code".
func (c *ClaudeCodeConverter) ToolID() string { return "claude-code" }

// FromPMTeam converts a prompt-manager team snapshot to a Claude Code team config.
func (c *ClaudeCodeConverter) FromPMTeam(snapshot *PMTeamSnapshot) (*ToolTeamConfig, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot must not be nil")
	}

	cfg := &ToolTeamConfig{
		TeamName:    snapshot.Team.ID,
		Description: snapshot.Team.Mission,
		Members:     make([]ToolMember, 0, len(snapshot.Members)),
		Metadata:    map[string]any{},
	}

	for _, m := range snapshot.Members {
		tm := ToolMember{
			Name:      m.Agent.ID,
			AgentType: resolveAgentType(m.Agent),
		}
		if len(m.Agent.Tags) > 0 {
			tm.Metadata = map[string]any{"tags": m.Agent.Tags}
		}
		cfg.Members = append(cfg.Members, tm)
	}

	return cfg, nil
}

// resolveAgentType derives a Claude Code agent type string from agent metadata.
func resolveAgentType(agent store.Agent) string {
	// Check tags first (most explicit signal).
	for _, tag := range agent.Tags {
		lower := strings.ToLower(tag)
		if strings.Contains(lower, "explore") {
			return "Explore"
		}
		if strings.Contains(lower, "bash") {
			return "Bash"
		}
	}

	// Check capabilities for secondary heuristics.
	if agent.Capabilities != nil {
		for _, cap := range agent.Capabilities.Provides {
			lower := strings.ToLower(cap.CapabilityID)
			if strings.Contains(lower, "explore") {
				return "Explore"
			}
			if strings.Contains(lower, "bash") {
				return "Bash"
			}
		}
	}

	// Check connectors as a tertiary signal.
	for _, conn := range agent.Connectors {
		lower := strings.ToLower(conn.Type)
		if strings.Contains(lower, "explore") {
			return "Explore"
		}
		if strings.Contains(lower, "bash") {
			return "Bash"
		}
	}

	return "general-purpose"
}

// ToPMTeam converts a Claude Code team config into prompt-manager import structs.
func (c *ClaudeCodeConverter) ToPMTeam(config *ToolTeamConfig) (*PMTeamImport, error) {
	if config == nil {
		return nil, fmt.Errorf("config must not be nil")
	}

	teamID := validation.Slugify(config.TeamName)
	if teamID == "" {
		return nil, fmt.Errorf("team name %q produces empty slug", config.TeamName)
	}

	imp := &PMTeamImport{
		Team: store.Team{
			BaseEntity: store.BaseEntity{
				Kind:          store.KindTeam,
				SchemaVersion: store.CurrentSchemaVersion,
			},
			ID:          teamID,
			DisplayName: config.TeamName,
			Mission:     config.Description,
			Enabled:     true,
			Timestamps:  store.NewTimestamps(),
		},
		Agents:   make([]store.Agent, 0, len(config.Members)),
		Members:  make([]store.TeamMemberRelation, 0, len(config.Members)),
		OrgEdges: make([]store.OrgEdge, 0),
	}

	var leadAgentID string
	for i, m := range config.Members {
		agentID := validation.Slugify(m.Name)
		if agentID == "" {
			agentID = fmt.Sprintf("member-%d", i)
		}

		agent := store.Agent{
			BaseEntity: store.BaseEntity{
				Kind:          store.KindAgent,
				SchemaVersion: store.CurrentSchemaVersion,
			},
			ID:          agentID,
			DisplayName: m.Name,
			Status:      store.AgentStatusActive,
			Timestamps:  store.NewTimestamps(),
		}
		imp.Agents = append(imp.Agents, agent)

		rel := store.TeamMemberRelation{
			BaseEntity: store.BaseEntity{
				Kind:          store.KindTeamMember,
				SchemaVersion: store.CurrentSchemaVersion,
			},
			TeamID:  teamID,
			AgentID: agentID,
			Status:  store.MemberStatusActive,
		}
		imp.Members = append(imp.Members, rel)

		// First member is the team lead.
		if i == 0 {
			leadAgentID = agentID
		} else {
			imp.OrgEdges = append(imp.OrgEdges, store.OrgEdge{
				ManagerAgentID: leadAgentID,
				ReportAgentID:  agentID,
			})
		}
	}

	return imp, nil
}

// FormatSpawnPrompt generates a structured markdown prompt that instructs
// a Claude Code team lead to bootstrap the team session.
func (c *ClaudeCodeConverter) FormatSpawnPrompt(config *ToolTeamConfig, ctx SpawnContext) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config must not be nil")
	}

	var b strings.Builder

	// Header
	b.WriteString("# Team Spawn Instructions\n\n")
	b.WriteString(fmt.Sprintf("You are the team lead for **%s**.\n\n", config.TeamName))
	if config.Description != "" {
		b.WriteString(fmt.Sprintf("**Mission:** %s\n\n", config.Description))
	}

	// Team roster
	b.WriteString("## 1. Team Roster\n\n")
	for i, m := range config.Members {
		b.WriteString(fmt.Sprintf("- **%s** (type: %s)", m.Name, m.AgentType))
		if i == 0 {
			b.WriteString(" [team lead — you]")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Member spawning
	b.WriteString("## 2. Spawn Teammates\n\n")
	b.WriteString("Spawn each teammate as a subagent:\n\n")
	for _, m := range config.Members[1:] { // skip lead (self)
		b.WriteString(fmt.Sprintf("- Spawn **%s** with type `%s`\n", m.Name, m.AgentType))
	}
	if len(config.Members) > 1 {
		b.WriteString("\n")
	}

	// Coordination
	b.WriteString("## 3. Coordination\n\n")
	b.WriteString("- Communicate with teammates via your messaging capability\n")
	b.WriteString("- Track progress via the shared task board (run `prompt-manager team task-list` via Bash)\n")
	b.WriteString("- Monitor teammate status and reassign work as needed\n\n")

	// Org chart
	b.WriteString("## 4. Org Chart\n\n")
	if len(config.Members) > 0 {
		b.WriteString(fmt.Sprintf("- **Lead:** %s\n", config.Members[0].Name))
		for _, m := range config.Members[1:] {
			b.WriteString(fmt.Sprintf("  - Reports to lead: %s\n", m.Name))
		}
		b.WriteString("\n")
	}

	// Context
	if ctx.WorkingDir != "" || ctx.VrooliRoot != "" || ctx.TeamID != "" || ctx.AdditionalCtx != "" {
		b.WriteString("## 5. Context\n\n")
		if ctx.WorkingDir != "" {
			b.WriteString(fmt.Sprintf("- **Working directory:** %s\n", ctx.WorkingDir))
		}
		if ctx.VrooliRoot != "" {
			b.WriteString(fmt.Sprintf("- **Vrooli root:** %s\n", ctx.VrooliRoot))
		}
		if ctx.TeamID != "" {
			b.WriteString(fmt.Sprintf("- **Team ID:** %s\n", ctx.TeamID))
		}
		if ctx.AdditionalCtx != "" {
			b.WriteString(fmt.Sprintf("\n%s\n", ctx.AdditionalCtx))
		}
	}

	return b.String(), nil
}
