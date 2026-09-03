package world

import (
	"context"
	"testing"
	"time"

	"prompt-manager/internal/heartbeat"
)

type fakeSource struct {
	runs     []ActiveRun
	upcoming []Upcoming
}

func (f fakeSource) ActiveRuns() []ActiveRun        { return f.runs }
func (f fakeSource) UpcomingHeartbeats() []Upcoming { return f.upcoming }

func TestHubPublishOrdersSeqAndReplaysSince(t *testing.T) {
	hub := NewHub(3, 8, nil)
	for i := 0; i < 5; i++ {
		hub.Publish(Event{Kind: KindAgentMessage, AgentID: "a", Message: string(rune('a' + i))})
	}
	replay, _ := hub.Subscribe(context.Background(), 2)
	if len(replay) != 3 {
		t.Fatalf("ring of 3 should replay 3 newest events, got %d", len(replay))
	}
	if replay[0].Seq != 3 || replay[2].Seq != 5 {
		t.Fatalf("replay must be ordered by seq: %+v", replay)
	}
	for _, e := range replay {
		if e.At.IsZero() {
			t.Fatal("published events must be stamped")
		}
	}
	replay, _ = hub.Subscribe(context.Background(), 5)
	if len(replay) != 0 {
		t.Fatalf("since latest seq must replay nothing, got %d", len(replay))
	}
}

func TestHubDeliversLiveAndCleansUpOnCancel(t *testing.T) {
	hub := NewHub(8, 8, nil)
	ctx, cancel := context.WithCancel(context.Background())
	_, live := hub.Subscribe(ctx, 0)
	if hub.SubscriberCount() != 1 {
		t.Fatalf("expected one subscriber, got %d", hub.SubscriberCount())
	}
	hub.RunStarted(heartbeat.ActiveRun{TeamID: "t", AgentID: "a", RunID: "r", StartedAt: time.Now()})
	select {
	case e := <-live:
		if e.Kind != KindRunStarted || e.RunID != "r" || e.Seq != 1 {
			t.Fatalf("unexpected event %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("no live event")
	}
	cancel()
	deadline := time.Now().Add(time.Second)
	for hub.SubscriberCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hub.SubscriberCount() != 0 {
		t.Fatal("subscriber must be removed after cancel")
	}
	if _, ok := <-live; ok {
		t.Fatal("channel must be closed after cancel")
	}
}

func TestHubDropsSlowSubscriber(t *testing.T) {
	hub := NewHub(8, 1, nil)
	_, live := hub.Subscribe(context.Background(), 0)
	hub.Publish(Event{Kind: KindAgentMessage})
	hub.Publish(Event{Kind: KindAgentMessage})
	if hub.SubscriberCount() != 0 {
		t.Fatal("a subscriber that fell behind must be dropped")
	}
	<-live
	if _, ok := <-live; ok {
		t.Fatal("dropped subscriber channel must be closed")
	}
}

func TestRunEndedMapsOutcome(t *testing.T) {
	hub := NewHub(8, 8, nil)
	run := heartbeat.ActiveRun{TeamID: "t", AgentID: "a", RunID: "r"}
	hub.RunEnded(run, time.Now(), true, "exit 1")
	hub.RunEnded(run, time.Now(), false, "")
	replay, _ := hub.Subscribe(context.Background(), 0)
	if replay[0].Kind != KindRunFailed || replay[0].Message != "exit 1" {
		t.Fatalf("failed run must map to run.failed with the error: %+v", replay[0])
	}
	if replay[1].Kind != KindRunFinished {
		t.Fatalf("clean run must map to run.finished: %+v", replay[1])
	}
}

func TestSnapshotCarriesLiveLists(t *testing.T) {
	at := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	hub := NewHub(8, 8, fakeSource{runs: []ActiveRun{{TeamID: "t", AgentID: "a", RunID: "r", StartedAt: at}}, upcoming: []Upcoming{{TeamID: "t", AgentID: "b", ScheduledAt: at.Add(time.Hour)}}})
	hub.Publish(Event{Kind: KindAgentMessage})
	snap := hub.Snapshot()
	if snap.Kind != KindSnapshot || snap.Seq != 1 || len(snap.ActiveRuns) != 1 || len(snap.Upcoming) != 1 {
		t.Fatalf("unexpected snapshot %+v", snap)
	}
}
