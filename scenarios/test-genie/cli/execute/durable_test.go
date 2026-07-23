package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	if got := RecommendedWaitSeconds(600, true); got != 1050 {
		t.Fatalf("recommendedWaitSeconds known = %d, want 1050", got)
	}
	if got := RecommendedWaitSeconds(10, true); got != minRecommendedWaitSeconds {
		t.Fatalf("recommendedWaitSeconds floor = %d, want %d", got, minRecommendedWaitSeconds)
	}
	if got := RecommendedWaitSeconds(0, false); got != unknownETAWaitSeconds {
		t.Fatalf("recommendedWaitSeconds unknown = %d, want %d", got, unknownETAWaitSeconds)
	}
	cmd := ReattachCommandWithTimeout("demo", "R1", 1050)
	if cmd != "test-genie runs wait --json --timeout=1050 demo R1" {
		t.Fatalf("reattachCommandWithTimeout = %q", cmd)
	}
}

func TestComprehensiveWaitCeilingIsIndependentOfETA(t *testing.T) {
	if got := recommendedWaitSecondsForRequest(110, true, Request{ScenarioName: "demo", Preset: "comprehensive"}); got != comprehensiveWaitSeconds {
		t.Fatalf("comprehensive wait = %d, want %d", got, comprehensiveWaitSeconds)
	}
	if got := recommendedWaitSecondsForRequest(110, true, Request{ScenarioName: "demo", Preset: "quick"}); got == comprehensiveWaitSeconds {
		t.Fatalf("quick wait unexpectedly used comprehensive ceiling")
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
		"test-genie runs findings R1 --scenario demo",
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
	phasesAcc := []Phase{
		{Name: "a", Status: "passed", DurationSeconds: 2},
		{Name: "b", Status: "failed", DurationSeconds: 3},
		{Name: "c", Status: "provider_unavailable", DurationSeconds: 4},
	}
	resp := buildResponse(&runspb.RunEvent{Event: evRunCompleted, Success: false, Verdict: "FAIL"}, phasesAcc)
	if resp.Success || resp.Verdict != "FAIL" {
		t.Fatalf("terminal not applied: %+v", resp)
	}
	if resp.PhaseSummary.Total != 3 || resp.PhaseSummary.Passed != 1 || resp.PhaseSummary.Failed != 2 || resp.PhaseSummary.DurationSeconds != 9 {
		t.Fatalf("summary = %+v", resp.PhaseSummary)
	}
}

// fakeRunsServer implements just enough of RunsService to drive RunDurable.
type fakeRunsServer struct {
	runs_v1connect.UnimplementedRunsServiceHandler
	eta          int32
	coalesced    bool
	startErr     error
	followErr    error
	events       []*runspb.RunEvent
	followCalled atomic.Bool
	startCalls   atomic.Int32
}

func (f *fakeRunsServer) StartRun(_ context.Context, req *connect.Request[runspb.StartRunRequest]) (*connect.Response[runspb.StartRunResponse], error) {
	f.startCalls.Add(1)
	if f.startErr != nil {
		return nil, f.startErr
	}
	return connect.NewResponse(&runspb.StartRunResponse{
		RunId: "20260101-000000-abcd1234", Scenario: req.Msg.GetScenario(),
		EstimatedTotalSeconds: f.eta, EtaKnown: f.eta > 0, Coalesced: f.coalesced,
	}), nil
}

func (f *fakeRunsServer) FollowRun(_ context.Context, _ *connect.Request[runspb.FollowRunRequest], stream *connect.ServerStream[runspb.RunEvent]) error {
	f.followCalled.Store(true)
	if f.followErr != nil {
		return f.followErr
	}
	for _, ev := range f.events {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	return nil
}

// captureStdoutStderr redirects BOTH os.Stdout and os.Stderr for fn, so tests
// can assert the machine result (stdout) and the structured start-handle (stderr)
// independently.
func captureStdoutStderr(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	runErr := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = origOut, origErr
	var outBuf, errBuf bytes.Buffer
	if _, err := io.Copy(&outBuf, rOut); err != nil {
		t.Fatalf("copy stdout: %v", err)
	}
	if _, err := io.Copy(&errBuf, rErr); err != nil {
		t.Fatalf("copy stderr: %v", err)
	}
	return outBuf.String(), errBuf.String(), runErr
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

// captureDurableStdout redirects os.Stdout for the duration of fn (RunDurable
// writes its --json/--jsonl output to os.Stdout directly).
func captureDurableStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stdout: %v", err)
	}
	return buf.String(), runErr
}

// TestRunDurableJSONSharesDurablePath is the Phase 5 contract: --json starts
// from the SAME durable StartRun path as human/--jsonl (it works against a fake
// server that ONLY implements StartRun+FollowRun — the legacy blocking REST
// /executions path is gone), blocks to completion even for a long ETA (never
// auto-backgrounds), and emits the final Response as one JSON object carrying the
// run id as executionId.
func TestRunDurableJSONSharesDurablePath(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60") // would background a long run in human mode
	h := &fakeRunsServer{eta: 600, events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "20260101-000000-abcd1234"},
		{Event: evPhaseCompleted, Phase: "unit", Status: "passed", DurationSeconds: 2},
		{Event: evRunCompleted, Success: true, Verdict: "PASS"},
	}}
	url := newFakeRunsServer(t, h)
	out, err := captureDurableStdout(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSON: true})
	})
	if err != nil {
		t.Fatalf("json mode success expected, got %v", err)
	}
	if h.startCalls.Load() != 1 {
		t.Fatalf("--json must start exactly one durable run, got %d StartRun calls", h.startCalls.Load())
	}
	if !h.followCalled.Load() {
		t.Fatal("--json must block+follow even for a long ETA (never auto-background)")
	}
	var resp Response
	if e := json.Unmarshal([]byte(out), &resp); e != nil {
		t.Fatalf("json output is not a parseable Response: %v (%q)", e, out)
	}
	if !resp.Success || resp.Verdict != "PASS" {
		t.Fatalf("resp success/verdict wrong: %+v", resp)
	}
	if resp.ExecutionID != "20260101-000000-abcd1234" {
		t.Fatalf("executionId must carry the durable run id, got %q", resp.ExecutionID)
	}
	if resp.PhaseSummary.Total != 1 || resp.PhaseSummary.Passed != 1 {
		t.Fatalf("phase summary from events wrong: %+v", resp.PhaseSummary)
	}
}

func TestRunDurableJSONFailureEmitsErrorObject(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
	h := &fakeRunsServer{events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "R"},
		{Event: evRunCompleted, Success: false, Verdict: "FAIL"},
	}}
	url := newFakeRunsServer(t, h)
	out, err := captureDurableStdout(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSON: true})
	})
	if err == nil {
		t.Fatal("expected a failure error for a FAIL verdict")
	}
	var resp Response
	if e := json.Unmarshal([]byte(out), &resp); e != nil {
		t.Fatalf("failure output must still be parseable JSON: %v (%q)", e, out)
	}
	if resp.Success {
		t.Fatal("expected success=false in the json output")
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

// TestRunDurableJSONEmitsStartHandleAndRunHandle is the Phase 3 machine-handle
// contract for --json: the durable run identity is exposed as structured JSON
// BOTH early (a run_started handle on stderr, so a long run is not opaque until
// completion) AND in the terminal object (runHandle on stdout), while stdout
// stays a single parseable SuiteExecutionResult.
func TestRunDurableJSONEmitsStartHandleAndRunHandle(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60") // long run would background in human mode
	h := &fakeRunsServer{eta: 600, events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "20260101-000000-abcd1234"},
		{Event: evPhaseCompleted, Phase: "unit", Status: "passed", DurationSeconds: 2},
		{Event: evRunCompleted, Success: true, Verdict: "PASS"},
	}}
	url := newFakeRunsServer(t, h)
	stdout, stderr, err := captureStdoutStderr(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSON: true})
	})
	if err != nil {
		t.Fatalf("json success expected, got %v", err)
	}
	if h.startCalls.Load() != 1 {
		t.Fatalf("--json must start exactly one durable run, got %d", h.startCalls.Load())
	}

	// Early start-handle: one parseable JSON line on stderr carrying run identity.
	var handle map[string]any
	if e := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &handle); e != nil {
		t.Fatalf("start-handle on stderr is not one parseable JSON object: %v (%q)", e, stderr)
	}
	if handle["event"] != evRunStarted || handle["run_id"] != "20260101-000000-abcd1234" {
		t.Fatalf("start-handle missing run identity: %v", handle)
	}
	if handle["reattach"] != "test-genie runs wait --json demo 20260101-000000-abcd1234" {
		t.Fatalf("start-handle reattach breadcrumb wrong: %v", handle["reattach"])
	}

	// Terminal object: single Response on stdout carrying the structured runHandle.
	var resp Response
	if e := json.Unmarshal([]byte(stdout), &resp); e != nil {
		t.Fatalf("stdout is not a single parseable Response: %v (%q)", e, stdout)
	}
	if resp.ExecutionID != "20260101-000000-abcd1234" {
		t.Fatalf("executionId must carry the run id, got %q", resp.ExecutionID)
	}
	if resp.RunHandle == nil || resp.RunHandle.RunID != "20260101-000000-abcd1234" {
		t.Fatalf("terminal object must carry runHandle identity, got %+v", resp.RunHandle)
	}
	if resp.RunHandle.Reattach == "" || resp.RunHandle.Follow == "" {
		t.Fatalf("runHandle must carry reattach+follow commands, got %+v", resp.RunHandle)
	}
}

// TestRunDurableJSONLFirstEventIsRunStarted proves --jsonl exposes run identity
// as its first stdout event (StartRun called once), the machine early-handle.
func TestRunDurableJSONLFirstEventIsRunStarted(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "60")
	h := &fakeRunsServer{eta: 600, events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "R7", Scenario: "demo"},
		{Event: evRunCompleted, Success: true, Verdict: "PASS"},
	}}
	url := newFakeRunsServer(t, h)
	stdout, _, err := captureStdoutStderr(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSONL: true})
	})
	if err != nil {
		t.Fatalf("jsonl success expected, got %v", err)
	}
	if h.startCalls.Load() != 1 {
		t.Fatalf("--jsonl must start exactly one durable run, got %d", h.startCalls.Load())
	}
	first := strings.SplitN(strings.TrimSpace(stdout), "\n", 2)[0]
	var ev map[string]any
	if e := json.Unmarshal([]byte(first), &ev); e != nil {
		t.Fatalf("first jsonl line is not parseable JSON: %v (%q)", e, first)
	}
	if ev["event"] != evRunStarted || ev["run_id"] != "R7" {
		t.Fatalf("first jsonl event must be run_started with run_id, got %v", ev)
	}
}

// TestRunDurableJSONStartErrorCarriesScenario proves a start failure emits a
// parseable, actionable JSON object (success=false + scenario), with no runId
// since no run started.
func TestRunDurableJSONStartErrorCarriesScenario(t *testing.T) {
	h := &fakeRunsServer{startErr: connect.NewError(connect.CodeFailedPrecondition, errors.New("busy with another run"))}
	url := newFakeRunsServer(t, h)
	stdout, _, err := captureStdoutStderr(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSON: true})
	})
	if err == nil {
		t.Fatal("expected a start error")
	}
	var obj map[string]any
	if e := json.Unmarshal([]byte(stdout), &obj); e != nil {
		t.Fatalf("start-error output must be parseable JSON: %v (%q)", e, stdout)
	}
	if obj["success"] != false || obj["scenario"] != "demo" {
		t.Fatalf("start-error object must carry success=false + scenario: %v", obj)
	}
	if _, hasRun := obj["runId"]; hasRun {
		t.Fatalf("runId must be absent before a successful start: %v", obj)
	}
}

// TestRunDurableJSONFollowErrorCarriesRunIdentity proves a follow failure after
// a successful start emits a JSON error object carrying scenario AND runId, so
// automation can reattach/abort instead of parsing an opaque error string.
func TestRunDurableJSONFollowErrorCarriesRunIdentity(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
	h := &fakeRunsServer{followErr: errors.New("stream broke")}
	url := newFakeRunsServer(t, h)
	stdout, _, err := captureStdoutStderr(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSON: true})
	})
	if err == nil {
		t.Fatal("expected a follow error")
	}
	var obj map[string]any
	if e := json.Unmarshal([]byte(stdout), &obj); e != nil {
		t.Fatalf("follow-error output must be parseable JSON: %v (%q)", e, stdout)
	}
	if obj["scenario"] != "demo" || obj["runId"] != "20260101-000000-abcd1234" {
		t.Fatalf("follow-error object must carry scenario+runId: %v", obj)
	}
}

// TestRunDurableJSONLFollowErrorCarriesRunIdentity proves the JSONL terminal
// error event carries run identity, not only an error string.
func TestRunDurableJSONLFollowErrorCarriesRunIdentity(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
	h := &fakeRunsServer{followErr: errors.New("stream broke")}
	url := newFakeRunsServer(t, h)
	stdout, _, err := captureStdoutStderr(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSONL: true})
	})
	if err == nil {
		t.Fatal("expected a follow error")
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var ev map[string]any
	if e := json.Unmarshal([]byte(lines[len(lines)-1]), &ev); e != nil {
		t.Fatalf("terminal jsonl line must be parseable JSON: %v (%q)", e, stdout)
	}
	if ev["event"] != evRunCompleted || ev["success"] != false {
		t.Fatalf("terminal jsonl event must be a run_completed failure: %v", ev)
	}
	if ev["run_id"] != "20260101-000000-abcd1234" || ev["scenario"] != "demo" {
		t.Fatalf("terminal jsonl error must carry run identity: %v", ev)
	}
}

// TestRunDurableJSONCoalescedSurfaces proves a coalesced run (rode an in-flight
// identical run) is exposed as structured data in both the start-handle and the
// terminal runHandle for --json consumers.
func TestRunDurableJSONCoalescedSurfaces(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
	h := &fakeRunsServer{coalesced: true, events: []*runspb.RunEvent{
		{Event: evRunStarted, RunId: "20260101-000000-abcd1234"},
		{Event: evRunCompleted, Success: true, Verdict: "PASS"},
	}}
	url := newFakeRunsServer(t, h)
	stdout, stderr, err := captureStdoutStderr(t, func() error {
		return RunDurable(url, Request{ScenarioName: "demo"}, DurableOptions{JSON: true})
	})
	if err != nil {
		t.Fatalf("coalesced json success expected, got %v", err)
	}
	var handle map[string]any
	if e := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &handle); e != nil {
		t.Fatalf("start-handle not parseable: %v (%q)", e, stderr)
	}
	if handle["coalesced"] != true {
		t.Fatalf("start-handle must mark coalesced: %v", handle)
	}
	var resp Response
	if e := json.Unmarshal([]byte(stdout), &resp); e != nil {
		t.Fatalf("stdout not a Response: %v (%q)", e, stdout)
	}
	if resp.RunHandle == nil || !resp.RunHandle.Coalesced {
		t.Fatalf("terminal runHandle must mark coalesced: %+v", resp.RunHandle)
	}
}
