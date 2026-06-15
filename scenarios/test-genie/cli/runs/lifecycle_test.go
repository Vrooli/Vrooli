package runs

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// streamServer implements FollowRun + WaitRun for the wait/follow tests.
type streamServer struct {
	runs_v1connect.UnimplementedRunsServiceHandler
	events     []*runspb.RunEvent
	waitStatus *runspb.RunLiveStatus
	waitTimed  bool
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
	return connect.NewResponse(&runspb.WaitRunResponse{Status: s.waitStatus, TimedOut: s.waitTimed}), nil
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
	withStreamServer(t, &streamServer{waitStatus: &runspb.RunLiveStatus{RunId: "R", Status: "passed"}})
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
