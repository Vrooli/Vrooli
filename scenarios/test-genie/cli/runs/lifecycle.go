package runs

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/term"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliutil"

	cliexec "test-genie/cli/execute"
	"test-genie/cli/execute/report"
	execTypes "test-genie/cli/internal/execute"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// (JSON output uses the shared writeJSON helper defined in command.go.)

// Exit code for a wait that returned before the run reached a terminal state.
const exitWaitTimeout = 124

// stderrOut is where a --json wait writes its non-JSON timeout hint, so stdout
// stays pure JSON for parsers. Overridable in tests.
var stderrOut io.Writer = os.Stderr

const eagerWaitWindow = 30 * time.Second

var (
	waitNow       = time.Now
	waitStatePath = defaultWaitStatePath
	parkForAwait  = cliutil.ParkForAwait
)

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

func defaultWaitStatePath() string {
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "vrooli", "test-genie", "wait-attempts.json")
}

func waitKey(scenario, runID string) string {
	return strings.TrimSpace(scenario) + "/" + strings.TrimSpace(runID)
}

func readWaitAttempts(path string) map[string]int64 {
	attempts := map[string]int64{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &attempts)
	}
	return attempts
}

func writeWaitAttempts(path string, attempts map[string]int64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(attempts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	return nil
}

func recordWaitAttempt(scenario, runID string) error {
	path := waitStatePath()
	attempts := readWaitAttempts(path)
	attempts[waitKey(scenario, runID)] = waitNow().Unix()
	return writeWaitAttempts(path, attempts)
}

func clearWaitAttempt(scenario, runID string) {
	path := waitStatePath()
	attempts := readWaitAttempts(path)
	delete(attempts, waitKey(scenario, runID))
	_ = writeWaitAttempts(path, attempts)
}

func warnIfEagerWait(w io.Writer, scenario, runID string) {
	attempts := readWaitAttempts(waitStatePath())
	prevUnix, hadPrev := attempts[waitKey(scenario, runID)]
	if !hadPrev {
		return
	}
	prev := time.Unix(prevUnix, 0)
	elapsed := waitNow().Sub(prev)
	if elapsed < 0 || elapsed > eagerWaitWindow {
		return
	}
	fmt.Fprintf(w, "recent wait detected for %s/%s (%s ago). Do not poll with short output checks.\n", scenario, runID, elapsed.Round(time.Second))
	fmt.Fprintf(w, "If a previous wait process is still running, attach to it instead:\n")
	fmt.Fprintf(w, "  pgrep -af 'test-genie runs wait --json .* %s %s'\n", scenario, runID)
	fmt.Fprintf(w, "  tail --pid=<pid> -f /dev/null\n")
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
	// The public usage documents flags after the run handle. flag.FlagSet stops
	// parsing at the first positional argument, so normalize the two supported
	// wait flags before parsing. Without this, `runs wait demo R --json` silently
	// took the human streaming path instead of emitting its machine snapshot.
	args = normalizeWaitFlags(args)
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
	// JSON wait is the documented machine contract: it must return a real
	// WaitRun snapshot and exit code. Parking here can return success before a
	// queued run is dispatched when the caller's agent host does not deliver the
	// wake callback. Human waits may still park to save an agent turn.
	if !*jsonOut {
		if park, parked, perr := parkForAwait(cliutil.ParkRequest{
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
	}

	cl, err := client(apiClient)
	if err != nil {
		return err
	}

	if *jsonOut {
		return waitSnapshot(cl, w, scenario, runID, *timeout)
	}
	if *timeout > 0 {
		warnIfEagerWait(w, scenario, runID)
	}

	// Human mode: stream to terminal (optionally bounded by --timeout).
	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeout)*time.Second)
		defer cancel()
	}
	terminal, phases, streamErr := streamRunEvents(ctx, cl, w, scenario, runID, suppressHeartbeatsFor(w, false))
	if ctx.Err() == context.DeadlineExceeded {
		_ = recordWaitAttempt(scenario, runID)
		fmt.Fprintln(w)
		printTimeoutHint(w, scenario, runID, *timeout, fetchNextCheck(cl, scenario, runID))
		return &exitErr{code: exitWaitTimeout, err: fmt.Errorf("run %s did not finish within the wait window", runID)}
	}
	clearWaitAttempt(scenario, runID)
	printTerminalStandingView(cl, w, scenario, runID, terminal, phases)
	return terminalExit(runID, terminal, streamErr)
}

// waitSnapshot is the scripted `--json` path: one WaitRun call, one structured
// status, the suite exit code.
func waitSnapshot(cl runs_v1connect.RunsServiceClient, w io.Writer, scenario, runID string, timeout int) error {
	if timeout > 0 {
		warnIfEagerWait(stderrOut, scenario, runID)
	}
	resp, err := cl.WaitRun(context.Background(), connect.NewRequest(&runspb.WaitRunRequest{
		Scenario: scenario, RunId: runID, TimeoutSeconds: int32(timeout),
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	st := resp.Msg.GetStatus()
	view := cliexec.BuildRunStandingViewFromWaitResponse(context.Background(), resp.Msg, report.RunScoreCLI)
	if err := cliexec.WriteRunStandingJSON(w, view); err != nil {
		return err
	}
	if resp.Msg.GetTimedOut() {
		_ = recordWaitAttempt(scenario, runID)
		// stdout stays pure JSON (the snapshot already carries
		// recommended_next_check_seconds); the human-readable cadence + re-invoke
		// line goes to stderr.
		printTimeoutHint(stderrOut, scenario, runID, timeout, int(st.GetRecommendedNextCheckSeconds()))
		return &exitErr{code: exitWaitTimeout, err: fmt.Errorf("run %s did not finish within the wait window", runID)}
	}
	clearWaitAttempt(scenario, runID)
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
	terminal, phases, streamErr := streamRunEvents(context.Background(), cl, w, scenario, runID, suppressHeartbeatsFor(w, *heartbeats))
	printTerminalStandingView(cl, w, scenario, runID, terminal, phases)
	return terminalExit(runID, terminal, streamErr)
}

// normalizeWaitFlags supports the documented trailing-flag spelling while
// preserving FlagSet's normal validation of each recognized flag. It is kept
// deliberately narrow: unknown arguments remain positional and are rejected by
// twoPositional rather than being silently reinterpreted.
func normalizeWaitFlags(args []string) []string {
	flags := make([]string, 0, 3)
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--json":
			flags = append(flags, arg)
		case arg == "--timeout":
			flags = append(flags, arg)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		case strings.HasPrefix(arg, "--timeout="):
			flags = append(flags, arg)
		default:
			positionals = append(positionals, arg)
		}
	}
	return append(flags, positionals...)
}

// streamRunEvents follows a run, rendering each event with printFollowEvent, and
// returns the terminal run_completed event (nil if the stream ended without one,
// e.g. the context deadline elapsed). It is the single stream→render loop shared
// by `runs follow`, human `runs wait`, and the inline execute follower.
// suppressHeartbeats opts this follower out of heartbeat keep-alives server-side.
func streamRunEvents(ctx context.Context, cl runs_v1connect.RunsServiceClient, w io.Writer, scenario, runID string, suppressHeartbeats bool) (*runspb.RunEvent, []execTypes.Phase, error) {
	stream, err := cl.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID, SuppressHeartbeats: suppressHeartbeats,
	}))
	if err != nil {
		return nil, nil, err
	}
	var terminal *runspb.RunEvent
	var phases []execTypes.Phase
	for stream.Receive() {
		ev := stream.Msg()
		printFollowEvent(w, ev)
		switch ev.GetEvent() {
		case "phase_completed", "phase_failed":
			phases = append(phases, phaseFromRunEvent(ev))
		}
		if ev.GetEvent() == "run_completed" {
			terminal = ev
		}
	}
	return terminal, phases, stream.Err()
}

func printTerminalStandingView(cl runs_v1connect.RunsServiceClient, w io.Writer, scenario, runID string, terminal *runspb.RunEvent, phases []execTypes.Phase) {
	if terminal == nil {
		return
	}
	// FollowRun is an event stream, not a durable report. A late subscriber to
	// an already-completed run can legitimately receive only run_completed;
	// rendering that event alone used to fabricate a 0/0/0 result. Rehydrate
	// from WaitRun's canonical terminal snapshot before printing the summary so
	// human wait/follow agree with JSON wait and `runs show`.
	if snapshot, err := cl.WaitRun(context.Background(), connect.NewRequest(&runspb.WaitRunRequest{
		Scenario: scenario,
		RunId:    runID,
	})); err == nil && !snapshot.Msg.GetTimedOut() && snapshot.Msg.GetTerminalRun() != nil && len(snapshot.Msg.GetDegradedReasons()) == 0 {
		view := cliexec.BuildRunStandingViewFromWaitResponse(context.Background(), snapshot.Msg, report.RunScoreCLI)
		if len(view.Phases) > 0 || len(phases) == 0 {
			pr := report.New(w, view.Scenario, "", nil, nil, false, nil, nil)
			pr.SetStreamedObservations(true)
			pr.PrintResultsView(view)
			return
		}
	}
	resp := cliexec.Response{
		Success:     terminal.GetSuccess(),
		Verdict:     terminal.GetVerdict(),
		ExecutionID: runID,
		Phases:      phases,
		Error:       terminal.GetError(),
	}
	resp.PhaseSummary = summarizeRunPhases(phases)
	view := cliexec.BuildRunStandingView(context.Background(), resp, firstNonEmpty(terminal.GetScenario(), scenario), "", runID, nil, false, 0, report.RunScoreCLI)
	pr := report.New(w, view.Scenario, "", nil, nil, false, nil, nil)
	pr.SetStreamedObservations(true)
	pr.PrintResultsView(view)
}

func phaseFromRunEvent(ev *runspb.RunEvent) execTypes.Phase {
	return execTypes.Phase{
		Name:              ev.GetPhase(),
		Status:            ev.GetStatus(),
		DurationSeconds:   float64(ev.GetDurationSeconds()),
		Error:             ev.GetError(),
		PhasePresentation: ev.GetPhasePresentation(),
		FindingsSummary:   cliexec.FindingsSummaryFromProto(ev.GetFindingsSummary()),
	}
}

func summarizeRunPhases(phases []execTypes.Phase) execTypes.PhaseSummary {
	var summary execTypes.PhaseSummary
	for _, phase := range phases {
		summary.Total++
		switch phase.Status {
		case "passed":
			summary.Passed++
		case "failed", "aborted":
			summary.Failed++
		case "skipped":
			summary.Skipped++
		}
	}
	return summary
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
	action := statusWaitAction(resp.Msg)
	if *jsonOut {
		return writeStatusJSON(w, resp.Msg, action)
	}
	printLiveStatus(w, resp.Msg)
	if action != nil {
		fmt.Fprintln(w, "Run exactly once:")
		fmt.Fprintf(w, "  %s\n", action.Command)
		fmt.Fprintln(w, "This blocks quietly until terminal; do not poll status for completion.")
	}
	return nil
}

type statusNextAction struct {
	Kind      string `json:"kind"`
	Command   string `json:"command"`
	Timeout   int    `json:"timeout"`
	DoNotPoll bool   `json:"doNotPoll"`
}

func statusWaitAction(st *runspb.RunLiveStatus) *statusNextAction {
	if st == nil || st.GetStatus() != "in_progress" {
		return nil
	}
	timeout := cliexec.RecommendedWaitSeconds(int(st.GetEstimatedRemainingSeconds()), st.GetEtaKnown())
	return &statusNextAction{
		Kind: "wait", Command: cliexec.ReattachCommandWithTimeout(st.GetScenario(), st.GetRunId(), timeout),
		Timeout: timeout, DoNotPoll: true,
	}
}

func writeStatusJSON(w io.Writer, st *runspb.RunLiveStatus, action *statusNextAction) error {
	raw, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(st)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if action != nil {
		payload["nextAction"] = action
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
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
		fmt.Fprintf(w, "eta:      ~%ds remaining\n", st.GetEstimatedRemainingSeconds())
		fmt.Fprintf(w, "status backoff metadata: %ds (for nonblocking dashboards only)\n", st.GetRecommendedNextCheckSeconds())
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
		printFollowStanding(w, ev)
	case "phase_failed":
		fmt.Fprintf(w, "✗  %s (%ds)\n", ev.GetPhase(), ev.GetDurationSeconds())
		printFollowStanding(w, ev)
	case "run_completed":
		fmt.Fprintf(w, "\n%s\n", ev.GetVerdict())
		if ev.GetScenario() != "" && ev.GetRunId() != "" {
			fmt.Fprintf(w, "findings: test-genie runs findings %s %s\n", ev.GetScenario(), ev.GetRunId())
		}
	}
}

func printFollowStanding(w io.Writer, ev *runspb.RunEvent) {
	presentation := ev.GetPhasePresentation()
	if presentation == nil {
		return
	}
	pr := report.New(w, ev.GetScenario(), "", nil, nil, false, nil, nil)
	pr.PrintPhaseStanding(execTypes.Phase{
		Name:              ev.GetPhase(),
		Status:            ev.GetStatus(),
		DurationSeconds:   float64(ev.GetDurationSeconds()),
		Error:             ev.GetError(),
		PhasePresentation: presentation,
		FindingsSummary:   cliexec.FindingsSummaryFromProto(ev.GetFindingsSummary()),
	})
}
