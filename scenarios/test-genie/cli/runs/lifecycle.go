package runs

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// (JSON output uses the shared writeJSON helper defined in command.go.)

// Exit code for a wait that returned before the run reached a terminal state.
const exitWaitTimeout = 124

// twoPositional extracts the <scenario> <runID> positional arguments shared by
// the by-handle verbs (wait/follow/abort/status).
func twoPositional(args []string, verb string) (scenario, runID string, err error) {
	if len(args) < 2 {
		return "", "", fmt.Errorf("usage: test-genie runs %s <scenario> <runID>", verb)
	}
	return args[0], args[1], nil
}

// runWait blocks until the run is terminal (or --timeout elapses) and exits with
// the suite's result code: 0 passed, 1 failed/aborted, 124 still-in-progress.
//
// Human mode STREAMS the run's live events (the same renderer as `runs follow`)
// so a waiting agent sees progress + heartbeats and never reads the wait as a
// silent hang. `--json` keeps the single quiet WaitRun snapshot for scripts that
// want one structured result and an exit code.
func runWait(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs wait", flag.ContinueOnError)
	fs.SetOutput(w)
	timeout := fs.Int("timeout", 0, "Max seconds to block (0 = until the run is terminal)")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scenario, runID, err := twoPositional(fs.Args(), "wait")
	if err != nil {
		return err
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}

	if *jsonOut {
		return waitSnapshot(cl, w, scenario, runID, *timeout)
	}

	// Human mode: stream to terminal (optionally bounded by --timeout).
	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeout)*time.Second)
		defer cancel()
	}
	terminal, streamErr := streamRunEvents(ctx, cl, w, scenario, runID)
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Fprintf(w, "\nstill running — re-attach: test-genie runs follow %s %s\n", scenario, runID)
		return &exitErr{code: exitWaitTimeout, err: fmt.Errorf("run %s did not finish within the wait window", runID)}
	}
	return terminalExit(runID, terminal, streamErr)
}

// waitSnapshot is the scripted `--json` path: one WaitRun call, one structured
// status, the suite exit code.
func waitSnapshot(cl runs_v1connect.RunsServiceClient, w io.Writer, scenario, runID string, timeout int) error {
	resp, err := cl.WaitRun(context.Background(), connect.NewRequest(&runspb.WaitRunRequest{
		Scenario: scenario, RunId: runID, TimeoutSeconds: int32(timeout),
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	st := resp.Msg.GetStatus()
	if err := writeJSON(w, st); err != nil {
		return err
	}
	if resp.Msg.GetTimedOut() {
		return &exitErr{code: exitWaitTimeout, err: fmt.Errorf("run %s did not finish within the wait window", runID)}
	}
	if st.GetStatus() == "passed" {
		return nil
	}
	return &exitErr{code: exitRegression, err: fmt.Errorf("run %s %s", runID, st.GetStatus())}
}

// runFollow streams the run's canonical events to completion, exiting with the
// suite's result code.
func runFollow(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs follow", flag.ContinueOnError)
	fs.SetOutput(w)
	if err := fs.Parse(args); err != nil {
		return err
	}
	scenario, runID, err := twoPositional(fs.Args(), "follow")
	if err != nil {
		return err
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	terminal, streamErr := streamRunEvents(context.Background(), cl, w, scenario, runID)
	return terminalExit(runID, terminal, streamErr)
}

// streamRunEvents follows a run, rendering each event with printFollowEvent, and
// returns the terminal run_completed event (nil if the stream ended without one,
// e.g. the context deadline elapsed). It is the single stream→render loop shared
// by `runs follow`, human `runs wait`, and the inline execute follower.
func streamRunEvents(ctx context.Context, cl runs_v1connect.RunsServiceClient, w io.Writer, scenario, runID string) (*runspb.RunEvent, error) {
	stream, err := cl.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return nil, err
	}
	var terminal *runspb.RunEvent
	for stream.Receive() {
		ev := stream.Msg()
		printFollowEvent(w, ev)
		if ev.GetEvent() == "run_completed" {
			terminal = ev
		}
	}
	return terminal, stream.Err()
}

// terminalExit maps a stream's terminal event + error to the suite exit code:
// stream error → not-comparable; terminal failure → regression; else pass. A
// nil terminal with no error (deadline handled by the caller) is treated as a
// clean detach.
func terminalExit(runID string, terminal *runspb.RunEvent, streamErr error) error {
	if streamErr != nil && terminal == nil {
		return &exitErr{code: exitNotComparable, err: streamErr}
	}
	if terminal != nil && !terminal.GetSuccess() {
		return &exitErr{code: exitRegression, err: fmt.Errorf("run %s %s", runID, terminal.GetVerdict())}
	}
	return nil
}

// runAbort cancels a running run and reports its terminal aborted status.
func runAbort(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs abort", flag.ContinueOnError)
	fs.SetOutput(w)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scenario, runID, err := twoPositional(fs.Args(), "abort")
	if err != nil {
		return err
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.AbortRun(context.Background(), connect.NewRequest(&runspb.AbortRunRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg.GetStatus())
	}
	printLiveStatus(w, resp.Msg.GetStatus())
	return nil
}

// runStatus prints a live snapshot (status, active phase, elapsed, remaining
// ETA, recommended next-check backoff).
func runStatus(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs status", flag.ContinueOnError)
	fs.SetOutput(w)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scenario, runID, err := twoPositional(fs.Args(), "status")
	if err != nil {
		return err
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.GetRunStatus(context.Background(), connect.NewRequest(&runspb.GetRunStatusRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg)
	}
	printLiveStatus(w, resp.Msg)
	return nil
}

func printLiveStatus(w io.Writer, st *runspb.RunLiveStatus) {
	if st == nil {
		return
	}
	fmt.Fprintf(w, "run:      %s\n", st.GetRunId())
	fmt.Fprintf(w, "scenario: %s\n", st.GetScenario())
	fmt.Fprintf(w, "status:   %s\n", st.GetStatus())
	if st.GetActivePhase() != "" {
		fmt.Fprintf(w, "phase:    %s (%d/%d)\n", st.GetActivePhase(), st.GetPhaseIndex(), st.GetPhaseTotal())
	}
	fmt.Fprintf(w, "elapsed:  %.0fs\n", st.GetElapsedSeconds())
	if st.GetEtaKnown() && st.GetStatus() == "in_progress" {
		fmt.Fprintf(w, "eta:      ~%ds remaining (check again in %ds)\n", st.GetEstimatedRemainingSeconds(), st.GetRecommendedNextCheckSeconds())
	}
	if st.GetVerdict() != "" {
		fmt.Fprintf(w, "verdict:  %s\n", st.GetVerdict())
	}
	if st.GetError() != "" {
		fmt.Fprintf(w, "error:    %s\n", st.GetError())
	}
}

func printFollowEvent(w io.Writer, ev *runspb.RunEvent) {
	switch ev.GetEvent() {
	case "phase_started":
		fmt.Fprintf(w, "▶  %s\n", ev.GetPhase())
	case "phase_progress":
		if ev.GetMessage() != "" {
			fmt.Fprintf(w, "   %s\n", ev.GetMessage())
		}
	case "phase_heartbeat":
		fmt.Fprintf(w, "   … %s still running (%.0fs quiet)\n", ev.GetPhase(), ev.GetQuietSeconds())
	case "phase_completed":
		fmt.Fprintf(w, "✓  %s (%ds)\n", ev.GetPhase(), ev.GetDurationSeconds())
	case "phase_failed":
		fmt.Fprintf(w, "✗  %s (%ds)\n", ev.GetPhase(), ev.GetDurationSeconds())
	case "run_completed":
		fmt.Fprintf(w, "\n%s\n", ev.GetVerdict())
	}
}
