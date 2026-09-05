package monitoring

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	domainmonitoring "network-manager/internal/monitoring"
	domainsnapshot "network-manager/internal/snapshot"

	"github.com/vrooli/api-core/scheduletest"

	monitoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/monitoring"
)

func TestHandlerUpsertMonitoringScheduleMapsProto(t *testing.T) {
	// [REQ:NM-P1-007] Monitoring API handlers stay thin and delegate
	// baseline-anchored schedule validation to the domain service.
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	service := domainmonitoring.NewService(domainmonitoring.Config{
		Repo: &fakeRepository{schedule: map[string]domainmonitoring.Schedule{}},
		Snapshots: &fakeSnapshotService{snapshots: map[string]domainsnapshot.Snapshot{
			"baseline-1": {ID: "baseline-1", Status: "baseline", Profile: "home"},
		}},
		Clock: scheduletest.New(now),
	})
	h := handler{service: service}

	resp, err := h.UpsertMonitoringSchedule(context.Background(), connect.NewRequest(&monitoringv1.UpsertMonitoringScheduleRequest{
		Schedule: &monitoringv1.MonitoringSchedule{
			Name:                 "Home baseline watch",
			Profile:              "home",
			BaselineSnapshotId:   "baseline-1",
			IntervalMinutes:      60,
			Enabled:              true,
			LatencyThresholdMs:   100,
			UnavailableThreshold: 1,
		},
	}))

	require.NoError(t, err)
	require.Equal(t, "schedule-1", resp.Msg.GetSchedule().GetId())
	require.Equal(t, "baseline-1", resp.Msg.GetSchedule().GetBaselineSnapshotId())
	require.Equal(t, int32(60), resp.Msg.GetSchedule().GetIntervalMinutes())
	require.NotEmpty(t, resp.Msg.GetSchedule().GetEffects())
}

type fakeRepository struct {
	schedule map[string]domainmonitoring.Schedule
}

func (r *fakeRepository) ListSchedules(_ context.Context, includeDisabled bool) ([]domainmonitoring.Schedule, error) {
	out := []domainmonitoring.Schedule{}
	for _, schedule := range r.clock {
		if includeDisabled || schedule.Enabled {
			out = append(out, schedule)
		}
	}
	return out, nil
}

func (r *fakeRepository) GetSchedule(_ context.Context, id string) (domainmonitoring.Schedule, error) {
	schedule, ok := r.clock[id]
	if !ok {
		return domainmonitoring.Schedule{}, domainmonitoring.ErrNotFound
	}
	return schedule, nil
}

func (r *fakeRepository) UpsertSchedule(_ context.Context, schedule domainmonitoring.Schedule) (domainmonitoring.Schedule, error) {
	if schedule.ID == "" {
		schedule.ID = "schedule-1"
	}
	r.clock[schedule.ID] = schedule
	return schedule, nil
}

func (r *fakeRepository) SaveRun(_ context.Context, run domainmonitoring.Run) (domainmonitoring.Run, error) {
	return run, nil
}

func (r *fakeRepository) SaveAlert(_ context.Context, alert domainmonitoring.Alert) (domainmonitoring.Alert, error) {
	return alert, nil
}

func (r *fakeRepository) ListAlerts(context.Context, string, bool) ([]domainmonitoring.Alert, error) {
	return nil, nil
}

type fakeSnapshotService struct {
	snapshots map[string]domainsnapshot.Snapshot
}

func (s *fakeSnapshotService) Run(_ context.Context, profile string, _ bool) (domainsnapshot.Snapshot, error) {
	return domainsnapshot.Snapshot{ID: "snapshot-2", Profile: profile}, nil
}

func (s *fakeSnapshotService) Get(_ context.Context, id string) (domainsnapshot.Snapshot, error) {
	snapshot, ok := s.snapshots[id]
	if !ok {
		return domainsnapshot.Snapshot{}, domainsnapshot.ErrNotFound
	}
	return snapshot, nil
}
