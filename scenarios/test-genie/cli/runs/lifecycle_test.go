package runs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// streamServer implements FollowRun + WaitRun for the wait/follow tests.
type streamServer struct {
	runs_v1connect.UnimplementedRunsServiceHandler
	events          []*runspb.RunEvent
	waitStatus      *runspb.RunLiveStatus
	waitTimed       bool
	terminalRun     *runspb.RunInfo
	snapshotVersion int32
	degradedReasons []string
}

func (s *streamServer) FollowRun(_ context.Context, _ *connect.Request[runspb.FollowRunRequest], stream *connect.ServerStream[runspb.RunEvent]) error {
	for _, ev := range s.events {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamServer) WaitRun(_ context.Context, _ *connect.Request[runspb.WaitRunRequest]) (*connect.Response[runspb.WaitRunResponse], error) {
	return connect.NewResponse(&runspb.WaitRunResponse{
		Status: s.waitStatus, TimedOut: s.waitTimed, TerminalRun: s.terminalRun,
		TerminalSnapshotSchemaVersion: s.snapshotVersion, DegradedReasons: s.degradedReasons,
	}), nil
}

func (s *streamServer) GetRun(_ context.Context, _ *connect.Request[runspb.GetRunRequest]) (*connect.Response[runspb.GetRunResponse], error) {
	return connect.NewResponse(&runspb.GetRunResponse{
		Run:                           s.terminalRun,
		TerminalSnapshotSchemaVersion: s.snapshotVersion,
		DegradedReasons:               s.degradedReasons,
	}), nil
}

// withStreamServer stands up a real Connect server for h and points newClient at
// it, so the wait/follow renderers exercise the genuine stream path.
func withStreamServer(t *testing.T, h *streamServer) {
	t.Helper()
	path, handler := runs_v1connect.NewRunsServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cl := runs_v1connect.NewRunsServiceClient(http.DefaultClient, srv.URL)
	prev := newClient
	newClient = func(*cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) { return cl, nil }
	t.Cleanup(func() { newClient = prev })
}

func passEvents() []*runspb.RunEvent {
	return []*runspb.RunEvent{
		{Event: "run_started", RunId: "R"},
		{Event: "phase_started", Phase: "unit"},
		{Event: "phase_completed", Phase: "unit", Status: "passed", DurationSeconds: 2},
		{Event: "run_completed", Success: true, Verdict: "PASS"},
	}
}

// TestRunWaitHumanStreams proves human `runs wait` renders the live stream (≥1
// progress line) rather than blocking silently — the anti-polling fix.
func TestRunWaitHumanStreams(t *testing.T) {
	withStreamServer(t, &streamServer{events: passEvents()})
	var buf bytes.Buffer
	if err := runWait(nil, []string{"demo", "R"}, &buf); err != nil {
		t.Fatalf("runWait pass: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "unit") {
		t.Fatalf("human wait must stream phase events, got: %q", out)
	}
	if !strings.Contains(out, "PASS") {
		t.Fatalf("human wait must render the terminal verdict, got: %q", out)
	}
}

// TestRunWaitHumanFailureExitCode proves a failed run exits with the regression
// code through the streamed path.
func TestRunWaitHumanFailureExitCode(t *testing.T) {
	withStreamServer(t, &streamServer{events: []*runspb.RunEvent{
		{Event: "run_started", RunId: "R"},
		{Event: "phase_failed", Phase: "unit", Status: "failed"},
		{Event: "run_completed", Success: false, Verdict: "FAIL"},
	}})
	var buf bytes.Buffer
	err := runWait(nil, []string{"demo", "R"}, &buf)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitRegression {
		t.Fatalf("expected regression exit, got %v", err)
	}
}

// TestRunWaitJSONSnapshot proves `--json` stays a single quiet snapshot (no
// streamed phase lines), preserving the scripted contract.
func TestRunWaitJSONSnapshot(t *testing.T) {
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{
		RunId:  "R",
		Status: "passed",
		TerminalStandings: []*runspb.PhaseMaturityStanding{{
			Provider:             "architecture-health",
			Phase:                "architecture",
			CurrentLevel:         "L2",
			NextLevel:            "L3",
			BlockingFindingCodes: []string{"arch.primitive_unverified"},
			NextMove:             "Prove each command primitive.",
		}},
	}})
	var buf bytes.Buffer
	if err := runWait(nil, []string{"--json", "demo", "R"}, &buf); err != nil {
		t.Fatalf("runWait --json: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "phase_started") || strings.Contains(out, "▶") {
		t.Fatalf("--json must not stream events, got: %q", out)
	}
	if !strings.Contains(out, "\"status\"") {
		t.Fatalf("--json must emit a structured snapshot, got: %q", out)
	}
	if !strings.Contains(out, "\"maturityStanding\"") {
		t.Fatalf("--json wait must surface terminal maturity standings in the curated phases, got: %q", out)
	}
	if !strings.Contains(out, "arch.primitive_unverified") {
		t.Fatalf("--json wait standing must include blocking finding codes, got: %q", out)
	}
	if !strings.Contains(out, "\"topPriority\"") || !strings.Contains(out, "architecture maturity next move") {
		t.Fatalf("--json wait must surface the cross-phase top priority, got: %q", out)
	}
}

func TestRunWaitJSONNeverParksBeforeWaitRun(t *testing.T) {
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{Status: "passed"}})
	previous := parkForAwait
	parkForAwait = func(cliutil.ParkRequest) (*cliutil.ParkResult, bool, error) {
		t.Fatal("JSON wait must call WaitRun, not park before the run is terminal")
		return nil, false, nil
	}
	t.Cleanup(func() { parkForAwait = previous })
	var out bytes.Buffer
	if err := runWait(nil, []string{"--json", "demo", "R"}, &out); err != nil {
		t.Fatalf("JSON wait: %v", err)
	}
	if !strings.Contains(out.String(), `"status": "passed"`) {
		t.Fatalf("expected WaitRun JSON snapshot, got %q", out.String())
	}
}

func TestRunWaitJSONUsesCanonicalTerminalPhasesAndDurations(t *testing.T) {
	withStreamServer(t, &streamServer{
		waitStatus: &runspb.RunLiveStatus{RunId: "R", Scenario: "demo", Status: "failed", Verdict: "FAIL"},
		terminalRun: &runspb.RunInfo{
			RunId: "R", Scenario: "demo", Status: "failed",
			Phases: []*runspb.PhaseInfo{
				{Name: "unit", Status: "passed", DurationSeconds: 2},
				{Name: "workflow", Status: "failed", DurationSeconds: 9},
			},
		},
		snapshotVersion: 1,
	})
	var buf bytes.Buffer
	err := runWait(nil, []string{"--json", "demo", "R"}, &buf)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitRegression {
		t.Fatalf("failed terminal wait exit = %v", err)
	}
	var payload struct {
		Phases []struct {
			Name            string  `json:"name"`
			Status          string  `json:"status"`
			DurationSeconds float64 `json:"durationSeconds"`
		} `json:"phases"`
		PhaseSummary struct {
			DurationSeconds int `json:"durationSeconds"`
		} `json:"phaseSummary"`
		TerminalSnapshotSchemaVersion int32 `json:"terminalSnapshotSchemaVersion"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode wait JSON: %v\n%s", err, buf.String())
	}
	if len(payload.Phases) != 2 || payload.Phases[0].Name != "unit" || payload.Phases[1].Status != "failed" || payload.Phases[1].DurationSeconds != 9 {
		t.Fatalf("canonical phases missing from wait JSON: %+v", payload.Phases)
	}
	if payload.PhaseSummary.DurationSeconds != 11 || payload.TerminalSnapshotSchemaVersion != 1 {
		t.Fatalf("terminal summary/schema = %+v/%d", payload.PhaseSummary, payload.TerminalSnapshotSchemaVersion)
	}
}

func TestRunWaitAndShowJSONAgreeOnCanonicalTerminalEvidence(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0]
	run := &runspb.RunInfo{
		RunId: "R", Scenario: "demo", Status: "failed",
		Phases: []*runspb.PhaseInfo{
			{Name: "unit", Status: "passed", DurationSeconds: 2},
			{Name: "workflow", Status: "failed", DurationSeconds: 9},
		},
	}
	withStreamServer(t, &streamServer{
		waitStatus:      &runspb.RunLiveStatus{RunId: "R", Scenario: "demo", Status: "failed", Verdict: "FAIL"},
		terminalRun:     run,
		snapshotVersion: 1,
	})
	var waitOut, showOut bytes.Buffer
	_ = runWait(nil, []string{"--json", "demo", "R"}, &waitOut)
	if err := runShow(nil, []string{"--scenario", "demo", "--json", "R"}, &showOut); err != nil {
		t.Fatalf("runs show --json: %v", err)
	}
	var waitPayload struct {
		RunID  string `json:"runId"`
		Status string `json:"status"`
		Phases []struct {
			Name            string  `json:"name"`
			Status          string  `json:"status"`
			DurationSeconds float64 `json:"durationSeconds"`
		} `json:"phases"`
	}
	var showPayload struct {
		Run struct {
			RunID  string `json:"runId"`
			Status string `json:"status"`
			Phases []struct {
				Name            string  `json:"name"`
				Status          string  `json:"status"`
				DurationSeconds float64 `json:"durationSeconds"`
			} `json:"phases"`
		} `json:"run"`
	}
	if err := json.Unmarshal(waitOut.Bytes(), &waitPayload); err != nil {
		t.Fatalf("decode wait JSON: %v", err)
	}
	if err := json.Unmarshal(showOut.Bytes(), &showPayload); err != nil {
		t.Fatalf("decode show JSON: %v", err)
	}
	if waitPayload.RunID != showPayload.Run.RunID || waitPayload.Status != showPayload.Run.Status || len(waitPayload.Phases) != len(showPayload.Run.Phases) {
		t.Fatalf("wait/show summary mismatch: wait=%+v show=%+v", waitPayload, showPayload)
	}
	for i := range waitPayload.Phases {
		waitPhase, showPhase := waitPayload.Phases[i], showPayload.Run.Phases[i]
		if waitPhase.Name != showPhase.Name || waitPhase.Status != showPhase.Status || waitPhase.DurationSeconds != showPhase.DurationSeconds {
			t.Fatalf("wait/show phase %d mismatch: wait=%+v show=%+v", i, waitPhase, showPhase)
		}
	}
}

// TestRunWaitJSONTimeoutSurfacesBackoff proves a `--json --timeout` wait that
// returns before terminal exits 124, keeps stdout pure JSON, and prints the
// cadence governor + the exact quiet re-invoke line to stderr.
func TestRunWaitJSONTimeoutSurfacesBackoff(t *testing.T) {
	withStreamServer(t, &streamServer{
		waitStatus: &runspb.RunLiveStatus{RunId: "R", Status: "in_progress", RecommendedNextCheckSeconds: 17},
		waitTimed:  true,
	})
	var out, errBuf bytes.Buffer
	prev := stderrOut
	stderrOut = &errBuf
	t.Cleanup(func() { stderrOut = prev })

	err := runWait(nil, []string{"--json", "--timeout", "30", "demo", "R"}, &out)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitWaitTimeout {
		t.Fatalf("expected 124 wait-timeout exit, got %v", err)
	}
	if strings.Contains(out.String(), "still running") {
		t.Fatalf("stdout must stay pure JSON (hint belongs on stderr), got: %q", out.String())
	}
	hint := errBuf.String()
	if !strings.Contains(hint, "~17s") {
		t.Fatalf("stderr must surface recommended_next_check_seconds, got: %q", hint)
	}
	if !strings.Contains(hint, "test-genie runs wait --json --timeout=30 demo R") {
		t.Fatalf("stderr must surface the exact quiet re-invoke line, got: %q", hint)
	}
}

func TestRunWaitJSONWarnsOnRecentRepeatedWaitWithoutPollutingStdout(t *testing.T) {
	withStreamServer(t, &streamServer{
		waitStatus: &runspb.RunLiveStatus{RunId: "R", Status: "in_progress", RecommendedNextCheckSeconds: 17},
		waitTimed:  true,
	})
	tmp := t.TempDir()
	prevPath := waitStatePath
	prevNow := waitNow
	prevErr := stderrOut
	waitStatePath = func() string { return filepath.Join(tmp, "waits.json") }
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	waitNow = func() time.Time { return base }
	var errBuf bytes.Buffer
	stderrOut = &errBuf
	t.Cleanup(func() {
		waitStatePath = prevPath
		waitNow = prevNow
		stderrOut = prevErr
	})

	var first bytes.Buffer
	err := runWait(nil, []string{"--json", "--timeout", "30", "demo", "R"}, &first)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitWaitTimeout {
		t.Fatalf("first runWait --json should time out, got: %v", err)
	}
	firstErr := errBuf.String()
	if strings.Contains(firstErr, "recent wait detected") {
		t.Fatalf("first wait should not emit eagerness warning, got: %q", firstErr)
	}

	waitNow = func() time.Time { return base.Add(12 * time.Second) }
	errBuf.Reset()
	var second bytes.Buffer
	err = runWait(nil, []string{"--json", "--timeout", "30", "demo", "R"}, &second)
	if !errors.As(err, &ee) || ee.ExitCode() != exitWaitTimeout {
		t.Fatalf("second runWait --json should time out, got: %v", err)
	}
	if !json.Valid(bytes.TrimSpace(second.Bytes())) {
		t.Fatalf("stdout must remain pure JSON, got: %q", second.String())
	}
	warn := errBuf.String()
	if !strings.Contains(warn, "recent wait detected for demo/R") {
		t.Fatalf("expected eager-wait warning, got: %q", warn)
	}
	if !strings.Contains(warn, "tail --pid=<pid> -f /dev/null") {
		t.Fatalf("expected attach guidance, got: %q", warn)
	}
}

// TestRunFollowStreams proves `runs follow` renders the stream and exits clean.
func TestRunFollowStreams(t *testing.T) {
	withStreamServer(t, &streamServer{events: passEvents()})
	var buf bytes.Buffer
	if err := runFollow(nil, []string{"demo", "R"}, &buf); err != nil {
		t.Fatalf("runFollow: %v", err)
	}
	if !strings.Contains(buf.String(), "unit") {
		t.Fatalf("follow must stream events, got: %q", buf.String())
	}
}

func TestRunFollowRendersStandingAndFindingsBreadcrumb(t *testing.T) {
	withStreamServer(t, &streamServer{events: []*runspb.RunEvent{
		{Event: "run_started", RunId: "R", Scenario: "demo"},
		{
			Event:           "phase_completed",
			RunId:           "R",
			Scenario:        "demo",
			Phase:           "architecture",
			Status:          "passed",
			DurationSeconds: 2,
			MaturityStanding: &runspb.PhaseMaturityStanding{
				Provider:                "architecture-health",
				Phase:                   "architecture",
				CurrentLevel:            "L2",
				CurrentLevelLabel:       "Ready",
				NextLevel:               "L3",
				CeilingLevel:            "L4",
				BlockingFindingCodes:    []string{"arch.primitive_unverified"},
				NextMove:                "Prove each command primitive.",
				PriorityCapabilityLabel: "Command Architecture",
				NorthStar:               "Renderer-separated primitives are verified.",
			},
		},
		{Event: "run_completed", RunId: "R", Scenario: "demo", Success: true, Verdict: "PASS"},
	}})
	var buf bytes.Buffer
	if err := runFollow(nil, []string{"demo", "R"}, &buf); err != nil {
		t.Fatalf("runFollow: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"standing:",
		"L2 Ready → L3",
		"gaps: arch.primitive_unverified",
		"next: Prove each command primitive.",
		`docs: search-hub query "architecture maturity next move" --type doc`,
		"findings: test-genie runs findings demo R",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("follow output missing %q\n---\n%s", want, out)
		}
	}
}
