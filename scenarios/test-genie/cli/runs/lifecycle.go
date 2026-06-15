package runs

import (
	"context"
	"flag"
	"fmt"
	"io"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
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
	resp, err := cl.WaitRun(context.Background(), connect.NewRequest(&runspb.WaitRunRequest{
		Scenario: scenario, RunId: runID, TimeoutSeconds: int32(*timeout),
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	st := resp.Msg.GetStatus()
	if *jsonOut {
		return writeJSON(w, st)
	}
	printLiveStatus(w, st)

	if resp.Msg.GetTimedOut() {
		fmt.Fprintf(w, "\nstill running — re-attach: test-genie runs wait %s %s\n", scenario, runID)
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
	stream, err := cl.FollowRun(context.Background(), connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	var terminal *runspb.RunEvent
	for stream.Receive() {
		ev := stream.Msg()
		printFollowEvent(w, ev)
		if ev.GetEvent() == "run_completed" {
			terminal = ev
		}
	}
	if streamErr := stream.Err(); streamErr != nil {
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
