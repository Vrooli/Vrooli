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

type planSource struct{ svc internalplans.Service }

func (a planSource) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug)
}

type planStore struct{ svc internalplans.Service }

func (a planStore) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug)
}

func (a planStore) UpdatePhase(ctx context.Context, planID string, phase internalplans.Phase) (internalplans.Plan, error) {
	return a.svc.UpdatePhase(ctx, planID, phase)
}

type validatorAdapter struct{ svc internalvalidation.Service }

func (a validatorAdapter) ComputeStaleness(ctx context.Context, planID, phaseID string) (internalplans.StalenessTier, error) {
	report, err := a.svc.ComputeStaleness(ctx, planID, phaseID)
	if err != nil {
		return internalplans.StalenessUnknown, err
	}
	return report.Overall, nil
}

func (a validatorAdapter) RunValidation(ctx context.Context, planID, phaseID string) (internalexecution.ValidationResult, error) {
	res, err := a.svc.RunValidation(ctx, planID, phaseID)
	if err != nil {
		return internalexecution.ValidationResult{}, err
	}
	return internalexecution.ValidationResult{
		ID: res.ID, PlanID: res.PlanID, PhaseID: res.PhaseID,
		Verdict: string(res.Verdict), Staleness: res.Staleness,
		CommandsRun: res.CommandsRun, Detail: res.Detail, RanAt: res.RanAt,
	}, nil
}

func newStack(t *testing.T) (*sql.DB, internalplans.Service, internalvalidation.Service, internalauthoring.Service, internalexecution.Service) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalplans.Schema),
		apidb.SchemaProviderFunc(internalauthoring.Schema),
		apidb.SchemaProviderFunc(internalexecution.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC))

	plansSvc := internalplans.NewService(internalplans.Deps{Repo: internalplans.NewSQLiteRepository(d, clk), Clock: clk})
	// Validation over a real filesystem resolver rooted at the scenario api dir
	// (the test runs from there) so references resolve against real files.
	resolver := internalvalidation.NewFileResolver("")
	validationSvc := internalvalidation.NewService(internalvalidation.Deps{
		Plans:     planSource{svc: plansSvc},
		Resolver:  resolver,
		Staleness: internalvalidation.NewExistenceStaleness(resolver),
		// Runner intentionally nil: RunValidation degrades to UNKNOWN (no live
		// git-control-tower in a unit test) — never a fabricated pass.
		Clock: clk,
	})
	authoringSvc := internalauthoring.NewService(internalauthoring.Deps{
		Store:  internalauthoring.NewSQLiteStore(d, clk),
		Writer: planWriter{svc: plansSvc},
		Clock:  clk,
	})
	executionSvc := internalexecution.NewService(internalexecution.Deps{
		Repo:      internalexecution.NewSQLiteRepository(d, clk),
		Plans:     planStore{svc: plansSvc},
		Validator: validatorAdapter{svc: validationSvc},
		Velocity:  internalexecution.DefaultVelocitySink(),
		Clock:     clk,
	})
	return d, plansSvc, validationSvc, authoringSvc, executionSvc
}

// contentFor returns plausible authored content for a section, tailored so the
// phases/references sections parse into structured data at Finalize.
func contentFor(key string) string {
	switch {
	case strings.Contains(key, "phase"):
		return "### Phase 1 — Implement\n- Intent: build it\n- Acceptance: it builds\n\n### Phase 2 — Validate\n- Intent: check it\n- Acceptance: dod met\n"
	case strings.Contains(key, "reference"):
		return "Touches [CODE: main.go] and [REQ: OT-P0-001]."
	case strings.Contains(key, "anchor"):
		return "Scenario baseline `plan-manager` name `impl`."
	default:
		return "Authored content for the " + key + " section."
	}
}

func TestCrossDomainAuthorToExecuteToHandoff(t *testing.T) {
	ctx := context.Background()
	_, plansSvc, validationSvc, authoringSvc, executionSvc := newStack(t)

	// 1) Author a plan via the guided wizard: fill every section the wizard asks
	// for, validate the structure gate, then finalize into the plans SSOT.
	session, err := authoringSvc.StartSession(ctx, "Cross-domain plan", "", "")
	require.NoError(t, err)

	// Fill every section the wizard seeded (mandatory + optional) so the
	// references + phases sections parse into structured data at Finalize. The
	// Next() pointer (which surfaces only mandatory-unfilled sections) is
	// exercised separately in the authoring unit tests.
	for _, sec := range session.Sections {
		_, violations, subErr := authoringSvc.SubmitSection(ctx, session.ID, sec.Key, contentFor(string(sec.Key)))
		require.NoError(t, subErr)
		require.Empty(t, violations, "submitted content should satisfy the section gate for %q", sec.Key)
	}

	// The Next() pointer reports the session structurally complete.
	_, complete, err := authoringSvc.Next(ctx, session.ID)
	require.NoError(t, err)
	require.True(t, complete, "all sections filled => wizard complete")

	valid, violations, err := authoringSvc.ValidateStructure(ctx, session.ID)
	require.NoError(t, err)
	require.True(t, valid, "structure gate should pass once all mandatory sections are filled; violations=%v", violations)

	plan, err := authoringSvc.Finalize(ctx, session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, plan.ID)
	require.NotEmpty(t, plan.Phases, "phases section parsed into structured phases")

	// 2) The finalized plan is the SSOT — readable through the plans service.
	persisted, err := plansSvc.Get(ctx, plan.ID)
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
	exec, err := executionSvc.Start(ctx, plan.ID, "run-xyz")
	require.NoError(t, err)
	require.Equal(t, plan.ID, exec.PlanID)
	require.Equal(t, "run-xyz", exec.RunID)

	_, phaseCtx, err := executionSvc.GetStatus(ctx, exec.ID)
	require.NoError(t, err)
	require.True(t, phaseCtx.HasCurrent, "context injection returns the current phase")
	require.NotEmpty(t, phaseCtx.CurrentPhase.ID)

	// Record an in-flow decision + candidate finding (feeds the handoff).
	_, err = executionSvc.RecordDecision(ctx, exec.ID, persisted.Phases[0].ID, "use the SSOT", "")
	require.NoError(t, err)
	_, err = executionSvc.RecordFinding(ctx, exec.ID, persisted.Phases[0].ID, "possible edge case", "")
	require.NoError(t, err)

	// Drive every phase to done (the runner delegates the transition to plans).
	for _, ph := range persisted.Phases {
		_, _, transErr := executionSvc.TransitionPhase(ctx, exec.ID, ph.ID, internalplans.PhaseStatusDone)
		require.NoError(t, transErr)
	}

	// Plan status is recomputed to complete via the delegated transitions.
	done, err := plansSvc.Get(ctx, plan.ID)
	require.NoError(t, err)
	require.Equal(t, internalplans.PlanStatusComplete, done.Status)

	// 5) Complete → canonical handoff assembled from captured state.
	handoff, _, err := executionSvc.Complete(ctx, exec.ID, internalexecution.CompletionInputs{Tokens: 1234, Iterations: 5})
	require.NoError(t, err)
	require.Equal(t, internalexecution.CompletenessFull, handoff.Completeness, "all phases done => full")
	require.Empty(t, handoff.ResumePhaseID, "no resume point when complete")
	require.Len(t, handoff.Decisions, 1)
	require.Len(t, handoff.CandidateFindings, 1)

	got, err := executionSvc.GetHandoff(ctx, exec.ID)
	require.NoError(t, err)
	require.Equal(t, handoff.ID, got.ID)

	// 6) Velocity captured locally.
	points, err := executionSvc.GetVelocity(ctx, plan.ID)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.EqualValues(t, 1234, points[0].Tokens)
}
