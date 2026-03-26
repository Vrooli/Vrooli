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
	return b.buildSections(ctx, req, true)
}

// BuildContext constructs a prompt with all context sections but omits HEARTBEAT.md.
// Used by the member-context endpoint for single-process spawn mode bootstrapping.
func (b *PromptBuilder) BuildContext(ctx context.Context, req PromptBuildRequest) (string, error) {
	return b.buildSections(ctx, req, false)
}

// buildSections is the shared implementation. When includeHeartbeat is false,
// the HEARTBEAT.md section is omitted.
func (b *PromptBuilder) buildSections(ctx context.Context, req PromptBuildRequest, includeHeartbeat bool) (string, error) {
	sections, err := b.buildSectionList(ctx, req, includeHeartbeat)
	if err != nil {
		return "", err
	}

	// Reassemble into the original flat string format.
	// Adjacent agent-file sections are merged into a single block prefixed
	// with "# Agent Files (Markdown)\n\n" for exact backward compatibility.
	var parts []string
	for i := 0; i < len(sections); i++ {
		if sections[i].Kind == "agent-file" {
			block := "# Agent Files (Markdown)\n\n"
			for i < len(sections) && sections[i].Kind == "agent-file" {
				block += sections[i].Content
				i++
			}
			i-- // compensate for outer loop increment
			parts = append(parts, block)
		} else {
			parts = append(parts, sections[i].Content)
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no content available for heartbeat prompt")
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// buildSectionList returns the structured list of prompt sections.
// This is the shared core used by both the flat-string and structured endpoints.
func (b *PromptBuilder) buildSectionList(ctx context.Context, req PromptBuildRequest, includeHeartbeat bool) ([]PromptSection, error) {
	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		return nil, fmt.Errorf("agentId is required")
	}
	if b.agentStore == nil {
		return nil, fmt.Errorf("agent store is not configured")
	}

	agent, err := b.agentStore.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	teamID := strings.TrimSpace(req.TeamID)
	includeTeam := teamID != ""
	if includeTeam {
		if b.teamStore == nil {
			return nil, fmt.Errorf("team store is not configured")
		}
		if _, err := b.teamStore.Get(ctx, teamID); err != nil {
			return nil, err
		}
	}

	var sections []PromptSection

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

			for _, entry := range markdownFiles {
				content, err := b.agentStore.ReadFile(ctx, agentID, entry.Path)
				if err != nil {
					continue
				}
				sections = append(sections, PromptSection{
					Kind:       "agent-file",
					Label:      entry.Path,
					SourcePath: fmt.Sprintf("agents/%s/%s", agentID, entry.Path),
					Content:    fmt.Sprintf("## %s\n\n%s\n\n", entry.Path, content),
				})
			}
		}
	}

	if includeTeam {
		// 1.5 Team shared charter / operating model
		if teamDoc, err := b.teamStore.ReadSharedFile(ctx, teamID, "TEAM.md"); err == nil && strings.TrimSpace(teamDoc) != "" {
			sections = append(sections, PromptSection{
				Kind:       "team-shared-charter",
				Label:      "shared/TEAM.md",
				SourcePath: fmt.Sprintf("teams/%s/shared/TEAM.md", teamID),
				Content:    "# Team Charter (shared/TEAM.md)\n\n" + teamDoc,
			})
		}

		// 2. Team member RESPONSIBILITIES.md
		responsibilities, err := b.teamStore.GetResponsibilities(ctx, teamID, agentID)
		if err == nil && responsibilities != "" {
			sections = append(sections, PromptSection{
				Kind:       "team-responsibilities",
				Label:      "RESPONSIBILITIES.md",
				SourcePath: fmt.Sprintf("teams/%s/members/%s/RESPONSIBILITIES.md", teamID, agentID),
				Content:    "# Team Responsibilities (RESPONSIBILITIES.md)\n\n" + responsibilities,
			})
		}

		// 3. Team relationships + coordination commands
		if section := b.buildRelationshipSection(ctx, teamID, agentID); section != "" {
			sections = append(sections, PromptSection{
				Kind:    "team-relationships",
				Label:   "Team Relationships",
				Content: section,
			})
		}

		// 3.5 Coordination skill reference
		if coordSection := b.buildCoordinationSkillSection(ctx, teamID); coordSection != "" {
			sections = append(sections, PromptSection{
				Kind:    "team-coordination",
				Label:   "Team Coordination",
				Content: coordSection,
			})
		}

		// 4. Team inbox messages
		if section := b.buildInboxSection(ctx, teamID, agentID); section != "" {
			sections = append(sections, PromptSection{
				Kind:    "team-inbox",
				Label:   "Team Inbox",
				Content: section,
			})
		}

		// 4.5 Previous handoff from last heartbeat
		if handoff, err := b.teamStore.GetLastHandoff(ctx, teamID, agentID); err == nil && handoff != "" {
			sections = append(sections, PromptSection{
				Kind:       "last-handoff",
				Label:      "Previous Handoff",
				SourcePath: fmt.Sprintf("teams/%s/members/%s/last-handoff.md", teamID, agentID),
				Content:    "# Previous Heartbeat Handoff\n\nThis is what you noted at the end of your last heartbeat:\n\n" + handoff,
			})
		}

		// 5. HEARTBEAT.md (the specific task) - only when includeHeartbeat is true
		if includeHeartbeat {
			heartbeatInstructions, err := b.teamStore.GetHeartbeatInstructions(ctx, teamID, agentID)
			if err == nil && heartbeatInstructions != "" {
				sections = append(sections, PromptSection{
					Kind:       "heartbeat-task",
					Label:      "HEARTBEAT.md",
					SourcePath: fmt.Sprintf("teams/%s/members/%s/HEARTBEAT.md", teamID, agentID),
					Content:    "# Heartbeat Task (HEARTBEAT.md)\n\n" + heartbeatInstructions,
				})
			} else {
				// No heartbeat instructions - use default task
				sections = append(sections, PromptSection{
					Kind:    "heartbeat-task",
					Label:   "Heartbeat Task",
					Content: "# Heartbeat Task\n\nNo specific heartbeat instructions defined. Please review your responsibilities and perform any pending work.",
				})
			}
		}
	}

	return sections, nil
}

// BuildStructured returns the prompt as a list of structured sections.
func (b *PromptBuilder) BuildStructured(ctx context.Context, req PromptBuildRequest) ([]PromptSection, error) {
	return b.buildSectionList(ctx, req, true)
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

func (b *PromptBuilder) buildCoordinationSkillSection(ctx context.Context, teamID string) string {
	team, err := b.teamStore.Get(ctx, teamID)
	if err != nil {
		return ""
	}

	skillID := "team-coordination-multi-process"
	if team.SpawnMode == "single-process" {
		skillID = "team-coordination-single-process"
	}
	var section strings.Builder
	section.WriteString("# Team Coordination\n\n")
	section.WriteString("For team coordination guidance, run:\n```\n")
	section.WriteString(fmt.Sprintf("prompt-manager skill read %s\n", skillID))
	section.WriteString("```\n\n")
	section.WriteString("## Runtime Contract\n\n")
	section.WriteString("- This prompt-manager team already exists. Do not create or import another team.\n")
	section.WriteString("- Use prompt-manager CLI commands only for durable state such as tasks, decisions, knowledge, and handoffs.\n")
	section.WriteString("- Prefer the planning surface named in your team charter or heartbeat instructions before falling back to broad repo scans.\n")
	section.WriteString("- End your final response with `## HANDOFF` as the last section so prompt-manager can persist continuity automatically.\n")
	section.WriteString("- After writing the handoff, stop. Do not wait for extra confirmation inside the same run.\n")
	if team.DecisionMode == "approval" {
		section.WriteString("- This team is in `approval` decision mode. Analyze, prioritize, and log pending decisions, but do not deploy teams, trigger external execution, or create external backlog items unless a human has already accepted that decision.\n")
	}
	return section.String()
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
func (b *PromptBuilder) BuildTeamLeadPrompt(ctx context.Context, teamID, agentID, vrooliRoot string) (string, error) {
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
	additionalCtx := b.buildCoordinationSkillSection(ctx, teamID)
	if agentID != "" {
		if leadContext, err := b.Build(ctx, PromptBuildRequest{
			TeamID:  teamID,
			AgentID: agentID,
		}); err == nil && strings.TrimSpace(leadContext) != "" {
			additionalCtx = "## Lead Member Context\n\nApply this stored prompt-manager context in addition to the generic heartbeat instructions.\n\n" + leadContext
		}
	}
	spawnCtx := interop.SpawnContext{
		WorkingDir:    vrooliRoot,
		VrooliRoot:    vrooliRoot,
		TeamID:        teamID,
		DecisionMode:  team.DecisionMode,
		AdditionalCtx: additionalCtx,
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
