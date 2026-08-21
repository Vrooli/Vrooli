package heartbeat

import (
	"context"
	"sort"
	"strings"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/sourceledger"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

func TestRenderTeamContextWakeSectionDisclosesTruncation(t *testing.T) {
	section := renderTeamContextWakeSection("marketing-crew", sourceledger.WakeResult{
		Overflow: true,
		Refused:  3,
		Entries:  []sourceledger.Entry{{Body: "bounded memory"}},
	})
	for _, want := range []string{"This view was truncated: 3 memories were refused", "source-ledger recall", "bounded memory"} {
		if !strings.Contains(section, want) {
			t.Errorf("wake section missing %q: %s", want, section)
		}
	}
}

func promptSectionContent(sections []PromptSection, kind string) string {
	for _, section := range sections {
		if section.Kind == kind {
			return section.Content
		}
	}
	return ""
}

func distinctSectionKinds(sections []PromptSection) []string {
	seen := make(map[string]bool, len(sections))
	result := make([]string, 0, len(sections))
	for _, section := range sections {
		if seen[section.Kind] {
			continue
		}
		seen[section.Kind] = true
		result = append(result, section.Kind)
	}
	return result
}

func sectionKindIndex(kinds []string, kind string) int {
	for i, candidate := range kinds {
		if candidate == kind {
			return i
		}
	}
	return -1
}

func TestPromptBuilderAgentOnlyWrapsReferenceContext(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agent := &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive, FileOrder: []string{"SOUL.md", "NOTES.md"}}
	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := agentStore.CreateFile(ctx, agent.ID, "NOTES.md", "Notes content", false); err != nil {
		t.Fatalf("create notes: %v", err)
	}

	prompt, err := NewPromptBuilder(teamStore, agentStore).Build(ctx, PromptBuildRequest{AgentID: agent.ID})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	for _, want := range []string{"<context>", "<agent-files", "Notes content", "</agent-files>", "</context>"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("agent-only prompt missing %q: %s", want, prompt)
		}
	}
	if strings.Contains(prompt, "<heartbeat-task>") {
		t.Fatal("agent-only context unexpectedly contains a task")
	}
}

func TestPromptBuilderUsesVolatilityOrderAndTaskOutsideContext(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agent := &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}
	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
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
		t.Fatalf("write charter: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, team.ID, agent.ID, "Do the work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, team.ID, agent.ID, "Ship the update"); err != nil {
		t.Fatalf("set heartbeat: %v", err)
	}

	builder := NewPromptBuilder(teamStore, agentStore)
	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: agent.ID})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: agent.ID})
	if err != nil {
		t.Fatalf("build structured prompt: %v", err)
	}
	wantKinds := []string{
		promptSectionKindSharedDoctrine,
		promptSectionKindOperatingPolicy,
		promptSectionKindStorageMap,
		promptSectionKindMemberPolicy,
		promptSectionKindTopicContract,
		promptSectionKindResponsibilities,
		promptSectionKindContinuityFallback,
		promptSectionKindHeartbeatTask,
	}
	gotKinds := distinctSectionKinds(sections)
	for _, kind := range wantKinds {
		if sectionKindIndex(gotKinds, kind) == -1 {
			t.Errorf("structured prompt missing %q; got %v", kind, gotKinds)
		}
	}
	for i := 1; i < len(wantKinds); i++ {
		if sectionKindIndex(gotKinds, wantKinds[i-1]) > sectionKindIndex(gotKinds, wantKinds[i]) {
			t.Errorf("sections are not ordered by volatility: %v", gotKinds)
		}
	}
	contextEnd := strings.Index(prompt, "</context>")
	taskStart := strings.Index(prompt, "# Heartbeat Task (HEARTBEAT.md)")
	if contextEnd == -1 || taskStart == -1 || taskStart < contextEnd {
		t.Fatalf("task must be prose after context; context=%d task=%d", contextEnd, taskStart)
	}
	if strings.Contains(prompt, "<active-task-brief>") || strings.Contains(prompt, "<task-reminder>") {
		t.Fatal("retired task wrapper was rendered")
	}
	if strings.Contains(prompt, "HANDOFF") {
		t.Fatal("healthy-looking prompt contains legacy handoff instructions")
	}
	for _, tag := range []string{"<standing-rules>", "<operating-policy-team", "<operating-policy-member", "<topic-contract", "<responsibilities"} {
		if !strings.Contains(prompt, tag) {
			t.Errorf("context missing named child %s", tag)
		}
	}
}

func TestPromptBuilderInjectsContinuityFallbackWhenLedgerIsUnhealthy(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	prompt, err := NewPromptBuilder(teamStore, agentStore).Build(ctx, PromptBuildRequest{TeamID: "team-1", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if !strings.Contains(prompt, "<continuity-fallback") || !strings.Contains(prompt, "not healthy") {
		t.Fatalf("unhealthy ledger did not inject fallback: %s", prompt)
	}
	if strings.Contains(prompt, "HANDOFF") {
		t.Fatal("fallback must not revive the retired handoff template")
	}
}

func TestOperatingPolicyTeamPartIsMemberIndependent(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForRepoStoreTest(t, "../../../store"))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	team, err := teamStore.Get(ctx, "meta-optimization")
	if err != nil {
		t.Fatalf("load team: %v", err)
	}
	builder := NewPromptBuilder(teamStore, fileStore.Agents().(*store.FileAgentStore))
	first, err := builder.buildOperatingPolicyTeamSection(team, "")
	if err != nil {
		t.Fatalf("build team policy: %v", err)
	}
	second, err := builder.buildOperatingPolicyTeamSection(team, "")
	if err != nil {
		t.Fatalf("build team policy twice: %v", err)
	}
	if first != second {
		t.Fatal("team operating policy changed between members")
	}
	if strings.Contains(first, "skill-optimizer") || strings.Contains(first, "debt-curator") {
		t.Fatal("team operating policy contains member-specific content")
	}
	member, err := builder.buildOperatingPolicyMemberSection(team, "skill-optimizer")
	if err != nil {
		t.Fatalf("build member policy: %v", err)
	}
	if !strings.Contains(member, "skill-optimizer") || strings.Contains(first, "Your Member Contract") {
		t.Fatal("team/member policy split is not distinguishable")
	}
}

func TestPromptSectionRegistryHasVolatilityAndXMLIdentity(t *testing.T) {
	for kind, entry := range promptSectionKinds {
		if entry.Label == "" || entry.Heading == "" || entry.Element == "" || entry.Scope == "" {
			t.Errorf("section %q has incomplete identity: %+v", kind, entry)
		}
	}
	if _, ok := promptSectionKinds["active-task-brief"]; ok {
		t.Fatal("active-task-brief remains registered")
	}
	if _, ok := promptSectionKinds["task-reminder"]; ok {
		t.Fatal("task-reminder remains registered")
	}
}

func TestBuildContextOmitsTaskButRetainsReferenceContext(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, "team-1", "agent-1", "Ship update"); err != nil {
		t.Fatalf("set heartbeat: %v", err)
	}
	prompt, err := NewPromptBuilder(teamStore, agentStore).BuildContext(ctx, PromptBuildRequest{TeamID: "team-1", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("build context: %v", err)
	}
	if !strings.Contains(prompt, "<context>") || strings.Contains(prompt, "heartbeat-task") || strings.Contains(prompt, "Ship update") {
		t.Fatalf("BuildContext included task content: %s", prompt)
	}
}

func TestBuildStructuredMatchesRenderedPromptContract(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForRepoStoreTest(t, "../../../store"))
	builder := NewPromptBuilder(fileStore.Teams().(*store.FileTeamStore), fileStore.Agents().(*store.FileAgentStore))
	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{TeamID: "meta-optimization", AgentID: "skill-optimizer"})
	if err != nil {
		t.Fatalf("build structured: %v", err)
	}
	if len(sections) == 0 || sections[len(sections)-1].Kind != promptSectionKindHeartbeatTask {
		t.Fatalf("task must be the final structured section: %v", distinctSectionKinds(sections))
	}
	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: "meta-optimization", AgentID: "skill-optimizer"})
	if err != nil {
		t.Fatalf("build flat: %v", err)
	}
	if !strings.Contains(prompt, promptSectionContent(sections, promptSectionKindHeartbeatTask)) {
		t.Fatal("flat prompt omitted the structured task content")
	}
	if got := strings.Count(prompt, "Record durable continuity in your declared Source Ledger topics"); got != 1 {
		t.Fatalf("task decision guidance rendered %d times, want once", got)
	}
}

func TestBundledMembersRenderWithoutRetiredSections(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForRepoStoreTest(t, "../../../store"))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	builder := NewPromptBuilder(teamStore, fileStore.Agents().(*store.FileAgentStore))
	teams, err := teamStore.List(ctx)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	sort.Slice(teams, func(i, j int) bool { return teams[i].ID < teams[j].ID })
	count := 0
	for _, team := range teams {
		if team.OperatingContract == nil {
			continue
		}
		members, err := teamStore.GetMembers(ctx, team.ID)
		if err != nil {
			t.Fatalf("list %s members: %v", team.ID, err)
		}
		for _, member := range members {
			if member.Status != "" && member.Status != store.MemberStatusActive {
				continue
			}
			if _, ok := team.OperatingContract.Members[member.AgentID]; !ok {
				continue
			}
			prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: member.AgentID})
			if err != nil {
				t.Fatalf("build %s/%s: %v", team.ID, member.AgentID, err)
			}
			if strings.Contains(prompt, "active-task-brief") || strings.Contains(prompt, "task-reminder") {
				t.Fatalf("retired section kind rendered for %s/%s", team.ID, member.AgentID)
			}
			count++
		}
	}
	if count != 24 {
		t.Fatalf("rendered %d bundled members, want 24", count)
	}
}

func TestMemberWriteCommandVisibilityStillFollowsContract(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForRepoStoreTest(t, "../../../store"))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	team, err := teamStore.Get(ctx, "director-swarm")
	if err != nil {
		t.Fatalf("load team: %v", err)
	}
	policy := buildMemberStoragePolicy(team, "portfolio-manager", teamStore.StoreDir())
	if !policy.CanWriteKnowledge || !policy.CanWriteTask {
		t.Fatal("fixture member policy lost declared write permissions")
	}
	if strings.Contains(buildSharedDoctrineSection(true), "HANDOFF") {
		t.Fatal("standing rules still contain handoff instructions")
	}
}
