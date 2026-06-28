package authoring_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

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

// fakeAnchor / fakeReading / fakeRefs are autofill seams whose behavior the test
// dials between "fills" and "errors" to exercise graceful degradation.
type fakeAnchor struct {
	out string
	err error
}

func (f fakeAnchor) Anchor(_ context.Context, _, _ string) (string, error) {
	return f.out, f.err
}

type fakeReading struct {
	out string
	err error
}

func (f fakeReading) RequiredReading(_ context.Context, _ string) (string, error) {
	return f.out, f.err
}

type fakeRefs struct {
	out string
	err error
}

func (f fakeRefs) References(_ context.Context, _, _ string) (string, error) {
	return f.out, f.err
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
		Writer:       &fakePlanWriter{},
		Anchor:       fakeAnchor{out: "captured anchor"},
		RequiredRead: fakeReading{out: "docs/TESTING.md"},
		References:   fakeRefs{out: "[CODE: internal/widget/core.go]"},
	})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	updated, results, _, err := svc.Autofill(ctx, sess.ID, nil) // nil => all sources
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, r := range results {
		require.True(t, r.Filled, "source %s should fill", r.Source)
		require.False(t, r.Degraded)
	}
	// The autofilled sections carry content + the autofilled marker.
	for _, key := range []authoring.SectionKey{
		authoring.SectionRegressionAnchor, authoring.SectionReferences,
	} {
		idx := -1
		for i := range updated.Sections {
			if updated.Sections[i].Key == key {
				idx = i
			}
		}
		require.GreaterOrEqual(t, idx, 0)
		require.True(t, updated.Sections[idx].Filled)
		require.True(t, updated.Sections[idx].Autofilled)
		require.NotEmpty(t, updated.Sections[idx].Content)
	}
}

func TestLegacyRequiredReadingAutofillIsExplicitOnly(t *testing.T) {
	svc := newService(t, authoring.Deps{
		Writer:       &fakePlanWriter{},
		RequiredRead: fakeReading{out: "docs/TESTING.md"},
	})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	updated, results, _, err := svc.Autofill(ctx, sess.ID, []authoring.AutofillSource{authoring.AutofillRequiredReading})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Degraded)
	require.False(t, results[0].Filled)
	require.Contains(t, results[0].Detail, "section not present")

	_, _, _, err = svc.SubmitSection(ctx, updated.ID, authoring.SectionRequiredReading, "prompt-manager skill read plan-skill-discovery")
	require.Error(t, err)
}

// [REQ:PM-AUTHOR-002]
func TestAutofilledReferencesSurviveFinalize(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{
		Writer:     writer,
		References: fakeRefs{out: "[CODE: scenarios/plan-manager/api/internal/validation/service.go]"},
	})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Improve validation", "improve-validation", "")
	require.NoError(t, err)

	_, results, _, err := svc.Autofill(ctx, sess.ID, []authoring.AutofillSource{authoring.AutofillReferences})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Filled)

	fillMandatory(t, svc, sess.ID)
	_, _, err = svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Len(t, writer.created.References, 1)
	require.Equal(t, "scenarios/plan-manager/api/internal/validation/service.go", writer.created.References[0].Target)
}

func TestAutofillDegradesPerSourceNeverFalseFill(t *testing.T) {
	ctx := context.Background()

	// Nil anchor seam, erroring reading seam, healthy refs seam.
	svc := newService(t, authoring.Deps{
		Writer:       &fakePlanWriter{},
		Anchor:       nil, // nil seam must degrade, not panic / fabricate
		RequiredRead: fakeReading{err: errors.New("prompt-manager down")},
		References:   fakeRefs{out: "[CODE: internal/widget/core.go]"},
	})
	sess, _, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	updated, results, _, err := svc.Autofill(ctx, sess.ID, nil)
	require.NoError(t, err)
	require.Len(t, results, 2)

	byKey := map[authoring.SectionKey]authoring.AutofillResult{}
	for _, r := range results {
		byKey[r.SectionKey] = r
	}

	// Anchor: nil seam => degraded, section left unfilled (NEVER a false fill).
	anchorRes := byKey[authoring.SectionRegressionAnchor]
	require.True(t, anchorRes.Degraded)
	require.False(t, anchorRes.Filled)

	// References: healthy => filled.
	refsRes := byKey[authoring.SectionReferences]
	require.True(t, refsRes.Filled)
	require.False(t, refsRes.Degraded)

	// The degraded sections are genuinely empty in the persisted session.
	for _, key := range []authoring.SectionKey{authoring.SectionRegressionAnchor} {
		got, _, err := svc.GetSection(ctx, sess.ID, key)
		require.NoError(t, err)
		require.Empty(t, got.Content, "degraded section %s must be left for the author", key)
		require.False(t, got.Filled)
		_ = updated
	}
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

func TestCommandAnchorAutofillerUsesVerifiedGCTShape(t *testing.T) {
	runner := &recordingRunner{out: []byte(`{"status":"ready","scenario":"plan-manager","name":"plan-manager","baseline":{"name":"impl"}}`)}
	got, err := authoring.NewCommandAnchorAutofiller(runner.Run).Anchor(context.Background(), "Ignored Title", "plan-manager")
	require.NoError(t, err)
	require.Equal(t, "impl", got)
	require.Equal(t, "git-control-tower", runner.name)
	require.Equal(t, []string{"baseline", "snapshot", "status", "--scenario", "plan-manager", "--name", "plan-manager", "--json"}, runner.args)
}

func TestCommandReferenceExtractorEmitsParseableCodeRefs(t *testing.T) {
	runner := &recordingRunner{out: []byte(`{"target":"scenario:plan-manager"}`)}
	got, err := authoring.NewCommandReferenceExtractor(runner.Run).References(
		context.Background(),
		"Improve validation",
		"Touch scenarios/plan-manager/api/internal/validation/service.go and scenarios/plan-manager/api/handlers/validation/module.go.",
	)
	require.NoError(t, err)
	require.Equal(t, "code-facts", runner.name)
	require.Equal(t, []string{"facts", "describe", "scenario:plan-manager", "--include", "surfaces,parse_units", "--json"}, runner.args)
	require.Contains(t, got, "[CODE: scenarios/plan-manager/api/internal/validation/service.go]")
	require.Contains(t, got, "[CODE: scenarios/plan-manager/api/handlers/validation/module.go]")
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
	require.Contains(t, writer.created.RegressionAnchor.Commands, "git-control-tower baseline diff --scenario plan-manager --name plan-manager-hardening-readiness")
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
