package authoring_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"plan-manager/internal/authoring"
	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

// testRenderer adapts the plans-domain renderer to the authoring PlanRenderer
// seam for preview tests (the same renderer the production wiring uses).
type testRenderer struct{}

func (testRenderer) Render(p internalplans.Plan) string { return internalplans.RenderMarkdown(p) }

// TestWizardAuthoredPlanRendersComprehensive is the wizard→render golden guard:
// a plan authored entirely through the Service finalizes and renders to a
// comprehensive review artifact with the Work Posture section, the automatic
// Greenfield block, and the professional plan/phase fields. This proves the
// wizard, model, and renderer stay aligned end to end.
func TestWizardAuthoredPlanRendersComprehensive(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Comprehensive", "comprehensive", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	// A phase-native draft overrides the blob phases with the full structure.
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Contract", "Lock the model.")
	require.NoError(t, err)
	for field, content := range map[authoring.PhaseField]string{
		authoring.PhaseFieldReferences:    "[CODE: scenarios/plan-manager/api/internal/plans/render.go]",
		authoring.PhaseFieldAffectedAreas: "render.go\nparse.go",
		authoring.PhaseFieldSteps:         "Add the section\nWire the parser",
		authoring.PhaseFieldValidation:    "go test ./internal/plans ./internal/planmodel",
		authoring.PhaseFieldAcceptance:    "Rendered markdown is comprehensive.",
	} {
		_, _, _, ferr := svc.SubmitPhaseField(ctx, sess.ID, phase.ID, field, content)
		require.NoError(t, ferr)
	}
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldRelevantContext, "NO_CONTEXT: covered by global setup.")
	require.NoError(t, err)

	_, _, err = svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)

	md := internalplans.RenderMarkdown(writer.created)
	for _, want := range []string{
		"## Work Posture",
		"**This is greenfield work.**",
		"## Problem / Need",
		"## Target Outcome",
		"## Technical Approach",
		"## Validation Strategy",
		"**Ordered Steps:**",
		"**Phase Validation:**",
		"**Affected Areas:**",
	} {
		require.Contains(t, md, want, "wizard-authored plan must render %q", want)
	}
}

// TestPreviewPlanRendersWithoutPersisting covers the render-preview path: a
// complete session previews to the markdown review artifact (with the automatic
// Work Posture section and Greenfield block) without writing a plan.
func TestPreviewPlanRendersWithoutPersisting(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer, Renderer: testRenderer{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Preview me", "preview-me", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	md, step, err := svc.PreviewPlan(ctx, sess.ID)
	require.NoError(t, err)
	require.Contains(t, md, "## Work Posture")
	require.Contains(t, md, "**This is greenfield work.**")
	require.Contains(t, md, "## Problem / Need")
	require.Equal(t, "final_review", step.StepKind)
	require.Equal(t, 0, writer.calls, "preview must not persist a plan")
}

// TestPreviewUnavailableWithoutRenderer asserts preview degrades honestly when no
// renderer is wired (never a silent empty render).
func TestPreviewUnavailableWithoutRenderer(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "No renderer", "", "")
	require.NoError(t, err)
	_, _, err = svc.PreviewPlan(ctx, sess.ID)
	require.Error(t, err)
}

// TestPostureConflictConstraintsAreFlagged covers conflicting-posture constraints:
// a greenfield plan whose constraints ask for a compatibility shim is rejected so
// the rendered plan never contradicts the injected Greenfield block.
func TestPostureConflictConstraintsAreFlagged(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Conflict", "", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionConstraints, "Add a compatibility shim for the old API.")
	require.NoError(t, err)
	_ = violations

	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.Contains(t, lastViolationMessage(violations), "greenfield work posture")
}

func lastViolationMessage(violations []authoring.StructureViolation) string {
	var msgs []string
	for _, v := range violations {
		msgs = append(msgs, v.Message)
	}
	return strings.Join(msgs, " | ")
}

// TestPhaseAcceptanceEqualsValidationRejected covers the acceptance≠validation
// gate: a phase whose acceptance merely restates its validation is rejected.
func TestPhaseAcceptanceEqualsValidationRejected(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Accept eq valid", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Work", "Do work")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldNoCodeRefsReason, "NO_CODE_REFS: fixture")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldSteps, "Run the suite")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldValidation, "go test ./...")
	require.NoError(t, err)
	_, violations, _, err := svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldAcceptance, "go test ./...")
	require.NoError(t, err)
	require.Contains(t, lastViolationMessage(violations), "must not be identical to its validation")
}

// newStore returns a real SQLite-backed SessionStore (the production persistence
// path) plus a fake clock — mirroring internal/plans/plans_test.go.
func newStore(t *testing.T) (authoring.SessionStore, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(authoring.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC))
	return authoring.NewSQLiteStore(d, clk), clk
}

// fakePlanWriter records the plan it was asked to persist and returns it with an
// assigned id (mirroring the plans Service Create contract).
type fakePlanWriter struct {
	created internalplans.Plan
	calls   int
	err     error
}

func (w *fakePlanWriter) CreatePlan(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	w.calls++
	if w.err != nil {
		return internalplans.Plan{}, w.err
	}
	p.ID = "plan-finalized"
	p.Status = internalplans.PlanStatusDraft
	w.created = p
	return p, nil
}

// fakeAnchor is an AnchorIntentDeriver whose returned intent block the test
// dials. Derivation is deterministic (no dependency), so there is no error path.
type fakeAnchor struct {
	out string
}

func (f fakeAnchor) DeriveAnchorIntent(_ context.Context, _, _ string, _ planmodel.ChangeBoundary) string {
	return f.out
}

// fakeSuggester is a ReferenceSuggester whose returned candidates / error the
// test dials to exercise the reference review lifecycle and honest degradation.
type fakeSuggester struct {
	candidates []authoring.ReferenceCandidate
	err        error
	gotQuery   string
}

func (f *fakeSuggester) Suggest(_ context.Context, query string) ([]authoring.ReferenceCandidate, error) {
	f.gotQuery = query
	if f.err != nil {
		return nil, f.err
	}
	return append([]authoring.ReferenceCandidate(nil), f.candidates...), nil
}

type fakeContextDiscoverer struct {
	candidates []authoring.ContextCandidate
	err        error
	gotTitle   string
	gotConcept []string
	gotComplex string
}

func (f *fakeContextDiscoverer) DiscoverContext(_ context.Context, title string, concepts []string, complexity string) ([]authoring.ContextCandidate, error) {
	f.gotTitle = title
	f.gotConcept = append([]string(nil), concepts...)
	f.gotComplex = complexity
	if f.err != nil {
		return nil, f.err
	}
	return append([]authoring.ContextCandidate(nil), f.candidates...), nil
}

type fakeCommandValidator struct {
	results map[string]authoring.CommandReferenceResult
	err     error
	calls   []authoring.CommandReferenceRequest
}

func (f *fakeCommandValidator) ValidateCommandReference(_ context.Context, req authoring.CommandReferenceRequest) (authoring.CommandReferenceResult, error) {
	f.calls = append(f.calls, req)
	if f.err != nil {
		return authoring.CommandReferenceResult{}, f.err
	}
	if got, ok := f.results[req.CommandText]; ok {
		return got, nil
	}
	return authoring.CommandReferenceResult{Verdict: "valid", ValidationLevel: "argument_shape_validated"}, nil
}

type recordingRunner struct {
	name string
	args []string
	out  []byte
	err  error
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	return r.out, r.err
}

func newService(t *testing.T, d authoring.Deps) authoring.Service {
	t.Helper()
	store, clk := newStore(t)
	if d.Store == nil {
		d.Store = store
	}
	if d.Clock == nil {
		d.Clock = clk
	}
	return authoring.NewService(d)
}

// [REQ:PM-AUTHOR-002] The continue loop must surface the plan-wide relevant-
// context checkpoint and not silently bypass it; an explicit NO_CONTEXT reason
// (or accepting a global item) resolves it.
func TestContinueSurfacesGlobalContextCheckpoint(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{})
	sess, _, err := svc.StartSession(ctx, "Context checkpoint", "context-checkpoint", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, _, _, ready, _, step, err := svc.ContinueAuthoring(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, ready)
	require.Equal(t, "global_relevant_context", step.StepKind, "continue surfaces the global context checkpoint before finishing")

	// Skip with an explicit reason resolves the checkpoint.
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_CONTEXT: unit fixture needs no plan-wide setup.")
	require.NoError(t, err)
	_, _, _, _, _, step, err = svc.ContinueAuthoring(ctx, sess.ID)
	require.NoError(t, err)
	require.NotEqual(t, "global_relevant_context", step.StepKind, "an explicit NO_CONTEXT reason advances past the checkpoint")
}

func TestContinueGlobalContextResolvedByAcceptedItem(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{})
	sess, _, err := svc.StartSession(ctx, "Context accept", "context-accept", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, _, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:         internalplans.RelevantContextSkill,
		Label:        "Load steer",
		Reason:       "Plan-wide setup.",
		Instruction:  "Load before any phase.",
		Target:       "api-steer",
		Required:     true,
		RepeatPolicy: internalplans.RelevantContextOncePerExecution,
	})
	require.NoError(t, err)
	require.Empty(t, violations)

	_, _, _, _, _, step, err := svc.ContinueAuthoring(ctx, sess.ID)
	require.NoError(t, err)
	require.NotEqual(t, "global_relevant_context", step.StepKind, "a submitted global context item resolves the checkpoint")
}

// fillMandatory submits non-empty content to every mandatory section + the
// regression anchor so the structure gate passes. Returns the final session.
func fillMandatory(t *testing.T, svc authoring.Service, sessionID string) authoring.Session {
	t.Helper()
	ctx := context.Background()
	content := []struct {
		key authoring.SectionKey
		val string
	}{
		{authoring.SectionPurpose, "Make widgets better."},
		{authoring.SectionProblemStatement, "Widgets are unreliable today."},
		{authoring.SectionTargetOutcome, "Widgets are reliable and reviewable."},
		{authoring.SectionScope, "In: widget core."},
		{authoring.SectionTechnicalApproach, "Refactor the widget core behind a seam."},
		{authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**"},
		{authoring.SectionReferences, "NO_CODE_REFS: unit test fixture has no connected production code"},
		{authoring.SectionRegressionAnchor, "baseline captured at HEAD abc123"},
		{authoring.SectionValidationStrategy, "Run the widget unit suite and compare against the baseline."},
		{authoring.SectionDefinitionOfDone, "Tests green; baseline diff exit 0."},
		{authoring.SectionPhases, "### Phase 1 — Anchor\n- Intent: Capture baseline\n- Status: todo\n"},
	}
	var sess authoring.Session
	for _, item := range content {
		existing, _, err := svc.GetSection(ctx, sessionID, item.key)
		require.NoError(t, err)
		if strings.TrimSpace(existing.Content) != "" {
			continue
		}
		s, _, _, err := svc.SubmitSection(ctx, sessionID, item.key, item.val)
		require.NoError(t, err)
		sess = s
	}
	return sess
}

func TestStartSessionSeedsSkeletonAndPointer(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	sess, step, err := svc.StartSession(ctx, "Improve widget", "improve-widget", "")
	require.NoError(t, err)
	require.NotEmpty(t, sess.ID)
	require.Equal(t, "Improve widget", sess.Title)
	require.NotEmpty(t, sess.Sections)
	// The first mandatory section is the current pointer.
	require.Equal(t, authoring.SectionPurpose, sess.CurrentSectionKey)
	require.Equal(t, "purpose", step.StepKind)
	require.Equal(t, []string{"author", "next", sess.ID}, step.NextActions[0].Argv)

	// Empty title is rejected.
	_, _, err = svc.StartSession(ctx, "  ", "", "")
	require.Error(t, err)
}

// [REQ:PM-AUTHOR-001]
func TestSessionProgressionToComplete(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// Next points at the first unfilled mandatory section, not complete.
	sec, step, complete, err := svc.Next(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, complete)
	require.Equal(t, authoring.SectionPurpose, sec.Key)
	require.Equal(t, []string{"author", "section-submit", sess.ID, "--section", "purpose", "--content", "<one concise purpose paragraph>"}, step.NextActions[0].Argv)

	// Submitting purpose advances the pointer past it.
	updated, violations, step, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.NotEqual(t, authoring.SectionPurpose, updated.CurrentSectionKey)
	require.Equal(t, string(updated.CurrentSectionKey), step.NextActions[0].Argv[4])

	// Fill every mandatory section => Next reports complete.
	fillMandatory(t, svc, sess.ID)
	_, _, complete, err = svc.Next(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, complete)
}

func TestGetSectionAndPersistenceAcrossCalls(t *testing.T) {
	// A fresh service built over the SAME store proves a session survives across
	// separate CLI invocations (each is a new Service instance over the store).
	store, clk := newStore(t)
	ctx := context.Background()

	svc1 := authoring.NewService(authoring.Deps{Store: store, Writer: &fakePlanWriter{}, Clock: clk})
	sess, _, err := svc1.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, _, _, err = svc1.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)

	svc2 := authoring.NewService(authoring.Deps{Store: store, Writer: &fakePlanWriter{}, Clock: clk})
	got, _, err := svc2.GetSection(ctx, sess.ID, authoring.SectionPurpose)
	require.NoError(t, err)
	require.Equal(t, "A purpose.", got.Content)
	require.True(t, got.Filled)

	// Unknown section / session are typed not-founds.
	_, _, err = svc2.GetSection(ctx, sess.ID, "nope")
	require.Error(t, err)
	_, _, err = svc2.GetSection(ctx, "no-such-session", authoring.SectionPurpose)
	require.Error(t, err)
}

// [REQ:PM-AUTHOR-001]
func TestStructureGateRejectsEmptyMandatoryAndAnchor(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// A brand-new session fails the gate: mandatory sections + the anchor are empty.
	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.NotEmpty(t, violations)

	// The regression anchor is called out specifically.
	var anchorFlagged bool
	for _, v := range violations {
		if v.SectionKey == authoring.SectionRegressionAnchor {
			anchorFlagged = true
		}
	}
	require.True(t, anchorFlagged, "empty regression anchor is a distinct violation")

	// Filling everything mandatory satisfies the gate.
	fillMandatory(t, svc, sess.ID)
	valid, violations, _, err = svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid)
	require.Empty(t, violations)
}

func TestSubmitSectionReportsPerSectionViolations(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// Submitting empty content to a mandatory section reports a violation.
	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "   ")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Equal(t, authoring.SectionPurpose, violations[0].SectionKey)

	// Non-empty content passes.
	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Real purpose.")
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestSubmitSectionValidatesCurrentCLIReferences(t *testing.T) {
	commands := &fakeCommandValidator{results: map[string]authoring.CommandReferenceResult{
		"vrooli scenario tost cli-health": {
			Verdict:     "invalid",
			Issues:      []authoring.CommandIssue{{Code: "command_not_found", Message: "unknown command path"}},
			Suggestions: []string{"vrooli scenario test"},
			Guidance:    []string{"fix this to a current command or mark it cli[future] if intentional"},
		},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Commands: commands})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Run `cli:vrooli scenario tost cli-health`.")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Equal(t, authoring.SectionPurpose, violations[0].SectionKey)
	require.Contains(t, violations[0].Message, "command_not_found")
	require.Contains(t, violations[0].Message, "vrooli scenario test")
	require.Len(t, commands.calls, 1)
	require.Equal(t, "vrooli scenario tost cli-health", commands.calls[0].CommandText)
}

func TestCommandReferenceQualifiersSkipAuthoringCurrentValidation(t *testing.T) {
	commands := &fakeCommandValidator{results: map[string]authoring.CommandReferenceResult{
		"vrooli scenario someday cli-health": {Verdict: "invalid"},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Commands: commands})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Planned: `cli[future]:vrooli scenario someday cli-health`.")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Empty(t, commands.calls, "future references are not current-command requirements")
}

func TestFinalizeRejectsInvalidCLIReferences(t *testing.T) {
	writer := &fakePlanWriter{}
	commands := &fakeCommandValidator{results: map[string]authoring.CommandReferenceResult{
		"vrooli scenario tost cli-health": {
			Verdict: "invalid",
			Issues:  []authoring.CommandIssue{{Code: "command_not_found", Message: "unknown command path"}},
		},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, Commands: commands})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Run `cli:vrooli scenario tost cli-health`.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionProblemStatement, "A problem.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTargetOutcome, "An outcome.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionScope, "In: widget core.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "An approach.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "NO_CODE_REFS: command validation fixture exercises CLI references only")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "baseline captured at HEAD abc123")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "Tests green.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPhases, "### Phase 1 — Anchor\n- Intent: Capture baseline\n- Status: todo\n")
	require.NoError(t, err)

	_, _, err = svc.Finalize(ctx, sess.ID)
	require.Error(t, err)
	var gate authoring.ErrStructureGate
	require.True(t, errors.As(err, &gate))
	require.Len(t, gate.Violations, 1)
	require.Contains(t, gate.Violations[0].Message, "command_not_found")
	require.Equal(t, 0, writer.calls)
}

// [REQ:PM-AUTHOR-002]
func TestAutofillFillsWhenSeamsHealthy(t *testing.T) {
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Anchor: fakeAnchor{out: "captured anchor"},
	})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	updated, results, _, err := svc.Autofill(ctx, sess.ID, nil) // nil => all sources (regression_anchor only)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Filled, "regression anchor should fill")
	require.False(t, results[0].Degraded)
	// The autofilled section carries content + the autofilled marker.
	idx := -1
	for i := range updated.Sections {
		if updated.Sections[i].Key == authoring.SectionRegressionAnchor {
			idx = i
		}
	}
	require.GreaterOrEqual(t, idx, 0)
	require.True(t, updated.Sections[idx].Filled)
	require.True(t, updated.Sections[idx].Autofilled)
	require.NotEmpty(t, updated.Sections[idx].Content)
}

// TestSuggestReferencesIsReviewedNotAutofilled proves a suggestion never writes
// the references section: only an accepted candidate does, and the references
// gate (in nextGuidedStep/sessionViolations) stays unsatisfied until then.
func TestSuggestReferencesIsReviewedNotAutofilled(t *testing.T) {
	suggester := &fakeSuggester{candidates: []authoring.ReferenceCandidate{
		{Reference: planmodel.Reference{Kind: planmodel.ReferenceCode, Target: "internal/widget/core.go"}, Source: "code-symbol", Confidence: 0.9},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Suggester: suggester})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, candidates, _, err := svc.SuggestReferences(ctx, sess.ID)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, authoring.ReferenceCandidatePending, candidates[0].Status)

	// Raw suggestion must NOT have written the references section.
	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	require.Empty(t, got.Content, "a raw suggestion must not satisfy the references gate")

	// Accepting finalizes the locator into the section.
	_, accepted, violations, _, err := svc.AcceptReferenceCandidate(ctx, sess.ID, candidates[0].ID, nil)
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, authoring.ReferenceCandidateAccepted, accepted.Status)
	got, _, err = svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	require.Contains(t, got.Content, "[CODE: internal/widget/core.go]")
}

// TestSuggestedReferencesSurviveFinalize proves an accepted suggestion parses
// into the structured references[] at finalize.
func TestSuggestedReferencesSurviveFinalize(t *testing.T) {
	writer := &fakePlanWriter{}
	suggester := &fakeSuggester{candidates: []authoring.ReferenceCandidate{
		{Reference: planmodel.Reference{Kind: planmodel.ReferenceCode, Target: "scenarios/plan-manager/api/internal/validation/service.go"}},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, Suggester: suggester})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve validation", "improve-validation", "")
	require.NoError(t, err)

	_, candidates, _, err := svc.SuggestReferences(ctx, sess.ID)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	_, _, violations, _, err := svc.AcceptReferenceCandidate(ctx, sess.ID, candidates[0].ID, nil)
	require.NoError(t, err)
	require.Empty(t, violations)

	fillMandatory(t, svc, sess.ID)
	_, _, err = svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Len(t, writer.created.References, 1)
	require.Equal(t, "scenarios/plan-manager/api/internal/validation/service.go", writer.created.References[0].Target)
}

// TestSuggestReferencesDegradesHonestly proves a nil/erroring suggester yields
// no candidates and never fabricates a reference.
func TestSuggestReferencesDegradesHonestly(t *testing.T) {
	ctx := context.Background()
	// Nil suggester.
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, candidates, step, err := svc.SuggestReferences(ctx, sess.ID)
	require.NoError(t, err)
	require.Empty(t, candidates)
	require.Equal(t, "reference_candidates", step.StepKind)
	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	require.Empty(t, got.Content)

	// Erroring suggester degrades the same way.
	svc2 := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Suggester: &fakeSuggester{err: errors.New("search-hub down")}})
	sess2, _, err := svc2.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, candidates2, _, err := svc2.SuggestReferences(ctx, sess2.ID)
	require.NoError(t, err)
	require.Empty(t, candidates2)
}

// TestRejectReferenceCandidateNeverEntersSection proves a rejected suggestion
// stays an audit trail and never writes the references section.
func TestRejectReferenceCandidateNeverEntersSection(t *testing.T) {
	suggester := &fakeSuggester{candidates: []authoring.ReferenceCandidate{
		{Reference: planmodel.Reference{Kind: planmodel.ReferenceCode, Target: "internal/widget/core.go"}},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Suggester: suggester})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, candidates, _, err := svc.SuggestReferences(ctx, sess.ID)
	require.NoError(t, err)
	require.Len(t, candidates, 1)

	_, rejected, _, err := svc.RejectReferenceCandidate(ctx, sess.ID, candidates[0].ID, "unrelated subsystem")
	require.NoError(t, err)
	require.Equal(t, authoring.ReferenceCandidateRejected, rejected.Status)
	require.Equal(t, "unrelated subsystem", rejected.RejectionReason)

	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	require.Empty(t, got.Content)

	// A rejected candidate cannot be accepted afterwards.
	_, _, _, _, err = svc.AcceptReferenceCandidate(ctx, sess.ID, candidates[0].ID, nil)
	require.Error(t, err)
}

// TestAcceptReferenceCandidateRejectsKindMismatch proves an inline edit that
// mislabels a docs path as [CODE:] is rejected before it enters the section.
func TestAcceptReferenceCandidateRejectsKindMismatch(t *testing.T) {
	suggester := &fakeSuggester{candidates: []authoring.ReferenceCandidate{
		{Reference: planmodel.Reference{Kind: planmodel.ReferenceCode, Target: "internal/widget/core.go"}},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Suggester: suggester})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, candidates, _, err := svc.SuggestReferences(ctx, sess.ID)
	require.NoError(t, err)

	edit := &planmodel.Reference{Kind: planmodel.ReferenceCode, Target: "docs/concepts/PLAN-MODEL.md"}
	_, _, violations, _, err := svc.AcceptReferenceCandidate(ctx, sess.ID, candidates[0].ID, edit)
	require.NoError(t, err)
	require.NotEmpty(t, violations)
	require.Contains(t, violations[0].Message, "[DOC:]")

	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	require.Empty(t, got.Content, "a kind-mismatched locator must not enter the references section")
}

func TestAutofillDegradesPerSourceNeverFalseFill(t *testing.T) {
	ctx := context.Background()

	// Nil anchor seam must degrade, not panic / fabricate.
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Anchor: nil,
	})
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, results, _, err := svc.Autofill(ctx, sess.ID, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Anchor: nil seam => degraded, section left unfilled (NEVER a false fill).
	anchorRes := results[0]
	require.Equal(t, authoring.SectionRegressionAnchor, anchorRes.SectionKey)
	require.True(t, anchorRes.Degraded)
	require.False(t, anchorRes.Filled)

	// The degraded section is genuinely empty in the persisted session.
	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionRegressionAnchor)
	require.NoError(t, err)
	require.Empty(t, got.Content, "degraded section must be left for the author")
	require.False(t, got.Filled)
}

func TestAutofillEmptyOutputDegrades(t *testing.T) {
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Anchor: fakeAnchor{out: "   "}, // whitespace-only is not a real fill
	})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, results, _, err := svc.Autofill(ctx, sess.ID, []authoring.AutofillSource{authoring.AutofillRegressionAnchor})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Degraded)
	require.False(t, results[0].Filled)
}

func TestPhaseNativeAuthoringValidatesAndFinalizesStructuredPhase(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve authoring", "improve-authoring", "")
	require.NoError(t, err)

	updated, phase, violations, step, err := svc.AddPhase(ctx, sess.ID, "Authoring contract", "Add phase-native RPCs.")
	require.NoError(t, err)
	require.NotEmpty(t, phase.ID)
	require.Equal(t, 1, phase.Order)
	require.NotEmpty(t, violations, "acceptance + refs are still required")
	require.Len(t, updated.PhaseDrafts, 1)
	require.Equal(t, []string{"author", "phase-submit", sess.ID, phase.ID, "--field", "references", "--content", "[CODE: path/to/file.go]"}, step.NextActions[0].Argv)

	updated, violations, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldReferences, "[CODE: scenarios/plan-manager/api/internal/authoring/service.go]")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "acceptance is still missing")
	require.Len(t, updated.PhaseDrafts[0].References, 1)

	updated, violations, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldRequiredReading, "docs/concepts/PLAN-MODEL.md\nprompt-manager skill read plan-skill-discovery")
	require.NoError(t, err)
	require.NotEmpty(t, violations)
	require.Len(t, updated.PhaseDrafts[0].RequiredReading, 2)

	updated, item, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:         internalplans.RelevantContextCommand,
		Label:        "Recall prior work",
		Reason:       "Prior records describe the relevant-context slices already completed.",
		Instruction:  "Run recall before implementation.",
		Command:      "search-hub query plan-manager relevant context --type record",
		Required:     true,
		RepeatPolicy: internalplans.RelevantContextOncePerExecution,
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextScopeGlobal, item.Scope)
	require.Len(t, updated.RelevantContext, 1)

	updated, phaseItem, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, phase.ID, internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextDoc,
		Label:       "Plan model docs",
		Reason:      "The phase changes plan model semantics.",
		Instruction: "Read before editing authoring finalization.",
		Target:      "scenarios/plan-manager/docs/concepts/PLAN-MODEL.md",
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextScopePhase, phaseItem.Scope)
	require.Len(t, updated.PhaseDrafts[0].RelevantContext, 1)

	_, violations, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldSteps, "Add the proto RPCs\nWire the handler\nWire the CLI group")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "validation + acceptance still missing")

	_, violations, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldValidation, "go test ./api/handlers/authoring ./cli/...")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "acceptance still missing")

	updated, violations, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldAcceptance, "API and CLI tests cover the new phase-native flow.")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Empty(t, updated.CurrentPhaseID)

	_, step, complete, err := svc.NextPhase(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, complete)
	require.Equal(t, "final_review", step.StepKind)

	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Make authoring smaller-model friendly.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionProblemStatement, "Authoring is too heavy for small models.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTargetOutcome, "A small model can author a full plan step by step.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionScope, "In: plan-manager authoring.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "Phase-native guided steps over one big blob.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE: scenarios/plan-manager/api/internal/authoring/service.go]")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "baseline captured")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run authoring + CLI suites, then the scenario test.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "Scenario tests pass.")
	require.NoError(t, err)

	plan, _, err := svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, "plan-finalized", plan.ID)
	require.Len(t, writer.created.Phases, 1)
	require.Equal(t, "Authoring contract", writer.created.Phases[0].Title)
	require.Equal(t, phase.ID, writer.created.Phases[0].ID)
	require.Len(t, writer.created.Phases[0].References, 1)
	require.Len(t, writer.created.Phases[0].RequiredReading, 2)
	require.Len(t, writer.created.RelevantContext, 1)
	require.Len(t, writer.created.Phases[0].RelevantContext, 3, "explicit phase context plus migrated required-reading items")
	require.Equal(t, internalplans.RelevantContextDoc, writer.created.Phases[0].RelevantContext[0].Kind)
	require.Equal(t, phase.ID, writer.created.Phases[0].RelevantContext[0].PhaseID)
	require.Equal(t, internalplans.RelevantContextSourceMigrated, writer.created.Phases[0].RelevantContext[1].Source)
	require.Equal(t, phase.ID, writer.created.Phases[0].RelevantContext[1].PhaseID)
}

func TestContextDiscoveryCandidateLifecycle(t *testing.T) {
	writer := &fakePlanWriter{}
	discovery := &fakeContextDiscoverer{candidates: []authoring.ContextCandidate{{
		ID:      "cand-global",
		Concept: "plan-manager relevant context",
		Source:  "search-hub-recall",
		Item: internalplans.RelevantContextItem{
			Kind:         internalplans.RelevantContextSearch,
			Label:        "Recall context records",
			Reason:       "Prior records explain completed relevant-context slices.",
			Instruction:  "Run recall before choosing the next slice.",
			Command:      "search-hub query plan-manager relevant context --type record,skill,doc",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextOncePerExecution,
			Source:       internalplans.RelevantContextSourceDiscovered,
		},
	}}}
	svc := newService(t, authoring.Deps{Writer: writer, Context: discovery})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve context", "improve-context", "")
	require.NoError(t, err)

	updated, candidates, step, err := svc.DiscoverContextCandidates(ctx, sess.ID, []string{"plan-manager relevant context"}, "architectural")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Len(t, updated.ContextCandidates, 1)
	require.Equal(t, "context_discovery", step.StepKind)
	require.Equal(t, []string{"plan-manager relevant context"}, discovery.gotConcept)
	require.Equal(t, "architectural", discovery.gotComplex)

	updated, candidate, item, violations, _, err := svc.AcceptContextCandidate(ctx, sess.ID, "cand-global", "")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, authoring.ContextCandidateAccepted, candidate.Status)
	require.Equal(t, internalplans.RelevantContextScopeGlobal, item.Scope)
	require.Len(t, updated.RelevantContext, 1)

	updated, rejected, _, err := svc.RejectContextCandidate(ctx, sess.ID, "cand-global", "duplicate after accept")
	require.NoError(t, err)
	require.Equal(t, authoring.ContextCandidateRejected, rejected.Status)
	require.Equal(t, "duplicate after accept", updated.ContextCandidates[0].RejectionReason)
}

func TestContextDiscoveryDegradesWhenSeamUnavailable(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve context", "", "")
	require.NoError(t, err)

	updated, candidates, _, err := svc.DiscoverContextCandidates(ctx, sess.ID, []string{"context discovery"}, "")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.True(t, candidates[0].Degraded)
	require.Equal(t, internalplans.RelevantContextStatusDegraded, candidates[0].Item.Status)
	require.Len(t, updated.ContextCandidates, 1)
}

func TestAcceptContextCandidateAssignsPhaseScope(t *testing.T) {
	discovery := &fakeContextDiscoverer{candidates: []authoring.ContextCandidate{{
		ID:      "cand-phase",
		Concept: "phase context",
		Source:  "prompt-manager-actions",
		Item: internalplans.RelevantContextItem{
			Kind:         internalplans.RelevantContextCommand,
			Label:        "Discover actions",
			Reason:       "This phase needs current operational commands.",
			Instruction:  "Run discovery before editing.",
			Command:      "prompt-manager discover phase context --type all",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextOncePerExecution,
			Source:       internalplans.RelevantContextSourceDiscovered,
		},
	}}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Context: discovery})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve context", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Phase context", "Wire candidate assignment.")
	require.NoError(t, err)
	_, _, _, err = svc.DiscoverContextCandidates(ctx, sess.ID, []string{"phase context"}, "")
	require.NoError(t, err)

	updated, _, item, violations, _, err := svc.AcceptContextCandidate(ctx, sess.ID, "cand-phase", phase.ID)
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextScopePhase, item.Scope)
	require.Equal(t, phase.ID, item.PhaseID)
	require.Equal(t, internalplans.RelevantContextPhaseEntry, item.RepeatPolicy)
	require.Len(t, updated.PhaseDrafts[0].RelevantContext, 1)
}

func TestReferenceGateRequiresReferenceOrExplicitReason(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionProblemStatement, "A problem.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTargetOutcome, "An outcome.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionScope, "In scope.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "An approach.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "anchor")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "done")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPhases, "### Phase 1 — Work\n- Intent: Work\n- Acceptance: Done\n")
	require.NoError(t, err)

	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.Contains(t, violations[0].Message, "references must include")

	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "NO_CODE_REFS: docs-only plan")
	require.NoError(t, err)
	valid, violations, _, err = svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid)
	require.Empty(t, violations)
}

func TestPhaseContextGateRequiresContextOrExplicitNoContextReason(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionProblemStatement, "A problem.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTargetOutcome, "An outcome.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionScope, "In scope.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "An approach.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "NO_CODE_REFS: fixture")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "anchor")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "done")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Implement", "Change code")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldNoCodeRefsReason, "NO_CODE_REFS: fixture")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldSteps, "Do the thing")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldValidation, "go test ./...")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldAcceptance, "Tests pass.")
	require.NoError(t, err)

	valid, violations, step, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.Contains(t, violations[len(violations)-1].Message, "relevant_context")
	require.Equal(t, "validation_recovery", step.StepKind)

	_, violations, step, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldRelevantContext, "NO_CONTEXT: no additional setup")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, "phase_review", step.StepKind)

	valid, violations, _, err = svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid)
	require.Empty(t, violations)
}

// TestAnchorIntentDerivesTypedIntentNoSnapshot proves authoring derives the typed
// anchor INTENT deterministically — no git-control-tower call, no snapshot — and
// that the derived block parses into typed regression-anchor fields rather than
// degrading to legacy prose.
func TestAnchorIntentDerivesTypedIntentNoSnapshot(t *testing.T) {
	ctx := context.Background()
	// No CommandRunner is involved at all; the default deriver is pure. The anchor
	// is boundary-native: affected scenarios + commands derive from the boundary,
	// with no hand-authored <scenario> placeholder.
	boundary := planmodel.ChangeBoundary{AcceptanceAllow: []string{"scenarios/plan-manager/**", "packages/proto/**"}}
	got := authoring.DefaultAnchorIntentDeriver().DeriveAnchorIntent(ctx, "Improve validation", "improve-validation", boundary)
	require.Contains(t, got, "Strategy: "+planmodel.AnchorStrategyChangeBoundary)
	require.Contains(t, got, "Baseline name: improve-validation-baseline")
	require.NotContains(t, got, "<scenario>", "boundary-native intent must not carry a scenario placeholder")
	require.Contains(t, got, "git-control-tower baseline diff --scenario plan-manager --name improve-validation-baseline")

	anchor := planmodel.ParseRegressionAnchorBlock(got)
	require.Equal(t, planmodel.AnchorStrategyChangeBoundary, anchor.Strategy)
	require.Equal(t, "improve-validation-baseline", anchor.BaselineName)
}

// TestAnchorAutofillDerivesIntentEndToEnd proves the autofill regression_anchor
// source fills the section with the derived typed intent via the live default
// deriver (no git-control-tower).
func TestAnchorAutofillDerivesIntentEndToEnd(t *testing.T) {
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Anchor: authoring.DefaultAnchorIntentDeriver(),
	})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve validation", "improve-validation", "")
	require.NoError(t, err)

	_, results, _, err := svc.Autofill(ctx, sess.ID, []authoring.AutofillSource{authoring.AutofillRegressionAnchor})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Filled)
	require.False(t, results[0].Degraded)

	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionRegressionAnchor)
	require.NoError(t, err)
	require.Contains(t, got.Content, "Baseline name: improve-validation-baseline")
}

func TestCommandReferenceSuggesterRoutesHitsByLocatorShape(t *testing.T) {
	// A search-hub QueryResponse (protojson, camelCase) mixing locator-shaped hits
	// (code/doc/req) with a non-locator hit that must be dropped.
	resp := `{
	  "ranked": [
	    {"providerId":"code-symbol","type":"code","path":"scenarios/plan-manager/api/internal/validation/service.go","score":0.91},
	    {"providerId":"docs","type":"doc","path":"docs/concepts/PLAN-MODEL.md","score":0.7},
	    {"providerId":"requirements","type":"req","id":"PM-AUTHOR-002","score":0.6},
	    {"providerId":"records","type":"record","title":"some prior record","path":"rec-123","score":0.5}
	  ]
	}`
	runner := &recordingRunner{out: []byte(resp)}
	got, err := authoring.NewCommandReferenceSuggester(runner.Run).Suggest(context.Background(), "improve validation references")
	require.NoError(t, err)
	require.Equal(t, "search-hub", runner.name)
	require.Equal(t, []string{"query", "improve validation references", "--json"}, runner.args)
	require.Len(t, got, 3, "only the three locator-shaped hits are kept; the record hit is dropped")

	byTarget := map[string]authoring.ReferenceCandidate{}
	for _, c := range got {
		byTarget[c.Reference.Target] = c
	}
	require.Equal(t, planmodel.ReferenceCode, byTarget["scenarios/plan-manager/api/internal/validation/service.go"].Reference.Kind)
	require.Equal(t, planmodel.ReferenceDoc, byTarget["docs/concepts/PLAN-MODEL.md"].Reference.Kind)
	require.Equal(t, planmodel.ReferenceReq, byTarget["PM-AUTHOR-002"].Reference.Kind)
	require.Equal(t, "code-symbol", byTarget["scenarios/plan-manager/api/internal/validation/service.go"].Source)
}

func TestCommandReferenceSuggesterDegradesOnBadJSON(t *testing.T) {
	runner := &recordingRunner{out: []byte("not json")}
	got, err := authoring.NewCommandReferenceSuggester(runner.Run).Suggest(context.Background(), "anything")
	require.NoError(t, err)
	require.Empty(t, got, "an unparseable response degrades to no candidates, never a fabricated reference")
}

func TestFinalizeWritesThroughWriterWhenStructureValid(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "improve-widget", "")
	require.NoError(t, err)

	// Add a references section so the plan carries a parsed reference.
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE: internal/widget/core.go]")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	plan, _, err := svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, 1, writer.calls)
	require.Equal(t, "plan-finalized", plan.ID)

	// The session's prose mapped through to the plan.
	require.Equal(t, "Make widgets better.", writer.created.Purpose)
	require.Equal(t, "Improve widget", writer.created.Title)
	require.NotEmpty(t, writer.created.Phases, "phases section parsed into structured phases")
	require.Equal(t, "Anchor", writer.created.Phases[0].Title)
	require.NotEmpty(t, writer.created.References, "references section parsed into structured references")
	require.Equal(t, "internal/widget/core.go", writer.created.References[0].Target)
	require.NotEmpty(t, writer.created.RegressionAnchor.BaselineName, "captured anchor carried forward")

	// The session is marked finalized + linked to the plan.
	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionPurpose)
	require.NoError(t, err)
	require.NotEmpty(t, got.Content)
}

func TestFinalizeParsesStructuredRegressionAnchor(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Harden plan-manager", "harden-plan-manager", "")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, strings.Join([]string{
		"- Strategy: scenario_baseline",
		"- Scenario baseline: `plan-manager` (name `plan-manager-hardening-readiness`)",
		"- HEAD sha: `abc123`",
	}, "\n"))
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, _, err = svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, "scenario_baseline", writer.created.RegressionAnchor.Strategy)
	require.Equal(t, "plan-manager", writer.created.RegressionAnchor.Scenario)
	require.Equal(t, "plan-manager-hardening-readiness", writer.created.RegressionAnchor.BaselineName)
	require.Contains(t, writer.created.RegressionAnchor.Commands, "git-control-tower baseline diff --scenario plan-manager --name plan-manager-hardening-readiness --wait")
	require.False(t, writer.created.RegressionAnchor.Unavailable)
}

func TestFinalizeRejectsMalformedAuthoredReferences(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE:]")
	require.NoError(t, err)

	_, _, err = svc.Finalize(ctx, sess.ID)
	require.Error(t, err)
	var markup authoring.ErrAuthoredMarkup
	require.True(t, errors.As(err, &markup), "malformed reference markup is a typed authoring error")
	require.Equal(t, authoring.SectionReferences, markup.SectionKey)
	require.Equal(t, 0, writer.calls, "no plan written after a lossy parse failure")
}

func TestFinalizeRejectsNonParseablePhaseMarkup(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Make widgets better.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionProblemStatement, "Widgets are unreliable.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTargetOutcome, "Widgets are reliable.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionScope, "In: widget core.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "Refactor behind a seam.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "NO_CODE_REFS: malformed phase markup fixture")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "baseline captured at HEAD abc123")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the widget suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "Tests green.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPhases, "Phase 1 - missing markdown heading")
	require.NoError(t, err)

	_, _, err = svc.Finalize(ctx, sess.ID)
	require.Error(t, err)
	var markup authoring.ErrAuthoredMarkup
	require.True(t, errors.As(err, &markup), "non-empty phase markup that parses to zero phases is rejected")
	require.Equal(t, authoring.SectionPhases, markup.SectionKey)
	require.Equal(t, 0, writer.calls)
}

func TestFinalizeRejectsWhenStructureInvalid(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// Fill only purpose — the gate still has violations.
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)

	_, _, err = svc.Finalize(ctx, sess.ID)
	require.Error(t, err)
	var gate authoring.ErrStructureGate
	require.True(t, errors.As(err, &gate), "structure gate failure is typed")
	require.NotEmpty(t, gate.Violations)
	require.Equal(t, 0, writer.calls, "no plan written when the gate fails")
}

// Sanity: a missing session id surfaces as a typed not-found from every read.
func TestUnknownSessionIsNotFound(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	_, _, err := svc.GetSection(ctx, "nope", authoring.SectionPurpose)
	var notFound authoring.ErrSessionNotFound
	require.True(t, errors.As(err, &notFound))

	_, _, _, err = svc.Next(ctx, "nope")
	require.Error(t, err)
	_, _, _, err = svc.ValidateStructure(ctx, "nope")
	require.Error(t, err)
	_ = sql.ErrNoRows
}

// brownfieldPosture is a PosturePreparer that stamps brownfield, standing in for
// the production resolver reading a pilot/production scenario's maturity.
type brownfieldPosture struct{}

func (brownfieldPosture) PreparePosture(_ context.Context, p internalplans.Plan) internalplans.Plan {
	p.WorkPosture = internalplans.WorkPostureBrownfield
	p.WorkPostureSource = internalplans.WorkPostureSourceServiceMaturity
	p.WorkPostureDetail = "Scenario is in production; preserve external contracts."
	return p
}

// TestPreviewAppliesPostureSeam proves preview uses the same posture derivation as
// finalize/render: with a brownfield-resolving seam, the preview shows the
// brownfield block, not the default greenfield block (the prior parity bug).
func TestPreviewAppliesPostureSeam(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Renderer: testRenderer{}, Posture: brownfieldPosture{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Preview posture", "preview-posture", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	md, _, err := svc.PreviewPlan(ctx, sess.ID)
	require.NoError(t, err)
	require.Contains(t, md, "deployed or limited-live", "brownfield posture block must appear in preview")
	require.NotContains(t, md, "**This is greenfield work.**", "preview must not show greenfield when the scenario resolves brownfield")
}

// TestPreviewShowsChangeBoundary proves preview/finalize parity for the boundary:
// the preview render uses the same boundary path as the persisted render, so the
// Change Boundary section and its allow globs appear before finalize.
func TestPreviewShowsChangeBoundary(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Renderer: testRenderer{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Preview boundary", "preview-boundary", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	md, _, err := svc.PreviewPlan(ctx, sess.ID)
	require.NoError(t, err)
	require.Contains(t, md, "## Change Boundary", "preview must render the change boundary")
	require.Contains(t, md, "scenarios/plan-manager/**", "preview must show the authored allow glob")
}

// TestUpdateAndRemoveRelevantContextItem covers the accepted-context recovery
// path: a bad accepted item is corrected (update) or deleted (remove) before
// finalize without dropping the whole session.
func TestUpdateAndRemoveRelevantContextItem(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Context edit", "", "")
	require.NoError(t, err)

	_, saved, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind: internalplans.RelevantContextNote, Label: "bad note", Instruction: "do X",
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.NotEmpty(t, saved.ID)

	_, got, vios, _, err := svc.UpdateRelevantContextItem(ctx, sess.ID, "", saved.ID, internalplans.RelevantContextItem{
		Kind: internalplans.RelevantContextNote, Label: "fixed note", Instruction: "do Y",
	})
	require.NoError(t, err)
	require.Empty(t, vios)
	require.Equal(t, saved.ID, got.ID, "update preserves the item id")
	require.Equal(t, "fixed note", got.Label)

	items, _, err := svc.ListRelevantContext(ctx, sess.ID, "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "fixed note", items[0].Label)

	_, _, _, err = svc.RemoveRelevantContextItem(ctx, sess.ID, "", saved.ID)
	require.NoError(t, err)
	items, _, err = svc.ListRelevantContext(ctx, sess.ID, "")
	require.NoError(t, err)
	require.Empty(t, items)

	_, _, _, err = svc.RemoveRelevantContextItem(ctx, sess.ID, "", "does-not-exist")
	require.Error(t, err, "removing an unknown item id is an error")
}

// TestPhaseFreeFormContextStaysNoteNotCommand reproduces the friction-run failure:
// a free-form phase relevant_context line that looks like a skill-read command
// mixed with prose must be classified as a NOTE, never an executable command with
// bad argv — so preview/render never contains an invalid migrated command.
func TestPhaseFreeFormContextStaysNoteNotCommand(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Notes", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Build", "Do the work")
	require.NoError(t, err)

	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldRelevantContext,
		"prompt-manager skill read api-steer then also do the migration carefully")
	require.NoError(t, err)

	items, _, err := svc.ListRelevantContext(ctx, sess.ID, phase.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, internalplans.RelevantContextNote, items[0].Kind, "free-form phase context must be a note, never an executable command")
	require.Empty(t, items[0].Argv, "a note must not carry executable argv")
	require.Empty(t, items[0].Command, "a note must not carry a command")
}

// TestRegressionAnchorStepOffersIntentNotSnapshot proves the anchor guided step
// guides the agent to derive/confirm the typed INTENT and NEVER offers a
// capture-a-snapshot action at authoring time (snapshot capture moved to
// execution start).
func TestRegressionAnchorStepOffersIntentNotSnapshot(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Anchor: authoring.DefaultAnchorIntentDeriver()})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Anchor recovery", "anchor-recovery", "")
	require.NoError(t, err)

	_, step, err := svc.GetSection(ctx, sess.ID, authoring.SectionRegressionAnchor)
	require.NoError(t, err)
	var hasDerive, hasConfirm bool
	for _, a := range step.NextActions {
		require.NotEqual(t, "capture-baseline-snapshot", a.ID, "authoring must NOT offer a baseline-snapshot capture action")
		require.NotContains(t, strings.Join(a.Argv, " "), "git-control-tower baseline snapshot", "authoring must not shell git-control-tower")
		if a.ID == "autofill-anchor" {
			hasDerive = true
		}
		if a.ID == "submit-anchor-intent" {
			hasConfirm = true
			require.Contains(t, a.ContentPlaceholder, "Strategy: "+planmodel.AnchorStrategyChangeBoundary)
			require.NotContains(t, a.ContentPlaceholder, "<scenario>", "boundary-native anchor must not carry a scenario placeholder")
		}
	}
	require.True(t, hasDerive, "anchor step must offer a derive-intent action")
	require.True(t, hasConfirm, "anchor step must offer a confirm/adjust-intent action")
}

// TestBoundaryGateRequiresAllowOrOperatorOnly proves the change-boundary section
// is mandatory but satisfiable by an OPERATOR_ONLY reason, and that the finalized
// plan carries the parsed acceptance_allow / acceptance_deny.
func TestBoundaryGateRequiresAllowOrOperatorOnly(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess := fillMandatorySession(t, svc)

	// Empty boundary fails the gate.
	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "   ")
	require.NoError(t, err)
	require.NotEmpty(t, violations)
	require.Contains(t, violations[0].Message, "change boundary must declare")

	// A placeholder allow glob is rejected.
	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/<scenario>/**")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "unresolved <scenario> placeholder must be rejected")

	// A real allow list passes and finalizes with the parsed boundary.
	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary,
		"acceptance_allow:\n- scenarios/plan-manager/**\n- packages/proto/**\nacceptance_deny:\n- scenarios/swarm-manager/**")
	require.NoError(t, err)
	require.Empty(t, violations)

	plan, _, err := svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"packages/proto/**", "scenarios/plan-manager/**"}, plan.ChangeBoundary.AcceptanceAllow)
	require.Equal(t, []string{"scenarios/swarm-manager/**"}, plan.ChangeBoundary.AcceptanceDeny)
}

// TestBoundaryOperatorOnlyEscape proves an operator-only/no-code plan satisfies
// the boundary gate without an allow list.
func TestBoundaryOperatorOnlyEscape(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess := fillMandatorySession(t, svc)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary,
		"OPERATOR_ONLY: documentation-only operator decision with no editable repo paths")
	require.NoError(t, err)
	require.Empty(t, violations)

	plan, _, err := svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Empty(t, plan.ChangeBoundary.AcceptanceAllow)
	require.Contains(t, plan.ChangeBoundary.OperatorOnlyReason, "documentation-only")
}

// fillMandatorySession starts a session and fills every mandatory section EXCEPT
// the change boundary plus a complete phase and resolved global context, leaving
// the boundary for the caller to exercise.
func fillMandatorySession(t *testing.T, svc authoring.Service) authoring.Session {
	t.Helper()
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Boundary fixture", "boundary-fixture", "")
	require.NoError(t, err)
	for _, item := range []struct {
		key authoring.SectionKey
		val string
	}{
		{authoring.SectionPurpose, "Purpose."},
		{authoring.SectionProblemStatement, "Problem."},
		{authoring.SectionTargetOutcome, "Outcome."},
		{authoring.SectionScope, "In: core."},
		{authoring.SectionTechnicalApproach, "Approach."},
		{authoring.SectionReferences, "NO_CODE_REFS: boundary fixture"},
		{authoring.SectionRegressionAnchor, "anchor"},
		{authoring.SectionValidationStrategy, "Run the suite."},
		{authoring.SectionDefinitionOfDone, "Done."},
		{authoring.SectionRelevantContext, "NO_CONTEXT: boundary fixture needs no plan-wide setup."},
	} {
		_, _, _, err := svc.SubmitSection(ctx, sess.ID, item.key, item.val)
		require.NoError(t, err)
	}
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Work", "Do the work.")
	require.NoError(t, err)
	for _, f := range []struct {
		field authoring.PhaseField
		val   string
	}{
		{authoring.PhaseFieldNoCodeRefsReason, "NO_CODE_REFS: fixture"},
		{authoring.PhaseFieldSteps, "Do the thing"},
		{authoring.PhaseFieldValidation, "go test ./..."},
		{authoring.PhaseFieldAcceptance, "Tests pass."},
		{authoring.PhaseFieldRelevantContext, "NO_CONTEXT: phase needs no extra setup."},
	} {
		_, _, _, err := svc.SubmitPhaseField(ctx, sess.ID, phase.ID, f.field, f.val)
		require.NoError(t, err)
	}
	s, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	return s
}
