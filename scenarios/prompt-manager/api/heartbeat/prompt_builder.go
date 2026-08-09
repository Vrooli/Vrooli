package heartbeat

import (
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/interop"
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

	// Reference sections live in one named XML context. The task stays outside
	// it so the model sees the difference between material to consult and the
	// job to do. This also leaves every stable band contiguous for prefix cache
	// reuse.
	var contextParts []string
	var taskParts []string
	for _, section := range sections {
		entry := promptSectionKinds[section.Kind]
		if entry.Scope == promptScopeTask {
			taskParts = append(taskParts, section.Content)
			continue
		}
		attrs := ""
		if section.SourcePath != "" {
			attrs = ` source="` + html.EscapeString(section.SourcePath) + `"`
		}
		contextParts = append(contextParts, fmt.Sprintf("<%s%s>\n%s\n</%s>", promptElement(section.Kind), attrs, strings.TrimSpace(section.Content), promptElement(section.Kind)))
	}

	var parts []string
	if len(contextParts) > 0 {
		parts = append(parts, "<context>\n\n"+strings.Join(contextParts, "\n\n")+"\n\n</context>")
	}
	parts = append(parts, taskParts...)
	if len(parts) == 0 {
		return "", fmt.Errorf("no content available for heartbeat prompt")
	}

	return strings.Join(parts, "\n\n"), nil
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

		// Emit in volatility order. The stable team band must stay ahead of all
		// member and live-run material; otherwise one member's task or one live
		// ledger entry destroys the cache prefix for everyone behind it.
		sections = append(sections, newPromptSection(promptSectionKindSharedDoctrine, "", buildSharedDoctrineSection(includeHeartbeat)))

		teamPolicy, err := b.buildOperatingPolicyTeamSection(team, teamCharter)
		if err != nil {
			return nil, err
		}
		sections = append(sections, newPromptSection(promptSectionKindOperatingPolicy, fmt.Sprintf("teams/%s/team.json", teamID), teamPolicy))

		if section, err := b.buildStorageMapSection(team, agentID); err != nil {
			return nil, err
		} else if section != "" {
			sections = append(sections, newPromptSection(promptSectionKindStorageMap, "", section))
		}

		memberPolicy, err := b.buildOperatingPolicyMemberSection(team, agentID)
		if err != nil {
			return nil, err
		}
		sections = append(sections, newPromptSection(promptSectionKindMemberPolicy, fmt.Sprintf("teams/%s/team.json#operatingContract.members.%s", teamID, agentID), memberPolicy))

		if teamconfig.ShouldShowOrgContext(contract) {
			if section := b.buildOrgContextSection(ctx, team, agentID); section != "" {
				sections = append(sections, newPromptSection(promptSectionKindOrgContext, "", section))
			}
		}

		if section := b.buildTopicContractSection(teamID, agentID); section != "" {
			sections = append(sections, newPromptSection(promptSectionKindTopicContract, fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID), section))
		}

		if section := b.buildInboxFlowSection(teamID, agentID); section != "" {
			sections = append(sections, newPromptSection(promptSectionKindInboxFlow, fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID), section))
		}

		responsibilities, err := b.teamStore.GetResponsibilities(ctx, teamID, agentID)
		if err == nil && responsibilities != "" {
			sections = append(sections, newPromptSection(promptSectionKindResponsibilities, fmt.Sprintf("teams/%s/members/%s/RESPONSIBILITIES.md", teamID, agentID), promptHeading(promptSectionKindResponsibilities)+"\n\n"+shiftMarkdownHeadings(responsibilities, 2)))
		}

		sections = append(sections, agentSections...)

		// Live inbox, Source Ledger wake, and contract findings form the volatile
		// band. They must remain below every stable member section.
		if teamconfig.ShouldInjectInbox(contract) {
			if section := b.buildInboxSection(ctx, teamID, agentID); section != "" {
				sections = append(sections, newPromptSection(promptSectionKindTeamInbox, "", section))
			}
		}
		ledgerHealthy := b.teamStore.HasSourceLedger()
		if section, err := b.buildTeamContextWakeSection(ctx, teamID); err != nil {
			sections = append(sections, newPromptSection(promptSectionKindContinuityFallback, "source-ledger:team:"+teamID, renderContinuityFallbackSection(teamID, err)))
		} else if section != "" {
			sections = append(sections, newPromptSection(promptSectionKindTeamWake, "source-ledger:team:"+teamID, section))
		}
		if !ledgerHealthy {
			sections = append(sections, newPromptSection(promptSectionKindContinuityFallback, "source-ledger:team:"+teamID, renderContinuityFallbackSection(teamID, nil)))
		}

		// Findings are run-volatile and belong after the declarations they
		// describe, not between stable team and member bands.
		if section := b.buildContractFindingsSection(ctx, teamID, agentID); section != "" {
			sections = append(sections, newPromptSection(promptSectionKindContractFindings, fmt.Sprintf("teams/%s/members/%s/topics.json", teamID, agentID), section))
		}

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
			sections[len(sections)-1].Content = buildHeartbeatTaskSection(team, agentID, heartbeatInstructions, sections[len(sections)-1].Content)
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

// renderContinuityFallbackSection is emitted only when the Source Ledger is
// unavailable or its wake read fails. Healthy ledger-backed prompts contain no
// continuity-template language; an unhealthy dependency gets a bounded,
// actionable fallback so the next run is not silently disconnected.
func renderContinuityFallbackSection(teamID string, err error) string {
	var section strings.Builder
	section.WriteString(promptHeading(promptSectionKindContinuityFallback) + "\n\n")
	section.WriteString(fmt.Sprintf("The Source Ledger scope `team:%s` is not healthy for this run", teamID))
	if err != nil {
		section.WriteString(fmt.Sprintf(" (%v)", err))
	}
	section.WriteString(". Record concise continuity in the final response so an operator can recover it, then stop.\n")
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
// mode. Keep it constant: one member-specific byte here destroys the shared
// prefix this section exists to create.
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
	section.WriteString("| Continuity for your next run | declared Source Ledger topic | Durable context, not a self-reported completion claim |\n")
	section.WriteString("| Evidence, measurements, findings | `source-ledger journal note` in scope `team:<id>` | Typed-topic registry: `docs/agent-system/TOPICS.md` |\n")
	section.WriteString("| Broken code or scenario behavior | `prompt-manager skill read report-bug` | Writes `bug-inbox/*` on `scenario-qa` |\n")
	section.WriteString("| Something missing, confusing, or repeatedly worked around | `prompt-manager skill read report-friction` | Writes `friction-inbox/*` on `meta-optimization` |\n")
	section.WriteString("| A raw observation | `swarm-manager captures create` | Read dispositions with `swarm-manager backlog list --actor-id=<verified-profile-key>` |\n")
	section.WriteString("| A shaped outcome | `swarm-manager backlog create` | Same disposition read path |\n")
	section.WriteString("| Live team objects you maintain | team working state | Only the files named in your operating contract |\n\n")
	section.WriteString("Record what a later run needs in your declared Source Ledger topics. Confirm writes from `X-Vrooli-Attribution` receipts; do not treat your final response as proof that a write happened.\n\n")
	section.WriteString("## Authority order — the world\n\n")
	section.WriteString("When sources disagree about what is true or accepted:\n\n")
	section.WriteString("1. Operator instruction in the current run\n2. Accepted plan-of-record docs\n3. Accepted work dispositions\n4. Team working state\n5. Source Ledger evidence\n\n")
	section.WriteString("## Authority order — this prompt\n\n")
	section.WriteString("When sections of this prompt disagree, the later rule yields to the earlier one:\n\n")
	rank := 1
	writeRank := func(text string) {
		section.WriteString(fmt.Sprintf("%d. %s\n", rank, text))
		rank++
	}
	writeRank("Operator instruction given during this run")
	writeRank("Your contract — `" + promptElement(promptSectionKindOperatingPolicy) + "`, `" + promptElement(promptSectionKindMemberPolicy) + "`, and `" + promptElement(promptSectionKindTopicContract) + "`. Write surfaces and safety-critical rules bind the task; the task cannot widen them.")
	if includeHeartbeat {
		writeRank("`" + promptElement(promptSectionKindHeartbeatTask) + "` — the job for this run")
	}
	writeRank("`" + promptElement(promptSectionKindResponsibilities) + "` and `" + promptElement(promptSectionKindAgentFile) + "` — standing guidance that a task may override")
	return strings.TrimRight(section.String(), "\n")
}

type memberStoragePolicy struct {
	CanWriteKnowledge bool
	CanWriteTask      bool
}

func buildHeartbeatTaskSection(team *store.Team, agentID, heartbeatInstructions, heartbeatSection string) string {
	teamID := ""
	teamName := ""
	lane := ""
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
		}
	}
	if lane == "" {
		lane = "Apply the team mission within this member's assigned scope."
	}
	var section strings.Builder
	section.WriteString(promptHeading(promptSectionKindHeartbeatTask) + "\n\n")
	section.WriteString(fmt.Sprintf("You are running one prompt-manager heartbeat as `%s` on `%s`", agentID, teamID))
	if teamName != "" && teamName != teamID {
		section.WriteString(fmt.Sprintf(" (`%s`)", teamName))
	}
	section.WriteString(".\n\n")
	section.WriteString("Your lane: " + lane + "\n\n")
	section.WriteString("Follow the task loop below. Keep standing duties in `responsibilities` and use only the declared storage surfaces.\n\n")
	section.WriteString(strings.TrimSpace(strings.TrimPrefix(heartbeatSection, promptHeading(promptSectionKindHeartbeatTask))))
	if !strings.Contains(heartbeatSection, "Run Decision") {
		section.WriteString("\n\nRecord durable continuity in your declared Source Ledger topics. Choose one disposition — `existing-action-reference`, `new-action-candidate`, `cli-backlog`, `capability-work-item`, `prune`, `improve`, `graduate`, or `no-action` — and give the observable reason.\n")
	}
	return strings.TrimRight(section.String(), "\n")
}

func buildMemberStoragePolicy(team *store.Team, agentID string, configDir string) memberStoragePolicy {
	if team == nil || team.OperatingContract == nil {
		return memberStoragePolicy{}
	}
	member, ok := team.OperatingContract.Members[agentID]
	if !ok {
		return memberStoragePolicy{}
	}
	policy := memberStoragePolicy{}
	for _, ref := range member.AllowedWrites {
		if ref.Kind != "" {
			switch ref.Kind {
			case "knowledge":
				policy.CanWriteKnowledge = true
			case "task":
				policy.CanWriteTask = true
			}
			continue
		}
	}
	for _, ref := range member.ForbiddenWrites {
		if ref.Kind == "" {
			continue
		}
		switch ref.Kind {
		case "knowledge":
			policy.CanWriteKnowledge = false
		case "task":
			policy.CanWriteTask = false
		}
	}
	return policy
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

func (b *PromptBuilder) buildOperatingPolicyTeamSection(team *store.Team, teamCharter string) (string, error) {
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
	return strings.TrimRight(section.String(), "\n"), nil
}

func (b *PromptBuilder) buildOperatingPolicyMemberSection(team *store.Team, agentID string) (string, error) {
	if team == nil {
		return "", nil
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
	var section strings.Builder
	section.WriteString(promptHeading(promptSectionKindMemberPolicy) + "\n\n")
	section.WriteString(memberPolicy)
	section.WriteString("\n\nThe member policy is authoritative for this member's declared permissions; the task cannot widen it.\n")
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
