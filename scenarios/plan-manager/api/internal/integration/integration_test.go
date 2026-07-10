// Package integration_test is the cross-domain wiring proof. The per-domain unit
// tests exercise each domain in isolation behind fakes; this test composes the
// REAL plans / validation / authoring / execution services over a single
// home-store SQLite — wired with the SAME adapters the handler modules use — and
// drives the core agent loop end to end: author a plan → finalize into the plans
// SSOT → run it phase by phase with just-in-time context → complete into the
// canonical handoff; plus the validation read path (resolve references, derive
// the baseline scope). It catches adapter/seam regressions that isolated tests
// cannot (e.g. the PlanWriter / PlanStore / Validator method shims).
package integration_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	internalauthoring "plan-manager/internal/authoring"
	internalexecution "plan-manager/internal/execution"
	internalplanlog "plan-manager/internal/planlog"
	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"
	internalvalidation "plan-manager/internal/validation"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

// --- adapters: identical to the handler modules (PlanWriter / PlanStore /
// PlanSource / Validator). Kept here so the test composes services exactly as
// production does. ---

type planWriter struct{ svc internalplans.Service }

func (a planWriter) CreatePlan(ctx context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	return a.svc.Create(ctx, p)
}

func (a planWriter) GetPlan(ctx context.Context, idOrSlug, workspaceRoot string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{Root: workspaceRoot})
}

func (a planWriter) RenderPlan(ctx context.Context, idOrSlug, workspaceRoot string) (string, error) {
	rendered, err := a.svc.Render(ctx, idOrSlug, internalplans.WorkspaceScope{Root: workspaceRoot}, internalplans.RenderOptions{})
	if err != nil {
		return "", err
	}
	return rendered.Markdown, nil
}

type planSource struct{ svc internalplans.Service }

func (a planSource) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{})
}

type planStore struct{ svc internalplans.Service }

func (a planStore) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{})
}

func (a planStore) UpdatePhase(ctx context.Context, planID, workspaceID, workspaceRoot string, phase internalplans.Phase) (internalplans.Plan, error) {
	return a.svc.UpdatePhase(ctx, planID, internalplans.WorkspaceScope{ID: workspaceID, Root: workspaceRoot}, phase)
}

type validatorAdapter struct{ svc internalvalidation.Service }

func (a validatorAdapter) LastValidation(ctx context.Context, planID, phaseID string) (internalexecution.ValidationResult, bool, error) {
	res, ok, err := a.svc.LastValidation(ctx, planID, phaseID)
	if err != nil || !ok {
		return internalexecution.ValidationResult{}, false, err
	}
	return internalexecution.ValidationResult{
		ID: res.ID, PlanID: res.PlanID, PhaseID: res.PhaseID,
		Verdict: string(res.Verdict), Staleness: res.Staleness,
		CommandsRun: res.CommandsRun, Detail: res.Detail, RanAt: res.RanAt,
	}, true, nil
}

type logLedger struct{ svc internalplanlog.Service }

func (a logLedger) Summarize(ctx context.Context, executionID string) (planmodel.LogSummary, []planmodel.LogEntry, error) {
	return a.svc.Summarize(ctx, internalplanlog.Filter{ExecutionID: executionID})
}

func (a logLedger) SummarizePhase(ctx context.Context, executionID, phaseID string) (planmodel.LogSummary, []planmodel.LogEntry, error) {
	return a.svc.Summarize(ctx, internalplanlog.Filter{ExecutionID: executionID, PhaseID: phaseID})
}

type logResolver struct {
	plans      internalplans.Service
	executions internalexecution.Repository
}

func (a logResolver) Resolve(ctx context.Context, handle string) (internalplanlog.Scope, bool, error) {
	if e, ok, err := a.executions.GetExecution(ctx, handle); err != nil {
		return internalplanlog.Scope{}, false, err
	} else if ok {
		plan, perr := a.plans.Get(ctx, e.PlanID, internalplans.WorkspaceScope{})
		if perr != nil {
			return internalplanlog.Scope{}, false, perr
		}
		return internalplanlog.Scope{
			PlanID:         e.PlanID,
			ExecutionID:    e.ID,
			CurrentPhaseID: e.CurrentPhaseID,
			Phases:         integrationPhaseRefs(plan.Phases),
		}, true, nil
	}
	plan, err := a.plans.Get(ctx, handle, internalplans.WorkspaceScope{})
	if err != nil {
		return internalplanlog.Scope{}, false, nil
	}
	return internalplanlog.Scope{PlanID: plan.ID, Phases: integrationPhaseRefs(plan.Phases)}, true, nil
}

func integrationPhaseRefs(phases []internalplans.Phase) []internalplanlog.PhaseRef {
	out := make([]internalplanlog.PhaseRef, 0, len(phases))
	for _, ph := range phases {
		out = append(out, internalplanlog.PhaseRef{ID: ph.ID, Order: ph.Order, Title: ph.Title})
	}
	return out
}

func newStack(t *testing.T) (*sql.DB, internalplans.Service, internalvalidation.Service, internalauthoring.Service, internalexecution.Service, internalplanlog.Service) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalplans.Schema),
		apidb.SchemaProviderFunc(internalvalidation.Schema),
		apidb.SchemaProviderFunc(internalauthoring.Schema),
		apidb.SchemaProviderFunc(internalexecution.Schema),
		apidb.SchemaProviderFunc(internalplanlog.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC))

	plansSvc := internalplans.NewService(internalplans.Deps{Repo: internalplans.NewSQLiteRepository(d, clk), Clock: clk})
	execRepo := internalexecution.NewSQLiteRepository(d, clk)
	logSvc := internalplanlog.NewService(internalplanlog.Deps{
		Repo:     internalplanlog.NewSQLiteRepository(d, clk),
		Resolver: logResolver{plans: plansSvc, executions: execRepo},
		Clock:    clk,
	})
	// Validation over a real filesystem resolver rooted at the scenario api dir
	// (the test runs from there) so references resolve against real files.
	resolver := internalvalidation.NewFileResolver("")
	validationSvc := internalvalidation.NewService(internalvalidation.Deps{
		Plans:     planSource{svc: plansSvc},
		Resolver:  resolver,
		Staleness: internalvalidation.NewExistenceStaleness(resolver),
		// Runner intentionally nil: RunValidation degrades to UNKNOWN (no live
		// git-control-tower in a unit test) — never a fabricated pass.
		Results: internalvalidation.NewSQLiteResultStore(d, clk),
		Clock:   clk,
	})
	authoringSvc := internalauthoring.NewService(internalauthoring.Deps{
		Store:  internalauthoring.NewSQLiteStore(d, clk),
		Writer: planWriter{svc: plansSvc},
		Reader: planWriter{svc: plansSvc},
		Clock:  clk,
	})
	executionSvc := internalexecution.NewService(internalexecution.Deps{
		Repo:      execRepo,
		Plans:     planStore{svc: plansSvc},
		Validator: validatorAdapter{svc: validationSvc},
		Log:       logLedger{svc: logSvc},
		Velocity:  internalexecution.DefaultVelocitySink(),
		Clock:     clk,
	})
	return d, plansSvc, validationSvc, authoringSvc, executionSvc, logSvc
}

// contentFor returns plausible authored content for a section, tailored so the
// phases/references sections parse into structured data at Finalize.
func contentFor(key string) string {
	switch {
	case strings.Contains(key, "phase"):
		return strings.Join([]string{
			"### Phase 1 — Implement",
			"- Intent: build it",
			"**Ordered Steps:**",
			"1. Implement the change",
			"**Phase Validation:**",
			"go test ./internal/integration",
			"- Acceptance: it builds",
			"**References:**",
			"- [CODE: scenarios/plan-manager/api/internal/integration/integration_test.go]",
			"**Phase Context Setup:**",
			"### Operator Notes",
			"- NO_CONTEXT: integration fixture has no extra phase setup.",
			"",
			"### Phase 2 — Validate",
			"- Intent: check it",
			"**Ordered Steps:**",
			"1. Run validation",
			"**Phase Validation:**",
			"go test ./internal/integration",
			"- Acceptance: dod met",
			"**References:**",
			"- [CODE: scenarios/plan-manager/api/internal/integration/integration_test.go]",
			"**Phase Context Setup:**",
			"### Operator Notes",
			"- NO_CONTEXT: integration fixture has no extra phase setup.",
			"",
		}, "\n")
	case strings.Contains(key, "reference"):
		return "Touches [CODE: main.go] and [REQ: OT-P0-001]."
	case strings.Contains(key, "anchor"):
		return "- Scenario baseline: `plan-manager` (name `impl`)"
	case strings.Contains(key, "context"):
		return "NO_CONTEXT: integration fixture needs no plan-wide setup."
	case key == "decisions":
		return "Storage shape: one SQLite store shared by every domain service."
	default:
		return "Authored content for the " + key + " section."
	}
}

func executionReadyPlan(title string, phases []internalplans.Phase) internalplans.Plan {
	for i := range phases {
		phases[i] = executionReadyPhase(phases[i])
	}
	return internalplans.Plan{
		Title:              title,
		Purpose:            "Integration fixture plan.",
		ProblemStatement:   "Cross-domain integration needs an execution-grade plan.",
		TargetOutcome:      "Execution can start without repair.",
		Scope:              "Plan Manager integration fixture.",
		TechnicalApproach:  "Use real domain services over one SQLite store.",
		ValidationStrategy: "Run integration tests.",
		DefinitionOfDone:   "The integration flow completes.",
		Constraints:        "NO_CODE_REFS: integration fixture has no separate plan-level code refs.",
		ChangeBoundary: internalplans.ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/plan-manager/**"},
		},
		RegressionAnchor: internalplans.RegressionAnchor{
			Strategy: internalplans.AnchorStrategyChangeBoundary,
		},
		RelevantContext: []internalplans.RelevantContextItem{{
			Kind:         internalplans.RelevantContextNote,
			Scope:        internalplans.RelevantContextScopeGlobal,
			Label:        "NO_CONTEXT: integration fixture has no plan-wide setup.",
			Instruction:  "NO_CONTEXT: integration fixture has no plan-wide setup.",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextOncePerExecution,
			Source:       internalplans.RelevantContextSourceAuthored,
			Status:       internalplans.RelevantContextStatusReady,
		}},
		Phases: phases,
	}
}

func executionReadyPhase(phase internalplans.Phase) internalplans.Phase {
	if phase.Intent == "" {
		phase.Intent = "Exercise integration behavior."
	}
	if len(phase.Steps) == 0 {
		phase.Steps = []string{"Run the integration flow."}
	}
	if phase.Validation == "" {
		phase.Validation = "go test ./internal/integration"
	}
	if phase.Acceptance == "" {
		phase.Acceptance = "Integration assertions pass."
	}
	if len(phase.References) == 0 {
		phase.Reminders = append(phase.Reminders, "NO_CODE_REFS: integration fixture has no phase refs.")
	}
	if len(phase.RelevantContext) == 0 {
		phase.RelevantContext = []internalplans.RelevantContextItem{{
			Kind:         internalplans.RelevantContextNote,
			Scope:        internalplans.RelevantContextScopePhase,
			Label:        "NO_CONTEXT: integration fixture has no phase setup.",
			Instruction:  "NO_CONTEXT: integration fixture has no phase setup.",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextPhaseEntry,
			Source:       internalplans.RelevantContextSourceAuthored,
			Status:       internalplans.RelevantContextStatusReady,
		}}
	}
	return phase
}

// TestValidationResultPersistsForCheapContextRead pins the context-server fix:
// status/next must NOT shell a live validation run on every poll. They inject the
// LAST STORED result — so before any explicit RunValidation there is no validation
// in context, and after one the stored result is read back cheaply.
func TestValidationResultPersistsForCheapContextRead(t *testing.T) {
	ctx := context.Background()
	_, plansSvc, validationSvc, _, executionSvc, _ := newStack(t)

	plan, err := plansSvc.Create(ctx, executionReadyPlan("Cheap context", []internalplans.Phase{{
		Title:      "Phase one",
		Steps:      []string{"Run explicit validation once."},
		Validation: "plan-manager validate run <plan> --phase <phase>",
		Acceptance: "done",
		Reminders:  []string{"NO_CONTEXT: validation persistence fixture does not require setup context."},
	}}))
	require.NoError(t, err)
	require.NotEmpty(t, plan.Phases)
	phaseID := plan.Phases[0].ID
	plan.References = []internalplans.Reference{
		{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/api/main.go"},
	}
	plan, err = plansSvc.Update(ctx, plan)
	require.NoError(t, err)

	exec, _, _, err := executionSvc.Start(ctx, plan.ID, "run-cheap")
	require.NoError(t, err)

	// Before any explicit validation run: NO validation in the injected context.
	// status answered the poll without triggering a live baseline.
	_, before, _, err := executionSvc.GetStatus(ctx, exec.ID)
	require.NoError(t, err)
	require.False(t, before.HasValidation, "status must not trigger a live validation run")

	// The agent runs validation explicitly; the result is persisted for cheap reads.
	res, err := validationSvc.RunValidation(ctx, plan.ID, phaseID)
	require.NoError(t, err)
	require.Equal(t, internalvalidation.VerdictUnknown, res.Verdict)

	// Now status reads the STORED result (a cheap store read, no subprocess).
	_, after, _, err := executionSvc.GetStatus(ctx, exec.ID)
	require.NoError(t, err)
	require.True(t, after.HasValidation, "status injects the last STORED validation result")
	require.Equal(t, "unknown", after.LastValidation.Verdict)
}

func TestCrossDomainAuthorToExecuteToHandoff(t *testing.T) {
	ctx := context.Background()
	_, plansSvc, validationSvc, authoringSvc, executionSvc, logSvc := newStack(t)

	// 1) Author a plan via the guided wizard: fill every section the wizard asks
	// for, validate the structure gate, then finalize into the plans SSOT.
	session, _, err := authoringSvc.StartSession(ctx, "Cross-domain plan", "", "")
	require.NoError(t, err)

	// Fill every section the wizard seeded (mandatory + optional) so the
	// references + phases sections parse into structured data at Finalize. The
	// Next() pointer (which surfaces only mandatory-unfilled sections) is
	// exercised separately in the authoring unit tests.
	for _, sec := range session.Sections {
		_, violations, _, subErr := authoringSvc.SubmitSection(ctx, session.ID, sec.Key, contentFor(string(sec.Key)))
		require.NoError(t, subErr)
		require.Empty(t, violations, "submitted content should satisfy the section gate for %q", sec.Key)
	}

	// The Next() pointer reports the session structurally complete.
	_, _, complete, err := authoringSvc.Next(ctx, session.ID)
	require.NoError(t, err)
	require.True(t, complete, "all sections filled => wizard complete")

	valid, violations, _, err := authoringSvc.ValidateStructure(ctx, session.ID)
	require.NoError(t, err)
	require.True(t, valid, "structure gate should pass once all mandatory sections are filled; violations=%v", violations)

	finalizeResult, _, err := authoringSvc.Finalize(ctx, session.ID, internalauthoring.FinalizeOptions{})
	plan := finalizeResult.Plan
	require.NoError(t, err)
	require.NotEmpty(t, plan.ID)
	require.NotEmpty(t, plan.Phases, "phases section parsed into structured phases")

	// 2) The finalized plan is the SSOT — readable through the plans service.
	persisted, err := plansSvc.Get(ctx, plan.ID, internalplans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, plan.ID, persisted.ID)
	require.Equal(t, internalplans.PlanStatusDraft, persisted.Status, "all phases start todo => draft")

	// 3) Validation read path over the persisted plan.
	refReport, err := validationSvc.ResolveReferences(ctx, plan.ID, "")
	require.NoError(t, err)
	require.NotEmpty(t, refReport.References, "the references section parsed into reference records")

	scope, err := validationSvc.DeriveBaselineScope(ctx, plan.ID, "")
	require.NoError(t, err)
	_ = scope // derivation must not error; exact commands depend on ref targets

	// 4) Execute the plan phase by phase with just-in-time context injection.
	exec, _, _, err := executionSvc.Start(ctx, plan.ID, "run-xyz")
	require.NoError(t, err)
	require.Equal(t, plan.ID, exec.PlanID)
	require.Equal(t, "run-xyz", exec.RunID)

	_, phaseCtx, _, err := executionSvc.GetStatus(ctx, exec.ID)
	require.NoError(t, err)
	require.True(t, phaseCtx.HasCurrent, "context injection returns the current phase")
	require.NotEmpty(t, phaseCtx.CurrentPhase.ID)

	// Record an in-flow decision + candidate finding through the log domain
	// (feeds the handoff via the LogLedger seam).
	_, _, _, err = logSvc.AddDecision(ctx, internalplanlog.AddInputs{PlanOrExecution: exec.ID, PhaseID: persisted.Phases[0].ID, Title: "use the SSOT"})
	require.NoError(t, err)
	_, _, _, err = logSvc.AddFinding(ctx, internalplanlog.AddInputs{PlanOrExecution: exec.ID, PhaseID: persisted.Phases[0].ID, Title: "possible edge case"})
	require.NoError(t, err)

	// Drive every phase to done (the runner delegates the transition to plans).
	for _, ph := range persisted.Phases {
		_, _, _, err = logSvc.AddNote(ctx, internalplanlog.AddInputs{
			PlanOrExecution: exec.ID,
			PhaseID:         ph.ID,
			Title:           internalexecution.NoFeedbackCheckpointTitle,
			Detail:          "integration fixture reviewed phase feedback before transition",
		})
		require.NoError(t, err)
		_, _, _, transErr := executionSvc.TransitionPhase(ctx, exec.ID, ph.ID, internalexecution.PhaseTransitionInputs{
			ToStatus:                 internalplans.PhaseStatusDone,
			ValidationOverrideReason: "integration fixture focuses on author-to-handoff flow; validation is covered separately",
		})
		require.NoError(t, transErr)
	}

	// Plan status is recomputed to complete via the delegated transitions.
	done, err := plansSvc.Get(ctx, plan.ID, internalplans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, internalplans.PlanStatusComplete, done.Status)

	// 5) Complete → canonical handoff assembled from captured state.
	handoff, _, _, err := executionSvc.Complete(ctx, exec.ID, internalexecution.CompletionInputs{Tokens: 1234, Iterations: 5})
	require.NoError(t, err)
	require.Equal(t, internalexecution.CompletenessFull, handoff.Completeness, "all phases done => full")
	require.Empty(t, handoff.ResumePhaseID, "no resume point when complete")
	require.Equal(t, 1, handoff.LogSummary.Decisions, "the decision is rolled into the handoff log summary")
	require.Equal(t, 1, handoff.LogSummary.Findings, "the candidate finding is rolled into the handoff log summary")
	require.Equal(t, 2, handoff.LogSummary.Notes, "phase feedback checkpoint notes are rolled into the handoff")
	require.Len(t, handoff.LogEntries, 4, "the handoff snapshots work products and feedback checkpoint notes")

	got, _, err := executionSvc.GetHandoff(ctx, exec.ID)
	require.NoError(t, err)
	require.Equal(t, handoff.ID, got.ID)

	// 6) Velocity captured locally.
	points, _, err := executionSvc.GetVelocity(ctx, plan.ID)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.EqualValues(t, 1234, points[0].Tokens)
}

func TestSmallAgentContinueLoopsAuthorAndExecute(t *testing.T) {
	ctx := context.Background()
	_, plansSvc, _, authoringSvc, executionSvc, logSvc := newStack(t)

	session, step, err := authoringSvc.StartSession(ctx, "Small-agent loop", "small-agent-loop", "")
	require.NoError(t, err)
	require.Equal(t, "author-next", step.NextActions[0].ID)

	var finalized internalplans.Plan
	for guard := 0; guard < 20 && finalized.ID == ""; guard++ {
		var section internalauthoring.Section
		var phase internalauthoring.PhaseDraft
		var ready bool
		var violations []internalauthoring.StructureViolation

		session, section, phase, ready, violations, step, err = authoringSvc.ContinueAuthoring(ctx, session.ID)
		require.NoError(t, err)
		require.Len(t, step.NextActions, 1, "continue should expose exactly one recommended action")
		require.Contains(t, []internalauthoring.NextActionKind{internalauthoring.NextActionRecommended, internalauthoring.NextActionRecovery}, step.NextActions[0].Kind)

		switch step.StepKind {
		case "purpose":
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "Prove the continue-loop can guide a small implementation agent.")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "problem_statement", "target_outcome", "technical_approach", "validation_strategy":
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "Continue-loop fixture content for "+string(section.Key)+".")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "scope":
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "In: Plan Manager authoring and execution loop. Out: consumer inversion.")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "acceptance_boundary":
			// The change boundary is mandatory (satisfiable by OPERATOR_ONLY); the
			// continue loop surfaces it before references.
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "acceptance_allow:\n- scenarios/plan-manager/**")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "references":
			// References is mandatory (satisfiable by NO_CODE_REFS); the continue
			// loop surfaces it as a first-class step now.
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "[CODE: scenarios/plan-manager/api/internal/integration/integration_test.go]")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "regression_anchor":
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "Strategy: change_boundary")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "definition_of_done":
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "Authoring finalizes and execution handoff completes with structured state.")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "global_relevant_context":
			// The continue loop may surface advisory plan-wide context; a direct
			// setup item is enough to move on because context is no longer a
			// candidate-sweep gate.
			require.Equal(t, "discover-skill-pack", step.NextActions[0].ID)
			session, _, violations, _, err = authoringSvc.SubmitRelevantContextItem(ctx, session.ID, "", internalplans.RelevantContextItem{
				Kind:         internalplans.RelevantContextSkill,
				Label:        "Load the planning runtime steer",
				Reason:       "Plan-wide setup so any phase agent reorients quickly.",
				Instruction:  "Load this skill before starting any phase.",
				Target:       "implementation-plan-authoring",
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextOncePerExecution,
			})
			require.NoError(t, err)
			require.Empty(t, violations)
		case "phase_outline":
			require.Equal(t, "add-phase", step.NextActions[0].ID)
			session, phase, violations, _, err = authoringSvc.AddPhase(ctx, session.ID, "Implement loop", "Exercise guided authoring and execution handoff.")
			require.NoError(t, err)
			require.NotEmpty(t, phase.ID)
			require.NotEmpty(t, violations, "new phase still needs references, acceptance, and context")
		case "phase_references":
			session, violations, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldReferences, "[CODE: scenarios/plan-manager/api/internal/integration/integration_test.go]")
			require.NoError(t, err)
			require.NotEmpty(t, violations, "steps/validation/acceptance/context still missing")
		case "phase_steps":
			session, violations, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldSteps, "Wire the handler\nWire the CLI")
			require.NoError(t, err)
			require.NotEmpty(t, violations, "validation/acceptance/context still missing")
		case "phase_validation":
			session, violations, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldValidation, "go test ./internal/integration")
			require.NoError(t, err)
			require.NotEmpty(t, violations, "acceptance/context still missing")
		case "phase_acceptance":
			session, violations, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldAcceptance, "The small-agent continue-loop integration test passes.")
			require.NoError(t, err)
			require.NotEmpty(t, violations, "context still missing")
		case "phase_relevant_context":
			require.Equal(t, "submit-phase-relevant_context", step.NextActions[0].ID)
			session, _, violations, _, err = authoringSvc.SubmitRelevantContextItem(ctx, session.ID, phase.ID, internalplans.RelevantContextItem{
				Kind:         internalplans.RelevantContextCodeRef,
				Label:        "Integration fixture",
				Reason:       "The phase changes the cross-domain proof itself.",
				Instruction:  "Read the integration test before changing the guided loop.",
				Target:       "scenarios/plan-manager/api/internal/integration/integration_test.go",
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextPhaseEntry,
			})
			require.NoError(t, err)
			require.Empty(t, violations)
		case "validation_recovery":
			require.NotEmpty(t, violations)
			require.Equal(t, internalauthoring.SectionReferences, violations[0].SectionKey)
			session, violations, _, err = authoringSvc.SubmitSection(ctx, session.ID, internalauthoring.SectionReferences, "NO_CODE_REFS: loop-level test has no separate product code reference beyond the phase reference")
			require.NoError(t, err)
			require.Empty(t, violations)
		case "final_review":
			require.True(t, ready)
			valid, violations, step, err := authoringSvc.ValidateStructure(ctx, session.ID)
			require.NoError(t, err)
			require.True(t, valid, "continue should only reach final review after structure is valid: %v", violations)
			require.Equal(t, "finalize-session", step.NextActions[0].ID)
			finalizedResult, fstep, ferr2 := authoringSvc.Finalize(ctx, session.ID, internalauthoring.FinalizeOptions{})
			finalized, step, err = finalizedResult.Plan, fstep, ferr2
			require.NoError(t, err)
			require.Equal(t, "finalized", step.StepKind)
		default:
			t.Fatalf("unexpected continue step %q action=%v", step.StepKind, step.NextActions)
		}
	}
	require.NotEmpty(t, finalized.ID, "authoring continue loop should finalize within guard")
	require.Len(t, finalized.Phases, 1)
	require.Len(t, finalized.Phases[0].RelevantContext, 1)
	finalized.RegressionAnchor = internalplans.RegressionAnchor{
		Strategy:     internalplans.AnchorStrategyScenarioBaseline,
		Scenario:     "plan-manager",
		BaselineName: "impl",
	}
	var updateErr error
	finalized, updateErr = plansSvc.Update(ctx, finalized)
	require.NoError(t, updateErr)

	exec, pctx, execStep, err := executionSvc.ContinueExecution(ctx, finalized.ID, "", "run-small-agent")
	require.NoError(t, err)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "transition-active", execStep.NextActions[0].ID)

	exec, _, execStep, err = executionSvc.TransitionPhase(ctx, exec.ID, pctx.CurrentPhase.ID, internalexecution.PhaseTransitionInputs{ToStatus: internalplans.PhaseStatusActive})
	require.NoError(t, err)
	require.Equal(t, "execution-next", execStep.NextActions[0].ID)

	exec, pctx, execStep, err = executionSvc.ContinueExecution(ctx, exec.ID, "", "")
	require.NoError(t, err)
	require.Equal(t, "run-validation", execStep.NextActions[0].ID, "continue must not recommend done without fresh passing validation")
	require.NotEmpty(t, execStep.NextActions[0].BlockedBy)

	_, _, _, err = logSvc.AddNote(ctx, internalplanlog.AddInputs{
		PlanOrExecution: exec.ID,
		PhaseID:         pctx.CurrentPhase.ID,
		Title:           internalexecution.NoFeedbackCheckpointTitle,
		Detail:          "small-agent fixture reviewed phase feedback before transition",
	})
	require.NoError(t, err)

	exec, _, execStep, err = executionSvc.TransitionPhase(ctx, exec.ID, pctx.CurrentPhase.ID, internalexecution.PhaseTransitionInputs{
		ToStatus:                 internalplans.PhaseStatusDone,
		ValidationOverrideReason: "small-agent integration fixture validates degraded guidance separately",
	})
	require.NoError(t, err)
	require.Equal(t, "complete-execution", execStep.NextActions[0].ID)

	handoff, _, execStep, err := executionSvc.Complete(ctx, exec.ID, internalexecution.CompletionInputs{Tokens: 321, Iterations: 2})
	require.NoError(t, err)
	require.Equal(t, internalexecution.CompletenessFull, handoff.Completeness)
	require.Equal(t, "execution-handoff", execStep.NextActions[0].ID)

	persisted, err := plansSvc.Get(ctx, finalized.ID, internalplans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, internalplans.PlanStatusComplete, persisted.Status)
}
