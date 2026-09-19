package drills_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	drills "data-backup-manager/internal/drills"
)

type fakePlans struct{ plan drills.Plan }

func (f fakePlans) PlanForDrill(context.Context, string) (drills.Plan, error) { return f.plan, nil }
func (f fakePlans) SchedulableDrillPlans(context.Context) ([]drills.Plan, error) {
	return []drills.Plan{f.plan}, nil
}

type fakeSnapshots struct {
	snapshot drills.Snapshot
	ok       bool
}

func (f fakeSnapshots) LatestSuccessfulSnapshot(context.Context, string, string, string) (drills.Snapshot, bool, error) {
	return f.snapshot, f.ok, nil
}

type fakeRestores struct{ done chan struct{} }

func (f fakeRestores) VerifyTarget(context.Context, string, string, string) (drills.Restore, error) {
	return drills.Restore{ID: "restore-1", Status: "requested"}, nil
}

func (f fakeRestores) GetRestore(context.Context, string) (drills.Restore, error) {
	select {
	case <-f.done:
	default:
		close(f.done)
	}
	return drills.Restore{ID: "restore-1", Status: "verified"}, nil
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func TestService_NoSnapshotPersistsFailedEvidenceAndIsIdempotent(t *testing.T) {
	repo := newDrillRepo(t)
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	svc := drills.NewService(drills.Deps{Repo: repo, Plans: fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}}, Snapshots: fakeSnapshots{}, Restores: fakeRestores{done: make(chan struct{})}, Clock: fixedClock{now: now}, BaseContext: context.Background()})
	defer svc.Shutdown(context.Background())
	first, err := svc.Run(context.Background(), "plan-1", "", "", "same-request", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != drills.StatusFailed || first.NextAction == "" {
		t.Fatalf("failed evidence = %+v", first)
	}
	second, err := svc.Run(context.Background(), "plan-1", "", "", "same-request", false)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("retry created duplicate %q vs %q", second.ID, first.ID)
	}
}

func TestService_SuccessfulDrillUsesVerifiedRestoreEvidence(t *testing.T) {
	repo := newDrillRepo(t)
	done := make(chan struct{})
	finishedPersisted := make(chan struct{})
	svc := drills.NewService(drills.Deps{Repo: signalFinishRepo{Repository: repo, finished: finishedPersisted}, Plans: fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}}, Snapshots: fakeSnapshots{snapshot: drills.Snapshot{ID: "snapshot-1"}, ok: true}, Restores: fakeRestores{done: done}, Clock: fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}, BaseContext: context.Background(), PollInterval: time.Millisecond})
	defer svc.Shutdown(context.Background())
	drill, err := svc.Run(context.Background(), "plan-1", "", "", "success-1", false)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("verified restore worker did not run")
	}
	select {
	case <-finishedPersisted:
	case <-time.After(time.Second):
		t.Fatal("drill worker did not persist verified evidence")
	}
	finished, err := repo.Get(context.Background(), drill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != drills.StatusVerified || finished.RestoreID != "restore-1" {
		t.Fatalf("finished drill = %+v", finished)
	}
}

func TestRunDueRejectsMalformedLegacySchedule(t *testing.T) {
	repo := newDrillRepo(t)
	svc := drills.NewService(drills.Deps{
		Repo:        repo,
		Plans:       fakePlans{plan: drills.Plan{ID: "bad-plan", Enabled: true, DrillSchedule: "not-a-duration"}},
		Snapshots:   fakeSnapshots{},
		Restores:    fakeRestores{done: make(chan struct{})},
		Clock:       fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)},
		BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())

	if err := svc.RunDue(context.Background()); err == nil || !strings.Contains(err.Error(), `bad-plan`) {
		t.Fatalf("RunDue error = %v, want plan-scoped malformed schedule error", err)
	}
}

type failVerifiedFinishRepo struct {
	drills.Repository
	finished chan struct{}
}

type signalFinishRepo struct {
	drills.Repository
	finished chan struct{}
}

func (r signalFinishRepo) Finish(ctx context.Context, id string, status drills.Status, errMsg, nextAction string, finishedAt time.Time) error {
	err := r.Repository.Finish(ctx, id, status, errMsg, nextAction, finishedAt)
	select {
	case <-r.finished:
	default:
		close(r.finished)
	}
	return err
}

func (r failVerifiedFinishRepo) Finish(ctx context.Context, id string, status drills.Status, errMsg, nextAction string, finishedAt time.Time) error {
	if status == drills.StatusVerified {
		return errors.New("catalog is unavailable")
	}
	select {
	case <-r.finished:
	default:
		close(r.finished)
	}
	return r.Repository.Finish(ctx, id, status, errMsg, nextAction, finishedAt)
}

type failMarkRunningRepo struct {
	drills.Repository
	called   chan struct{}
	finished chan struct{}
}

func (r failMarkRunningRepo) MarkRunning(context.Context, string, string, time.Time) error {
	select {
	case <-r.called:
	default:
		close(r.called)
	}
	return errors.New("catalog is unavailable")
}

func (r failMarkRunningRepo) Finish(ctx context.Context, id string, status drills.Status, errMsg, nextAction string, finishedAt time.Time) error {
	select {
	case <-r.finished:
	default:
		close(r.finished)
	}
	return r.Repository.Finish(ctx, id, status, errMsg, nextAction, finishedAt)
}

func TestService_MarkRunningFailureLeavesActionableFailedEvidence(t *testing.T) {
	base := newDrillRepo(t)
	called := make(chan struct{})
	finishedPersisted := make(chan struct{})
	done := make(chan struct{})
	svc := drills.NewService(drills.Deps{
		Repo:         failMarkRunningRepo{Repository: base, called: called, finished: finishedPersisted},
		Plans:        fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}},
		Snapshots:    fakeSnapshots{snapshot: drills.Snapshot{ID: "snapshot-1"}, ok: true},
		Restores:     fakeRestores{done: done},
		Clock:        fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)},
		BaseContext:  context.Background(),
		PollInterval: time.Millisecond,
	})
	defer svc.Shutdown(context.Background())

	drill, err := svc.Run(context.Background(), "plan-1", "", "", "mark-running-failure", false)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("drill worker did not attempt to persist its running state")
	}
	select {
	case <-finishedPersisted:
	case <-time.After(time.Second):
		t.Fatal("drill worker did not persist failed evidence after running-state failure")
	}

	finished, err := base.Get(context.Background(), drill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != drills.StatusFailed || finished.Error == "" || finished.NextAction == "" {
		t.Fatalf("mark-running failure evidence = %+v", finished)
	}
}

func TestService_FailedEvidenceCommitIsNotSilentlyDiscarded(t *testing.T) {
	base := newDrillRepo(t)
	done := make(chan struct{})
	finishedPersisted := make(chan struct{})
	svc := drills.NewService(drills.Deps{
		Repo:         failVerifiedFinishRepo{Repository: base, finished: finishedPersisted},
		Plans:        fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}},
		Snapshots:    fakeSnapshots{snapshot: drills.Snapshot{ID: "snapshot-1"}, ok: true},
		Restores:     fakeRestores{done: done},
		Clock:        fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)},
		BaseContext:  context.Background(),
		PollInterval: time.Millisecond,
	})
	defer svc.Shutdown(context.Background())

	drill, err := svc.Run(context.Background(), "plan-1", "", "", "evidence-failure", false)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("verified restore worker did not run")
	}
	select {
	case <-finishedPersisted:
	case <-time.After(time.Second):
		t.Fatal("drill worker did not persist fallback failed evidence")
	}

	finished, err := base.Get(context.Background(), drill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != drills.StatusFailed {
		t.Fatalf("evidence failure left drill in %q: %+v", finished.Status, finished)
	}
	if finished.Error == "" || finished.NextAction == "" {
		t.Fatalf("evidence failure was not actionable: %+v", finished)
	}
}
