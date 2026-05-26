package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	runsH "data-backup-manager/handlers/runs"

	"data-backup-manager/internal/clock"
	destint "data-backup-manager/internal/destinations"
	plansint "data-backup-manager/internal/plans"
	restoresint "data-backup-manager/internal/restores"
	runsint "data-backup-manager/internal/runs"
	schedint "data-backup-manager/internal/scheduler"
	targetsint "data-backup-manager/internal/targets"
)

// This file holds the composition-root adapters that satisfy the narrow
// reader/effect seams the runs domain and the scheduler declare, backed by the
// concrete sibling services. Keeping the adapters here (package main) means the
// domains stay decoupled — runs and scheduler depend only on their own
// interfaces, and main is the single place that knows the whole graph.

// planLookup adapts plans.Service to runs.PlanLookup.
type planLookup struct{ svc plansint.Service }

func (a planLookup) PlanForRun(ctx context.Context, planID string) (runsint.PlanForRun, error) {
	p, err := a.svc.Get(ctx, planID)
	if err != nil {
		return runsint.PlanForRun{}, err
	}
	return runsint.PlanForRun{
		ID:             p.ID,
		TargetIDs:      p.TargetIDs,
		DestinationIDs: p.DestinationIDs,
		KeepLatest:     int(p.KeepLatest),
	}, nil
}

// targetLookup adapts targets.Service to runs.TargetLookup.
type targetLookup struct{ svc targetsint.Service }

func (a targetLookup) TargetForRun(ctx context.Context, targetID string) (runsint.TargetForRun, error) {
	t, err := a.svc.Get(ctx, targetID)
	if err != nil {
		return runsint.TargetForRun{}, err
	}
	return runsint.TargetForRun{ID: t.ID, Kind: t.SourceKind, Locator: t.Locator}, nil
}

// destinationLookup adapts destinations.Service to runs.DestinationLookup.
type destinationLookup struct{ svc destint.Service }

func (a destinationLookup) DestinationForRun(ctx context.Context, destID string) (runsint.DestinationForRun, error) {
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return runsint.DestinationForRun{}, err
	}
	return runsint.DestinationForRun{ID: d.ID, Name: d.Name}, nil
}

func (a destinationLookup) WouldBlock(ctx context.Context, destID string, pendingBytes int64) (bool, string, error) {
	return a.svc.WouldBlock(ctx, destID, pendingBytes)
}

// restoreTargetLookup adapts targets.Service to restores.TargetLookup.
type restoreTargetLookup struct{ svc targetsint.Service }

func (a restoreTargetLookup) TargetForRestore(ctx context.Context, targetID string) (restoresint.TargetForRestore, error) {
	t, err := a.svc.Get(ctx, targetID)
	if err != nil {
		return restoresint.TargetForRestore{}, err
	}
	return restoresint.TargetForRestore{ID: t.ID, Kind: t.SourceKind, Locator: t.Locator}, nil
}

// restoreDestinationLookup adapts destinations.Service to restores.DestinationLookup.
type restoreDestinationLookup struct{ svc destint.Service }

func (a restoreDestinationLookup) DestinationForRestore(ctx context.Context, destID string) (restoresint.DestinationForRestore, error) {
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return restoresint.DestinationForRestore{}, err
	}
	return restoresint.DestinationForRestore{ID: d.ID, Name: d.Name}, nil
}

// verifiedLookup adapts restores.Service to the runs handler's VerifiedLookup
// seam: it rolls the latest successful verify per target into a map so
// ListTargetStatus can report proven-restorable posture in one call. This is
// the composition seam that keeps the runs domain from importing restores.
type verifiedLookup struct{ svc restoresint.Service }

func (a verifiedLookup) LastVerifiedByTarget(ctx context.Context) (map[string]runsH.VerifiedInfo, error) {
	statuses, err := a.svc.LastVerifiedByTarget(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[string]runsH.VerifiedInfo, len(statuses))
	for _, s := range statuses {
		out[s.TargetID] = runsH.VerifiedInfo{LastVerifiedAt: s.LastVerifiedAt, SnapshotID: s.SnapshotID}
	}
	return out, nil
}

// runTrigger adapts runs.Service to scheduler.RunTrigger (scheduler-fired runs).
type runTrigger struct{ svc runsint.Service }

func (a runTrigger) TriggerRun(ctx context.Context, planID string) error {
	_, err := a.svc.TriggerRun(ctx, planID, runsint.TriggerScheduler)
	return err
}

// planSource adapts plans.Service to scheduler.PlanSource.
type planSource struct{ svc plansint.Service }

func (a planSource) SchedulablePlans(ctx context.Context) ([]schedint.DuePlan, error) {
	ps, err := a.svc.SchedulablePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schedint.DuePlan, 0, len(ps))
	for _, p := range ps {
		out = append(out, schedint.DuePlan{ID: p.ID, Schedule: p.Schedule, Enabled: p.Enabled})
	}
	return out, nil
}

// logEventSink is the production runs.EventSink: it logs backup outcomes for
// platform monitoring (infra-health / system-monitor) to pick up. A routed
// notification sink (OT-P1-006) replaces this later.
type logEventSink struct{ logger *log.Logger }

func (s logEventSink) BackupOutcome(_ context.Context, ev runsint.RunOutcomeEvent) {
	s.logger.Printf("backup-outcome run=%s plan=%s status=%s succeeded=%d failed=%d blocked=%d",
		ev.RunID, ev.PlanID, ev.Status, ev.Succeeded, ev.Failed, ev.Blocked)
}

// BackupPostureDegraded satisfies health.PostureEventSink: it logs a degraded
// backup posture so platform monitoring can flag overdue/failed backups.
func (s logEventSink) BackupPostureDegraded(_ context.Context, detail string) {
	s.logger.Printf("backup-posture degraded: %s", detail)
}

// overdueAfter returns the age past which a target's last success counts as
// overdue. Configurable via DBM_OVERDUE_AFTER (a Go duration); default 36h.
func overdueAfter() time.Duration {
	const def = 36 * time.Hour
	if raw := strings.TrimSpace(os.Getenv("DBM_OVERDUE_AFTER")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return def
}

// backupPosture adapts runs.Service to health.BackupPosture. A target is
// "overdue or failed" when its last run failed/partial-failed, it has never
// succeeded, or its last success is older than overdueAfter.
type backupPosture struct {
	runs         runsint.Service
	clock        clock.Clock
	overdueAfter time.Duration
}

func (a backupPosture) OverdueOrFailed(ctx context.Context) (bool, string, error) {
	statuses, err := a.runs.ListTargetStatus(ctx, nil)
	if err != nil {
		return false, "", err
	}
	now := a.clock.Now()
	bad := 0
	for _, s := range statuses {
		switch s.LastRunStatus {
		case runsint.RunFailed, runsint.RunPartialFailed:
			bad++
			continue
		}
		if s.LastSuccessAt.IsZero() || now.Sub(s.LastSuccessAt) > a.overdueAfter {
			bad++
		}
	}
	if bad > 0 {
		return true, fmt.Sprintf("%d of %d target(s) overdue or last run failed", bad, len(statuses)), nil
	}
	return false, "", nil
}

// startScheduler runs the in-process scheduler on a ticker in a background
// goroutine. Interval comes from DBM_SCHEDULER_INTERVAL (a Go duration);
// default 60s. Setting it to "0" disables the ticker (the manual/on-demand
// trigger path still works via the RunsService). With no schedulable plans,
// each tick is a no-op.
func startScheduler(ctx context.Context, sched *schedint.Scheduler, logger *log.Logger) {
	interval := 60 * time.Second
	if raw := strings.TrimSpace(os.Getenv("DBM_SCHEDULER_INTERVAL")); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			logger.Printf("scheduler: invalid DBM_SCHEDULER_INTERVAL %q, using %s", raw, interval)
		} else if d == 0 {
			logger.Printf("scheduler: disabled via DBM_SCHEDULER_INTERVAL=0")
			return
		} else {
			interval = d
		}
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sched.Tick(ctx); err != nil {
					logger.Printf("scheduler tick: %v", err)
				}
			}
		}
	}()
}
