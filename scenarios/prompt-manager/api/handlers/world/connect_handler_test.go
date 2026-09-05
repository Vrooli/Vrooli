package world

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	worldv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/world"
	worldconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/world/world_v1connect"

	"prompt-manager/internal/heartbeat"
	domain "prompt-manager/internal/world"
)

func newServer(t *testing.T) (worldconnect.WorldServiceClient, *domain.Hub) {
	t.Helper()
	store := domain.NewStore(t.TempDir())
	hub := domain.NewHub(16, 16, nil)
	path, handler := NewConnectMount(store, hub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return worldconnect.NewWorldServiceClient(srv.Client(), srv.URL), hub
}

func TestConfigAndLayoutOverConnect(t *testing.T) {
	client, _ := newServer(t)
	ctx := context.Background()
	got, err := client.GetWorldConfig(ctx, connect.NewRequest(&worldv1.GetWorldConfigRequest{}))
	if err != nil || got.Msg.GetScene() != "park" {
		t.Fatalf("default config: %v %+v", err, got)
	}
	set, err := client.SetWorldConfig(ctx, connect.NewRequest(&worldv1.SetWorldConfigRequest{Config: &worldv1.WorldConfig{Scene: "office", QualityProfile: "low", PeriodMode: "dusk", Scale: 1}}))
	if err != nil || set.Msg.GetScene() != "office" || set.Msg.GetUpdatedAt() == "" {
		t.Fatalf("set config: %v %+v", err, set)
	}
	if _, err := client.SetWorldConfig(ctx, connect.NewRequest(&worldv1.SetWorldConfigRequest{Config: &worldv1.WorldConfig{Scene: "moon", QualityProfile: "low", PeriodMode: "dusk", Scale: 1}})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid config must be InvalidArgument, got %v", err)
	}
	rot := 0.5
	layout, err := client.SetLayout(ctx, connect.NewRequest(&worldv1.SetLayoutRequest{Layout: &worldv1.WorldLayout{Scene: "office", Overrides: []*worldv1.LayoutOverride{{PlaceId: "room:x", Position: &worldv1.Vec2{X: 1, Z: 2}, Rotation: &rot}}}}))
	if err != nil || len(layout.Msg.GetOverrides()) != 1 || layout.Msg.GetOverrides()[0].GetRotation() != rot {
		t.Fatalf("set layout: %v %+v", err, layout)
	}
	back, err := client.GetLayout(ctx, connect.NewRequest(&worldv1.GetLayoutRequest{Scene: "office"}))
	if err != nil || back.Msg.GetOverrides()[0].GetPosition().GetZ() != 2 {
		t.Fatalf("get layout: %v %+v", err, back)
	}
}

func TestStreamWorldFeedSendsSnapshotThenLiveEvents(t *testing.T) {
	client, hub := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.StreamWorldFeed(ctx, connect.NewRequest(&worldv1.StreamWorldFeedRequest{}))
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("no snapshot: %v", stream.Err())
	}
	if stream.Msg().GetKind() != worldv1.WorldEventKind_WORLD_EVENT_KIND_SNAPSHOT {
		t.Fatalf("first event must be the snapshot, got %v", stream.Msg().GetKind())
	}
	hub.RunStarted(heartbeat.ActiveRun{TeamID: "t", AgentID: "a", RunID: "r1", StartedAt: time.Now()})
	hub.RunEnded(heartbeat.ActiveRun{TeamID: "t", AgentID: "a", RunID: "r1"}, time.Now(), true, "boom")
	want := []worldv1.WorldEventKind{worldv1.WorldEventKind_WORLD_EVENT_KIND_RUN_STARTED, worldv1.WorldEventKind_WORLD_EVENT_KIND_RUN_FAILED}
	for i, kind := range want {
		if !stream.Receive() {
			t.Fatalf("event %d missing: %v", i, stream.Err())
		}
		msg := stream.Msg()
		if msg.GetKind() != kind || msg.GetRunId() != "r1" || msg.GetSeq() != uint64(i+1) {
			t.Fatalf("event %d: got %+v", i, msg)
		}
		if kind == worldv1.WorldEventKind_WORLD_EVENT_KIND_RUN_FAILED && msg.GetMessage() != "boom" {
			t.Fatalf("run.failed must carry the error, got %q", msg.GetMessage())
		}
	}
	cancel()
}

func TestEventToProtoFormatsTimes(t *testing.T) {
	at := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	p := EventToProto(domain.Event{Kind: domain.KindHeartbeatUpcoming, Seq: 7, At: at, TeamID: "t", ScheduledAt: at.Add(time.Hour), Upcoming: []domain.Upcoming{{TeamID: "t", AgentID: "a", ScheduledAt: at}}})
	if p.GetSeq() != 7 || p.GetScheduledAt() != "2026-09-02T13:00:00Z" || len(p.GetUpcoming()) != 1 {
		t.Fatalf("unexpected proto %+v", p)
	}
}
