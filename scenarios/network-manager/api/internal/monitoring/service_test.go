package monitoring

import (
	"context"
	"testing"
	"time"

	"network-manager/internal/snapshot"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

func TestServiceScheduleRequiresBaselineSnapshot(t *testing.T) {
	// [REQ:NM-P1-007] Monitoring schedules must be anchored to a real
	// baseline snapshot before recurring regression detection is enabled.
	service := NewService(Config{
		Repo:      newFakeRepository(),
		Snapshots: &fakeSnapshotService{},
		Clock:     scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)),
	})

	_, err := service.UpsertSchedule(context.Background(), Schedule{Name: "watch", BaselineSnapshotID: "missing"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "baseline snapshot is required")
}

func TestServiceRunCheckDetectsAndPersistsRegressions(t *testing.T) {
	// [REQ:NM-P1-007] Recurring checks compare a fresh snapshot to the
	// baseline and persist open alerts when quality regresses.
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	snapshots := &fakeSnapshotService{
		byID: map[string]snapshot.Snapshot{
			"baseline-1": {
				ID:      "baseline-1",
				Status:  "baseline",
				Profile: "home",
				Metrics: []snapshot.Metric{
					{Name: "dns_lookup_latency", Value: "20", Unit: "ms", Status: "healthy"},
					{Name: "wan_https_reachability", Value: "15", Unit: "ms", Status: "healthy"},
				},
			},
		},
		next: snapshot.Snapshot{
			ID:      "snapshot-2",
			Status:  "complete",
			Profile: "home",
			Metrics: []snapshot.Metric{
				{Name: "dns_lookup_latency", Value: "180", Unit: "ms", Status: "degraded"},
				{Name: "wan_https_reachability", Value: "failed", Unit: "ms", Status: "failed"},
			},
		},
	}
	service := NewService(Config{Repo: repo, Snapshots: snapshots, Clock: scheduletest.New(now)})
	schedule, err := service.UpsertSchedule(context.Background(), Schedule{
		Name:                 "Home baseline watch",
		BaselineSnapshotID:   "baseline-1",
		IntervalMinutes:      60,
		Enabled:              true,
		LatencyThresholdMS:   100,
		UnavailableThreshold: 1,
	})
	require.NoError(t, err)

	run, err := service.RunCheck(context.Background(), schedule.ID, false)
	require.NoError(t, err)
	require.Equal(t, "regression_detected", run.Status)
	require.True(t, run.RegressionDetected)
	require.NotEmpty(t, run.Alerts)
	require.Len(t, repo.alerts, len(run.Alerts))
	require.Contains(t, run.Alerts[0].Evidence[0], "dns_lookup_latency increased")
}

func TestServiceDisabledRunIsAdvisory(t *testing.T) {
	// [REQ:NM-P1-007] Disabled schedules do not capture snapshots or emit
	// fake health success; they record the advisory disabled state.
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	repo.schedules["schedule-1"] = Schedule{
		ID:                 "schedule-1",
		Name:               "disabled",
		Profile:            "home",
		BaselineSnapshotID: "baseline-1",
		Enabled:            false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	snapshots := &fakeSnapshotService{byID: map[string]snapshot.Snapshot{"baseline-1": {ID: "baseline-1"}}}
	service := NewService(Config{Repo: repo, Snapshots: snapshots, Clock: scheduletest.New(now)})

	run, err := service.RunCheck(context.Background(), "schedule-1", false)
	require.NoError(t, err)
	require.Equal(t, "disabled", run.Status)
	require.Empty(t, snapshots.runProfiles)
	require.Len(t, repo.runs, 1)
}

type fakeSnapshotService struct {
	byID        map[string]snapshot.Snapshot
	next        snapshot.Snapshot
	runProfiles []string
}

func (s *fakeSnapshotService) Run(_ context.Context, profile string, dryRun bool) (snapshot.Snapshot, error) {
	s.runProfiles = append(s.runProfiles, profile)
	next := s.next
	if dryRun {
		next.ID = "snapshot-dry-run"
		next.Status = "dry_run"
	}
	return next, nil
}

func (s *fakeSnapshotService) Get(_ context.Context, id string) (snapshot.Snapshot, error) {
	if s.byID == nil {
		return snapshot.Snapshot{}, snapshot.ErrNotFound
	}
	snap, ok := s.byID[id]
	if !ok {
		return snapshot.Snapshot{}, snapshot.ErrNotFound
	}
	return snap, nil
}

type fakeRepository struct {
	schedules map[string]Schedule
	runs      []Run
	alerts    []Alert
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{schedules: map[string]Schedule{}}
}

func (r *fakeRepository) ListSchedules(_ context.Context, includeDisabled bool) ([]Schedule, error) {
	out := make([]Schedule, 0, len(r.schedules))
	for _, schedule := range r.schedules {
		if includeDisabled || schedule.Enabled {
			out = append(out, schedule)
		}
	}
	return out, nil
}

func (r *fakeRepository) GetSchedule(_ context.Context, id string) (Schedule, error) {
	schedule, ok := r.schedules[id]
	if !ok {
		return Schedule{}, ErrNotFound
	}
	return schedule, nil
}

func (r *fakeRepository) UpsertSchedule(_ context.Context, schedule Schedule) (Schedule, error) {
	if schedule.ID == "" {
		schedule.ID = "schedule-1"
	}
	r.schedules[schedule.ID] = schedule
	return schedule, nil
}

func (r *fakeRepository) SaveRun(_ context.Context, run Run) (Run, error) {
	r.runs = append(r.runs, run)
	return run, nil
}

func (r *fakeRepository) SaveAlert(_ context.Context, alert Alert) (Alert, error) {
	r.alerts = append(r.alerts, alert)
	return alert, nil
}

func (r *fakeRepository) ListAlerts(_ context.Context, scheduleID string, openOnly bool) ([]Alert, error) {
	out := []Alert{}
	for _, alert := range r.alerts {
		if scheduleID != "" && alert.ScheduleID != scheduleID {
			continue
		}
		if openOnly && alert.Status != "open" {
			continue
		}
		out = append(out, alert)
	}
	return out, nil
}
