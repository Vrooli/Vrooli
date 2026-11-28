package heartbeat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
)

type stubTeamObjectives struct {
	byTeam map[string]TeamObjectiveContext
	err    error
	calls  []string
}

func (s *stubTeamObjectives) TeamObjectives(_ context.Context, teamID string) (TeamObjectiveContext, error) {
	s.calls = append(s.calls, teamID)
	if s.err != nil {
		return TeamObjectiveContext{}, s.err
	}
	return s.byTeam[teamID], nil
}

// newTeamObjectivesFixture builds one agent that belongs to two teams so the
// same member can be measured against different setpoints. The base build (no
// team) and both team builds share the fixture.
func newTeamObjectivesFixture(t *testing.T) (context.Context, *store.FileTeamStore, *PromptBuilder) {
	t.Helper()
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	for _, id := range []string{"team-a", "team-b"} {
		if err := teamStore.Create(ctx, newIndependentTestTeam(id, id)); err != nil {
			t.Fatalf("create team %s: %v", id, err)
		}
		if err := teamStore.SetHeartbeatInstructions(ctx, id, "agent-1", "Ship the update"); err != nil {
			t.Fatalf("set heartbeat instructions for %s: %v", id, err)
		}
	}
	return ctx, teamStore, NewPromptBuilder(teamStore, agentStore)
}

func teamObjectiveContext(teamID, revision string, objectives ...TeamObjective) TeamObjectiveContext {
	return TeamObjectiveContext{TeamID: teamID, AttachmentRevision: revision, Objectives: objectives}
}

// An unwired builder must not look like a team with no obligations. Omitting the
// section says "not read"; rendering an empty one would assert a setpoint the
// prompt never checked.
func TestTeamObjectivesSectionAbsentWithoutProvider(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(prompt, promptHeading(promptSectionKindTeamObjectives)) {
		t.Fatalf("unwired builder rendered an objectives section:\n%s", prompt)
	}
}

// A team with no attachments renders nothing rather than an empty setpoint.
func TestTeamObjectivesSectionAbsentWhenTeamHasNoAttachments(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)
	stub := &stubTeamObjectives{byTeam: map[string]TeamObjectiveContext{}}
	builder.SetTeamObjectiveProvider(stub)

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(prompt, promptHeading(promptSectionKindTeamObjectives)) {
		t.Fatalf("team with no attachments rendered an objectives section:\n%s", prompt)
	}
	if len(stub.calls) == 0 {
		t.Fatal("provider was never consulted")
	}
}

func TestTeamObjectivesSectionRendersOrderedRevisions(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)
	builder.SetTeamObjectiveProvider(&stubTeamObjectives{byTeam: map[string]TeamObjectiveContext{
		"team-a": teamObjectiveContext("team-a", "rev-a",
			TeamObjective{ObjectiveID: "T1", Title: "Income", Class: "terminal", MeaningRevision: "m1", Priority: 0, AcknowledgedRevision: "m1"},
			TeamObjective{ObjectiveID: "T3", Title: "Contribution", Class: "terminal", MeaningRevision: "m3", Priority: 1, AcknowledgedRevision: "m0", RestatementPending: true},
		),
	}})

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, want := range []string{
		promptHeading(promptSectionKindTeamObjectives),
		"<team-objectives",
		"Income",
		"`T1`",
		"meaning `m1`",
		"Contribution",
		"restatement pending (last acknowledged `m0`)",
		"Attachment revision: `rev-a`",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	incomeIndex := strings.Index(prompt, "Income")
	contributionIndex := strings.Index(prompt, "Contribution")
	if incomeIndex < 0 || contributionIndex < 0 || incomeIndex > contributionIndex {
		t.Fatalf("objectives were not rendered in priority order:\n%s", prompt)
	}
}

// The same member in two teams must see each team's own setpoint, and a base
// (team-less) build must see none.
func TestTeamObjectivesSectionDistinguishesTeamsAndBase(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)
	builder.SetTeamObjectiveProvider(&stubTeamObjectives{byTeam: map[string]TeamObjectiveContext{
		"team-a": teamObjectiveContext("team-a", "rev-a",
			TeamObjective{ObjectiveID: "T1", Title: "Income", MeaningRevision: "m1", AcknowledgedRevision: "m1"}),
		"team-b": teamObjectiveContext("team-b", "rev-b",
			TeamObjective{ObjectiveID: "I1", Title: "Capability compounding", MeaningRevision: "m2", AcknowledgedRevision: "m2"}),
	}})

	teamA, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("build team-a: %v", err)
	}
	teamB, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-b", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("build team-b: %v", err)
	}
	base, err := builder.BuildContext(ctx, PromptBuildRequest{AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("build base: %v", err)
	}

	if !strings.Contains(teamA, "Income") || strings.Contains(teamA, "Capability compounding") {
		t.Fatalf("team-a saw the wrong setpoint:\n%s", teamA)
	}
	if !strings.Contains(teamB, "Capability compounding") || strings.Contains(teamB, "Income") {
		t.Fatalf("team-b saw the wrong setpoint:\n%s", teamB)
	}
	if strings.Contains(base, promptHeading(promptSectionKindTeamObjectives)) {
		t.Fatalf("base (team-less) build rendered an objectives section:\n%s", base)
	}
}

// A changed meaning revision must reach fresh composition. There is no cache
// between the authority and the prompt, so the new revision replaces the old.
func TestTeamObjectivesMeaningRevisionChangeAppearsInFreshComposition(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)
	stub := &stubTeamObjectives{byTeam: map[string]TeamObjectiveContext{
		"team-a": teamObjectiveContext("team-a", "rev-1",
			TeamObjective{ObjectiveID: "T1", Title: "Income", MeaningRevision: "m1", AcknowledgedRevision: "m1"}),
	}}
	builder.SetTeamObjectiveProvider(stub)

	first, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("first build: %v", err)
	}
	if !strings.Contains(first, "meaning `m1`") {
		t.Fatalf("first build missing initial revision:\n%s", first)
	}

	stub.byTeam["team-a"] = teamObjectiveContext("team-a", "rev-2",
		TeamObjective{ObjectiveID: "T1", Title: "Income", MeaningRevision: "m2", AcknowledgedRevision: "m1", RestatementPending: true})

	second, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("second build: %v", err)
	}
	if !strings.Contains(second, "meaning `m2`") || !strings.Contains(second, "Attachment revision: `rev-2`") {
		t.Fatalf("fresh composition did not pick up the changed revision:\n%s", second)
	}
	if strings.Contains(second, "meaning `m1`") {
		t.Fatalf("stale revision survived a fresh composition:\n%s", second)
	}
}

// The objectives section is a stable team band: changing member and task inputs
// must not alter its bytes.
func TestTeamObjectivesSectionStableAcrossMemberAndTaskInputs(t *testing.T) {
	ctx, teamStore, builder := newTeamObjectivesFixture(t)
	builder.SetTeamObjectiveProvider(&stubTeamObjectives{byTeam: map[string]TeamObjectiveContext{
		"team-a": teamObjectiveContext("team-a", "rev-a",
			TeamObjective{ObjectiveID: "T1", Title: "Income", MeaningRevision: "m1", AcknowledgedRevision: "m1"}),
	}})

	first, err := builder.BuildStructured(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("first structured build: %v", err)
	}
	firstObjectives := promptSectionContent(first, promptSectionKindTeamObjectives)
	if firstObjectives == "" {
		t.Fatal("objectives section missing from first build")
	}

	if err := teamStore.SetHeartbeatInstructions(ctx, "team-a", "agent-1", "A different job entirely"); err != nil {
		t.Fatalf("update heartbeat: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-a", "agent-1", "Different duties"); err != nil {
		t.Fatalf("update responsibilities: %v", err)
	}

	second, err := builder.BuildStructured(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("second structured build: %v", err)
	}
	if got := promptSectionContent(second, promptSectionKindTeamObjectives); got != firstObjectives {
		t.Fatalf("objectives band changed with volatile inputs:\nfirst: %s\nsecond: %s", firstObjectives, got)
	}
}

// The objectives section sits in the stable team band, after the shared
// doctrine and before member-scoped sections.
func TestTeamObjectivesSectionPrecedesMemberSections(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)
	builder.SetTeamObjectiveProvider(&stubTeamObjectives{byTeam: map[string]TeamObjectiveContext{
		"team-a": teamObjectiveContext("team-a", "rev-a",
			TeamObjective{ObjectiveID: "T1", Title: "Income", MeaningRevision: "m1", AcknowledgedRevision: "m1"}),
	}})

	sections, err := builder.BuildStructured(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("BuildStructured: %v", err)
	}
	kinds := distinctSectionKinds(sections)
	objectivesIndex := sectionKindIndex(kinds, promptSectionKindTeamObjectives)
	memberIndex := sectionKindIndex(kinds, promptSectionKindMemberPolicy)
	doctrineIndex := sectionKindIndex(kinds, promptSectionKindSharedDoctrine)
	if objectivesIndex < 0 {
		t.Fatalf("objectives section missing from %v", kinds)
	}
	if doctrineIndex >= 0 && doctrineIndex > objectivesIndex {
		t.Fatalf("objectives precede the shared doctrine: %v", kinds)
	}
	if memberIndex >= 0 && memberIndex < objectivesIndex {
		t.Fatalf("objectives follow a member section: %v", kinds)
	}
}

// A provider failure is required-context failure: it must be visible, not
// silently dropped, so a member cannot read "unreadable" as "no obligations".
func TestTeamObjectivesProviderErrorIsVisible(t *testing.T) {
	ctx, _, builder := newTeamObjectivesFixture(t)
	builder.SetTeamObjectiveProvider(&stubTeamObjectives{err: errors.New("authority offline")})

	prompt, err := builder.Build(ctx, PromptBuildRequest{TeamID: "team-a", AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("Build should tolerate an objectives provider error: %v", err)
	}
	for _, want := range []string{
		promptHeading(promptSectionKindTeamObjectives),
		"could not be read",
		"unknown, not empty",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("visible failure missing %q:\n%s", want, prompt)
		}
	}
}
