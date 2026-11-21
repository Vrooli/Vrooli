package events

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
	eventsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events/events_v1connect"
)

type eventsTestClient struct {
	eventsconnect.EventsServiceClient
}

func (eventsTestClient) List(context.Context, *connect.Request[eventsv1.ListRequest]) (*connect.Response[eventsv1.ListResponse], error) {
	return connect.NewResponse(&eventsv1.ListResponse{}), nil
}

func TestRunRendersResponse(t *testing.T) {
	h := &handlers{client: eventsTestClient{}}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit"}}}, JSON: true})
	if err := h.run(ctx); err != nil {
		t.Fatal(err)
	}
}
