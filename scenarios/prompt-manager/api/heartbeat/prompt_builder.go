package heartbeat

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"prompt-manager/interop"
	"prompt-manager/store"
)

// PromptBuildRequest defines the inputs for assembling a heartbeat prompt.
// TeamID is optional; when omitted, only agent markdown files are included.
type PromptBuildRequest struct {
	TeamID  string
	AgentID string
}

// PromptBuilder assembles prompts using agent + team context.
type PromptBuilder struct {
	teamStore  *store.FileTeamStore
	agentStore *store.FileAgentStore
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder(teamStore *store.FileTeamStore, agentStore *store.FileAgentStore) *PromptBuilder {
	return &PromptBuilder{
		teamStore:  teamStore,
		agentStore: agentStore,
	}
}

// Build constructs a prompt from agent files, team responsibilities, relationships, inbox, and heartbeat instructions.
func (b *PromptBuilder) Build(ctx context.Context, req PromptBuildRequest) (string, error) {
	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		return "", fmt.Errorf("agentId is required")
	}
	if b.agentStore == nil {
		return "", fmt.Errorf("agent store is not configured")
	}

	agent, err := b.agentStore.Get(ctx, agentID)
	if err != nil {
		return "", err
	}

	teamID := strings.TrimSpace(req.TeamID)
	includeTeam := teamID != ""
	if includeTeam {
		if b.teamStore == nil {
			return "", fmt.Errorf("team store is not configured")
		}
		if _, err := b.teamStore.Get(ctx, teamID); err != nil {
			return "", err
		}
	}

	var parts []string

	// 1. Agent markdown files (global personality + notes)
	agentFiles, err := b.agentStore.ListFiles(ctx, agentID)
	if err == nil && len(agentFiles) > 0 {
		var markdownFiles []store.AgentFileEntry
		for _, entry := range agentFiles {
			if entry.IsDir {
				continue
			}
			if strings.HasSuffix(strings.ToLower(entry.Path), ".md") {
				markdownFiles = append(markdownFiles, entry)
			}
		}

		if len(markdownFiles) > 0 {
			markdownFiles = orderAgentMarkdownFiles(markdownFiles, agent.FileOrder)

			section := "# Agent Files (Markdown)\n\n"
			for _, entry := range markdownFiles {
				content, err := b.agentStore.ReadFile(ctx, agentID, entry.Path)
				if err != nil {
					continue
				}
				section += fmt.Sprintf("## %s\n\n%s\n\n", entry.Path, content)
			}
			parts = append(parts, section)
		}
	}

	if includeTeam {
		// 2. Team member RESPONSIBILITIES.md
		responsibilities, err := b.teamStore.GetResponsibilities(ctx, teamID, agentID)
		if err == nil && responsibilities != "" {
			parts = append(parts, "# Team Responsibilities (RESPONSIBILITIES.md)\n\n"+responsibilities)
		}

		// 3. Team relationships + coordination commands
		if section := b.buildRelationshipSection(ctx, teamID, agentID); section != "" {
			parts = append(parts, section)
		}

		// 4. Team inbox messages
		if section := b.buildInboxSection(ctx, teamID, agentID); section != "" {
			parts = append(parts, section)
		}

		// 5. HEARTBEAT.md (the specific task)
		heartbeatInstructions, err := b.teamStore.GetHeartbeatInstructions(ctx, teamID, agentID)
		if err == nil && heartbeatInstructions != "" {
			parts = append(parts, "# Heartbeat Task (HEARTBEAT.md)\n\n"+heartbeatInstructions)
		} else {
			// No heartbeat instructions - use default task
			parts = append(parts, "# Heartbeat Task\n\nNo specific heartbeat instructions defined. Please review your responsibilities and perform any pending work.")
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no content available for heartbeat prompt")
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

func orderAgentMarkdownFiles(files []store.AgentFileEntry, fileOrder []string) []store.AgentFileEntry {
	if len(files) == 0 {
		return files
	}
	if len(fileOrder) == 0 {
		sort.Slice(files, func(i, j int) bool {
			a := strings.ToLower(files[i].Path)
			b := strings.ToLower(files[j].Path)
			if a == "soul.md" {
				return b != "soul.md"
			}
			if b == "soul.md" {
				return false
			}
			return a < b
		})
		return files
	}

	byPath := make(map[string]store.AgentFileEntry, len(files))
	for _, entry := range files {
		byPath[strings.ToLower(entry.Path)] = entry
	}

	ordered := make([]store.AgentFileEntry, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, path := range fileOrder {
		key := strings.ToLower(path)
		entry, ok := byPath[key]
		if !ok {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		ordered = append(ordered, entry)
		seen[key] = struct{}{}
	}

	var remaining []store.AgentFileEntry
	for _, entry := range files {
		key := strings.ToLower(entry.Path)
		if _, exists := seen[key]; exists {
			continue
		}
		remaining = append(remaining, entry)
	}

	sort.Slice(remaining, func(i, j int) bool {
		return strings.ToLower(remaining[i].Path) < strings.ToLower(remaining[j].Path)
	})

	return append(ordered, remaining...)
}

func (b *PromptBuilder) buildRelationshipSection(ctx context.Context, teamID, agentID string) string {
	org, err := b.teamStore.GetOrgChart(ctx, teamID)
	if err != nil {
		return ""
	}

	teamName := teamID
	if team, err := b.teamStore.Get(ctx, teamID); err == nil && team.DisplayName != "" {
		teamName = team.DisplayName
	}

	var managerID string
	var reportIDs []string
	for _, edge := range org.Edges {
		if edge.ReportAgentID == agentID {
			managerID = edge.ManagerAgentID
		}
		if edge.ManagerAgentID == agentID {
			reportIDs = append(reportIDs, edge.ReportAgentID)
		}
	}

	resolveAgentLabel := func(id string) string {
		if id == "" {
			return ""
		}
		agent, err := b.agentStore.Get(ctx, id)
		if err != nil || agent.DisplayName == "" || agent.DisplayName == id {
			return id
		}
		return fmt.Sprintf("%s (%s)", agent.DisplayName, id)
	}

	managerLabel := "None"
	if managerID != "" {
		managerLabel = resolveAgentLabel(managerID)
	}

	var reportsLabel string
	if len(reportIDs) == 0 {
		reportsLabel = "None"
	} else {
		labels := make([]string, 0, len(reportIDs))
		for _, reportID := range reportIDs {
			labels = append(labels, resolveAgentLabel(reportID))
		}
		sort.Strings(labels)
		reportsLabel = strings.Join(labels, ", ")
	}

	section := "# Team Relationships\n\n"
	section += fmt.Sprintf("Team: %s (%s)\n", teamName, teamID)
	section += fmt.Sprintf("Your agent ID: %s\n\n", agentID)
	section += fmt.Sprintf("- Reports to: %s\n", managerLabel)
	section += fmt.Sprintf("- Direct reports: %s\n\n", reportsLabel)
	section += "## Coordination Commands\n\n"
	section += fmt.Sprintf("- Send a directive: `prompt-manager team message-send %s <recipient-agent-id> --from=%s --content \"...\"`\n", teamID, agentID)
	section += fmt.Sprintf("- Check your inbox: `prompt-manager team message-list %s %s`\n", teamID, agentID)
	section += fmt.Sprintf("- Delete a message: `prompt-manager team message-delete %s %s <message-id>`\n", teamID, agentID)
	section += fmt.Sprintf("- Clear inbox: `prompt-manager team message-clear %s %s`\n", teamID, agentID)
	return section
}

// BuildTeamLeadPrompt constructs a prompt for single-process spawn mode.
// It loads the full team snapshot, converts to CC config, and generates spawn instructions.
func (b *PromptBuilder) BuildTeamLeadPrompt(ctx context.Context, teamID, vrooliRoot string) (string, error) {
	if b.teamStore == nil {
		return "", fmt.Errorf("team store is not configured")
	}

	team, err := b.teamStore.Get(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("loading team: %w", err)
	}

	// Build snapshot
	snapshot := &interop.PMTeamSnapshot{
		Team: *team,
	}

	// Load members
	members, err := b.teamStore.GetMembers(ctx, teamID)
	if err == nil {
		for _, rel := range members {
			pm := interop.PMTeamMember{
				Relation: rel,
			}
			if b.agentStore != nil {
				if agent, err := b.agentStore.Get(ctx, rel.AgentID); err == nil {
					pm.Agent = *agent
				}
			}
			if resp, err := b.teamStore.GetResponsibilities(ctx, teamID, rel.AgentID); err == nil {
				pm.Responsibilities = resp
			}
			if instr, err := b.teamStore.GetHeartbeatInstructions(ctx, teamID, rel.AgentID); err == nil {
				pm.HeartbeatInstr = instr
			}
			snapshot.Members = append(snapshot.Members, pm)
		}
	}

	// Load roles
	if roles, err := b.teamStore.GetRoles(ctx, teamID); err == nil {
		snapshot.Roles = roles.Roles
	}

	// Load org chart
	if org, err := b.teamStore.GetOrgChart(ctx, teamID); err == nil {
		snapshot.OrgEdges = org.Edges
	}

	// Convert to CC config
	converter := interop.ClaudeCodeConverter{}
	ccConfig, err := converter.FromPMTeam(snapshot)
	if err != nil {
		return "", fmt.Errorf("converting to CC config: %w", err)
	}

	// Generate spawn prompt
	spawnCtx := interop.SpawnContext{
		WorkingDir: vrooliRoot,
		VrooliRoot: vrooliRoot,
		TeamID:     teamID,
	}

	prompt, err := converter.FormatSpawnPrompt(ccConfig, spawnCtx)
	if err != nil {
		return "", fmt.Errorf("generating spawn prompt: %w", err)
	}

	return prompt, nil
}

func (b *PromptBuilder) buildInboxSection(ctx context.Context, teamID, agentID string) string {
	inbox, err := b.teamStore.GetInbox(ctx, teamID, agentID)
	if err != nil || len(inbox.Messages) == 0 {
		return ""
	}

	messages := make([]store.TeamMessage, len(inbox.Messages))
	copy(messages, inbox.Messages)
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	resolveAgentLabel := func(id string) string {
		agent, err := b.agentStore.Get(ctx, id)
		if err != nil || agent.DisplayName == "" || agent.DisplayName == id {
			return id
		}
		return fmt.Sprintf("%s (%s)", agent.DisplayName, id)
	}

	section := "# Team Inbox\n\n"
	section += fmt.Sprintf("You have %d pending message(s):\n\n", len(messages))
	for _, message := range messages {
		fromLabel := resolveAgentLabel(message.FromAgentID)
		section += fmt.Sprintf("## %s\n\n", message.ID)
		section += fmt.Sprintf("From: %s\n", fromLabel)
		section += fmt.Sprintf("Sent: %s\n\n", message.CreatedAt)
		section += message.Content + "\n\n"
	}
	return section
}
