package world

import (
	"context"
	"sort"
	"time"

	"prompt-manager/internal/heartbeat"
)

// ScheduleLister is the scheduler surface the feed needs.
type ScheduleLister interface {
	ListScheduled() []heartbeat.ScheduledRun
}

// RunLister is the registry surface the snapshot needs.
type RunLister interface {
	ListActive() []heartbeat.ActiveRun
}

// LiveSource adapts the heartbeat registry and scheduler into a SnapshotSource.
type LiveSource struct {
	Runs     RunLister
	Schedule ScheduleLister
}

// ActiveRuns implements SnapshotSource.
func (s LiveSource) ActiveRuns() []ActiveRun {
	if s.Runs == nil {
		return nil
	}
	active := s.Runs.ListActive()
	out := make([]ActiveRun, 0, len(active))
	for _, run := range active {
		out = append(out, ActiveRun{TeamID: run.TeamID, AgentID: run.AgentID, RunID: run.RunID, StartedAt: run.StartedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AgentID < out[j].AgentID })
	return out
}

// UpcomingHeartbeats implements SnapshotSource.
func (s LiveSource) UpcomingHeartbeats() []Upcoming {
	if s.Schedule == nil {
		return nil
	}
	return upcomingFrom(s.Schedule.ListScheduled())
}

func upcomingFrom(runs []heartbeat.ScheduledRun) []Upcoming {
	out := make([]Upcoming, 0, len(runs))
	for _, run := range runs {
		if run.NextRun.IsZero() {
			continue
		}
		out = append(out, Upcoming{TeamID: run.TeamID, AgentID: run.AgentID, ScheduledAt: run.NextRun})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScheduledAt.Equal(out[j].ScheduledAt) {
			return out[i].AgentID < out[j].AgentID
		}
		return out[i].ScheduledAt.Before(out[j].ScheduledAt)
	})
	return out
}

// ScheduleWatcher publishes HEARTBEAT_UPCOMING when a heartbeat's next run
// enters the horizon or changes, and HEARTBEAT_CANCELLED when a scheduled
// heartbeat disappears. It polls the scheduler because the scheduler has no
// change hook; the interval is a lever of the caller.
type ScheduleWatcher struct {
	hub      *Hub
	schedule ScheduleLister
	horizon  time.Duration
	known    map[string]time.Time
	now      func() time.Time
}

// NewScheduleWatcher creates a watcher announcing runs within horizon.
func NewScheduleWatcher(hub *Hub, schedule ScheduleLister, horizon time.Duration) *ScheduleWatcher {
	return &ScheduleWatcher{hub: hub, schedule: schedule, horizon: horizon, known: map[string]time.Time{}, now: func() time.Time { return time.Now().UTC() }}
}

// Tick compares the scheduler with what was announced and publishes the differences.
func (w *ScheduleWatcher) Tick() {
	current := map[string]heartbeat.ScheduledRun{}
	for _, run := range w.schedule.ListScheduled() {
		current[run.TeamID+"/"+run.AgentID] = run
	}
	now := w.now()
	for key, previous := range w.known {
		run, still := current[key]
		if !still || run.NextRun.IsZero() {
			delete(w.known, key)
			w.hub.Publish(Event{Kind: KindHeartbeatCancelled, TeamID: teamOf(key), AgentID: agentOf(key), At: now, ScheduledAt: previous})
		}
	}
	for key, run := range current {
		if run.NextRun.IsZero() || run.NextRun.Sub(now) > w.horizon {
			continue
		}
		if previous, seen := w.known[key]; seen && previous.Equal(run.NextRun) {
			continue
		}
		w.known[key] = run.NextRun
		w.hub.Publish(Event{Kind: KindHeartbeatUpcoming, TeamID: run.TeamID, AgentID: run.AgentID, At: now, ScheduledAt: run.NextRun})
	}
}

// Run ticks until ctx ends.
func (w *ScheduleWatcher) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.Tick()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Tick()
		}
	}
}

func teamOf(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			return key[:i]
		}
	}
	return key
}

func agentOf(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			return key[i+1:]
		}
	}
	return ""
}
