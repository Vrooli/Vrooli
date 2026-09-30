package events

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
)

type fakeEventsService struct{}

func (fakeEventsService) Recent(context.Context, int) []Event {
	return []Event{{Type: "session", SessionID: "s1", Timestamp: "now", Details: map[string]string{"k": "v"}}}
}
func (fakeEventsService) Count(context.Context) int { return 123 }

func TestListClampsLimitAndProjectsEvents(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeEventsService{}})
	for _, limit := range []int32{0, -1, 50, 2000} {
		resp, err := h.List(context.Background(), connect.NewRequest(&eventsv1.ListRequest{Limit: limit}))
		if err != nil || len(resp.Msg.Events) != 1 || resp.Msg.Total != 123 || resp.Msg.Events[0].Details["k"] != "v" {
			t.Fatalf("limit %d: %#v %v", limit, resp, err)
		}
	}
}
