package agentmanager

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func newTestClient(f *fakeAgentManager) *Client {
	c := NewClient(0)
	c.baseURLResolver = func(context.Context) (string, error) { return f.server.URL, nil }
	return c
}

func TestClientEnsureProfileAcceptsCreatedAndSurfacesRejection(t *testing.T) {
	f := newFakeAgentManager(t)
	c := newTestClient(f)
	f.reply(http.MethodPost, "/api/v1/profiles/ensure", http.StatusCreated, &apipb.EnsureProfileResponse{Created: true})

	resp, err := c.EnsureProfile(context.Background(), &apipb.EnsureProfileRequest{ProfileKey: "pk"})
	if err != nil || !resp.GetCreated() {
		t.Fatalf("EnsureProfile = %+v, %v; want created", resp, err)
	}
	var sent apipb.EnsureProfileRequest
	decodeBody(t, f.callsTo(http.MethodPost, "/api/v1/profiles/ensure")[0], &sent)
	if sent.GetProfileKey() != "pk" {
		t.Fatalf("ensured profile key = %q, want pk", sent.GetProfileKey())
	}

	f.fail(http.MethodPost, "/api/v1/profiles/ensure", http.StatusBadRequest, `{"error":"bad defaults"}`)
	if _, err := c.EnsureProfile(context.Background(), &apipb.EnsureProfileRequest{ProfileKey: "pk"}); err == nil || !strings.Contains(err.Error(), "bad defaults") {
		t.Fatalf("EnsureProfile rejection = %v", err)
	}
}

func TestClientGetTaskMapsFoundAndMissing(t *testing.T) {
	f := newFakeAgentManager(t)
	c := newTestClient(f)
	f.reply(http.MethodGet, "/api/v1/tasks/task-1", http.StatusOK, &apipb.GetTaskResponse{Task: &domainpb.Task{Id: "task-1", Title: "T"}})

	task, err := c.GetTask(context.Background(), "task-1")
	if err != nil || task.GetTitle() != "T" {
		t.Fatalf("GetTask = %+v, %v", task, err)
	}
	if task, err := c.GetTask(context.Background(), "task-missing"); err != nil || task != nil {
		t.Fatalf("GetTask missing = %+v, %v; want nil, nil", task, err)
	}
}

func TestClientStopRunByTagPostsTagStop(t *testing.T) {
	f := newFakeAgentManager(t)
	c := newTestClient(f)
	f.reply(http.MethodPost, "/api/v1/runs/tag/system-monitor-i/stop", http.StatusOK, &apipb.StopRunByTagResponse{})

	if err := c.StopRunByTag(context.Background(), "system-monitor-i"); err != nil {
		t.Fatalf("StopRunByTag: %v", err)
	}
	if err := c.StopRunByTag(context.Background(), "unknown"); err == nil || !strings.Contains(err.Error(), "no route") {
		t.Fatalf("StopRunByTag unknown = %v", err)
	}
}

func TestClientGetRunEventsResumesAfterSequence(t *testing.T) {
	f := newFakeAgentManager(t)
	c := newTestClient(f)
	f.reply(http.MethodGet, "/api/v1/runs/run-1/events", http.StatusOK, &apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: 6}}})

	events, err := c.GetRunEvents(context.Background(), "run-1", 5)
	if err != nil || len(events) != 1 || events[0].GetSequence() != 6 {
		t.Fatalf("GetRunEvents = %v, %v", events, err)
	}
	if q := f.callsTo(http.MethodGet, "/api/v1/runs/run-1/events")[0].Query; q != "after_sequence=5" {
		t.Fatalf("events query = %q, want after_sequence=5", q)
	}
}

func TestNewUUIDReturnsDistinctUUIDs(t *testing.T) {
	a, b := NewUUID(), NewUUID()
	if len(a) != 36 || a == b {
		t.Fatalf("NewUUID = %q, %q; want distinct 36-char UUIDs", a, b)
	}
}
