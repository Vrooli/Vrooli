package metricshttp

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type connectTracker struct {
	event metrics.Event
	err   error
}

func (f *connectTracker) TrackEvent(event metrics.Event) error { f.event = event; return f.err }
func TestConnectTrackEventRequiresVariantSlug(t *testing.T) {
	handler := NewConnectHandler(ConnectDependencies{Tracker: &connectTracker{}})
	_, err := handler.TrackEvent(context.Background(), connect.NewRequest(&lpbsv1.TrackEventRequest{EventType: "page_view", SessionId: "session"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestConnectTrackEventMapsSlugAndPayload(t *testing.T) {
	tracker := &connectTracker{}
	handler := NewConnectHandler(ConnectDependencies{Tracker: tracker})
	response, err := handler.TrackEvent(context.Background(), connect.NewRequest(&lpbsv1.TrackEventRequest{EventType: "page_view", VariantSlug: "control", SessionId: "session", EventId: "event"}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetSuccess() || tracker.event.VariantSlug != "control" || tracker.event.SessionID != "session" {
		t.Fatalf("unexpected event: %+v", tracker.event)
	}
}
