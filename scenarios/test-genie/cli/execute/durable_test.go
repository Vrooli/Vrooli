package execute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// TestReattachCommandIsQuietWait pins the agent-facing breadcrumb to the quiet,
// single-return `runs wait --json` verb — never the heartbeating `runs follow`
// stream. A backgrounded stream re-wakes an agent on every heartbeat ("still
// waiting…" spam); a single quiet wait does not. followCommand is the separate
// human live-watch verb.
func TestReattachCommandIsQuietWait(t *testing.T) {
	got := reattachCommand("demo", "R1")
	if got != "test-genie runs wait --json demo R1" {
		t.Fatalf("reattachCommand = %q, want it to name `runs wait --json`", got)
	}
	if strings.Contains(got, "runs follow") {
		t.Fatalf("reattachCommand must not steer agents to the heartbeating `runs follow`: %q", got)
	}
	live := followCommand("demo", "R1")
	if live != "test-genie runs follow demo R1" {
		t.Fatalf("followCommand = %q, want it to name `runs follow`", live)
	}
}

func TestRecommendedWaitCommandIncludesBufferedTimeout(t *testing.T) {
	if got := recommendedWaitSeconds(600, true); got != 1050 {
		t.Fatalf("recommendedWaitSeconds known = %d, want 1050", got)
	}
	if got := recommendedWaitSeconds(10, true); got != minRecommendedWaitSeconds {
		t.Fatalf("recommendedWaitSeconds floor = %d, want %d", got, minRecommendedWaitSeconds)
	}
	if got := recommendedWaitSeconds(0, false); got != unknownETAWaitSeconds {
		t.Fatalf("recommendedWaitSeconds unknown = %d, want %d", got, unknownETAWaitSeconds)
	}
	cmd := reattachCommandWithTimeout("demo", "R1", 1050)
	if cmd != "test-genie runs wait --json --timeout=1050 demo R1" {
		t.Fatalf("reattachCommandWithTimeout = %q", cmd)
	}
}

func TestAgentWaitBlockIsProviderAgnosticAndActionable(t *testing.T) {
	var buf strings.Builder
	printAgentWaitBlock(&buf, "demo", "R1", 600, true, 1050)
	out := buf.String()
	for _, want := range []string{
		"Agent wait protocol",
		"Run exactly once",
		"test-genie runs wait --json --timeout=1050 demo R1",
		"recommended wait timeout: 17m30s",
		"coding-agent tool execution",
		"tail --pid=<pid> -f /dev/null",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("agent wait block missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"Codex", "Claude"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("agent wait block must stay provider-agnostic; found %q in:\n%s", forbidden, out)
		}
	}
}

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

// TestRunDurableUnknownETABackgrounds proves a run whose ETA is unknown
// auto-backgrounds (returns fast) instead of blocking inline — closing the
// first-run/unestimatable-run gap.
func TestRunDurableUnknownETABackgrounds(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60")
	h := &fakeRunsServer{eta: 0} // eta_known=false
	url := newFakeRunsServer(t, h)
	if err := RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{}); err != nil {
		t.Fatalf("unknown-ETA should background and return nil, got %v", err)
	}
	if h.followCalled.Load() {
		t.Fatal("an unknown-ETA run must background, not follow inline")
	}
}

// TestRunDurableUnknownETAInlineWhenLeverOff proves the lever can force inline
// follow for unknown-ETA runs.
func TestRunDurableUnknownETAInlineWhenLeverOff(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60")
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_ON_UNKNOWN_ETA", "0")
	h := &fakeRunsServer{eta: 0, events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "R"},
		{Event: evRunCompleted, Success: true, Verdict: "PASS"},
	}}
	url := newFakeRunsServer(t, h)
	if err := RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{}); err != nil {
		t.Fatalf("inline follow should succeed, got %v", err)
	}
	if !h.followCalled.Load() {
		t.Fatal("lever off: unknown-ETA must follow inline")
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
