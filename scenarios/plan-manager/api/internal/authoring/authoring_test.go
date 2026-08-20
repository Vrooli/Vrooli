package authoring_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"plan-manager/internal/authoring"
	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

// testRenderer adapts the plans-domain renderer to the authoring PlanRenderer
// seam for preview tests (the same renderer the production wiring uses).
type testRenderer struct{}

func (testRenderer) Render(p internalplans.Plan) string { return internalplans.RenderMarkdown(p) }
func (testRenderer) RenderDraft(p internalplans.Plan, sessionID string) string {
	return internalplans.RenderMarkdownWithOptions(p, internalplans.RenderOptions{AuthoringSessionID: sessionID})
}

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

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.NoError(t, err)

	md := internalplans.RenderMarkdown(writer.created)
	for _, want := range []string{
		"### Work Posture",
		"**This is greenfield work.**",
		"### Execution Feedback",
		"plan-manager log decision-add <execution-id> --phase <phase-id> --title",
		"## Problem",
		"## Outcome",
		"## Approach & Decisions",
		"### Validation Strategy",
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
	require.Contains(t, md, "### Work Posture")
	require.Contains(t, md, "**This is greenfield work.**")
	require.Contains(t, md, "### Execution Feedback")
	require.Contains(t, md, "plan-manager log decision-add <execution-id> --phase <phase-id> --title")
	require.Contains(t, md, "## Problem")
	require.Equal(t, "final_review", step.StepKind)
	require.Equal(t, 0, writer.calls, "preview must not persist a plan")
}

func TestPreviewPlanQualityNoticeUsesAuthorValidateForDraft(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer, Renderer: testRenderer{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Thin preview", "thin-preview", "")
	require.NoError(t, err)
	fillMandatorySectionsOnly(t, svc, sess.ID)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_CONTEXT: fixture has no global setup.\nNO_SKILL_CONTEXT: fixture has no skill setup.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPhases, "### Phase 1 — Thin\n- Intent: The legacy phase lacks execution-grade fields.\n- Context: none needed — fixture focuses on quality.")
	require.NoError(t, err)

	md, _, err := svc.PreviewPlan(ctx, sess.ID)
	require.NoError(t, err)
	require.Contains(t, md, "Plan quality: **fail**")
	require.Contains(t, md, "plan-manager author validate "+sess.ID)
	require.NotContains(t, md, "plan-manager validate run thin-preview")
	require.Equal(t, 0, writer.calls, "preview must not persist a plan")
}

func TestAuthoringGuidanceReferencesDefaultFeedbackWorkflow(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Feedback guidance", "feedback-guidance", "")
	require.NoError(t, err)

	_, relevantStep, err := svc.GetSection(ctx, sess.ID, authoring.SectionRelevantContext)
	require.NoError(t, err)
	require.Contains(t, strings.Join(relevantStep.Instructions, " | "), "default plan-manager log capture workflow")

	_, phasesStep, err := svc.GetSection(ctx, sess.ID, authoring.SectionPhases)
	require.NoError(t, err)
	require.Contains(t, strings.Join(phasesStep.Instructions, " | "), "default feedback capture is rendered automatically")
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
func newStore(t *testing.T) (authoring.SessionStore, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(authoring.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC))
	return authoring.NewSQLiteStore(d, clk), clk
}

// fakePlanWriter records the plan it was asked to persist and returns it with an
// assigned id (mirroring the plans Service Create contract).
type fakePlanWriter struct {
	mu      sync.Mutex
	created internalplans.Plan
	calls   int
	err     error
	// mirror, when set, is stamped on the created plan — the computed publish
	// result Create threads back (mirrors the production plans service).
	mirror internalplans.RenderedPlanMirror
}

func (w *fakePlanWriter) CreatePlan(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls++
	if w.err != nil {
		return internalplans.Plan{}, w.err
	}
	p.ID = "plan-finalized"
	p.Status = internalplans.PlanStatusDraft
	p.Mirror = w.mirror
	w.created = p
	return p, nil
}

type fakePlanReader struct {
	mu               sync.Mutex
	plans            map[string]internalplans.Plan
	getCalls         int
	renderCalls      int
	getWorkspaces    []string
	renderWorkspaces []string
	getErr           error
	renderErr        error
}

func (r *fakePlanReader) GetPlan(_ context.Context, idOrSlug, workspaceRoot string) (internalplans.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getCalls++
	r.getWorkspaces = append(r.getWorkspaces, workspaceRoot)
	if r.getErr != nil {
		return internalplans.Plan{}, r.getErr
	}
	if r.plans != nil {
		if p, ok := r.plans[idOrSlug]; ok {
			return p, nil
		}
	}
	return internalplans.Plan{}, internalplans.ErrPlanNotFound{ID: idOrSlug}
}

func (r *fakePlanReader) RenderPlan(_ context.Context, idOrSlug, workspaceRoot string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.renderCalls++
	r.renderWorkspaces = append(r.renderWorkspaces, workspaceRoot)
	if r.renderErr != nil {
		return "", r.renderErr
	}
	return "# " + idOrSlug, nil
}

type fakeSkillPackDiscoverer struct {
	result        authoring.SkillPackResult
	err           error
	gotTitle      string
	gotConcepts   []string
	gotComplexity string
}

type fakeSourceEvidenceAdvisor struct {
	result authoring.SourceEvidenceAdvisory
	err    error
	paths  []string
}

type fakeDiagramValidator struct {
	result authoring.DiagramValidationResult
	err    error
	calls  int
}

func (f *fakeDiagramValidator) ValidateMarkdownDiagrams(_ context.Context, _, _ string) (authoring.DiagramValidationResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeSourceEvidenceAdvisor) AdviseSourceEvidence(_ context.Context, paths []string) (authoring.SourceEvidenceAdvisory, error) {
	f.paths = append([]string(nil), paths...)
	return f.result, f.err
}

func (f *fakeSkillPackDiscoverer) DiscoverSkillPack(_ context.Context, title string, concepts []string, complexity string) (authoring.SkillPackResult, error) {
	f.gotTitle = title
	f.gotConcepts = append([]string(nil), concepts...)
	f.gotComplexity = complexity
	if f.err != nil {
		return authoring.SkillPackResult{}, f.err
	}
	return f.result, nil
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

func TestSubmitSectionBlocksInvalidMermaidAndSkipsNonMermaidContent(t *testing.T) {
	validator := &fakeDiagramValidator{result: authoring.DiagramValidationResult{Findings: []authoring.DiagramFinding{{Code: "mermaid_invalid", Line: 3, Message: "Parse error"}}}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Diagrams: validator})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Diagram gate", "diagram-gate", "")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "```mermaid\nsequenceDiagram\nA->>B: one; two\n```")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0].Message, "Mermaid diagram line 3: Parse error")
	require.Equal(t, 1, validator.calls)

	validator.result = authoring.DiagramValidationResult{}
	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "Plain prose without a diagram.")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, 1, validator.calls, "non-Mermaid sections must not call Knowledge Observatory")
}

func TestSubmitSectionBlocksWhenMermaidValidationIsUnavailable(t *testing.T) {
	validator := &fakeDiagramValidator{result: authoring.DiagramValidationResult{Unverified: true}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Diagrams: validator})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Unavailable diagram gate", "unavailable-diagram-gate", "")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "```mermaid\nflowchart TD\nA --> B\n```")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0].Message, "could not be validated")
}

func TestValidateStructureAndFinalizeBlockInvalidMermaid(t *testing.T) {
	validator := &fakeDiagramValidator{result: authoring.DiagramValidationResult{Findings: []authoring.DiagramFinding{{Code: "mermaid_invalid", Line: 3, Message: "Parse error"}}}}
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer, Diagrams: validator})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Diagram finalization gate", "diagram-finalization-gate", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionTechnicalApproach, "```mermaid\nsequenceDiagram\nA->>B: one; two\n```")
	require.NoError(t, err)

	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.Contains(t, strings.Join(violationMessages(violations), "\n"), "Mermaid diagram line 3: Parse error")
	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.Error(t, err)
	require.Zero(t, writer.calls, "finalize must never persist a plan with an invalid diagram")
}

func violationMessages(violations []authoring.StructureViolation) []string {
	messages := make([]string, 0, len(violations))
	for _, violation := range violations {
		messages = append(messages, violation.Message)
	}
	return messages
}

func fillMandatorySectionsOnly(t *testing.T, svc authoring.Service, sessionID string) authoring.Session {
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
		{authoring.SectionRegressionAnchor, "Strategy: change_boundary"},
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

// fillMandatory submits a readiness-clean minimal plan. Tests that intentionally
// exercise pre-readiness checkpoints use fillMandatorySectionsOnly instead.
func fillMandatory(t *testing.T, svc authoring.Service, sessionID string) authoring.Session {
	t.Helper()
	ctx := context.Background()
	sess := fillMandatorySectionsOnly(t, svc, sessionID)
	_, _, _, err := svc.SubmitSection(ctx, sessionID, authoring.SectionRelevantContext, "NO_CONTEXT: unit fixture has no plan-wide setup.")
	require.NoError(t, err)
	if len(sess.PhaseDrafts) == 0 {
		var phase authoring.PhaseDraft
		var violations []authoring.StructureViolation
		sess, phase, violations, _, err = svc.AddPhase(ctx, sessionID, "Fixture phase", "Exercise authoring behavior.")
		require.NoError(t, err)
		require.NotEmpty(t, violations)
		_, _, _, err = svc.SubmitPhaseField(ctx, sessionID, phase.ID, authoring.PhaseFieldNoCodeRefsReason, "NO_CODE_REFS: unit fixture has no phase refs.")
		require.NoError(t, err)
		_, _, _, err = svc.SubmitPhaseField(ctx, sessionID, phase.ID, authoring.PhaseFieldSteps, "Run the focused test fixture.")
		require.NoError(t, err)
		_, _, _, err = svc.SubmitPhaseField(ctx, sessionID, phase.ID, authoring.PhaseFieldValidation, "go test ./internal/authoring")
		require.NoError(t, err)
		_, _, _, err = svc.SubmitPhaseField(ctx, sessionID, phase.ID, authoring.PhaseFieldAcceptance, "The authoring fixture passes.")
		require.NoError(t, err)
		sess, _, _, err = svc.SubmitPhaseField(ctx, sessionID, phase.ID, authoring.PhaseFieldRelevantContext, "NO_CONTEXT: unit fixture has no phase setup.")
		require.NoError(t, err)
	}
	return sess
}

func TestDiscoverSkillPackAutoAddsMultipleSkillsAndIsIdempotent(t *testing.T) {
	discovery := &fakeSkillPackDiscoverer{result: authoring.SkillPackResult{
		Items: []planmodel.RelevantContextItem{
			{Kind: planmodel.RelevantContextSkill, Label: "Implementation Plan Authoring", Target: "implementation-plan-authoring", Reason: "plans", Instruction: "load", Source: planmodel.RelevantContextSourceDiscovered, Status: planmodel.RelevantContextStatusReady},
			{Kind: planmodel.RelevantContextSkill, Label: "Boundary", Target: "boundary-of-responsibility-enforcement", Reason: "boundaries", Instruction: "load", Source: planmodel.RelevantContextSourceDiscovered, Status: planmodel.RelevantContextStatusReady},
		},
		ReadCommand:  "prompt-manager skill read implementation-plan-authoring boundary-of-responsibility-enforcement",
		BudgetStatus: "ok",
		Summary:      "prompt-manager returned 2 skill(s)",
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Skills: discovery})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Skill pack cutover", "", "")
	require.NoError(t, err)

	updated, result, added, kept, violations, _, err := svc.DiscoverSkillPack(ctx, sess.ID, "", []string{"plan-manager authoring", "skill discovery"}, "architectural")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Len(t, added, 2)
	require.Empty(t, kept)
	require.Len(t, updated.RelevantContext, 2)
	require.Equal(t, result.ReadCommand, discovery.result.ReadCommand)
	require.Equal(t, []string{"plan-manager authoring", "skill discovery"}, discovery.gotConcepts)
	require.Equal(t, "architectural", discovery.gotComplexity)

	updated, _, added, kept, violations, _, err = svc.DiscoverSkillPack(ctx, sess.ID, "", []string{"plan-manager authoring"}, "architectural")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Empty(t, added)
	require.Len(t, kept, 2)
	require.Len(t, updated.RelevantContext, 2)
}

func TestDiscoverSkillPackPhaseScopedLandsOnPhaseNotGlobal(t *testing.T) {
	discovery := &fakeSkillPackDiscoverer{result: authoring.SkillPackResult{
		Items: []planmodel.RelevantContextItem{
			{Kind: planmodel.RelevantContextSkill, Label: "Storage Steer", Target: "storage-steer", Reason: "embedding metadata", Instruction: "load", Source: planmodel.RelevantContextSourceDiscovered, Status: planmodel.RelevantContextStatusReady},
		},
		ReadCommand: "prompt-manager skill read storage-steer",
		Summary:     "prompt-manager returned 1 skill(s)",
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Skills: discovery})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Phase-scoped skills", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Embeddings", "Embed each facet text.")
	require.NoError(t, err)

	updated, _, added, kept, violations, _, err := svc.DiscoverSkillPack(ctx, sess.ID, phase.ID, []string{"embedding metadata"}, "moderate")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Len(t, added, 1)
	require.Empty(t, kept)
	// The pack lands on the phase and NOT in global context — the whole point of
	// the flag is that a phase-only skill does not enter every phase's setup.
	require.Empty(t, updated.RelevantContext)
	require.Len(t, updated.PhaseDrafts, 1)
	require.Len(t, updated.PhaseDrafts[0].RelevantContext, 1)
	item := updated.PhaseDrafts[0].RelevantContext[0]
	require.Equal(t, planmodel.RelevantContextScopePhase, item.Scope)
	require.Equal(t, phase.ID, item.PhaseID)
	require.Equal(t, "storage-steer", item.Target)

	// Re-running the same phase-scoped discovery is idempotent.
	updated, _, added, kept, violations, _, err = svc.DiscoverSkillPack(ctx, sess.ID, phase.ID, []string{"embedding metadata"}, "moderate")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Empty(t, added)
	require.Len(t, kept, 1)
	require.Len(t, updated.PhaseDrafts[0].RelevantContext, 1)
}

func TestDiscoverSkillPackUnknownPhaseFailsInsteadOfFallingBackToGlobal(t *testing.T) {
	discovery := &fakeSkillPackDiscoverer{result: authoring.SkillPackResult{
		Items: []planmodel.RelevantContextItem{
			{Kind: planmodel.RelevantContextSkill, Label: "Storage Steer", Target: "storage-steer", Reason: "embedding metadata", Instruction: "load", Source: planmodel.RelevantContextSourceDiscovered, Status: planmodel.RelevantContextStatusReady},
		},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Skills: discovery})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Unknown phase ref", "", "")
	require.NoError(t, err)

	_, _, _, _, _, _, err = svc.DiscoverSkillPack(ctx, sess.ID, "no-such-phase", []string{"embedding metadata"}, "moderate")
	require.Error(t, err)

	// A bad phase ref must not silently demote the pack into global context.
	reloaded, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.Empty(t, reloaded.RelevantContext)
}

func TestDiscoverSkillPackPreservesManualSkillsAndDegradesWhenUnavailable(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, Skills: &fakeSkillPackDiscoverer{err: errors.New("prompt-manager down")}})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Skill pack degraded", "", "")
	require.NoError(t, err)

	manual := planmodel.RelevantContextItem{Kind: planmodel.RelevantContextSkill, Label: "Manual", Target: "manual-skill", Reason: "known", Instruction: "load", Source: planmodel.RelevantContextSourceAuthored, Status: planmodel.RelevantContextStatusReady}
	sess, _, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", manual)
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Len(t, sess.RelevantContext, 1)

	updated, result, added, kept, violations, _, err := svc.DiscoverSkillPack(ctx, sess.ID, "", []string{"anything"}, "")
	require.NoError(t, err)
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradedReason, "prompt-manager down")
	require.Empty(t, added)
	require.Empty(t, kept)
	require.Empty(t, violations)
	require.Len(t, updated.RelevantContext, 1)
	require.Equal(t, "manual-skill", updated.RelevantContext[0].Target)
}

func TestReferenceGateRequiresReferenceOrExplicitReason(t *testing.T) {
	writer := &fakePlanWriter{}
	advisor := &fakeSourceEvidenceAdvisor{result: authoring.SourceEvidenceAdvisory{
		RepairRequired:  true,
		Issues:          []authoring.SourceEvidenceIssue{{Code: "generated_output_too_broad", Severity: "repair-required", Detail: "packages/proto/gen/** is too broad"}},
		Recommendations: []authoring.SourceEvidenceRecommendation{{Selection: "packages/proto/gen/go/plan-manager/**", Reason: "affected namespace"}},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, SourceEvidence: advisor})
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
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**\n- packages/proto/**")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "Strategy: change_boundary")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "done")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_CONTEXT: unit fixture needs no plan-wide setup.\nNO_SKILL_CONTEXT: unit fixture has no skill setup.")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Work", "Work")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldNoCodeRefsReason, "NO_CODE_REFS: fixture")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldSteps, "Do the work.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldValidation, "go test ./internal/authoring")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldAcceptance, "The authoring fixture passes.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldRelevantContext, "NO_CONTEXT: unit fixture has no phase setup.")
	require.NoError(t, err)

	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.Contains(t, violations[0].Message, "run search-hub directly")

	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "NO_CODE_REFS: docs-only plan")
	require.NoError(t, err)
	valid, violations, step, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid)
	require.Empty(t, violations)
	require.Equal(t, []string{"packages/proto/**"}, advisor.paths)
	require.Contains(t, strings.Join(step.Instructions, "\n"), "generated_output_too_broad")
	require.Contains(t, strings.Join(step.Instructions, "\n"), "packages/proto/gen/go/plan-manager/**")

	advisor.err = errors.New("git-control-tower unavailable")
	valid, violations, step, err = svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid, "an advisory outage must not create a false readiness failure")
	require.Empty(t, violations)
	require.Contains(t, strings.Join(step.Instructions, "\n"), "source_evidence_advisory_unavailable")
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
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "Strategy: change_boundary")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "done")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_CONTEXT: unit fixture needs no plan-wide setup.\nNO_SKILL_CONTEXT: unit fixture has no skill setup.")
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

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	plan := result.Plan
	require.NoError(t, err)
	require.Equal(t, 1, writer.calls)
	require.Equal(t, "plan-finalized", plan.ID)

	// The session's prose mapped through to the plan.
	require.Equal(t, "Make widgets better.", writer.created.Purpose)
	require.Equal(t, "Improve widget", writer.created.Title)
	require.NotEmpty(t, writer.created.Phases, "phases section parsed into structured phases")
	require.Equal(t, "Fixture phase", writer.created.Phases[0].Title)
	require.NotEmpty(t, writer.created.References, "references section parsed into structured references")
	require.Equal(t, "internal/widget/core.go", writer.created.References[0].Target)
	require.Equal(t, planmodel.AnchorStrategyChangeBoundary, writer.created.RegressionAnchor.Strategy, "typed anchor carried forward")

	// The session is marked finalized + linked to the plan.
	got, _, err := svc.GetSection(ctx, sess.ID, authoring.SectionPurpose)
	require.NoError(t, err)
	require.NotEmpty(t, got.Content)
}

func TestFinalizeVerifiesReadbackAndIsIdempotent(t *testing.T) {
	writer := &fakePlanWriter{}
	reader := &fakePlanReader{plans: map[string]internalplans.Plan{
		"plan-finalized": {
			ID:          "plan-finalized",
			Slug:        "improve-widget",
			Title:       "Improve widget",
			Status:      internalplans.PlanStatusDraft,
			ContentHash: "hash-1",
		},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, Reader: reader})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "improve-widget", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	firstResult, _, err := svc.Finalize(ctx, sess.Slug, authoring.FinalizeOptions{})
	first := firstResult.Plan
	require.NoError(t, err)
	require.Equal(t, "plan-finalized", first.ID)
	require.Equal(t, "improve-widget", first.Slug)
	require.Equal(t, 1, writer.calls)
	require.Equal(t, 1, reader.getCalls)
	require.Equal(t, 1, reader.renderCalls)

	secondResult, _, err := svc.Finalize(ctx, sess.Slug, authoring.FinalizeOptions{})
	second := secondResult.Plan
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.Slug, second.Slug)
	require.Equal(t, 1, writer.calls, "retrying finalized session must not create a second plan")
	require.Equal(t, 2, reader.getCalls)
	require.Equal(t, 2, reader.renderCalls)
}

func TestConcurrentFinalizeDoesNotCreateDuplicatePlans(t *testing.T) {
	writer := &fakePlanWriter{}
	reader := &fakePlanReader{plans: map[string]internalplans.Plan{
		"plan-finalized": {
			ID:          "plan-finalized",
			Slug:        "concurrent-finalize",
			Title:       "Concurrent finalize",
			Status:      internalplans.PlanStatusDraft,
			ContentHash: "hash-1",
		},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, Reader: reader})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Concurrent finalize", "concurrent-finalize", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	start := make(chan struct{})
	errs := make(chan error, 2)
	ids := make(chan string, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			finalizeResult, _, finalizeErr := svc.Finalize(ctx, sess.Slug, authoring.FinalizeOptions{})
			plan := finalizeResult.Plan
			if finalizeErr == nil {
				ids <- plan.ID
			}
			errs <- finalizeErr
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(ids)

	for finalizeErr := range errs {
		require.NoError(t, finalizeErr)
	}
	for id := range ids {
		require.Equal(t, "plan-finalized", id)
	}
	require.Equal(t, 1, writer.calls, "concurrent finalize must not create duplicate plans")
}

func TestFinalizeFailsWhenReadbackCannotResolvePlan(t *testing.T) {
	writer := &fakePlanWriter{}
	reader := &fakePlanReader{}
	svc := newService(t, authoring.Deps{Writer: writer, Reader: reader})
	ctx := context.Background()

	sess, _, err := svc.StartSession(ctx, "Improve widget", "improve-widget", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.Error(t, err)
	var readback authoring.ErrFinalizeReadback
	require.ErrorAs(t, err, &readback)
	require.Equal(t, 1, writer.calls)

	got, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, got.Finalized, "fake-success finalize must not mark the session finalized")
	require.Empty(t, got.PlanID)
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

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
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

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
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
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor, "Strategy: change_boundary")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionValidationStrategy, "Run the widget suite.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDefinitionOfDone, "Tests green.")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPhases, "Phase 1 - missing markdown heading")
	require.NoError(t, err)

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
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

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
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
	require.Contains(t, md, "### Change Boundary", "preview must render the change boundary")
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

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	plan := result.Plan
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

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	plan := result.Plan
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
		{authoring.SectionRegressionAnchor, "Strategy: change_boundary"},
		{authoring.SectionValidationStrategy, "Run the suite."},
		{authoring.SectionDefinitionOfDone, "Done."},
		{authoring.SectionRelevantContext, "NO_CONTEXT: boundary fixture needs no plan-wide setup.\nNO_SKILL_CONTEXT: boundary fixture has no skill setup."},
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
