package runs_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"data-backup-manager/internal/runs"
	runsmocks "data-backup-manager/internal/runs/mocks"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

// blockedService builds a run service whose every run is stopped (cap-blocked),
// with a clock the test can move.
func blockedService(t *testing.T) (runs.Service, *scheduletest.FakeClock) {
	t.Helper()
	clock := scheduletest.New(time.Date(2026, 11, 15, 13, 0, 0, 0, time.UTC))
	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: []string{"t1"}, DestinationIDs: []string{"dst-1"}}
	svc := runs.NewService(runs.Deps{
		Repo:  runs.NewSQLiteRepository(newRunsDB(t), clock),
		Plans: &runsmocks.FakePlanLookup{Plans: map[string]runs.PlanForRun{plan.ID: plan}},
		Targets: &runsmocks.FakeTargetLookup{Targets: map[string]runs.TargetForRun{
			"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "a"},
		}},
		Destinations: &runsmocks.FakeDestinationLookup{
			Destinations: map[string]runs.DestinationForRun{"dst-1": {ID: "dst-1", Name: "nightly"}},
			BlockFn:      func(string, int64) (bool, string, error) { return true, "over cap", nil },
		},
		Engine:      &mocks.FakeKopiaEngine{},
		Sources:     sources.NewRegistry(&sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}),
		Events:      &runsmocks.FakeEventSink{},
		Clock:       clock,
		StagingRoot: t.TempDir(),
		Executor:    runsmocks.NewSyncExecutor(),
	})
	return svc, clock
}

func wantHeld(t *testing.T, err error, failed int, wait time.Duration, clock *scheduletest.FakeClock) {
	t.Helper()
	var held runs.ErrRetryBackoff
	if !errors.As(err, &held) {
		t.Fatalf("err = %v, want ErrRetryBackoff", err)
	}
	if held.Failed != failed {
		t.Errorf("failed retries = %d, want %d", held.Failed, failed)
	}
	if got := held.Until.Sub(clock.Now()); got != wait {
		t.Errorf("held for %v more, want %v", got, wait)
	}
}

// A retry that keeps ending the same way must not be repeated every minute: the
// 1000 failed runs of one blocked plan were exactly that loop.
func TestRetriesOfABlockedPlanBackOff(t *testing.T) {
	ctx := context.Background()
	svc, clock := blockedService(t)

	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry); err != nil {
		t.Fatalf("first retry is allowed (nothing has failed twice yet): %v", err)
	}
	// One minute later, as the monitor ticks: still held for the other 4.
	clock.Advance(time.Minute)
	_, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry)
	wantHeld(t, err, 1, 4*time.Minute, clock)

	clock.Advance(4 * time.Minute)
	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry); err != nil {
		t.Fatalf("retry after the 5 minute wait: %v", err)
	}
	_, err = svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry)
	wantHeld(t, err, 2, 15*time.Minute, clock)

	list, err := svc.ListRuns(ctx, "plan-1", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("%d runs recorded, want 2: the held retries must not create runs", len(list))
	}
	for _, r := range list {
		if r.Trigger != runs.TriggerRetry {
			t.Errorf("run %s trigger = %s, want retry", r.ID, r.Trigger)
		}
	}
}

func TestAnOperatorsOwnTriggerIsNeverHeldAndRestartsTheCount(t *testing.T) {
	ctx := context.Background()
	svc, clock := blockedService(t)

	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Second)
	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual); err != nil {
		t.Fatalf("a manual trigger must go ahead while retries are held: %v", err)
	}
	// The manual run is now the newest, so the retry count starts over.
	clock.Advance(time.Second)
	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry); err != nil {
		t.Fatalf("a retry after a fresh attempt starts a new count: %v", err)
	}
	// A scheduled attempt does the same.
	clock.Advance(time.Second)
	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerScheduler); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Second)
	if _, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerRetry); err != nil {
		t.Fatalf("a retry after a scheduled attempt starts a new count: %v", err)
	}
}

// Run stats cover the newest attempts and only the retries inside that span, so a
// burst of retries (3,728 of them, once) cannot push every real attempt out.
func TestStatsWindowIsCountedInAttemptsNotRows(t *testing.T) {
	ctx := context.Background()
	repo := runs.NewSQLiteRepository(newRunsDB(t), scheduletest.New(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)))
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	seed := func(trigger runs.TriggerSource, hour int) string {
		t.Helper()
		at := base.Add(time.Duration(hour) * time.Hour)
		r, err := repo.CreateRun(ctx, runs.Run{PlanID: fmt.Sprintf("plan-%d", hour), Trigger: trigger, Status: runs.RunPending, StartedAt: at, UpdatedAt: at})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	seed(runs.TriggerRetry, 0) // older than the window's attempts
	seed(runs.TriggerScheduler, 1)
	seed(runs.TriggerRetry, 2)
	a2 := seed(runs.TriggerManual, 3)
	r3 := seed(runs.TriggerRetry, 4)
	a3 := seed(runs.TriggerScheduler, 5)
	r4 := seed(runs.TriggerRetry, 6)

	reader, ok := repo.(runs.StatsWindowReader)
	if !ok {
		t.Fatal("the sqlite repository must bound the stats window in attempts")
	}
	got, err := reader.StatsWindow(ctx, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, r := range got {
		ids[r.ID] = true
	}
	want := map[string]bool{a3: true, a2: true, r3: true, r4: true}
	if len(ids) != len(want) {
		t.Fatalf("window holds %d runs, want the 2 newest attempts and the 2 retries inside their span: %v", len(ids), ids)
	}
	for id := range want {
		if !ids[id] {
			t.Errorf("run %s missing from the stats window", id)
		}
	}
}

// A run that fails (or is cut off by a restart) after most targets were captured
// is "partial-failed" as a run. For a target, what counts is what happened to
// that target: 205 backed-up targets were listed as failing because 4 others
// could not be read.
func TestTargetStatusFollowsTheTargetsOwnOutcomeNotTheRuns(t *testing.T) {
	ctx := context.Background()
	repo := runs.NewSQLiteRepository(newRunsDB(t), scheduletest.New(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)))
	start := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	run, err := repo.CreateRun(ctx, runs.Run{PlanID: "plan", Trigger: runs.TriggerManual, Status: runs.RunPending, StartedAt: start, UpdatedAt: start})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []runs.RunEvent{runs.EventStartCapture, runs.EventStartSnapshot} {
		if err := repo.UpdateRunStatus(ctx, run.ID, e); err != nil {
			t.Fatal(err)
		}
	}
	for target, status := range map[string]runs.OutcomeStatus{"ok": runs.OutcomeSucceeded, "bad": runs.OutcomeFailed, "sealed": runs.OutcomeBlocked} {
		if err := repo.SaveOutcome(ctx, run.ID, runs.TargetOutcome{TargetID: target, DestinationID: "d", Status: status, StartedAt: start, FinishedAt: start.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.FinishRun(ctx, run.ID, runs.EventPartialFail, "", start.Add(time.Minute), 0); err != nil {
		t.Fatal(err)
	}
	got := map[string]runs.RunStatus{}
	statuses, err := repo.TargetStatuses(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statuses {
		got[s.TargetID] = s.LastRunStatus
	}
	if got["ok"] != runs.RunCompleted || got["bad"] != runs.RunFailed || got["sealed"] != runs.RunFailed {
		t.Fatalf("per-target status = %v, want ok completed, bad and sealed failed", got)
	}
}
