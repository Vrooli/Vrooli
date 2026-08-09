package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/interop"
	"prompt-manager/memberflow"
	"prompt-manager/sourceledger"
	"prompt-manager/store"
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
)

// PromptBuildRequest defines the inputs for assembling a heartbeat prompt.
// TeamID is optional; when omitted, only agent markdown files are included.
type PromptBuildRequest struct {
	TeamID  string
	AgentID string
}

// PromptBuilder assembles prompts using agent + team context.
type PromptBuilder struct {
	teamStore        *store.FileTeamStore
	agentStore       *store.FileAgentStore
	contractFindings ContractFindingsProvider
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
		if sections[i].Kind == promptSectionKindAgentFile {
			block := promptHeading(promptSectionKindAgentFile) + "\n\n"
			for i < len(sections) && sections[i].Kind == promptSectionKindAgentFile {
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
				agentSections = append(agentSections, newPromptSection(promptSectionKindAgentFile, fmt.Sprintf("agents/%s/%s", agentID, entry.Path), fmt.Sprintf("## %s\n\n%s\n\n", entry.Path, shiftMarkdownHeadings(content, 3))))
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

		// Standing rules lead so the prompt opens with a block that is identical
		// for every member; the task sandwich still holds because the brief
		// precedes every *context* section, and standing rules are framing
		// rather than context.
		sections = append(sections, newPromptSection(promptSectionKindSharedDoctrine, "", buildSharedDoctrineSection(includeHeartbeat)))

		// The heartbeat prompt uses an intentional task sandwich: the active
		// brief comes first so every later context section is read through the
		// current job, while the full HEARTBEAT.md and final reminder stay near
		// the end for recency. Middle sections should not duplicate doctrine
		// already owned by generated storage, contract, or coordination context.
		sections = append(sections, newPromptSection(promptSectionKindActiveTaskBrief, fmt.Sprintf("teams/%s/team.json#operatingContract.members.%s", teamID, agentID), buildActiveTaskBriefSection(team, agentID, includeHeartbeat, heartbeatInstructions, b.teamStore.StoreDir())))

		if teamconfig.ShouldInjectInbox(contract) {
			if section := b.buildInboxSection(ctx, teamID, agentID); section != "" {
				sections = append(sections, newPromptSection(promptSectionKindTeamInbox, "", section))
			}
		}

		if section, err := b.buildTeamContextWakeSection(ctx, teamID); err != nil {
			return nil, err
		} else if section != "" {
			sections = append(sections, newPromptSection(promptSectionKindTeamWake, "source-ledger:team:"+teamID, section))
		}

		if handoff, err := b.teamStore.GetLastHandoff(ctx, teamID, agentID); err == nil && handoff != "" {
			sections = append(sections, newPromptSection(promptSectionKindLastHandoff, fmt.Sprintf("teams/%s/members/%s/last-handoff.md", teamID, agentID), promptHeading(promptSectionKindLastHandoff)+"\n\nThis is what you noted at the end of your last heartbeat:\n\n"+shiftMarkdownHeadings(handoff, 2)))
		}

		if section, err := b.buildStorageMapSection(team, agentID); err != nil {
			return nil, err
		} else if section != "" {
			sections = append(sections, newPromptSection(promptSectionKindStorageMap, "", section))
		}

		if teamconfig.ShouldShowOrgContext(contract) {
			if section := b.buildOrgContextSection(ctx, team, agentID); section != "" {
				sections = append(sections, newPromptSection(promptSectionKindOrgContext, "", section))
			}
		}

		operatingPolicy, err := b.buildOperatingPolicySection(team, agentID, teamCharter)
		if err != nil {
			return nil, err
		}
		sections = append(sections, newPromptSection(promptSectionKindOperatingPolicy, fmt.Sprintf("teams/%s/team.json", teamID), operatingPolicy))

		if section := b.buildTopicContractSection(teamID, agentID); section != "" {
			sections = append(sections, newPromptSection(promptSectionKindTopicContract, fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID), section))
		}

		if section := b.buildInboxFlowSection(teamID, agentID); section != "" {
			sections = append(sections, newPromptSection(promptSectionKindInboxFlow, fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID), section))
		}

		// Sits directly after the declarations it reports on, so the contract
		// and the ways this member is currently breaking it read together.
		if section := b.buildContractFindingsSection(ctx, teamID, agentID); section != "" {
			sections = append(sections, newPromptSection(promptSectionKindContractFindings, fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID), section))
		}

		responsibilities, err := b.teamStore.GetResponsibilities(ctx, teamID, agentID)
		if err == nil && responsibilities != "" {
			sections = append(sections, newPromptSection(promptSectionKindResponsibilities, fmt.Sprintf("teams/%s/members/%s/RESPONSIBILITIES.md", teamID, agentID), promptHeading(promptSectionKindResponsibilities)+"\n\n"+shiftMarkdownHeadings(responsibilities, 2)))
		}

		sections = append(sections, agentSections...)

		if includeHeartbeat {
			if heartbeatInstructions != "" {
				sections = append(sections, newPromptSection(promptSectionKindHeartbeatTask, fmt.Sprintf("teams/%s/members/%s/HEARTBEAT.md", teamID, agentID), promptHeading(promptSectionKindHeartbeatTask)+"\n\n"+shiftMarkdownHeadings(heartbeatInstructions, 2)))
			} else {
				// No HEARTBEAT.md for this member. The heading is the same as
				// the populated branch because the precedence list names it
				// unconditionally; a second spelling here would rank a heading
				// the reader cannot find.
				sections = append(sections, newPromptSection(promptSectionKindHeartbeatTask, "",
					promptHeading(promptSectionKindHeartbeatTask)+"\n\nNo specific heartbeat instructions defined. Please review your responsibilities and perform any pending work."))
			}
			sections = append(sections, newPromptSection(promptSectionKindTaskReminder, fmt.Sprintf("teams/%s/team.json#operatingContract.members.%s", teamID, agentID), buildTaskReminderSection(team, agentID, heartbeatInstructions)))
		}
	} else {
		sections = append(sections, agentSections...)
	}

	if err := validatePromptSections(sections); err != nil {
		return nil, err
	}
	return sections, nil
}

// buildTeamContextWakeSection injects only Source Ledger's bounded ambient
// view. The wake is context, not authority: the section tells the member to
// use the authoritative storage map and unified work feed for current state.
func (b *PromptBuilder) buildTeamContextWakeSection(ctx context.Context, teamID string) (string, error) {
	wake, err := b.teamStore.WakeTeamCorpus(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("reading team source-ledger wake: %w", err)
	}
	if !b.teamStore.HasSourceLedger() {
		return "", nil
	}
	return renderTeamContextWakeSection(teamID, wake), nil
}

func renderTeamContextWakeSection(teamID string, wake sourceledger.WakeResult) string {
	var section strings.Builder
	section.WriteString(promptHeading(promptSectionKindTeamWake) + "\n\n")
	section.WriteString("Bounded ambient context from the `team:" + teamID + "` Source Ledger scope. This is orientation, not authority; use the Storage Map and unified work feed for current state.\n\n")
	if wake.Overflow {
		section.WriteString(fmt.Sprintf("This view was truncated: %d memories were refused by the Source Ledger ceilings. Retrieve the complete context with `source-ledger recall \"<query>\" --scope=team:%s`.", wake.Refused, teamID))
		if len(wake.Entries) == 0 {
			return section.String()
		}
		section.WriteString("\n\n")
	}
	if len(wake.Entries) == 0 {
		section.WriteString("No durable team context has been recorded yet.")
		return section.String()
	}
	for _, entry := range wake.Entries {
		body := strings.TrimSpace(entry.Body)
		if body == "" {
			continue
		}
		section.WriteString("- ")
		section.WriteString(strings.ReplaceAll(body, "\n", "\n  "))
		section.WriteString("\n")
	}
	return strings.TrimRight(section.String(), "\n")
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

// buildSharedDoctrineSection renders the standing rules every member gets.
//
// It is emitted first and is byte-identical for every member in a given build
// mode. That is the entire point: the doctrine below was previously interleaved
// with member-specific text inside `# Storage Map` and `# Active Task Brief`,
// so no two prompts shared a prefix and the same ~1.6k tokens were re-sent to
// every member on every tick with nothing cacheable. Keep it constant — one
// member-specific byte here and the shared prefix is gone.
//
// includeHeartbeat is the only permitted variation: naming the heartbeat
// section in the precedence list when the build omits it would rank a heading
// the reader cannot find. All heartbeat builds still share one prefix.
func buildSharedDoctrineSection(includeHeartbeat bool) string {
	var section strings.Builder
	section.WriteString(promptHeading(promptSectionKindSharedDoctrine) + "\n\n")
	section.WriteString("## Where things go\n\n")
	section.WriteString("Use persistent storage only when information must survive this run.\n\n")
	section.WriteString("| To record | Use | Notes |\n|---|---|---|\n")
	section.WriteString("| Continuity for your next run | final `## HANDOFF` section | Next-run memory, not canonical truth |\n")
	section.WriteString("| Evidence, measurements, findings | `source-ledger journal note` in scope `team:<id>` | Typed-topic registry: `docs/agent-system/TOPICS.md` |\n")
	section.WriteString("| Broken code or scenario behavior | `prompt-manager skill read report-bug` | Writes `bug-inbox/*` on `scenario-qa` |\n")
	section.WriteString("| Something missing, confusing, or repeatedly worked around | `prompt-manager skill read report-friction` | Writes `friction-inbox/*` on `meta-optimization` |\n")
	section.WriteString("| A raw observation | `swarm-manager captures create` | Read dispositions with `swarm-manager backlog list --actor-id=<verified-profile-key>` |\n")
	section.WriteString("| A shaped outcome | `swarm-manager backlog create` | Same disposition read path |\n")
	section.WriteString("| Live team objects you maintain | team working state | Only the files named in your operating contract |\n\n")
	section.WriteString("One-off friction not worth filing goes in handoff. When your contract requires a handoff, write it as the last section of your final response and then stop — do not wait for confirmation inside the same run.\n\n")
	section.WriteString("## Authority order — the world\n\n")
	section.WriteString("When sources disagree about what is true or accepted:\n\n")
	section.WriteString("1. Operator instruction in the current run\n2. Accepted plan-of-record docs\n3. Accepted work dispositions\n4. Team working state\n5. Knowledge log evidence\n6. Handoff\n\n")
	section.WriteString("## Authority order — this prompt\n\n")
	section.WriteString("When sections of this prompt disagree, the later rule yields to the earlier one:\n\n")
	rank := 1
	writeRank := func(text string) {
		section.WriteString(fmt.Sprintf("%d. %s\n", rank, text))
		rank++
	}
	writeRank("Operator instruction given during this run")
	writeRank("Your contract — `" + promptHeading(promptSectionKindActiveTaskBrief) + "`, `" + promptHeading(promptSectionKindOperatingPolicy) + "`, `" + promptHeading(promptSectionKindTopicContract) + "`. Write surfaces and safety-critical rules bind the task; the task cannot widen them.")
	if includeHeartbeat {
		writeRank("`" + promptHeading(promptSectionKindHeartbeatTask) + "` — the job for this run")
	}
	writeRank("`" + promptHeading(promptSectionKindResponsibilities) + "` and `" + promptHeading(promptSectionKindAgentFile) + "` — standing guidance that a task may override")
	writeRank("`" + promptHeading(promptSectionKindLastHandoff) + "` — prior-run notes, never authority")
	return strings.TrimRight(section.String(), "\n")
}

type memberStoragePolicy struct {
	CanWriteKnowledge         bool
	RequiresHandoff           bool
	CanWriteTask              bool
	CanWriteWorkingStatePaths []string
	AllowedWriteLabels        []string
	ForbiddenWriteLabels      []string
	RequiredReadPrefixes      []string
}

func buildActiveTaskBriefSection(team *store.Team, agentID string, includeHeartbeat bool, heartbeatInstructions string, configDir string) string {
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
			policy = buildMemberStoragePolicy(team, agentID, configDir)
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
	section.WriteString(promptHeading(promptSectionKindActiveTaskBrief) + "\n\n")
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
	section.WriteString(renderWriteSurface(policy))
	// Omit rather than negate: a line saying a member has nothing declared
	// costs tokens on every tick and tells the reader nothing it could act on.
	if len(policy.RequiredReadPrefixes) > 0 {
		section.WriteString("## Required Memory\n\nKnowledge topics:\n")
		for _, topic := range policy.RequiredReadPrefixes {
			section.WriteString("- `" + topic + "`\n")
		}
		section.WriteString("\n")
	}
	if policy.RequiresHandoff {
		section.WriteString("Handoff: end with `## HANDOFF` as the final response section.\n")
	}
	// Conflict precedence and work-filing routes are in `# Standing Rules`.
	return strings.TrimRight(section.String(), "\n")
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
	section.WriteString(promptHeading(promptSectionKindTaskReminder) + "\n\n")
	section.WriteString(fmt.Sprintf("Run this heartbeat as `%s` on `%s`.\n\n", agentID, teamID))
	section.WriteString(fmt.Sprintf("Focus on: %s.\n\n", focus))
	section.WriteString("Do now:\n")
	section.WriteString("1. Follow the task loop in `HEARTBEAT.md`.\n")
	section.WriteString("2. Use only the write surfaces allowed in `# Active Task Brief`.\n")
	itemThree := "3. Record observations, friction, and shaped work"
	if len(policy.CanWriteWorkingStatePaths) > 0 {
		itemThree += ", working-state updates"
	}
	if policy.RequiresHandoff {
		itemThree += ", and handoff"
	}
	itemThree += " according to `# Storage Map`"
	section.WriteString(itemThree + ".\n")
	if policy.RequiresHandoff {
		section.WriteString("4. End with `## HANDOFF` when required, then stop.")
	}
	return section.String()
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

// renderWriteSurface renders the member's declared write surface for the
// Active Task Brief. It is the one place that turns the storage policy into
// prose, so the brief and `# Storage Map` cannot describe different surfaces
// for the same member: both read the same policy, computed once per build.
// TestWriteSurfaceSitesMoveTogether pins that across a declaration change.
func renderWriteSurface(policy memberStoragePolicy) string {
	if len(policy.AllowedWriteLabels) == 0 && len(policy.ForbiddenWriteLabels) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("## Write Surface\n\n")
	writeList := func(heading string, labels []string) {
		if len(labels) == 0 {
			return
		}
		out.WriteString(heading + ":\n")
		for _, label := range labels {
			out.WriteString("- " + label + "\n")
		}
		out.WriteString("\n")
	}
	writeList("Allowed", policy.AllowedWriteLabels)
	writeList("Forbidden", policy.ForbiddenWriteLabels)
	return out.String()
}

func buildMemberStoragePolicy(team *store.Team, agentID string, configDir string) memberStoragePolicy {
	if team == nil || team.OperatingContract == nil {
		return memberStoragePolicy{}
	}
	member, ok := team.OperatingContract.Members[agentID]
	if !ok {
		return memberStoragePolicy{}
	}
	policy := memberStoragePolicy{
		RequiresHandoff:      teamconfig.RequiresHandoff(team.Contract()) && !writeRefsContainKind(member.ForbiddenWrites, "handoff"),
		RequiredReadPrefixes: loadRequiredReadPrefixes(configDir, team.ID, agentID),
	}
	for _, ref := range member.AllowedWrites {
		label := describeWriteRef(ref, team, agentID, configDir)
		if label != "" {
			policy.AllowedWriteLabels = append(policy.AllowedWriteLabels, label)
		}
		if ref.Kind != "" {
			switch ref.Kind {
			case "knowledge":
				policy.CanWriteKnowledge = true
			case "handoff":
				policy.RequiresHandoff = teamconfig.RequiresHandoff(team.Contract())
			case "task":
				policy.CanWriteTask = true
			}
			continue
		}
		normalized := normalizedWritePath(ref, team, agentID, configDir)
		if normalized != "" {
			policy.CanWriteWorkingStatePaths = append(policy.CanWriteWorkingStatePaths, normalized)
		}
	}
	for _, ref := range member.ForbiddenWrites {
		label := describeWriteRef(ref, team, agentID, configDir)
		if label != "" {
			policy.ForbiddenWriteLabels = append(policy.ForbiddenWriteLabels, label)
		}
		if ref.Kind == "" {
			continue
		}
		switch ref.Kind {
		case "knowledge":
			policy.CanWriteKnowledge = false
		case "handoff":
			policy.RequiresHandoff = false
		case "task":
			policy.CanWriteTask = false
		}
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
// of truth for read relationships. When configDir is empty (e.g., the
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
func loadRequiredReadPrefixes(configDir, teamID, agentID string) []string {
	if strings.TrimSpace(configDir) == "" {
		return nil
	}
	mt, err := memberflow.LoadMember(configDir, teamID, agentID)
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

func describeWriteRef(ref teamcontract.WriteRef, team *store.Team, agentID string, configDir string) string {
	if ref.Kind != "" {
		switch ref.Kind {
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
	if path := normalizedWritePath(ref, team, agentID, configDir); path != "" {
		if label := describeKnownWritePath(path, team, agentID, configDir); label != "" {
			return label
		}
		return "`" + path + "`"
	}
	return ""
}

func describeKnownWritePath(path string, team *store.Team, agentID string, configDir string) string {
	switch {
	case strings.HasSuffix(path, "/tasks.json"):
		return "team task board updates"
	}
	if team == nil || team.OperatingContract == nil {
		return ""
	}
	for _, doc := range team.OperatingContract.Documents.SharedState {
		normalized, err := teamcontract.NormalizePath(doc.Path, teamcontract.ValidationInput{
			TeamID:   team.ID,
			StoreDir: configDir,
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

func normalizedWritePath(ref teamcontract.WriteRef, team *store.Team, agentID string, configDir string) string {
	if team == nil {
		return ""
	}
	path, err := teamcontract.NormalizePath(teamcontract.PathRef{
		Base:     ref.Base,
		Path:     ref.Path,
		MemberID: ref.MemberID,
		AgentID:  ref.AgentID,
	}, teamcontract.ValidationInput{
		TeamID:   team.ID,
		StoreDir: configDir,
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
	section.WriteString(promptHeading(promptSectionKindOperatingPolicy) + "\n\n")
	if strings.TrimSpace(teamCharter) != "" {
		section.WriteString("## Team Charter\n\n")
		section.WriteString(fmt.Sprintf("Source: `teams/%s/shared/TEAM.md`\n\n", team.ID))
		section.WriteString(shiftMarkdownHeadings(strings.TrimSpace(teamCharter), 3))
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
	memberPolicy, err := teamcontract.RenderMemberPolicy(team.OperatingContract, teamcontract.RenderInput{
		TeamID:   team.ID,
		TeamName: team.DisplayName,
		MemberID: agentID,
		StoreDir: b.teamStore.StoreDir(),
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
		teamconfig.AllowsPeerTriggers(contract)
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

	section := promptHeading(promptSectionKindOrgContext) + "\n\n"
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

	// Doctrine that is the same for every member lives in
	// `# Standing Rules`. This section carries only what differs by team and
	// member: the declared storage surfaces and the commands this member may
	// actually run.
	var section strings.Builder
	section.WriteString(promptHeading(promptSectionKindStorageMap) + "\n\nYour declared surfaces. Routing rules and authority order are in `" + promptHeading(promptSectionKindSharedDoctrine) + "`.\n")

	teamStorage, err := teamcontract.RenderTeamStorage(team.OperatingContract, teamcontract.RenderInput{
		TeamID:         team.ID,
		TeamName:       team.DisplayName,
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

	// Every block below is omitted when the member cannot use it. A command
	// list that says a command is unavailable is pure prompt weight: the
	// member cannot act on it, and its absence carries the same information.
	if teamconfig.ShouldShowTaskBoardGuidance(contract) {
		lines = append(lines, "### Task Board")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Review current work: `prompt-manager team task-list %s`", team.ID))
		if policy.CanWriteTask {
			lines = append(lines, fmt.Sprintf("- Add a task: `prompt-manager team task-add %s --by=<agent-id> --title \"...\"`", team.ID))
			lines = append(lines, fmt.Sprintf("- Update a task: `prompt-manager team task-update %s <task-id> --status=in-progress`", team.ID))
		}
		lines = append(lines, "")
	}

	if teamconfig.ShouldShowKnowledgeLogGuidance(contract) {
		lines = append(lines, "### Knowledge Log")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Recall team context: `source-ledger recall \"<query>\" --scope=team:%s`", team.ID))
		if policy.CanWriteKnowledge {
			lines = append(lines, fmt.Sprintf("- Record durable team context: `source-ledger journal note \"<prose>\" --scope=team:%s --kind=team-knowledge`", team.ID))
		}
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
	configDir := b.teamStore.StoreDir()
	repoRoot := deriveRepoRoot(configDir)
	in, ok, err := LoadInboxFlowInputs(configDir, repoRoot, teamID, agentID)
	if err != nil || !ok {
		return ""
	}
	return RenderInboxFlow(in)
}

// deriveRepoRoot resolves the repository root for taxonomy lookup. Prefers
// VROOLI_ROOT (set by the lifecycle), then walks up from an absolute
// configDir (.../scenarios/prompt-manager/store -> repo root). Returns
// empty when neither path produces a usable directory; in that case
// taxonomy resolution is skipped and the Inbox Flow section renders
// without dispatch tables (operators see _NOT FOUND_ markers).
func deriveRepoRoot(configDir string) string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	if strings.TrimSpace(configDir) == "" {
		return ""
	}
	abs, err := filepath.Abs(configDir)
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

	section := promptHeading(promptSectionKindTeamInbox) + "\n\n"
	section += fmt.Sprintf("You have %d pending message(s):\n\n", len(messages))
	for _, message := range messages {
		fromLabel := resolveAgentLabel(message.FromAgentID)
		section += fmt.Sprintf("## %s\n\n", message.ID)
		section += fmt.Sprintf("From: %s\n", fromLabel)
		section += fmt.Sprintf("Sent: %s\n\n", message.CreatedAt)
		section += shiftMarkdownHeadings(message.Content, 3) + "\n\n"
	}
	return section
}
