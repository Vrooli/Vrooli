package heartbeat

import (
	"context"
	"strings"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/memberflow"
	"prompt-manager/store"
)

type stubContractFindings struct {
	findings []ContractFinding
	err      error
	calls    int
}

func (s *stubContractFindings) MemberContractFindings(_ context.Context, _, _ string) ([]ContractFinding, error) {
	s.calls++
	return s.findings, s.err
}

func newContractFindingsFixture(t *testing.T) (context.Context, *PromptBuilder, *store.Team, string) {
	t.Helper()
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := store.NewFileStore(roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	agent := &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}
	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	team := newIndependentTestTeam("team-1", "Team One")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := teamStore.SetHeartbeatInstructions(ctx, team.ID, agent.ID, "Ship the update"); err != nil {
		t.Fatalf("set heartbeat instructions: %v", err)
	}
	return ctx, NewPromptBuilder(teamStore, agentStore), team, agent.ID
}

// An unwired builder must not look like a clean member. Rendering "no findings"
// without having checked would assert something the prompt does not know.
func TestContractFindingsSectionAbsentWithoutProvider(t *testing.T) {
	ctx, builder, team, agentID := newContractFindingsFixture(t)

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: agentID})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(prompt, promptHeadingContractFindings) {
		t.Fatalf("unwired builder rendered a findings section:\n%s", prompt)
	}
}

// The section exists to shrink to nothing when there is nothing to say. A
// clean member's prompt must not grow by a single line.
func TestContractFindingsSectionAbsentWhenMemberIsClean(t *testing.T) {
	ctx, builder, team, agentID := newContractFindingsFixture(t)
	stub := &stubContractFindings{}
	builder.SetContractFindingsProvider(stub)

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: agentID})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(prompt, promptHeadingContractFindings) {
		t.Fatalf("clean member rendered a findings section:\n%s", prompt)
	}
	if stub.calls == 0 {
		t.Fatal("provider was never consulted")
	}
}

// A provider failure must not take the heartbeat with it. The member still has
// a job to do; a missing advisory section is the correct degradation.
func TestContractFindingsProviderErrorDoesNotFailTheBuild(t *testing.T) {
	ctx, builder, team, agentID := newContractFindingsFixture(t)
	builder.SetContractFindingsProvider(&stubContractFindings{err: context.DeadlineExceeded})

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: agentID})
	if err != nil {
		t.Fatalf("Build should tolerate a findings provider error: %v", err)
	}
	if strings.Contains(prompt, promptHeadingContractFindings) {
		t.Fatalf("errored provider rendered a findings section:\n%s", prompt)
	}
}

func TestContractFindingsSectionRendersAfterTopicContract(t *testing.T) {
	ctx, builder, team, agentID := newContractFindingsFixture(t)
	builder.SetContractFindingsProvider(&stubContractFindings{findings: []ContractFinding{{
		Rule:     "actual_writer_undeclared",
		Severity: "error",
		Prefix:   "contrarian-scan/*",
		Detail:   `wrote 20 entries under "contrarian-scan/*" but member's output[] declares no overlapping prefix`,
	}}})

	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{TeamID: team.ID, AgentID: agentID})
	if err != nil {
		t.Fatalf("BuildStructured: %v", err)
	}
	kinds := distinctSectionKinds(sections)
	findingsIndex := sectionKindIndex(kinds, promptSectionKindContractFindings)
	if findingsIndex < 0 {
		t.Fatalf("findings section missing from %v", kinds)
	}
	// The findings comment on the topic contract, so they must follow it.
	if contractIndex := sectionKindIndex(kinds, promptSectionKindTopicContract); contractIndex >= 0 && contractIndex > findingsIndex {
		t.Fatalf("findings section precedes the contract it reports on: %v", kinds)
	}
	// The heartbeat task keeps the last word; findings are context, not the job.
	if taskIndex := sectionKindIndex(kinds, "heartbeat-task"); taskIndex >= 0 && taskIndex < findingsIndex {
		t.Fatalf("findings section displaced the heartbeat task: %v", kinds)
	}
}

func TestRenderContractFindingsOrdersErrorsFirst(t *testing.T) {
	rendered := renderContractFindings("team-1", []ContractFinding{
		{Rule: "loop_kind_missing", Severity: "warning", Detail: "member declares no loop_kind"},
		{Rule: "actual_writer_undeclared", Severity: "error", Prefix: "contrarian-scan/*", Detail: "undeclared prefix"},
	})

	errIndex := strings.Index(rendered, "actual_writer_undeclared")
	warnIndex := strings.Index(rendered, "loop_kind_missing")
	if errIndex < 0 || warnIndex < 0 {
		t.Fatalf("both findings should render:\n%s", rendered)
	}
	if errIndex > warnIndex {
		t.Fatalf("errors must rank ahead of warnings:\n%s", rendered)
	}
	for _, want := range []string{
		"2 items",
		"`contrarian-scan/*`",
		// The member cannot edit its own topics.json, so the section has to
		// name the routes it can actually take.
		"propose the change as a decision",
		"report it as friction",
		"prompt-manager graph topics --team team-1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered findings missing %q:\n%s", want, rendered)
		}
	}
}

// The inferred-backtick prose pattern matches any lowercase slash-separated
// string, so it also flags paths and package names. Those are fine in an
// operator sweep and wrong in a prompt: the member would be asked to correct
// a declaration that was never wrong. The provider drops them by the Advisory
// flag rather than by rule name, so a future heuristic opts in the same way.
func TestMemberflowProviderWithholdsAdvisoryFindings(t *testing.T) {
	findings := []memberflow.Finding{
		{
			Rule:     "prose_topic_leak",
			Severity: memberflow.SeverityWarning,
			Member:   memberflow.MemberRef{Team: "team-1", Member: "agent-1"},
			Prefix:   "docs/configuration/host",
			Detail:   "inferred-backtick match on a docs path",
			Advisory: true,
		},
		{
			Rule:     "actual_writer_undeclared",
			Severity: memberflow.SeverityError,
			Member:   memberflow.MemberRef{Team: "team-1", Member: "agent-1"},
			Prefix:   "contrarian-scan/*",
			Detail:   "undeclared prefix",
		},
		{
			// No member attribution: belongs to the corpus, not to anyone's
			// heartbeat. Routing it to a member would be a guess.
			Rule:     "prose_topic_leak",
			Severity: memberflow.SeverityWarning,
			Prefix:   "some/prefix",
			Detail:   "unattributed",
		},
	}

	kept := filterActionableFindings(findings)
	if len(kept["team-1/agent-1"]) != 1 {
		t.Fatalf("expected exactly the actionable finding, got %#v", kept)
	}
	if kept["team-1/agent-1"][0].Rule != "actual_writer_undeclared" {
		t.Fatalf("wrong finding survived: %#v", kept["team-1/agent-1"][0])
	}
	if len(kept) != 1 {
		t.Fatalf("unattributed finding was routed to a member: %#v", kept)
	}
}

func TestRenderContractFindingsSingularCount(t *testing.T) {
	rendered := renderContractFindings("team-1", []ContractFinding{
		{Rule: "orphan_output", Severity: "warning", Prefix: "x/*", Detail: "no declared consumer"},
	})
	if !strings.Contains(rendered, "1 item ") {
		t.Fatalf("single finding should read as one item:\n%s", rendered)
	}
}
