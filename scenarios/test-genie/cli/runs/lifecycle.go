package runs

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/term"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// (JSON output uses the shared writeJSON helper defined in command.go.)

// Exit code for a wait that returned before the run reached a terminal state.
const exitWaitTimeout = 124

// stderrOut is where a --json wait writes its non-JSON timeout hint, so stdout
// stays pure JSON for parsers. Overridable in tests.
var stderrOut io.Writer = os.Stderr

// printTimeoutHint emits the cadence governor + the exact re-invoke command after
// a --timeout wait returned before the run was terminal. nextCheck is the
// server's recommended_next_check_seconds (0 → the cadence line is omitted). The
// re-invoke line is the quiet wait verb, never a faster poll.
func printTimeoutHint(w io.Writer, scenario, runID string, timeout, nextCheck int) {
	if nextCheck > 0 {
		fmt.Fprintf(w, "still running — re-check in ~%ds (do not poll faster):\n", nextCheck)
	} else {
		fmt.Fprintf(w, "still running — re-attach with:\n")
	}
	fmt.Fprintf(w, "  test-genie runs wait --json --timeout=%d %s %s\n", timeout, scenario, runID)
}

// fetchNextCheck best-effort reads the server's recommended backoff for a run
// (used by the human-stream timeout path, which has no live status in hand).
// Returns 0 on any error.
func fetchNextCheck(cl runs_v1connect.RunsServiceClient, scenario, runID string) int {
	resp, err := cl.GetRunStatus(context.Background(), connect.NewRequest(&runspb.GetRunStatusRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return 0
	}
	return int(resp.Msg.GetRecommendedNextCheckSeconds())
}

// twoPositional extracts the <scenario> <runID> positional arguments shared by
// the by-handle verbs (wait/follow/abort/status).
func twoPositional(args []string, verb string) (scenario, runID string, err error) {
	if len(args) < 2 {
		return "", "", fmt.Errorf("usage: test-genie runs %s <scenario> <runID>", verb)
	}
	return args[0], args[1], nil
}

// isTTY reports whether w is a terminal (mirrors report/color.go). Heartbeat
// keep-alives are kept for an interactive terminal and dropped otherwise.
func isTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// suppressHeartbeatsFor decides whether THIS follower opts out of heartbeat
// keep-alives: a non-interactive consumer (piped/backgrounded stdout) does,
// unless forceKeep (the --heartbeats override) says otherwise. A backgrounded
// stream that beats every ~30s re-wakes an agent on each beat — the spam this
// suppression exists to prevent.
func suppressHeartbeatsFor(w io.Writer, forceKeep bool) bool {
	if forceKeep {
		return false
	}
	return !isTTY(w)
}

// runWait blocks until the run is terminal (or --timeout elapses) and exits with
// the suite's result code: 0 passed, 1 failed/aborted, 124 still-in-progress.
//
// `--json` is the quiet agent path: a single WaitRun snapshot + exit code, no
// stream. Human mode STREAMS the run's live events (the same renderer as `runs
// follow`); on a non-TTY (piped) stdout it suppresses heartbeat keep-alives so a
// backgrounded wait is not re-woken on every beat.
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

	// Inside an agent-manager run, park instead of blocking: agent-manager owns
	// the agent process and performs the wait on its behalf, waking the run with
	// the result injected as the next turn (zero tokens while parked). Outside an
	// AM run this is a no-op (parked=false) and we fall through to the normal
	// blocking wait — human / CI / raw-terminal behaviour is unchanged.
	if park, parked, perr := cliutil.ParkForAwait(cliutil.ParkRequest{
		Producer: cliutil.ParkProducerTestGenie,
		Key:      scenario + "/" + runID,
	}); parked {
		if perr == nil {
			fmt.Fprintln(w, park.Message)
			return nil
		}
		// We are in an AM run but park failed — degrade gracefully to the inline
		// wait (no worse than before park existed).
		fmt.Fprintf(stderrOut, "agent-manager park unavailable (%v) — waiting inline instead\n", perr)
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
	terminal, streamErr := streamRunEvents(ctx, cl, w, scenario, runID, suppressHeartbeatsFor(w, false))
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Fprintln(w)
		printTimeoutHint(w, scenario, runID, *timeout, fetchNextCheck(cl, scenario, runID))
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
		// stdout stays pure JSON (the snapshot already carries
		// recommended_next_check_seconds); the human-readable cadence + re-invoke
		// line goes to stderr.
		printTimeoutHint(stderrOut, scenario, runID, timeout, int(st.GetRecommendedNextCheckSeconds()))
		return &exitErr{code: exitWaitTimeout, err: fmt.Errorf("run %s did not finish within the wait window", runID)}
	}
	if st.GetStatus() == "passed" {
		return nil
	}
	return &exitErr{code: exitRegression, err: fmt.Errorf("run %s %s", runID, st.GetStatus())}
}

// runFollow streams the run's canonical events to completion, exiting with the
// suite's result code. On a non-TTY (piped) stdout it suppresses heartbeat
// keep-alives unless --heartbeats forces them on.
func runFollow(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs follow", flag.ContinueOnError)
	fs.SetOutput(w)
	heartbeats := fs.Bool("heartbeats", false, "Keep heartbeat keep-alive lines even when stdout is not a terminal")
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
	terminal, streamErr := streamRunEvents(context.Background(), cl, w, scenario, runID, suppressHeartbeatsFor(w, *heartbeats))
	return terminalExit(runID, terminal, streamErr)
}

// streamRunEvents follows a run, rendering each event with printFollowEvent, and
// returns the terminal run_completed event (nil if the stream ended without one,
// e.g. the context deadline elapsed). It is the single stream→render loop shared
// by `runs follow`, human `runs wait`, and the inline execute follower.
// suppressHeartbeats opts this follower out of heartbeat keep-alives server-side.
func streamRunEvents(ctx context.Context, cl runs_v1connect.RunsServiceClient, w io.Writer, scenario, runID string, suppressHeartbeats bool) (*runspb.RunEvent, error) {
	stream, err := cl.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID, SuppressHeartbeats: suppressHeartbeats,
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
