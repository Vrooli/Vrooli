package execute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

func TestAutoBackgroundThreshold(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "")
		s, ok := autoBackgroundThreshold()
		if !ok || s != defaultAutoBackgroundSeconds {
			t.Fatalf("default = (%d,%v), want (%d,true)", s, ok, defaultAutoBackgroundSeconds)
		}
	})
	t.Run("disabled", func(t *testing.T) {
		t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
		if _, ok := autoBackgroundThreshold(); ok {
			t.Fatal("0 must disable auto-background")
		}
	})
	t.Run("floor", func(t *testing.T) {
		t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "3")
		s, ok := autoBackgroundThreshold()
		if !ok || s != minAutoBackgroundSeconds {
			t.Fatalf("below floor = (%d,%v), want (%d,true)", s, ok, minAutoBackgroundSeconds)
		}
	})
}

func TestEventToMapDropsEmptyFields(t *testing.T) {
	m := eventToMap(&runspb.RunEvent{Event: evRunStarted, RunId: "R1", ElapsedSeconds: 0})
	if m["event"] != evRunStarted || m["run_id"] != "R1" {
		t.Fatalf("run_started map = %v", m)
	}
	if _, ok := m["phase"]; ok {
		t.Fatalf("empty phase must be omitted: %v", m)
	}
	done := eventToMap(&runspb.RunEvent{Event: evRunCompleted, Success: false, Verdict: "FAIL"})
	if done["success"] != false || done["verdict"] != "FAIL" {
		t.Fatalf("run_completed map = %v", done)
	}
}

func TestBuildResponseAggregates(t *testing.T) {
	phasesAcc := []Phase{{Name: "a", Status: "passed"}, {Name: "b", Status: "failed"}}
	resp := buildResponse(&runspb.RunEvent{Event: evRunCompleted, Success: false, Verdict: "FAIL"}, phasesAcc)
	if resp.Success || resp.Verdict != "FAIL" {
		t.Fatalf("terminal not applied: %+v", resp)
	}
	if resp.PhaseSummary.Total != 2 || resp.PhaseSummary.Passed != 1 || resp.PhaseSummary.Failed != 1 {
		t.Fatalf("summary = %+v", resp.PhaseSummary)
	}
}

// fakeRunsServer implements just enough of RunsService to drive RunDurable.
type fakeRunsServer struct {
	runs_v1connect.UnimplementedRunsServiceHandler
	eta          int32
	events       []*runspb.RunEvent
	followCalled atomic.Bool
}

func (f *fakeRunsServer) StartRun(_ context.Context, req *connect.Request[runspb.StartRunRequest]) (*connect.Response[runspb.StartRunResponse], error) {
	return connect.NewResponse(&runspb.StartRunResponse{
		RunId: "20260101-000000-abcd1234", Scenario: req.Msg.GetScenario(),
		EstimatedTotalSeconds: f.eta, EtaKnown: f.eta > 0,
	}), nil
}

func (f *fakeRunsServer) FollowRun(_ context.Context, _ *connect.Request[runspb.FollowRunRequest], stream *connect.ServerStream[runspb.RunEvent]) error {
	f.followCalled.Store(true)
	for _, ev := range f.events {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	return nil
}

func newFakeRunsServer(t *testing.T, h *fakeRunsServer) string {
	t.Helper()
	path, handler := runs_v1connect.NewRunsServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestRunDurableInlineFollow(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0") // never background
	t.Run("success", func(t *testing.T) {
		h := &fakeRunsServer{events: []*runspb.RunEvent{
			{Event: evRunStarted, RunId: "R"},
			{Event: evPhaseStarted, Phase: "unit"},
			{Event: evPhaseCompleted, Phase: "unit", Status: "passed", DurationSeconds: 2},
			{Event: evRunCompleted, Success: true, Verdict: "PASS"},
		}}
		url := newFakeRunsServer(t, h)
		if err := RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{}); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !h.followCalled.Load() {
			t.Fatal("expected FollowRun to be called inline")
		}
	})
	t.Run("failure", func(t *testing.T) {
		h := &fakeRunsServer{events: []*runspb.RunEvent{
			{Event: evRunStarted, RunId: "R"},
			{Event: evPhaseFailed, Phase: "unit", Status: "failed", DurationSeconds: 1},
			{Event: evRunCompleted, Success: false, Verdict: "FAIL"},
		}}
		url := newFakeRunsServer(t, h)
		if err := RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{}); err == nil {
			t.Fatal("expected failure error for FAIL verdict")
		}
	})
}

func TestRunDurableAutoBackgroundsLongRun(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60")
	h := &fakeRunsServer{eta: 600} // 10m known-long
	url := newFakeRunsServer(t, h)
	if err := RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{}); err != nil {
		t.Fatalf("auto-background should return nil, got %v", err)
	}
	if h.followCalled.Load() {
		t.Fatal("a known-long run must NOT be followed inline (should background)")
	}
}

func TestRunDurableWaitForcesInline(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60")
	h := &fakeRunsServer{eta: 600, events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "R"},
		{Event: evRunCompleted, Success: true, Verdict: "PASS"},
	}}
	url := newFakeRunsServer(t, h)
	if err := RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{Wait: true}); err != nil {
		t.Fatalf("--wait inline should succeed, got %v", err)
	}
	if !h.followCalled.Load() {
		t.Fatal("--wait must follow inline even for a long ETA")
	}
}
