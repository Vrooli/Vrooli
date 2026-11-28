package drills_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	drills "data-backup-manager/internal/drills"
	"github.com/vrooli/api-core/eventbus"
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

type fixedDrillReadiness struct {
	ready  bool
	reason string
	err    error
}

func (f fixedDrillReadiness) ReadyForDrill(context.Context, string) (bool, string, error) {
	return f.ready, f.reason, f.err
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

type fakeNotificationPublisher struct {
	events   []eventbus.DomainEvent
	fail     int
	disabled bool
}

func (p *fakeNotificationPublisher) Enabled() bool { return !p.disabled }

func (p *fakeNotificationPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	if p.fail > 0 {
		p.fail--
		return errors.New("events unavailable")
	}
	return nil
}

func TestFailedDrillNoticeStaysPendingWhenEventsAreDisabled(t *testing.T) {
	repo := newDrillRepo(t)
	notifications := &fakeNotificationPublisher{disabled: true}
	svc := drills.NewService(drills.Deps{
		Repo: repo, Plans: fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}},
		Snapshots: fakeSnapshots{}, Restores: fakeRestores{done: make(chan struct{})},
		Notifications: notifications, Clock: fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}, BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())

	drill, err := svc.Run(context.Background(), "plan-1", "", "", "events-disabled", true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), drill.ID)
	if err != nil || got.NotificationState != drills.NotificationPending || len(notifications.events) != 0 {
		t.Fatalf("disabled Events delivery was reported as sent: drill=%+v events=%d err=%v", got, len(notifications.events), err)
	}
	if err := svc.Reconcile(context.Background()); err != nil {
		t.Fatalf("disabled Events delivery blocked drill service startup reconciliation: %v", err)
	}
}

func TestMigrationSnapshotUsesExactUnboundDestinationAndSnapshot(t *testing.T) {
	repo := newDrillRepo(t)
	done := make(chan struct{})
	svc := drills.NewService(drills.Deps{
		Repo:      repo,
		Plans:     fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"old-destination"}, Enabled: true}},
		Snapshots: fakeSnapshots{}, Restores: fakeRestores{done: done},
		Clock: fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}, BaseContext: context.Background(), PollInterval: time.Millisecond,
	})
	defer svc.Shutdown(context.Background())
	drill, err := svc.RunMigrationSnapshot(context.Background(), "plan-1", "target-1", "operator-selected-new", "captured-snapshot", "db4-drill-key")
	if err != nil {
		t.Fatal(err)
	}
	if drill.DestinationID != "operator-selected-new" || drill.SnapshotID != "captured-snapshot" {
		t.Fatalf("migration drill lost exact destination/snapshot: %+v", drill)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		got, getErr := svc.Get(context.Background(), drill.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if got.Status == drills.StatusVerified {
			return
		}
		if got.Status == drills.StatusFailed {
			t.Fatalf("migration drill failed: %+v", got)
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("migration drill did not finish through the normal verified restore worker")
}

func TestFailedDrillPublishesOneDurableNotification(t *testing.T) {
	repo := newDrillRepo(t)
	notifications := &fakeNotificationPublisher{}
	svc := drills.NewService(drills.Deps{
		Repo: repo, Plans: fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}},
		Snapshots: fakeSnapshots{}, Restores: fakeRestores{done: make(chan struct{})},
		Notifications: notifications, Clock: fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}, BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())

	drill, err := svc.Run(context.Background(), "plan-1", "", "", "missing-snapshot", true)
	if err != nil {
		t.Fatal(err)
	}
	if drill.Status != drills.StatusFailed || drill.NotificationState != drills.NotificationSent {
		t.Fatalf("failed drill notification state = %+v", drill)
	}
	if drill.NotificationSentAt.IsZero() || len(notifications.events) != 1 {
		t.Fatalf("sent_at=%s events=%d", drill.NotificationSentAt, len(notifications.events))
	}
	event := notifications.events[0]
	if event.EventType != "data_backup_manager.backup_health.notification.v1" || event.Payload["category"] != "backup-recovery-needs-attention" || event.Payload["dedupe_key"] != "recovery-drill-failed:"+drill.ID {
		t.Fatalf("event does not carry one deduplicated drill failure notice: %+v", event)
	}
	if _, err := svc.Run(context.Background(), "plan-1", "", "", "missing-snapshot", true); err != nil {
		t.Fatal(err)
	}
	if len(notifications.events) != 1 {
		t.Fatalf("idempotent replay published %d events, want one", len(notifications.events))
	}
}

func TestFailedDrillNotificationRetriesFromDurableOutbox(t *testing.T) {
	repo := newDrillRepo(t)
	notifications := &fakeNotificationPublisher{fail: 1}
	svc := drills.NewService(drills.Deps{
		Repo: repo, Plans: fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}},
		Snapshots: fakeSnapshots{}, Restores: fakeRestores{done: make(chan struct{})},
		Notifications: notifications, Clock: fixedClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}, BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())

	drill, err := svc.Run(context.Background(), "plan-1", "", "", "retry-notification", true)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := repo.Get(context.Background(), drill.ID); err != nil || got.NotificationState != drills.NotificationPending {
		t.Fatalf("notification after publish failure = %+v, err=%v", got, err)
	}
	if err := svc.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), drill.ID)
	if err != nil || got.NotificationState != drills.NotificationSent || len(notifications.events) != 2 {
		t.Fatalf("retried notification = %+v events=%d err=%v", got, len(notifications.events), err)
	}
	if notifications.events[0].Payload["dedupe_key"] != notifications.events[1].Payload["dedupe_key"] {
		t.Fatalf("retry did not preserve dedupe identity: %v vs %v", notifications.events[0].Payload["dedupe_key"], notifications.events[1].Payload["dedupe_key"])
	}
	if !notifications.events[0].Occurred.Equal(notifications.events[1].Occurred) {
		t.Fatalf("retry changed event occurrence and therefore event identity: %s vs %s", notifications.events[0].Occurred, notifications.events[1].Occurred)
	}
}

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

func TestPreviewBlocksWhenDestinationReadinessIsNotProven(t *testing.T) {
	svc := drills.NewService(drills.Deps{
		Repo:        newDrillRepo(t),
		Plans:       fakePlans{plan: drills.Plan{ID: "plan-1", TargetIDs: []string{"target-1"}, DestinationIDs: []string{"dest-1"}, Enabled: true}},
		Snapshots:   fakeSnapshots{snapshot: drills.Snapshot{ID: "snapshot-1"}, ok: true},
		Readiness:   fixedDrillReadiness{ready: false, reason: "volume identity or writable state is not proven"},
		BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())

	preview, err := svc.Preview(context.Background(), "plan-1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Eligible || !strings.Contains(preview.Reason, "volume identity") || preview.SnapshotID != "snapshot-1" {
		t.Fatalf("preview = %+v, want snapshot present but blocked by readiness", preview)
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

func TestRunDueRespectsExplicitManualSchedule(t *testing.T) {
	repo := newDrillRepo(t)
	svc := drills.NewService(drills.Deps{
		Repo:      repo,
		Plans:     fakePlans{plan: drills.Plan{ID: "manual-plan", Enabled: true, DrillSchedule: "manual", TargetIDs: []string{"target"}, DestinationIDs: []string{"destination"}}},
		Snapshots: fakeSnapshots{}, Restores: fakeRestores{done: make(chan struct{})},
		Clock: fixedClock{now: time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC)}, BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())
	if err := svc.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.List(context.Background(), "manual-plan", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("manual plan auto-created %d drills", len(rows))
	}
}

func TestRunDueSchedulesEveryTargetDestinationUnitAndIsIdempotentWithinInterval(t *testing.T) {
	repo := newDrillRepo(t)
	now := time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC)
	plan := drills.Plan{
		ID: "weekly-protected", Enabled: true, DrillSchedule: "168h",
		TargetIDs:      []string{"target-a", "target-b"},
		DestinationIDs: []string{"destination-offhost", "destination-local"},
	}
	svc := drills.NewService(drills.Deps{
		Repo: repo, Plans: fakePlans{plan: plan}, Snapshots: fakeSnapshots{},
		Restores: fakeRestores{done: make(chan struct{})}, Clock: fixedClock{now: now},
		BaseContext: context.Background(),
	})
	defer svc.Shutdown(context.Background())

	if err := svc.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	first, err := repo.List(context.Background(), plan.ID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(plan.TargetIDs)*len(plan.DestinationIDs) {
		t.Fatalf("scheduled %d units, want full target × destination matrix of %d: %+v", len(first), len(plan.TargetIDs)*len(plan.DestinationIDs), first)
	}
	seen := map[string]bool{}
	for _, drill := range first {
		if !drill.Scheduled {
			t.Fatalf("unit %s × %s was not marked scheduled", drill.TargetID, drill.DestinationID)
		}
		seen[drill.TargetID+"/"+drill.DestinationID] = true
	}
	for _, targetID := range plan.TargetIDs {
		for _, destinationID := range plan.DestinationIDs {
			if !seen[targetID+"/"+destinationID] {
				t.Errorf("missing scheduled unit %s × %s", targetID, destinationID)
			}
		}
	}

	if err := svc.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	second, err := repo.List(context.Background(), plan.ID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != len(first) {
		t.Fatalf("same schedule interval created %d rows after retry, want stable %d units", len(second), len(first))
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

func (r signalFinishRepo) Finish(ctx context.Context, id string, event drills.Event, errMsg, nextAction string, finishedAt time.Time) error {
	err := r.Repository.Finish(ctx, id, event, errMsg, nextAction, finishedAt)
	select {
	case <-r.finished:
	default:
		close(r.finished)
	}
	return err
}

func (r failVerifiedFinishRepo) Finish(ctx context.Context, id string, event drills.Event, errMsg, nextAction string, finishedAt time.Time) error {
	if event == drills.EventVerify {
		return errors.New("catalog is unavailable")
	}
	err := r.Repository.Finish(ctx, id, event, errMsg, nextAction, finishedAt)
	select {
	case <-r.finished:
	default:
		close(r.finished)
	}
	return err
}

type failMarkRunningRepo struct {
	drills.Repository
	called   chan struct{}
	finished chan struct{}
}

func (r failMarkRunningRepo) Start(context.Context, string, time.Time) error {
	select {
	case <-r.called:
	default:
		close(r.called)
	}
	return errors.New("catalog is unavailable")
}

func (r failMarkRunningRepo) Finish(ctx context.Context, id string, event drills.Event, errMsg, nextAction string, finishedAt time.Time) error {
	err := r.Repository.Finish(ctx, id, event, errMsg, nextAction, finishedAt)
	select {
	case <-r.finished:
	default:
		close(r.finished)
	}
	return err
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
