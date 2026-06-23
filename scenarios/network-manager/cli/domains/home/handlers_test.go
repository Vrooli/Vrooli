package home

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	clitest "network-manager/cli/internal/testutil"

	homev1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration"
	homeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration/home_integration_v1connect"
)

func TestHandlersActionsCallsGeneratedClient(t *testing.T) {
	// [REQ:NM-P0-007] The Home Automation actions CLI reaches the Connect API contract.
	var called bool
	core := newHomeTestApp(t, &fakeHomeService{
		listActions: func(context.Context, *connect.Request[homev1.ListActionsRequest]) (*connect.Response[homev1.ListActionsResponse], error) {
			called = true
			return connect.NewResponse(&homev1.ListActionsResponse{Actions: []*homev1.HomeAction{{Name: "network.health.run", Effect: "read"}}}), nil
		},
	})
	h := newHandlers(core)
	var out bytes.Buffer

	err := h.actions(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, JSON: true, Stdout: &out}))

	if err != nil {
		t.Fatalf("actions: %v", err)
	}
	if !called {
		t.Fatal("expected ListActions API call")
	}
	if !strings.Contains(out.String(), "network.health.run") {
		t.Fatalf("expected action in output, got %s", out.String())
	}
}

func TestHandlersInvokePassesApproval(t *testing.T) {
	// [REQ:NM-P0-007] The Home Automation invoke CLI forwards approval acknowledgements.
	var gotName string
	var gotApproved bool
	core := newHomeTestApp(t, &fakeHomeService{
		invoke: func(_ context.Context, req *connect.Request[homev1.InvokeActionRequest]) (*connect.Response[homev1.InvokeActionResponse], error) {
			gotName = req.Msg.GetName()
			gotApproved = req.Msg.GetApproved()
			return connect.NewResponse(&homev1.InvokeActionResponse{
				Status:  "manual_required",
				Message: "manual required",
				Event:   &homev1.HomeEvent{Id: "event-1", Type: "network.quality.degraded", Summary: "redacted", OccurredAt: "2026-06-23T17:30:00Z"},
			}), nil
		},
	})
	h := newHandlers(core)
	var out bytes.Buffer

	err := h.invoke(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core:   core,
		JSON:   true,
		Stdout: &out,
		Schema: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "name", Required: true}},
			Flags:       []cliapp.Flag{{Name: "approved", Bool: true}},
		},
		Positionals: map[string]string{"name": "network.adblock.pause_device"},
		BoolFlags:   map[string]bool{"approved": true},
	}))

	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if gotName != "network.adblock.pause_device" || !gotApproved {
		t.Fatalf("request = name %q approved %t", gotName, gotApproved)
	}
	if !strings.Contains(out.String(), "manual_required") {
		t.Fatalf("expected status in output, got %s", out.String())
	}
}

func TestHandlersEventsCallsGeneratedClient(t *testing.T) {
	// [REQ:NM-P0-007] The Home Automation events CLI reaches the Connect API contract.
	var called bool
	core := newHomeTestApp(t, &fakeHomeService{
		listEvents: func(context.Context, *connect.Request[homev1.ListRecentEventsRequest]) (*connect.Response[homev1.ListRecentEventsResponse], error) {
			called = true
			return connect.NewResponse(&homev1.ListRecentEventsResponse{Events: []*homev1.HomeEvent{{Id: "event-1", Type: "network.device.new_seen", Summary: "redacted", OccurredAt: "2026-06-23T17:30:00Z"}}}), nil
		},
	})
	h := newHandlers(core)
	var out bytes.Buffer

	err := h.events(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, JSON: true, Stdout: &out}))

	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if !called {
		t.Fatal("expected ListRecentEvents API call")
	}
	if !strings.Contains(out.String(), "network.device.new_seen") {
		t.Fatalf("expected event in output, got %s", out.String())
	}
}

func newHomeTestApp(t *testing.T, svc homeconnect.HomeIntegrationServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := homeconnect.NewHomeIntegrationServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return clitest.NewTestApp(t, mux)
}

type fakeHomeService struct {
	listActions func(context.Context, *connect.Request[homev1.ListActionsRequest]) (*connect.Response[homev1.ListActionsResponse], error)
	invoke      func(context.Context, *connect.Request[homev1.InvokeActionRequest]) (*connect.Response[homev1.InvokeActionResponse], error)
	listEvents  func(context.Context, *connect.Request[homev1.ListRecentEventsRequest]) (*connect.Response[homev1.ListRecentEventsResponse], error)
}

func (f *fakeHomeService) ListActions(ctx context.Context, req *connect.Request[homev1.ListActionsRequest]) (*connect.Response[homev1.ListActionsResponse], error) {
	if f.listActions != nil {
		return f.listActions(ctx, req)
	}
	return connect.NewResponse(&homev1.ListActionsResponse{}), nil
}

func (f *fakeHomeService) InvokeAction(ctx context.Context, req *connect.Request[homev1.InvokeActionRequest]) (*connect.Response[homev1.InvokeActionResponse], error) {
	if f.invoke != nil {
		return f.invoke(ctx, req)
	}
	return connect.NewResponse(&homev1.InvokeActionResponse{}), nil
}

func (f *fakeHomeService) ListRecentEvents(ctx context.Context, req *connect.Request[homev1.ListRecentEventsRequest]) (*connect.Response[homev1.ListRecentEventsResponse], error) {
	if f.listEvents != nil {
		return f.listEvents(ctx, req)
	}
	return connect.NewResponse(&homev1.ListRecentEventsResponse{}), nil
}
