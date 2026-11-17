package plans_test

import (
	"context"
	"database/sql"
	"testing"

	"plan-manager/internal/execution"
	"plan-manager/internal/plans"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func newCandidateService(t *testing.T) (plans.Service, *sql.DB) {
	t.Helper()
	d, clk := newDB(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(execution.Schema)))
	return plans.NewService(plans.Deps{Repo: plans.NewSQLiteRepository(d, clk), Clock: clk, Mirror: newFakeMirrorStore(t.TempDir())}), d
}

func candidateReadyPlan() plans.Plan {
	p := samplePlan()
	p.ProblemStatement = "The current widget workflow loses required plan context."
	p.TargetOutcome = "Agents receive a complete, validated widget plan."
	p.TechnicalApproach = "Persist structured plan data and render it deterministically."
	p.ValidationStrategy = "Run focused package tests and inspect the rendered plan."
	p.ChangeBoundary = plans.ChangeBoundary{AcceptanceAllow: []string{"scenarios/plan-manager/**"}}
	for index := range p.Phases {
		p.Phases[index].Steps = []string{"Implement the phase change."}
		p.Phases[index].Validation = "Run the focused plan-manager test package."
		if len(p.Phases[index].References) == 0 {
			p.Phases[index].Reminders = append(p.Phases[index].Reminders, "NO_CODE_REFS: this fixture phase has no separate code reference.")
		}
		if len(p.Phases[index].RelevantContext) == 0 {
			p.Phases[index].RelevantContext = []plans.RelevantContextItem{{
				Kind: plans.RelevantContextNote, Scope: plans.RelevantContextScopePhase, Label: "Implementation context",
				Instruction: "Read the canonical plan contract before editing.", Required: true,
				RepeatPolicy: plans.RelevantContextPhaseEntry, Source: plans.RelevantContextSourceAuthored, Status: plans.RelevantContextStatusReady,
			}}
		}
	}
	return p
}

func TestCandidateRevisionPreviewsAndAppliesInPlace(t *testing.T) {
	svc, _ := newCandidateService(t)
	ctx := context.Background()
	base, err := svc.Create(ctx, candidateReadyPlan())
	require.NoError(t, err)
	next := base
	next.Purpose = "Updated by workshop candidate."
	candidate, err := svc.CreateCandidate(ctx, plans.CandidateRevision{
		PlanID:                  base.ID,
		ExpectedBaseContentHash: base.ContentHash,
		ProposalProvenance:      "workshop:session-1:response-1",
		CandidatePlan:           next,
	})
	require.NoError(t, err)
	preview, err := svc.PreviewCandidate(ctx, candidate.ID)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Rendered)
	require.Contains(t, preview.Diff.Changes, plans.CandidateFieldChange{Field: "Purpose", BeforeJSON: `"` + base.Purpose + `"`, AfterJSON: `"Updated by workshop candidate."`})
	applied, updated, _, err := svc.ApplyCandidate(ctx, candidate.ID, base.ContentHash, true)
	require.NoError(t, err)
	require.Equal(t, plans.CandidateRevisionApplied, applied.State)
	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, "Updated by workshop candidate.", updated.Purpose)
	require.NotEqual(t, base.ContentHash, updated.ContentHash)

	// Application is idempotent: replay returns the exact canonical revision.
	replay, replayPlan, _, err := svc.ApplyCandidate(ctx, candidate.ID, base.ContentHash, true)
	require.NoError(t, err)
	require.Equal(t, applied.AppliedContentHash, replay.AppliedContentHash)
	require.Equal(t, updated.ContentHash, replayPlan.ContentHash)
}

func TestCandidateRevisionStateGraph(t *testing.T) {
	valid := map[plans.CandidateRevisionState][]plans.CandidateRevisionState{
		plans.CandidateRevisionPending: {plans.CandidateRevisionApplied, plans.CandidateRevisionDiscarded, plans.CandidateRevisionExpired},
	}
	states := []plans.CandidateRevisionState{
		plans.CandidateRevisionPending, plans.CandidateRevisionApplied,
		plans.CandidateRevisionDiscarded, plans.CandidateRevisionExpired,
	}
	for _, from := range states {
		for _, to := range states {
			want := false
			for _, allowed := range valid[from] {
				want = want || allowed == to
			}
			require.Equal(t, want, from.CanTransitionTo(to), "%s -> %s", from, to)
		}
	}
}

func TestCandidateRevisionStoreRejectsTerminalRewrite(t *testing.T) {
	svc, db := newCandidateService(t)
	ctx := context.Background()
	base, err := svc.Create(ctx, candidateReadyPlan())
	require.NoError(t, err)
	candidate, err := svc.CreateCandidate(ctx, plans.CandidateRevision{
		PlanID: base.ID, ExpectedBaseContentHash: base.ContentHash,
		ProposalProvenance: "workshop:state-guard", CandidatePlan: base,
	})
	require.NoError(t, err)
	repo := plans.NewSQLiteRepository(db, nil)

	terminal := candidate
	terminal.State = plans.CandidateRevisionDiscarded
	terminal.DiscardReason = "operator declined candidate"
	require.NoError(t, repo.SaveCandidate(ctx, terminal))

	resurrection := terminal
	resurrection.State = plans.CandidateRevisionApplied
	err = repo.SaveCandidate(ctx, resurrection)
	var stateErr plans.ErrCandidateState
	require.ErrorAs(t, err, &stateErr)
	require.Equal(t, plans.CandidateRevisionDiscarded, stateErr.State)
	require.Equal(t, plans.CandidateRevisionApplied, stateErr.Next)

	unknownTarget := terminal
	unknownTarget.State = "unknown"
	err = repo.SaveCandidate(ctx, unknownTarget)
	require.ErrorAs(t, err, &stateErr)
	require.Equal(t, plans.CandidateRevisionPending, stateErr.State)
	require.Equal(t, plans.CandidateRevisionState("unknown"), stateErr.Next)

	stored, found, err := repo.GetCandidate(ctx, candidate.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, plans.CandidateRevisionDiscarded, stored.State)
	require.Equal(t, "operator declined candidate", stored.DiscardReason)
}

func TestCandidateApplyCommitsPlanAndLifecycleTogether(t *testing.T) {
	svc, db := newCandidateService(t)
	ctx := context.Background()
	base, err := svc.Create(ctx, candidateReadyPlan())
	require.NoError(t, err)
	next := base
	next.Purpose = "This candidate must not partially apply."
	candidate, err := svc.CreateCandidate(ctx, plans.CandidateRevision{
		PlanID: base.ID, ExpectedBaseContentHash: base.ContentHash,
		ProposalProvenance: "workshop:atomic-apply", CandidatePlan: next,
	})
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_candidate_applied BEFORE UPDATE ON candidate_revisions
WHEN NEW.state='applied' BEGIN SELECT RAISE(ABORT, 'injected lifecycle save failure'); END`)
	require.NoError(t, err)

	_, _, _, err = svc.ApplyCandidate(ctx, candidate.ID, base.ContentHash, true)
	require.ErrorContains(t, err, "injected lifecycle save failure")

	storedPlan, err := svc.Get(ctx, base.ID, plans.WorkspaceScope{ID: base.WorkspaceID, Root: base.WorkspaceRoot})
	require.NoError(t, err)
	require.Equal(t, base.ContentHash, storedPlan.ContentHash)
	require.Equal(t, base.Purpose, storedPlan.Purpose)
	storedCandidate, err := svc.GetCandidate(ctx, candidate.ID)
	require.NoError(t, err)
	require.Equal(t, plans.CandidateRevisionPending, storedCandidate.State)
}

func TestCandidateRevisionRejectsStaleBaseAndActiveExecution(t *testing.T) {
	svc, d := newCandidateService(t)
	ctx := context.Background()
	base, err := svc.Create(ctx, candidateReadyPlan())
	require.NoError(t, err)
	_, err = svc.CreateCandidate(ctx, plans.CandidateRevision{
		PlanID: base.ID, ExpectedBaseContentHash: "stale", ProposalProvenance: "workshop:stale", CandidatePlan: plans.Plan{Title: base.Title},
	})
	require.ErrorAs(t, err, new(plans.ErrCandidateStaleBase))
	next := base
	next.Purpose = "Changed"
	candidate, err := svc.CreateCandidate(ctx, plans.CandidateRevision{
		PlanID: base.ID, ExpectedBaseContentHash: base.ContentHash, ProposalProvenance: "workshop:active", CandidatePlan: next,
	})
	require.NoError(t, err)
	for _, row := range []struct{ id, state string }{
		{"active-execution", "active"},
		{"ready-execution", "ready"},
		{"completed-execution", "completed"},
		{"abandoned-execution", "abandoned"},
	} {
		_, err = d.ExecContext(ctx, `INSERT INTO executions (id, plan_id, run_id, lifecycle_state, started_at, updated_at) VALUES (?, ?, '', ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`, row.id, base.ID, row.state)
		require.NoError(t, err)
	}
	_, _, _, err = svc.ApplyCandidate(ctx, candidate.ID, base.ContentHash, true)
	var activeErr plans.ErrCandidateExecutionActive
	require.ErrorAs(t, err, &activeErr)
	require.Equal(t, []string{"active-execution", "ready-execution"}, activeErr.ExecutionIDs)
	_, err = d.ExecContext(ctx, `DELETE FROM executions WHERE id IN ('active-execution', 'ready-execution')`)
	require.NoError(t, err)
	_, updated, _, err := svc.ApplyCandidate(ctx, candidate.ID, base.ContentHash, true)
	require.NoError(t, err, "terminal executions must not block candidate application")
	require.Equal(t, "Changed", updated.Purpose)
	current, err := svc.Get(ctx, base.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, updated.ContentHash, current.ContentHash)
}
