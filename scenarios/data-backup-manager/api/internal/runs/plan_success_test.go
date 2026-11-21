package runs_test

import (
	"context"
	"testing"
	"time"

	"data-backup-manager/internal/runs"

	"github.com/vrooli/api-core/scheduletest"
)

// A plan's success history counts only succeeded outcomes recorded under that
// plan: another plan's success for the same target is not this plan's.
func TestPlanSuccessCountsOnlySucceededOutcomesOfThatPlan(t *testing.T) {
	ctx := context.Background()
	repo := runs.NewSQLiteRepository(newRunsDB(t), scheduletest.New(time.Time{}))
	reader, ok := repo.(runs.PlanSuccessReader)
	if !ok {
		t.Fatal("the production repository must expose plan success history")
	}

	finish := func(planID, targetID string, status runs.OutcomeStatus, at time.Time) {
		t.Helper()
		run, err := repo.CreateRun(ctx, runs.Run{PlanID: planID, Trigger: runs.TriggerScheduler})
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.SaveOutcome(ctx, run.ID, runs.TargetOutcome{TargetID: targetID, DestinationID: "dst", Status: status, StartedAt: at, FinishedAt: at}); err != nil {
			t.Fatal(err)
		}
		if status == runs.OutcomeSucceeded {
			finishSeededRun(t, ctx, repo, run.ID, at)
			return
		}
		for _, ev := range []runs.RunEvent{runs.EventStartCapture, runs.EventStartSnapshot} {
			if err := repo.UpdateRunStatus(ctx, run.ID, ev); err != nil {
				t.Fatal(err)
			}
		}
		if err := repo.FinishRun(ctx, run.ID, runs.EventFail, "failed", at, 0); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Date(2026, 8, 19, 18, 58, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 20, 1, 0, 0, 0, time.UTC)
	finish("plan-a", "t1", runs.OutcomeSucceeded, old)
	finish("plan-a", "t1", runs.OutcomeSucceeded, newer)
	finish("plan-a", "t2", runs.OutcomeFailed, newer.Add(time.Hour))
	finish("plan-b", "t2", runs.OutcomeSucceeded, newer.Add(2*time.Hour))

	got, err := reader.PlanSuccess(ctx, "plan-a")
	if err != nil {
		t.Fatalf("PlanSuccess: %v", err)
	}
	if !got.At.Equal(newer) || !got.ByTarget["t1"].Equal(newer) {
		t.Fatalf("success = %+v, want t1 at %s", got, newer)
	}
	if _, has := got.ByTarget["t2"]; has {
		t.Fatalf("a failed outcome, or another plan's success, was counted for t2: %+v", got.ByTarget)
	}
	empty, err := reader.PlanSuccess(ctx, "never-ran")
	if err != nil || !empty.At.IsZero() || len(empty.ByTarget) != 0 {
		t.Fatalf("a plan that never succeeded must report no success: %+v %v", empty, err)
	}
}
