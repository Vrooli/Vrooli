package monitoring

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"network-manager/internal/clock"
	"network-manager/internal/snapshot"
)

type SnapshotService interface {
	Run(ctx context.Context, profile string, dryRun bool) (snapshot.Snapshot, error)
	Get(ctx context.Context, id string) (snapshot.Snapshot, error)
}

type Service struct {
	repo      Repository
	snapshots SnapshotService
	clock     clock.Clock
}

type Config struct {
	Repo      Repository
	Snapshots SnapshotService
	Clock     clock.Clock
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, snapshots: cfg.Snapshots, clock: cfg.Clock}
	if s.clock == nil {
		s.clock = clock.System{}
	}
	return s
}

func (s *Service) ListSchedules(ctx context.Context, includeDisabled bool) ([]Schedule, error) {
	return s.repo.ListSchedules(ctx, includeDisabled)
}

func (s *Service) UpsertSchedule(ctx context.Context, schedule Schedule) (Schedule, error) {
	normalized, err := s.normalizeSchedule(ctx, schedule)
	if err != nil {
		return Schedule{}, err
	}
	return s.repo.UpsertSchedule(ctx, normalized)
}

func (s *Service) RunCheck(ctx context.Context, scheduleID string, dryRun bool) (Run, error) {
	scheduleID = strings.TrimSpace(scheduleID)
	if scheduleID == "" {
		return Run{}, fmt.Errorf("schedule_id is required")
	}
	schedule, err := s.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		return Run{}, err
	}
	now := s.clock.Now().UTC()
	if !schedule.Enabled {
		run := Run{
			ID:         uuid.NewString(),
			ScheduleID: schedule.ID,
			Status:     "disabled",
			Summary:    "Monitoring schedule is disabled.",
			Effects:    []string{"No snapshot was captured because this schedule is disabled."},
			CreatedAt:  now,
		}
		if dryRun {
			run.ID = "monitoring-dry-run"
			return run, nil
		}
		return s.repo.SaveRun(ctx, run)
	}
	if s.snapshots == nil {
		return Run{}, fmt.Errorf("snapshot service is required")
	}
	baseline, err := s.snapshots.Get(ctx, schedule.BaselineSnapshotID)
	if err != nil {
		return Run{}, fmt.Errorf("get baseline snapshot: %w", err)
	}
	current, err := s.snapshots.Run(ctx, schedule.Profile, dryRun)
	if err != nil {
		return Run{}, fmt.Errorf("run monitoring snapshot: %w", err)
	}
	alerts := detectRegressions(schedule, baseline, current, now)
	run := Run{
		ID:                 uuid.NewString(),
		ScheduleID:         schedule.ID,
		SnapshotID:         current.ID,
		Status:             "healthy",
		Summary:            "No regressions detected against the configured baseline.",
		RegressionDetected: len(alerts) > 0,
		Alerts:             alerts,
		Effects: []string{
			fmt.Sprintf("Compared snapshot %s against baseline %s.", current.ID, baseline.ID),
			"Monitoring is advisory; persistent network changes still require explicit approval.",
		},
		CreatedAt: now,
	}
	if len(alerts) > 0 {
		run.Status = "regression_detected"
		run.Summary = fmt.Sprintf("%d regression alert(s) detected.", len(alerts))
	}
	if dryRun {
		run.ID = "monitoring-dry-run"
		run.Status = "dry_run"
		run.Effects = append(run.Effects, "Dry run did not persist the monitoring run or alerts.")
		return run, nil
	}
	saved, err := s.repo.SaveRun(ctx, run)
	if err != nil {
		return Run{}, err
	}
	saved.Alerts = make([]Alert, 0, len(alerts))
	for _, alert := range alerts {
		alert.ScheduleID = schedule.ID
		alert.SnapshotID = current.ID
		savedAlert, err := s.repo.SaveAlert(ctx, alert)
		if err != nil {
			return Run{}, err
		}
		saved.Alerts = append(saved.Alerts, savedAlert)
	}
	return saved, nil
}

func (s *Service) ListAlerts(ctx context.Context, scheduleID string, openOnly bool) ([]Alert, error) {
	return s.repo.ListAlerts(ctx, strings.TrimSpace(scheduleID), openOnly)
}

func (s *Service) normalizeSchedule(ctx context.Context, schedule Schedule) (Schedule, error) {
	schedule.ID = strings.TrimSpace(schedule.ID)
	schedule.Name = strings.TrimSpace(schedule.Name)
	if schedule.Name == "" {
		return Schedule{}, fmt.Errorf("name is required")
	}
	schedule.Profile = strings.TrimSpace(schedule.Profile)
	if schedule.Profile == "" {
		schedule.Profile = "home"
	}
	schedule.BaselineSnapshotID = strings.TrimSpace(schedule.BaselineSnapshotID)
	if schedule.BaselineSnapshotID == "" {
		return Schedule{}, fmt.Errorf("baseline_snapshot_id is required")
	}
	if s.snapshots != nil {
		if _, err := s.snapshots.Get(ctx, schedule.BaselineSnapshotID); err != nil {
			return Schedule{}, fmt.Errorf("baseline snapshot is required: %w", err)
		}
	}
	if schedule.IntervalMinutes <= 0 {
		schedule.IntervalMinutes = 60
	}
	if schedule.IntervalMinutes < 5 {
		return Schedule{}, fmt.Errorf("interval_minutes must be at least 5")
	}
	if schedule.LatencyThresholdMS <= 0 {
		schedule.LatencyThresholdMS = 100
	}
	if schedule.UnavailableThreshold < 0 {
		return Schedule{}, fmt.Errorf("unavailable_threshold cannot be negative")
	}
	if schedule.UnavailableThreshold == 0 {
		schedule.UnavailableThreshold = 1
	}
	now := s.clock.Now().UTC()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}
	schedule.UpdatedAt = now
	schedule.Effects = []string{
		fmt.Sprintf("Recurring snapshots every %d minute(s) for profile %s.", schedule.IntervalMinutes, schedule.Profile),
		"Regression detection compares future snapshots against the selected baseline.",
	}
	return schedule, nil
}

func detectRegressions(schedule Schedule, baseline, current snapshot.Snapshot, now time.Time) []Alert {
	var alerts []Alert
	if delta, ok := latencyDeltaMS(baseline, current, "dns_lookup_latency"); ok && delta > schedule.LatencyThresholdMS {
		alerts = append(alerts, Alert{
			ID:         uuid.NewString(),
			ScheduleID: schedule.ID,
			SnapshotID: current.ID,
			Severity:   "warning",
			Status:     "open",
			Summary:    "DNS lookup latency regressed against baseline.",
			Evidence: []string{
				fmt.Sprintf("dns_lookup_latency increased by %dms; threshold is %dms.", delta, schedule.LatencyThresholdMS),
			},
			CreatedAt: now,
		})
	}
	baselineUnavailable := countUnavailable(baseline)
	currentUnavailable := countUnavailable(current)
	if currentUnavailable-baselineUnavailable >= schedule.UnavailableThreshold {
		alerts = append(alerts, Alert{
			ID:         uuid.NewString(),
			ScheduleID: schedule.ID,
			SnapshotID: current.ID,
			Severity:   "critical",
			Status:     "open",
			Summary:    "Unavailable or failed probe count increased.",
			Evidence: []string{
				fmt.Sprintf("Unavailable/failed probes changed from %d to %d; threshold is %d.", baselineUnavailable, currentUnavailable, schedule.UnavailableThreshold),
			},
			CreatedAt: now,
		})
	}
	for _, metric := range current.Metrics {
		if metric.Status == "failed" && (metric.Name == "wan_https_reachability" || metric.Name == "resolver_addresses") {
			alerts = append(alerts, Alert{
				ID:         uuid.NewString(),
				ScheduleID: schedule.ID,
				SnapshotID: current.ID,
				Severity:   "critical",
				Status:     "open",
				Summary:    fmt.Sprintf("%s is failing.", metric.Name),
				Evidence:   []string{fmt.Sprintf("%s status is %s in snapshot %s.", metric.Name, metric.Status, current.ID)},
				CreatedAt:  now,
			})
		}
	}
	return alerts
}

func latencyDeltaMS(baseline, current snapshot.Snapshot, name string) (int, bool) {
	base, ok := metricInt(baseline, name)
	if !ok {
		return 0, false
	}
	now, ok := metricInt(current, name)
	if !ok {
		return 0, false
	}
	return now - base, true
}

func metricInt(s snapshot.Snapshot, name string) (int, bool) {
	for _, metric := range s.Metrics {
		if metric.Name != name {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(metric.Value))
		if err != nil {
			return 0, false
		}
		return value, true
	}
	return 0, false
}

func countUnavailable(s snapshot.Snapshot) int {
	count := 0
	for _, metric := range s.Metrics {
		switch metric.Status {
		case "unavailable", "unsupported", "failed":
			count++
		}
	}
	return count
}
