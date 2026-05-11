package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"prompt-manager/interop"
	"prompt-manager/memberflow"
	"prompt-manager/store"
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
	"sort"
	"strings"
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

// Build constructs the full runtime heartbeat prompt, including HEARTBEAT.md
// when team context is provided.
func (b *PromptBuilder) Build(ctx context.Context, req PromptBuildRequest) (string, error) {
	return b.buildSections(ctx, req, true)
}

// BuildContext constructs a prompt with all context sections but omits HEARTBEAT.md.
// Used by the member-context endpoint and single-process leader bootstrapping.
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

	// Reassemble structured sections into the flat prompt format used by
	// heartbeat executors and prompt preview endpoints.
	// Adjacent agent-file sections are merged into a single block prefixed
	// with "# Agent Files (Markdown)\n\n".
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
	var team *store.Team
	if includeTeam {
		if b.teamStore == nil {
			return nil, fmt.Errorf("team store is not configured")
		}
		team, err = b.teamStore.Get(ctx, teamID)
		if err != nil {
			return nil, err
		}
	}

	var sections []PromptSection
	var agentSections []PromptSection

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
				agentSections = append(agentSections, PromptSection{
					Kind:       "agent-file",
					Label:      entry.Path,
					SourcePath: fmt.Sprintf("agents/%s/%s", agentID, entry.Path),
					Content:    fmt.Sprintf("## %s\n\n%s\n\n", entry.Path, content),
				})
			}
		}
	}

	if includeTeam {
		contract := team.Contract()

		var teamCharter string
		if teamDoc, err := b.teamStore.ReadSharedFile(ctx, teamID, "TEAM.md"); err == nil && strings.TrimSpace(teamDoc) != "" {
			teamCharter = teamDoc
		}

		var heartbeatInstructions string
		if includeHeartbeat {
			if instructions, err := b.teamStore.GetHeartbeatInstructions(ctx, teamID, agentID); err == nil {
				heartbeatInstructions = instructions
			}
		}

		// The heartbeat prompt uses an intentional task sandwich: the active
		// brief comes first so every later context section is read through the
		// current job, while the full HEARTBEAT.md and final reminder stay near
		// the end for recency. Middle sections should not duplicate doctrine
		// already owned by generated storage, contract, or coordination context.
		sections = append(sections, PromptSection{
			Kind:       promptSectionKindActiveTaskBrief,
			Label:      promptSectionLabelActiveTaskBrief,
			SourcePath: fmt.Sprintf("teams/%s/team.json#operatingContract.members.%s", teamID, agentID),
			Content:    buildActiveTaskBriefSection(team, agentID, includeHeartbeat, heartbeatInstructions, b.teamStore.StoreDir()),
		})

		if teamconfig.ShouldInjectInbox(contract) {
			if section := b.buildInboxSection(ctx, teamID, agentID); section != "" {
				sections = append(sections, PromptSection{
					Kind:    "team-inbox",
					Label:   "Team Inbox",
					Content: section,
				})
			}
		}

		if handoff, err := b.teamStore.GetLastHandoff(ctx, teamID, agentID); err == nil && handoff != "" {
			sections = append(sections, PromptSection{
				Kind:       "last-handoff",
				Label:      "Previous Handoff",
				SourcePath: fmt.Sprintf("teams/%s/members/%s/last-handoff.md", teamID, agentID),
				Content:    "# Previous Heartbeat Handoff\n\nThis is what you noted at the end of your last heartbeat:\n\n" + handoff,
			})
		}

		if section := b.buildChallengeReviewSection(ctx, team, agentID); section != "" {
			sections = append(sections, PromptSection{
				Kind:       promptSectionKindChallengeReview,
				Label:      promptSectionLabelChallengeReview,
				SourcePath: fmt.Sprintf("teams/%s/shared/knowledge.jsonl", teamID),
				Content:    section,
			})
		}

		if section, err := b.buildStorageMapSection(team, agentID); err != nil {
			return nil, err
		} else if section != "" {
			sections = append(sections, PromptSection{
				Kind:    "team-storage-map",
				Label:   "Storage Map",
				Content: section,
			})
		}

		if teamconfig.ShouldShowOrgContext(contract) {
			if section := b.buildOrgContextSection(ctx, team, agentID); section != "" {
				sections = append(sections, PromptSection{
					Kind:    "team-org-context",
					Label:   "Team Org Context",
					Content: section,
				})
			}
		}

		operatingPolicy, err := b.buildOperatingPolicySection(team, agentID, teamCharter)
		if err != nil {
			return nil, err
		}
		sections = append(sections, PromptSection{
			Kind:       promptSectionKindOperatingPolicy,
			Label:      promptSectionLabelOperatingPolicy,
			SourcePath: fmt.Sprintf("teams/%s/team.json", teamID),
			Content:    operatingPolicy,
		})

		if section := b.buildTopicContractSection(teamID, agentID); section != "" {
			sections = append(sections, PromptSection{
				Kind:       promptSectionKindTopicContract,
				Label:      promptSectionLabelTopicContract,
				SourcePath: fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID),
				Content:    section,
			})
		}

		if section := b.buildInboxFlowSection(teamID, agentID); section != "" {
			sections = append(sections, PromptSection{
				Kind:       promptSectionKindInboxFlow,
				Label:      promptSectionLabelInboxFlow,
				SourcePath: fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID),
				Content:    section,
			})
		}

		responsibilities, err := b.teamStore.GetResponsibilities(ctx, teamID, agentID)
		if err == nil && responsibilities != "" {
			sections = append(sections, PromptSection{
				Kind:       "team-responsibilities",
				Label:      "RESPONSIBILITIES.md",
				SourcePath: fmt.Sprintf("teams/%s/members/%s/RESPONSIBILITIES.md", teamID, agentID),
				Content:    "# Team Responsibilities (RESPONSIBILITIES.md)\n\n" + responsibilities,
			})
		}

		sections = append(sections, agentSections...)

		if includeHeartbeat {
			if heartbeatInstructions != "" {
				sections = append(sections, PromptSection{
					Kind:       "heartbeat-task",
					Label:      "HEARTBEAT.md",
					SourcePath: fmt.Sprintf("teams/%s/members/%s/HEARTBEAT.md", teamID, agentID),
					Content:    promptHeadingHeartbeatTask + "\n\n" + heartbeatInstructions,
				})
			} else {
				// No heartbeat instructions - use default task
				sections = append(sections, PromptSection{
					Kind:    "heartbeat-task",
					Label:   "Heartbeat Task",
					Content: "# Heartbeat Task\n\nNo specific heartbeat instructions defined. Please review your responsibilities and perform any pending work.",
				})
			}
			sections = append(sections, PromptSection{
				Kind:       promptSectionKindTaskReminder,
				Label:      promptSectionLabelTaskReminder,
				SourcePath: fmt.Sprintf("teams/%s/team.json#operatingContract.members.%s", teamID, agentID),
				Content:    buildTaskReminderSection(team, agentID, heartbeatInstructions),
			})
		}
	} else {
		sections = append(sections, agentSections...)
	}

	return sections, nil
}

// BuildStructured returns the prompt as a list of structured sections.
func (b *PromptBuilder) BuildStructured(ctx context.Context, req PromptBuildRequest) ([]PromptSection, error) {
	return b.buildSectionList(ctx, req, true)
}

func (b *PromptBuilder) buildTopicContractSection(teamID, agentID string) string {
	if b == nil || b.teamStore == nil {
		return ""
	}
	in, err := LoadTopicContractInputs(b.teamStore.StoreDir(), teamID, agentID)
	if err != nil {
		return ""
	}
	return RenderTopicContract(in)
}

type memberStoragePolicy struct {
	CanWriteDecision          bool
	CanWriteKnowledge         bool
	RequiresHandoff           bool
	CanWriteTask              bool
	CanWriteWorkingStatePaths []string
	AllowedWriteLabels        []string
	ForbiddenWriteLabels      []string
	RequiredReadPrefixes      []string
	DecisionCapPerHeartbeat   *int
	PendingOwnedDecisionCap   *int
}

func buildActiveTaskBriefSection(team *store.Team, agentID string, includeHeartbeat bool, heartbeatInstructions string, storeDir string) string {
	teamID := ""
	teamName := ""
	lane := ""
	policy := memberStoragePolicy{}
	if team != nil {
		teamID = team.ID
		teamName = team.DisplayName
		if teamName == "" {
			teamName = team.ID
		}
		if team.OperatingContract != nil {
			if member, ok := team.OperatingContract.Members[agentID]; ok {
				lane = strings.TrimSpace(member.Lane)
			}
			policy = buildMemberStoragePolicy(team, agentID, storeDir)
		}
	}
	if lane == "" {
		lane = "Apply the team mission within this member's assigned scope."
	}
	task := firstHeartbeatTaskHeading(heartbeatInstructions)
	if task == "" {
		task = lane
	}

	var section strings.Builder
	section.WriteString(promptHeadingActiveTaskBrief + "\n\n")
	section.WriteString(fmt.Sprintf("You are running one prompt-manager heartbeat as `%s` on `%s`", agentID, teamID))
	if teamName != "" && teamName != teamID {
		section.WriteString(fmt.Sprintf(" (`%s`)", teamName))
	}
	section.WriteString(".\n\n")
	section.WriteString("## Mission This Run\n\n")
	section.WriteString(lane + "\n\n")
	section.WriteString("## Primary Task\n\n")
	if includeHeartbeat {
		section.WriteString(task + "\n\n")
		section.WriteString("The complete task source is included later in `# Heartbeat Task (HEARTBEAT.md)`. Use this brief to stay oriented while reading the context pack.\n\n")
	} else {
		section.WriteString("The active heartbeat task is intentionally omitted from member context.\n\n")
	}
	section.WriteString("## Write Surface\n\n")
	if len(policy.AllowedWriteLabels) == 0 {
		section.WriteString("Allowed: none declared\n\n")
	} else {
		section.WriteString("Allowed:\n")
		for _, label := range policy.AllowedWriteLabels {
			section.WriteString("- " + label + "\n")
		}
		section.WriteString("\n")
	}
	if len(policy.ForbiddenWriteLabels) == 0 {
		section.WriteString("Forbidden: none declared\n\n")
	} else {
		section.WriteString("Forbidden:\n")
		for _, label := range policy.ForbiddenWriteLabels {
			section.WriteString("- " + label + "\n")
		}
		section.WriteString("\n")
	}
	section.WriteString("## Required Memory\n\n")
	if len(policy.RequiredReadPrefixes) == 0 {
		section.WriteString("Knowledge topics: none declared\n\n")
	} else {
		section.WriteString("Knowledge topics:\n")
		for _, topic := range policy.RequiredReadPrefixes {
			section.WriteString("- `" + topic + "`\n")
		}
		section.WriteString("\n")
	}
	section.WriteString("Handoff:\n")
	if policy.RequiresHandoff {
		section.WriteString("- End with `## HANDOFF` as the final response section when the team requires handoff persistence.\n\n")
	} else {
		section.WriteString("- No required handoff declared for this member.\n\n")
	}
	if policy.CanWriteDecision {
		section.WriteString("Decision cap: ")
		if policy.DecisionCapPerHeartbeat != nil {
			section.WriteString(fmt.Sprintf("%d new decisions this heartbeat", *policy.DecisionCapPerHeartbeat))
		} else {
			section.WriteString("no explicit per-heartbeat cap")
		}
		if policy.PendingOwnedDecisionCap != nil {
			section.WriteString(fmt.Sprintf("; skip new decisions when %d owned-context decisions are already pending", *policy.PendingOwnedDecisionCap))
		}
		section.WriteString(".\n\n")
	} else {
		section.WriteString("Decision writes: not allowed for this member. Review decisions when useful; do not create them.\n\n")
	}
	section.WriteString("## Operating Rule\n\n")
	section.WriteString("If context sections disagree, follow the authority order in `# Storage Map`. If the heartbeat task conflicts with lower-priority context, follow the heartbeat task unless it violates the operator instruction or your write rules.")
	return section.String()
}

func buildTaskReminderSection(team *store.Team, agentID string, heartbeatInstructions string) string {
	teamID := ""
	lane := "Apply the team mission within this member's assigned scope."
	policy := memberStoragePolicy{}
	if team != nil {
		teamID = team.ID
		if team.OperatingContract != nil {
			if member, ok := team.OperatingContract.Members[agentID]; ok && strings.TrimSpace(member.Lane) != "" {
				lane = strings.TrimSpace(member.Lane)
			}
			policy = buildMemberStoragePolicy(team, agentID, "")
		}
	}
	focus := firstHeartbeatTaskHeading(heartbeatInstructions)
	if focus == "" {
		focus = lane
	}

	var section strings.Builder
	section.WriteString(promptHeadingTaskReminder + "\n\n")
	section.WriteString(fmt.Sprintf("Run this heartbeat as `%s` on `%s`.\n\n", agentID, teamID))
	section.WriteString(fmt.Sprintf("Focus on: %s.\n\n", focus))
	section.WriteString("Do now:\n")
	section.WriteString("1. Follow the task loop in `HEARTBEAT.md`.\n")
	section.WriteString("2. Use only the write surfaces allowed in `# Active Task Brief`.\n")
	itemThree := "3. Record observations, friction"
	if policy.CanWriteDecision {
		itemThree += ", decisions"
	}
	if len(policy.CanWriteWorkingStatePaths) > 0 {
		itemThree += ", working-state updates"
	}
	if policy.RequiresHandoff {
		itemThree += ", and handoff"
	}
	itemThree += " according to `# Storage Map`"
	if !policy.CanWriteDecision {
		itemThree += "; do not create decisions"
	}
	section.WriteString(itemThree + ".\n")
	if policy.RequiresHandoff {
		section.WriteString("4. End with `## HANDOFF` when required, then stop.")
	}
	return section.String()
}

type pendingChallengeView struct {
	Decision   store.DecisionEntry
	Report     *store.KnowledgeEntry
	Resolution *store.KnowledgeEntry
	Status     string
}

func (b *PromptBuilder) buildChallengeReviewSection(ctx context.Context, team *store.Team, agentID string) string {
	if b == nil || b.teamStore == nil || team == nil || team.OperatingContract == nil {
		return ""
	}
	member, ok := team.OperatingContract.Members[agentID]
	if !ok {
		return ""
	}

	ownedContexts := make(map[string]struct{}, len(member.OwnedDecisionContexts))
	for _, contextID := range member.OwnedDecisionContexts {
		contextID = strings.TrimSpace(contextID)
		if contextID != "" {
			ownedContexts[contextID] = struct{}{}
		}
	}

	decisions, _, err := b.teamStore.GetDecisions(ctx, team.ID, "", store.DecisionStatusPending, 0)
	if err != nil || len(decisions) == 0 {
		return ""
	}

	challenges := make([]pendingChallengeView, 0)
	for _, decision := range decisions {
		if strings.TrimSpace(decision.ID) == "" || !memberOwnsChallengeTarget(decision, agentID, ownedContexts) {
			continue
		}
		reports, err := b.teamStore.GetKnowledge(ctx, team.ID, "challenge-report/"+decision.ID, "", 0)
		if err != nil || len(reports) == 0 {
			continue
		}
		resolutions, err := b.teamStore.GetKnowledge(ctx, team.ID, "challenge-resolution-record/"+decision.ID, "", 0)
		if err != nil {
			continue
		}

		view := pendingChallengeView{
			Decision: decision,
			Report:   &reports[len(reports)-1],
			Status:   "open",
		}
		if len(resolutions) > 0 {
			view.Resolution = &resolutions[len(resolutions)-1]
			view.Status = challengeResolutionStatus(view.Resolution.Content)
			if view.Status == "" {
				view.Status = "open"
			}
		}
		if challengeStatusClosed(view.Status) {
			continue
		}
		challenges = append(challenges, view)
	}
	if len(challenges) == 0 {
		return ""
	}

	var section strings.Builder
	section.WriteString(promptHeadingChallengeReview + "\n\n")
	section.WriteString("These pending decisions you authored or own by context have unresolved contrarian challenges. Respond before filing unrelated new decisions when possible.\n\n")
	for _, challenge := range challenges {
		section.WriteString(fmt.Sprintf("## `%s`\n\n", challenge.Decision.ID))
		section.WriteString(fmt.Sprintf("- Context: `%s`\n", emptyAs(challenge.Decision.Context, "none")))
		section.WriteString(fmt.Sprintf("- Author: `%s`\n", emptyAs(challenge.Decision.By, "unknown")))
		section.WriteString(fmt.Sprintf("- Challenge report: `%s`", challenge.Report.Topic))
		if challenge.Report.ID != "" {
			section.WriteString(fmt.Sprintf(" (`%s`)", challenge.Report.ID))
		}
		section.WriteString("\n")
		if challenge.Resolution != nil {
			section.WriteString(fmt.Sprintf("- Latest resolution: `%s`", challenge.Resolution.Topic))
			if challenge.Resolution.ID != "" {
				section.WriteString(fmt.Sprintf(" (`%s`)", challenge.Resolution.ID))
			}
			section.WriteString(fmt.Sprintf(" status `%s`\n", challenge.Status))
		} else {
			section.WriteString("- Latest resolution: none yet; treat status as `open`\n")
		}
		if summary := firstContentLine(challenge.Report.Content); summary != "" {
			section.WriteString("- Report summary: " + summary + "\n")
		}
		section.WriteString("- Response path: revise or supersede the decision, accept the challenge, or disagree with evidence; if you cannot attach the response to the decision, write `challenge-resolution-record/" + challenge.Decision.ID + "` with `status: author-responded`.\n")
		section.WriteString("- Inspect with: `prompt-manager team knowledge-list " + team.ID + " --topic challenge-report/" + challenge.Decision.ID + "` and `prompt-manager team knowledge-list " + team.ID + " --topic challenge-resolution-record/" + challenge.Decision.ID + "`.\n\n")
	}
	section.WriteString("Challenge lifecycle canon: `docs/agent-system/CONTRARIAN_REVIEW.md`.")
	return section.String()
}

func memberOwnsChallengeTarget(decision store.DecisionEntry, agentID string, ownedContexts map[string]struct{}) bool {
	if strings.TrimSpace(decision.By) == agentID {
		return true
	}
	if _, ok := ownedContexts[strings.TrimSpace(decision.Context)]; ok {
		return true
	}
	return false
}

func challengeResolutionStatus(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "status:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "status:")), `"'`)
		}
		if strings.HasPrefix(line, `"status"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(strings.TrimRight(parts[1], ",")), `"'`)
			}
		}
	}
	return ""
}

func challengeStatusClosed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "overridden", "stale":
		return true
	default:
		return false
	}
}

func firstContentLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		const maxLen = 220
		if len(line) > maxLen {
			line = line[:maxLen] + "..."
		}
		return line
	}
	return ""
}

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstHeartbeatTaskHeading(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if title == "" || strings.EqualFold(title, "Heartbeat") {
			continue
		}
		return title
	}
	return ""
}

func buildMemberStoragePolicy(team *store.Team, agentID string, storeDir string) memberStoragePolicy {
	if team == nil || team.OperatingContract == nil {
		return memberStoragePolicy{}
	}
	member, ok := team.OperatingContract.Members[agentID]
	if !ok {
		return memberStoragePolicy{}
	}
	policy := memberStoragePolicy{
		RequiresHandoff:         teamconfig.RequiresHandoff(team.Contract()) && !writeRefsContainKind(member.ForbiddenWrites, "handoff"),
		RequiredReadPrefixes:    loadRequiredReadPrefixes(storeDir, team.ID, agentID),
		DecisionCapPerHeartbeat: member.NewDecisionCapPerHeartbeat,
		PendingOwnedDecisionCap: member.PendingOwnedDecisionCap,
	}
	for _, ref := range member.AllowedWrites {
		label := describeWriteRef(ref, team, agentID, storeDir)
		if label != "" {
			policy.AllowedWriteLabels = append(policy.AllowedWriteLabels, label)
		}
		if ref.Kind != "" {
			switch ref.Kind {
			case "decision":
				policy.CanWriteDecision = true
			case "knowledge":
				policy.CanWriteKnowledge = true
			case "handoff":
				policy.RequiresHandoff = teamconfig.RequiresHandoff(team.Contract())
			case "task":
				policy.CanWriteTask = true
			}
			continue
		}
		normalized := normalizedWritePath(ref, team, agentID, storeDir)
		switch {
		case strings.HasSuffix(normalized, "/decisions.jsonl"):
			policy.CanWriteDecision = true
		case strings.HasSuffix(normalized, "/knowledge.jsonl"):
			policy.CanWriteKnowledge = true
		default:
			if normalized != "" {
				policy.CanWriteWorkingStatePaths = append(policy.CanWriteWorkingStatePaths, normalized)
			}
		}
	}
	for _, ref := range member.ForbiddenWrites {
		label := describeWriteRef(ref, team, agentID, storeDir)
		if label != "" {
			policy.ForbiddenWriteLabels = append(policy.ForbiddenWriteLabels, label)
		}
		if ref.Kind == "" {
			continue
		}
		switch ref.Kind {
		case "decision":
			policy.CanWriteDecision = false
		case "knowledge":
			policy.CanWriteKnowledge = false
		case "handoff":
			policy.RequiresHandoff = false
		case "task":
			policy.CanWriteTask = false
		}
	}
	if member.NewDecisionCapPerHeartbeat != nil && *member.NewDecisionCapPerHeartbeat == 0 {
		policy.CanWriteDecision = false
	}
	policy.AllowedWriteLabels = sortedUniqueStrings(policy.AllowedWriteLabels)
	policy.ForbiddenWriteLabels = sortedUniqueStrings(policy.ForbiddenWriteLabels)
	sort.Strings(policy.RequiredReadPrefixes)
	sort.Strings(policy.CanWriteWorkingStatePaths)
	return policy
}

// loadRequiredReadPrefixes returns the topic prefixes the member must keep
// in working memory every heartbeat. The list comes from the member's
// topics.json `required_read[]` declaration — the single declaration source
// of truth for read relationships. When storeDir is empty (e.g., the
// task-reminder section, which only needs decision/handoff hints) or
// topics.json is absent (a positive empty declaration), the function
// returns nil and the rendering falls back to "Knowledge topics: none
// declared".
//
// Errors loading topics.json are intentionally swallowed as "no required
// reads" rather than propagated: the prompt builder must not refuse to
// emit a heartbeat just because a topics.json read failed. The validator
// (api/memberflow/validation.go) catches malformed topics.json on every
// `prompt-manager team validate` run so drift surfaces there, not here.
func loadRequiredReadPrefixes(storeDir, teamID, agentID string) []string {
	if strings.TrimSpace(storeDir) == "" {
		return nil
	}
	mt, err := memberflow.LoadMember(storeDir, teamID, agentID)
	if err != nil || len(mt.Topics.RequiredRead) == 0 {
		return nil
	}
	prefixes := make([]string, 0, len(mt.Topics.RequiredRead))
	for _, e := range mt.Topics.RequiredRead {
		if p := strings.TrimSpace(e.Prefix); p != "" {
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func describeWriteRef(ref teamcontract.WriteRef, team *store.Team, agentID string, storeDir string) string {
	if ref.Kind != "" {
		switch ref.Kind {
		case "decision":
			return "decision proposals"
		case "knowledge":
			return "knowledge observations and friction signals"
		case "handoff":
			return "final `## HANDOFF` continuity"
		case "task":
			return "team task board updates"
		case "inbox-message":
			return "async inbox messages"
		default:
			return ref.Kind
		}
	}
	if path := normalizedWritePath(ref, team, agentID, storeDir); path != "" {
		if label := describeKnownWritePath(path, team, agentID, storeDir); label != "" {
			return label
		}
		return "`" + path + "`"
	}
	return ""
}

func describeKnownWritePath(path string, team *store.Team, agentID string, storeDir string) string {
	switch {
	case strings.HasSuffix(path, "/decisions.jsonl"):
		return "decision proposals"
	case strings.HasSuffix(path, "/knowledge.jsonl"):
		return "knowledge observations and friction signals"
	case strings.HasSuffix(path, "/tasks.json"):
		return "team task board updates"
	}
	if team == nil || team.OperatingContract == nil {
		return ""
	}
	for _, doc := range team.OperatingContract.Documents.SharedState {
		normalized, err := teamcontract.NormalizePath(doc.Path, teamcontract.ValidationInput{
			TeamID:       team.ID,
			DecisionMode: team.DecisionMode,
			StoreDir:     storeDir,
		}, agentID)
		if err != nil || normalized != path {
			continue
		}
		meta, ok := teamcontract.TeamWorkingStateKindMetadata(doc.Kind)
		if !ok {
			return fmt.Sprintf("team working state `%s`", path)
		}
		return fmt.Sprintf("team working state `%s` (%s)", path, strings.ToLower(meta.Label))
	}
	return ""
}

func normalizedWritePath(ref teamcontract.WriteRef, team *store.Team, agentID string, storeDir string) string {
	if team == nil {
		return ""
	}
	path, err := teamcontract.NormalizePath(teamcontract.PathRef{
		Base:     ref.Base,
		Path:     ref.Path,
		MemberID: ref.MemberID,
		AgentID:  ref.AgentID,
	}, teamcontract.ValidationInput{
		TeamID:       team.ID,
		DecisionMode: team.DecisionMode,
		StoreDir:     storeDir,
	}, agentID)
	if err != nil {
		return ""
	}
	return path
}

func writeRefsContainKind(refs []teamcontract.WriteRef, kind string) bool {
	for _, ref := range refs {
		if ref.Kind == kind {
			return true
		}
	}
	return false
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

func (b *PromptBuilder) buildOperatingPolicySection(team *store.Team, agentID string, teamCharter string) (string, error) {
	if team == nil {
		return "", nil
	}

	contract := team.Contract()
	var section strings.Builder
	section.WriteString(promptHeadingOperatingPolicy + "\n\n")
	if strings.TrimSpace(teamCharter) != "" {
		section.WriteString("## Team Charter\n\n")
		section.WriteString(fmt.Sprintf("Source: `teams/%s/shared/TEAM.md`\n\n", team.ID))
		section.WriteString(strings.TrimSpace(teamCharter))
		section.WriteString("\n\n")
	}
	section.WriteString("## Runtime\n\n")
	section.WriteString(fmt.Sprintf("- Runtime mode: `%s`\n", team.Runtime.Mode))
	section.WriteString(fmt.Sprintf("- Queue policy: `%s` (max concurrent: %d)\n", team.Execution.QueuePolicy, team.Execution.MaxConcurrentRuns))
	section.WriteString("\n## Coordination\n\n")
	section.WriteString(fmt.Sprintf("- Coordination pattern: `%s`\n", team.Coordination.Pattern))
	section.WriteString(fmt.Sprintf("- Messaging mode: `%s`\n", team.Coordination.MessagingMode))
	section.WriteString(fmt.Sprintf("- Reporting mode: `%s`\n", team.Coordination.ReportingMode))
	if team.Coordination.LeadAgentID != "" {
		section.WriteString(fmt.Sprintf("- Lead agent: `%s`\n", team.Coordination.LeadAgentID))
	}
	section.WriteString("\n")
	if shouldIncludeCoordinationSkillReference(team) {
		skillID := teamconfig.CoordinationSkillID(contract)
		section.WriteString("Coordination guidance:\n```\n")
		section.WriteString(fmt.Sprintf("prompt-manager skill read %s\n", skillID))
		section.WriteString("```\n\n")
	}
	section.WriteString("Coordination rules:\n")
	section.WriteString("- This prompt-manager team already exists. Do not create or import another team.\n")
	section.WriteString("- Prefer the planning surface named in your team charter or heartbeat instructions before falling back to broad repo scans.\n")
	if teamconfig.MessagingUsesAsyncInbox(contract) {
		section.WriteString("- Team coordination uses asynchronous inbox messages that survive across heartbeat runs.\n")
	} else if teamconfig.MessagingUsesInSession(contract) {
		section.WriteString("- Coordinate with teammates using in-session subagent messaging, not durable inbox messages.\n")
	} else {
		section.WriteString("- This team does not use agent-to-agent messaging by default. Stay within your own scope unless the heartbeat task tells you otherwise.\n")
	}
	if teamconfig.AllowsPeerTriggers(contract) {
		section.WriteString(fmt.Sprintf("- Peer triggers are enabled. You may request a teammate run by calling `prompt-manager team heartbeat-trigger %s <agent-id>` when the heartbeat task explicitly benefits from it.\n", team.ID))
	}
	if team.DecisionMode == "approval" {
		section.WriteString("- This team is in `approval` decision mode. Analyze, prioritize, and log pending decisions, but do not deploy teams, trigger external execution, or create external backlog items unless a human has already accepted that decision.\n")
	}

	memberPolicy, err := teamcontract.RenderMemberPolicy(team.OperatingContract, teamcontract.RenderInput{
		TeamID:       team.ID,
		TeamName:     team.DisplayName,
		DecisionMode: team.DecisionMode,
		MemberID:     agentID,
		StoreDir:     b.teamStore.StoreDir(),
	})
	if err != nil {
		return "", err
	}
	section.WriteString(memberPolicy)
	return strings.TrimRight(section.String(), "\n"), nil
}

func shouldIncludeCoordinationSkillReference(team *store.Team) bool {
	if team == nil {
		return false
	}
	contract := team.Contract()
	return contract.Coordination.Pattern != teamconfig.CoordinationPatternIndependent ||
		teamconfig.MessagingEnabled(contract) ||
		teamconfig.AllowsPeerTriggers(contract) ||
		team.DecisionMode == "approval"
}

func (b *PromptBuilder) buildOrgContextSection(ctx context.Context, team *store.Team, agentID string) string {
	if team == nil {
		return ""
	}

	teamName := team.ID
	if team.DisplayName != "" {
		teamName = team.DisplayName
	}

	contract := team.Contract()
	var managerID string
	var reportIDs []string

	switch team.Coordination.ReportingMode {
	case teamconfig.ReportingModeLeader:
		leadID := team.Coordination.LeadAgentID
		if leadID == "" {
			return ""
		}
		if agentID != leadID {
			managerID = leadID
		} else if members, err := b.teamStore.GetMembers(ctx, team.ID); err == nil {
			for _, member := range members {
				if member.AgentID != leadID {
					reportIDs = append(reportIDs, member.AgentID)
				}
			}
		}
	case teamconfig.ReportingModeOrgChart:
		org, err := b.teamStore.GetOrgChart(ctx, team.ID)
		if err != nil {
			return ""
		}
		for _, edge := range org.Edges {
			if edge.ReportAgentID == agentID {
				managerID = edge.ManagerAgentID
			}
			if edge.ManagerAgentID == agentID {
				reportIDs = append(reportIDs, edge.ReportAgentID)
			}
		}
	default:
		return ""
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

	section := "# Team Org Context\n\n"
	section += fmt.Sprintf("Team: %s (%s)\n", teamName, team.ID)
	section += fmt.Sprintf("Your agent ID: %s\n\n", agentID)
	section += fmt.Sprintf("- Reporting mode: %s\n", contract.Coordination.ReportingMode)
	section += fmt.Sprintf("- Reports to: %s\n", managerLabel)
	section += fmt.Sprintf("- Direct reports: %s\n", reportsLabel)
	return section
}

func (b *PromptBuilder) buildStorageMapSection(team *store.Team, agentID string) (string, error) {
	if team == nil {
		return "", nil
	}

	var section strings.Builder
	section.WriteString(`# Storage Map

Use persistent storage only when information should survive this run.

## Continue

Use your final ` + "`## HANDOFF`" + ` for short-term continuity.

Write what your next run needs to know: what changed, what remains open, what to check first, and any blockers. Handoff is not canonical truth. It is next-run memory.

## Observe

Use the knowledge log for structured observations from this heartbeat: evidence, measurements, snapshots, findings, and concrete friction signals.

When an observation has a clear typed destination — another member's inbox topic, or a concrete-typed topic that another member drains — write to that topic with ` + "`prompt-manager team knowledge-add`" + ` so the right drainer picks it up. If no typed topic fits, use ` + "`report-friction`" + ` for system/process friction or raise a ` + "`meta-self-improvement`" + ` proposal to add the missing topic/taxonomy. The full registry of inboxes and other typed topics is ` + "`docs/agent-system/TOPICS.md`" + `.

For broken scenario or code behavior — bugs of any kind: code defects, regressions, prompt confusion, data-shape mismatches, unexpected errors — load ` + "`prompt-manager skill read report-bug`" + ` and follow it. The skill writes to ` + "`bug-inbox/<signal-type>/<slug>`" + ` on ` + "`scenario-qa`" + `, where the bug-investigator drains. ` + "`bug-inbox/*`" + ` is the universal-source intake any team's members write to.

If something expected was missing, broken, confusing, slow, undocumented, or harder than it should have been, capture it as friction. For structural friction — a skill that misled you, a tool that's missing, a doc you couldn't find, a workaround you keep repeating — load ` + "`prompt-manager skill read report-friction`" + ` and follow it. The skill writes to ` + "`friction-inbox/<scope>/<slug>`" + ` on ` + "`meta-optimization`" + `, where the friction-curator drains and routes to the appropriate sub-member or to debt-curator for synthesis. For situational, one-off friction not worth filing, mention it in handoff.

## Propose

Use decisions for changes that need review.

Create a decision when something durable should change: plan-of-record docs, skills, actions, CLIs, team config, scenarios, backlog, or another member's operating surface. Include the proposed change, rationale, evidence, and target destination.

## Operate

Use team working state for live team objects you are assigned to maintain.

Working state is team-local operational memory: task boards, ledgers, registers, rolling audits, append-only logs, and operator input files. It is not automatically canonical outside the team. Update only the working-state files named in your operating contract.

## Authority Order

When sources disagree, prefer:

1. Operator instruction in the current run
2. Accepted plan-of-record docs
3. Accepted decisions
4. Team working state
5. Knowledge log evidence
6. Handoff
`)

	teamStorage, err := teamcontract.RenderTeamStorage(team.OperatingContract, teamcontract.RenderInput{
		TeamID:         team.ID,
		TeamName:       team.DisplayName,
		DecisionMode:   team.DecisionMode,
		MemberID:       agentID,
		StoreDir:       b.teamStore.StoreDir(),
		RequireHandoff: teamconfig.RequiresHandoff(team.Contract()),
	})
	if err != nil {
		return "", err
	}
	section.WriteString("\n")
	section.WriteString(teamStorage)

	commandSection := b.buildAvailableStorageCommandsSection(team, agentID)
	if commandSection != "" {
		section.WriteString("\n")
		section.WriteString(commandSection)
	}

	return strings.TrimRight(section.String(), "\n"), nil
}

func (b *PromptBuilder) buildAvailableStorageCommandsSection(team *store.Team, agentID string) string {
	if team == nil {
		return ""
	}

	contract := team.Contract()
	policy := buildMemberStoragePolicy(team, agentID, b.teamStore.StoreDir())
	var lines []string

	if teamconfig.MessagingUsesAsyncInbox(contract) {
		lines = append(lines, "### Async Inbox")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Send a directive: `prompt-manager team message-send %s <recipient-agent-id> --from=<agent-id> --content \"...\"`", team.ID))
		lines = append(lines, fmt.Sprintf("- List inbox messages: `prompt-manager team message-list %s <agent-id>`", team.ID))
		lines = append(lines, fmt.Sprintf("- Delete a message: `prompt-manager team message-delete %s <agent-id> <message-id>`", team.ID))
		lines = append(lines, fmt.Sprintf("- Clear inbox: `prompt-manager team message-clear %s <agent-id>`", team.ID))
		lines = append(lines, "")
	}

	if teamconfig.ShouldShowTaskBoardGuidance(contract) {
		lines = append(lines, "### Task Board")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Review current work: `prompt-manager team task-list %s`", team.ID))
		if policy.CanWriteTask {
			lines = append(lines, fmt.Sprintf("- Add a task: `prompt-manager team task-add %s --by=<agent-id> --title \"...\"`", team.ID))
			lines = append(lines, fmt.Sprintf("- Update a task: `prompt-manager team task-update %s <task-id> --status=in-progress`", team.ID))
		} else {
			lines = append(lines, "- Task writes are not allowed for this member.")
		}
		lines = append(lines, "")
	}

	if teamconfig.ShouldShowDecisionLogGuidance(contract) {
		lines = append(lines, "### Decision Log")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Review decisions: `prompt-manager team decision-list %s`", team.ID))
		if policy.CanWriteDecision {
			lines = append(lines, fmt.Sprintf("- Record a pending decision: `prompt-manager team decision-add %s --by=<agent-id> --decision \"...\" --rationale \"...\"`", team.ID))
		} else {
			lines = append(lines, "- Decision writes are not allowed for this member.")
		}
		lines = append(lines, "")
	}

	if teamconfig.ShouldShowKnowledgeLogGuidance(contract) {
		lines = append(lines, "### Knowledge Log")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Review team knowledge: `prompt-manager team knowledge-list %s`", team.ID))
		if policy.CanWriteKnowledge {
			lines = append(lines, fmt.Sprintf("- Record durable knowledge: `prompt-manager team knowledge-add %s --by=<agent-id> --topic \"...\" --content \"...\"`", team.ID))
		} else {
			lines = append(lines, "- Knowledge writes are not allowed for this member.")
		}
		lines = append(lines, "")
	}

	if teamconfig.RequiresHandoff(contract) {
		lines = append(lines, "### Handoff")
		lines = append(lines, "")
		lines = append(lines, "- End your final response with `## HANDOFF` as the last section so prompt-manager can persist continuity automatically.")
		lines = append(lines, "- After writing the handoff, stop. Do not wait for extra confirmation inside the same run.")
		lines = append(lines, "")
	}

	if len(lines) == 0 {
		return ""
	}

	return "## Available Storage Commands\n\n" + strings.Join(lines, "\n")
}

// BuildTeamLeadPrompt constructs a prompt for leader-led single-process teams.
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
	additionalCtx := ""
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

// buildInboxFlowSection generates the "Inbox Flow" prompt section from the
// member's topics.json + the taxonomy registry. Returns empty when the
// member declares no intake; in that case the caller skips the section.
//
// All errors are swallowed and logged-via-empty-section by design: a
// malformed topics.json should not block the whole heartbeat. The
// `unknown_taxonomy` validator surfaces those problems separately.
func (b *PromptBuilder) buildInboxFlowSection(teamID, agentID string) string {
	if b.teamStore == nil {
		return ""
	}
	storeDir := b.teamStore.StoreDir()
	repoRoot := deriveRepoRoot(storeDir)
	in, ok, err := LoadInboxFlowInputs(storeDir, repoRoot, teamID, agentID)
	if err != nil || !ok {
		return ""
	}
	return RenderInboxFlow(in)
}

// deriveRepoRoot resolves the repository root for taxonomy lookup. Prefers
// VROOLI_ROOT (set by the lifecycle), then walks up from an absolute
// storeDir (.../scenarios/prompt-manager/store -> repo root). Returns
// empty when neither path produces a usable directory; in that case
// taxonomy resolution is skipped and the Inbox Flow section renders
// without dispatch tables (operators see _NOT FOUND_ markers).
func deriveRepoRoot(storeDir string) string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	if strings.TrimSpace(storeDir) == "" {
		return ""
	}
	abs, err := filepath.Abs(storeDir)
	if err != nil {
		return ""
	}
	// store -> prompt-manager -> scenarios -> repo
	cur := abs
	for i := 0; i < 3; i++ {
		next := filepath.Dir(cur)
		if next == cur {
			return ""
		}
		cur = next
	}
	return cur
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
