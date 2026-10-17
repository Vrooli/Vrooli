package calendar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"
)

func TestTestModeHTTPClientAddsRoutingHeaderWithoutMutatingRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "1", r.Header.Get("X-Vrooli-Test-Mode"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	response, err := (testModeHTTPClient{HTTPClient: server.Client()}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.Empty(t, request.Header.Get("X-Vrooli-Test-Mode"))
}

func TestEventCLIJourneyUsesGeneratedConnectEventClient(t *testing.T) {
	eventClient := &cliEventClient{}
	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/calendar/event-commands/synthetic-key-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eventClient.event)
	}))
	defer statusServer.Close()
	h := &handlers{eventClient: eventClient, httpClient: statusServer.Client(), apiBase: statusServer.URL}

	createCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{
		"title": "DST weekend", "subject": "synthetic owner", "notes": "fixture only", "availability": "busy", "timezone": "America/New_York", "all-day": "true", "start-date": "2026-11-01", "end-date-exclusive": "2026-11-03", "idempotency-key": "synthetic-key-1",
	}, Schema: eventTestSchema("title", "subject", "notes", "availability", "timezone", "all-day", "start-date", "end-date-exclusive", "idempotency-key")})
	created, err := h.createEventCall(createCtx)
	require.NoError(t, err)
	require.Equal(t, "synthetic-cli-event", created.Event.Id)
	require.EqualValues(t, 1, created.Event.Revision)
	require.Equal(t, "synthetic-key-1", eventClient.createKey)

	updateCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{
		"event-id": created.Event.Id, "revision": "1", "title": "Updated DST weekend", "subject": "synthetic owner", "notes": "edited fixture", "availability": "free", "timezone": "America/New_York", "all-day": "true", "start-date": "2026-11-01", "end-date-exclusive": "2026-11-03",
	}, Schema: eventTestSchema("event-id", "revision", "title", "subject", "notes", "availability", "timezone", "all-day", "start-date", "end-date-exclusive")})
	updated, err := h.updateEventCall(updateCtx)
	require.NoError(t, err)
	require.Equal(t, created.Event.Id, updated.Event.Id)
	require.EqualValues(t, 2, updated.Event.Revision)
	require.EqualValues(t, 1, eventClient.expectedRevision)

	getCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{"event-id": created.Event.Id}, Schema: eventTestSchema("event-id")})
	reopened, err := h.getEventCall(getCtx)
	require.NoError(t, err)
	require.Equal(t, updated.Event.Id, reopened.Event.Id)
	require.EqualValues(t, 2, reopened.Event.Revision)
	require.Equal(t, updated.Event.Title, reopened.Event.Title)

	statusCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{"idempotency-key": "synthetic-key-1"}, Schema: eventTestSchema("idempotency-key")})
	status, err := h.eventStatusCall(statusCtx)
	require.NoError(t, err)
	require.Equal(t, updated.Event.Id, status.ID)
	require.EqualValues(t, 2, status.Revision)

	listCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{"start-date": "2026-11-01", "end-date": "2026-11-30"}, Schema: eventTestSchema("start-date", "end-date")})
	listed, err := h.listEventsCall(listCtx)
	require.NoError(t, err)
	require.Len(t, listed.Events, 1)
	require.Equal(t, updated.Event.Id, listed.Events[0].Id)
	require.EqualValues(t, 2, listed.Events[0].Revision)
	require.Equal(t, "2026-11-01", eventClient.listRequest.StartLocalDate)
	require.Equal(t, "2026-11-30", eventClient.listRequest.EndLocalDate)
}

func TestEventCLIStaleRevisionConflictPreservesConnectStatus(t *testing.T) {
	eventClient := &cliEventClient{event: &v.CalendarEvent{Id: "event-1", Title: "Current", Revision: 4}}
	h := &handlers{eventClient: eventClient}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{
		"event-id": "event-1", "revision": "3", "title": "Stale", "subject": "owner", "availability": "busy", "timezone": "America/New_York", "all-day": "true", "start-date": "2026-11-01", "end-date-exclusive": "2026-11-03",
	}, Schema: eventTestSchema("event-id", "revision", "title", "subject", "availability", "timezone", "all-day", "start-date", "end-date-exclusive")})
	_, err := h.updateEventCall(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "aborted")
}

type cliEventClient struct {
	event            *v.CalendarEvent
	createKey        string
	expectedRevision int64
	listRequest      *v.ListEventsRequest
}

func (c *cliEventClient) ListEvents(_ context.Context, req *connect.Request[v.ListEventsRequest]) (*connect.Response[v.ListEventsResponse], error) {
	c.listRequest = req.Msg
	return connect.NewResponse(&v.ListEventsResponse{Events: []*v.CalendarEvent{c.event}}), nil
}

func (c *cliEventClient) GetEvent(_ context.Context, _ *connect.Request[v.GetEventRequest]) (*connect.Response[v.GetEventResponse], error) {
	return connect.NewResponse(&v.GetEventResponse{Event: c.event}), nil
}

func (c *cliEventClient) CreateEvent(_ context.Context, req *connect.Request[v.CreateEventRequest]) (*connect.Response[v.CreateEventResponse], error) {
	c.createKey = req.Msg.IdempotencyKey
	c.event = req.Msg.Event
	c.event.Id, c.event.Revision = "synthetic-cli-event", 1
	return connect.NewResponse(&v.CreateEventResponse{Event: c.event}), nil
}

func (c *cliEventClient) UpdateEvent(_ context.Context, req *connect.Request[v.UpdateEventRequest]) (*connect.Response[v.UpdateEventResponse], error) {
	c.expectedRevision = req.Msg.ExpectedRevision
	if c.event != nil && c.event.Revision != req.Msg.ExpectedRevision {
		return nil, connect.NewError(connect.CodeAborted, nil)
	}
	updated := req.Msg.Event
	updated.Revision = req.Msg.ExpectedRevision + 1
	c.event = updated
	return connect.NewResponse(&v.UpdateEventResponse{Event: updated}), nil
}

func eventTestSchema(_ ...string) cliapp.ArgSchema {
	names := []string{"event-id", "revision", "title", "subject", "notes", "availability", "timezone", "all-day", "start-date", "end-date", "end-date-exclusive", "start-at", "end-at", "idempotency-key"}
	flags := make([]cliapp.Flag, len(names))
	for i, name := range names {
		flags[i] = cliapp.Flag{Name: name}
	}
	return cliapp.ArgSchema{Flags: flags}
}
