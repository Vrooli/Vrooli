package world

import (
	"context"
	"testing"
	"time"

	"prompt-manager/internal/heartbeat"
)

type fakeSchedule struct{ runs []heartbeat.ScheduledRun }

func (f *fakeSchedule) ListScheduled() []heartbeat.ScheduledRun { return f.runs }

func TestScheduleWatcherAnnouncesChangesAndCancellations(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	hub := NewHub(16, 16, nil)
	sched := &fakeSchedule{runs: []heartbeat.ScheduledRun{
		{TeamID: "t", AgentID: "soon", NextRun: now.Add(10 * time.Minute)},
		{TeamID: "t", AgentID: "later", NextRun: now.Add(48 * time.Hour)},
	}}
	w := NewScheduleWatcher(hub, sched, time.Hour)
	w.now = func() time.Time { return now }

	w.Tick()
	replay, _ := hub.Subscribe(context.Background(), 0)
	if len(replay) != 1 || replay[0].Kind != KindHeartbeatUpcoming || replay[0].AgentID != "soon" {
		t.Fatalf("only the run inside the horizon is announced: %+v", replay)
	}

	w.Tick()
	replay, _ = hub.Subscribe(context.Background(), 0)
	if len(replay) != 1 {
		t.Fatalf("an unchanged schedule must not re-announce: %d events", len(replay))
	}

	sched.runs[0].NextRun = now.Add(20 * time.Minute)
	w.Tick()
	replay, _ = hub.Subscribe(context.Background(), 0)
	if len(replay) != 2 || replay[1].ScheduledAt != now.Add(20*time.Minute) {
		t.Fatalf("a moved next run is announced again: %+v", replay)
	}

	sched.runs = sched.runs[1:]
	w.Tick()
	replay, _ = hub.Subscribe(context.Background(), 0)
	if len(replay) != 3 || replay[2].Kind != KindHeartbeatCancelled || replay[2].AgentID != "soon" || replay[2].TeamID != "t" {
		t.Fatalf("a vanished schedule is cancelled: %+v", replay)
	}
}

func TestLiveSourceSortsSnapshotLists(t *testing.T) {
	now := time.Now()
	src := LiveSource{Schedule: &fakeSchedule{runs: []heartbeat.ScheduledRun{
		{TeamID: "t", AgentID: "z", NextRun: now.Add(time.Hour)},
		{TeamID: "t", AgentID: "a", NextRun: now.Add(time.Minute)},
		{TeamID: "t", AgentID: "never"},
	}}}
	up := src.UpcomingHeartbeats()
	if len(up) != 2 || up[0].AgentID != "a" {
		t.Fatalf("upcoming must be sorted by time and drop zero times: %+v", up)
	}
	if src.ActiveRuns() != nil {
		t.Fatal("nil run lister yields nil")
	}
}
