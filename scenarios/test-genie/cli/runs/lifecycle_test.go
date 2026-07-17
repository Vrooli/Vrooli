package runs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

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

func (s *streamServer) GetRunStatus(_ context.Context, _ *connect.Request[runspb.GetRunStatusRequest]) (*connect.Response[runspb.RunLiveStatus], error) {
	return connect.NewResponse(s.waitStatus), nil
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

func TestRunStatusPendingDirectsOneCanonicalQuietWait(t *testing.T) { // [REQ:TESTGENIE-ORCH-P0]
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{
		RunId: "R", Scenario: "demo", Status: "in_progress", EtaKnown: true,
		EstimatedRemainingSeconds: 41, RecommendedNextCheckSeconds: 7,
		Standing: &commonv1.OperationStanding{Lifecycle: "executing", Directive: "wait", ReattachCommand: "test-genie runs wait --json demo R"},
	}})
	var out bytes.Buffer
	if err := runStatus(nil, []string{"demo", "R"}, &out); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "action: wait") || !strings.Contains(text, "test-genie runs wait --json") {
		t.Fatalf("pending status lacks canonical wait action: %q", text)
	}
	for _, forbidden := range []string{"check again", "re-check", "run status again"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("pending status contains poll-first wording %q: %q", forbidden, text)
		}
	}
	if !strings.Contains(strings.ToLower(text), "do not poll") {
		t.Fatalf("pending status must make the anti-polling contract explicit: %q", text)
	}
}

func TestRunStatusJSONCarriesTypedWaitActionOnlyWhilePending(t *testing.T) { // [REQ:TESTGENIE-ORCH-P0]
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{
		RunId: "R", Scenario: "demo", Status: "in_progress", EstimatedRemainingSeconds: 41,
		Standing: &commonv1.OperationStanding{Lifecycle: "executing", Directive: "wait", ReattachCommand: "test-genie runs wait --json demo R"},
	}})
	var out bytes.Buffer
	if err := runStatus(nil, []string{"--json", "demo", "R"}, &out); err != nil {
		t.Fatalf("runStatus --json: %v", err)
	}
	var payload struct {
		Standing struct {
			Lifecycle string `json:"lifecycle"`
			Directive string `json:"directive"`
			Reattach  string `json:"reattachCommand"`
		} `json:"standing"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode status JSON: %v\n%s", err, out.String())
	}
	if payload.Standing.Lifecycle != "executing" || payload.Standing.Directive != "wait" || !strings.Contains(payload.Standing.Reattach, "test-genie runs wait --json") {
		t.Fatalf("typed operation standing = %+v", payload.Standing)
	}
}

func TestRunStatusTerminalOmitsWaitAction(t *testing.T) {
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{RunId: "R", Scenario: "demo", Status: "failed", Verdict: "FAIL"}})
	var out bytes.Buffer
	if err := runStatus(nil, []string{"--json", "demo", "R"}, &out); err != nil {
		t.Fatalf("terminal runStatus --json: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode terminal status JSON: %v", err)
	}
	if _, exists := payload["nextAction"]; exists {
		t.Fatalf("terminal status must omit wait action: %s", out.String())
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

// A completed run's event replay may contain only run_completed. The human
// renderer must hydrate the durable terminal record instead of claiming the
// suite had no phases.
func TestRunWaitHumanHydratesCanonicalTerminalPhasesAfterEventOnlyReplay(t *testing.T) {
	withStreamServer(t, &streamServer{
		events:     []*runspb.RunEvent{{Event: "run_completed", RunId: "R", Scenario: "demo", Success: false, Verdict: "FAIL"}},
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
	err := runWait(nil, []string{"demo", "R"}, &buf)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitRegression {
		t.Fatalf("expected failed terminal exit, got %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "Results: 0 passed • 0 failed • Duration: 0s") || strings.Contains(out, "(no phases recorded)") {
		t.Fatalf("event-only replay must use canonical terminal phases, got: %q", out)
	}
	if !strings.Contains(out, "unit") || !strings.Contains(out, "workflow") || !strings.Contains(out, "Duration: 11s") {
		t.Fatalf("canonical terminal report missing phases/duration, got: %q", out)
	}
}

// TestRunWaitJSONSnapshot proves `--json` stays a single quiet snapshot (no
// streamed phase lines), preserving the scripted contract.
func TestRunWaitJSONSnapshot(t *testing.T) {
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{
		RunId:  "R",
		Status: "passed",
		TerminalPresentations: []*commonv1.PhasePresentation{{
			Provider:             "architecture-health",
			Phase:                "architecture",
			CurrentLevel:         "L2",
			NextLevel:            "L3",
			BlockingFindingCodes: []string{"arch.primitive_unverified"},
			NextAction:           "Prove each command primitive.",
			DocumentationTopics:  []string{"architecture maturity next move"},
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
	if !strings.Contains(out, "\"phasePresentation\"") {
		t.Fatalf("--json wait must surface the terminal phase presentation in the curated phases, got: %q", out)
	}
	if !strings.Contains(out, "arch.primitive_unverified") {
		t.Fatalf("--json wait standing must include blocking finding codes, got: %q", out)
	}
	if !strings.Contains(out, "\"topPriority\"") || !strings.Contains(out, "architecture maturity next move") {
		t.Fatalf("--json wait must surface the cross-phase top priority, got: %q", out)
	}
}

func TestRunWaitAcceptsTrailingJSONFlag(t *testing.T) {
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{RunId: "R", Scenario: "demo", Status: "passed"}})
	var buf bytes.Buffer
	if err := runWait(nil, []string{"demo", "R", "--json"}, &buf); err != nil {
		t.Fatalf("trailing --json wait: %v", err)
	}
	if !json.Valid(bytes.TrimSpace(buf.Bytes())) {
		t.Fatalf("trailing --json must emit a machine snapshot, got: %q", buf.String())
	}
	if strings.Contains(buf.String(), "TEST SUITE COMPLETE") {
		t.Fatalf("trailing --json must not enter the human streaming path, got: %q", buf.String())
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
			PhasePresentation: &commonv1.PhasePresentation{
				Provider:             "architecture-health",
				Phase:                "architecture",
				CurrentLevel:         "L2",
				CurrentLevelLabel:    "Ready",
				NextLevel:            "L3",
				CeilingLevel:         "L4",
				BlockingFindingCodes: []string{"arch.primitive_unverified"},
				NextAction:           "Prove each command primitive.",
				FocusCapabilityLabel: "Command Architecture",
				NorthStar:            "Renderer-separated primitives are verified.",
				DocumentationTopics:  []string{"architecture maturity next move"},
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
