package execute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"connectrpc.com/connect"

	"test-genie/cli/execute/report"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// Canonical run-event kinds (mirrors the server's run manager vocabulary).
const (
	evRunStarted     = "run_started"
	evPhaseStarted   = "phase_started"
	evPhaseProgress  = "phase_progress"
	evPhaseHeartbeat = "phase_heartbeat"
	evPhaseCompleted = "phase_completed"
	evPhaseFailed    = "phase_failed"
	evRunCompleted   = "run_completed"
)

// DurableOptions configures the durable execute flow.
type DurableOptions struct {
	// Wait forces a block-to-completion inline follow regardless of ETA (used by
	// CI and the lifecycle test phase).
	Wait bool
	// JSONL emits the canonical newline-delimited event stream to stdout instead
	// of human rendering.
	JSONL bool
	// Printer renders the final human summary (nil in JSONL mode).
	Printer *report.Printer
}

// RunDurable starts a server-owned, cancel-survivable run and either follows it
// inline (printing the run id and re-attach command up front) or — for a
// known-long run without --wait — launches it in the background and returns
// immediately with the re-attach command. The run is NEVER aborted by this
// process exiting or being interrupted.
func RunDurable(baseURL string, req Request, opts DurableOptions) error {
	client := newRunsClient(baseURL)
	ctx := context.Background()

	start, err := client.StartRun(ctx, connect.NewRequest(toStartRunRequest(req)))
	if err != nil {
		if msg, ok := runBusyGuidance(err); ok {
			fmt.Fprint(os.Stderr, msg)
			return fmt.Errorf("scenario %s is busy with another run", req.ScenarioName)
		}
		return fmt.Errorf("start run: %w", err)
	}
	runID := start.Msg.GetRunId()
	eta := int(start.Msg.GetEstimatedTotalSeconds())
	etaKnown := start.Msg.GetEtaKnown()

	// Coalesced: this request matched an already-in-flight identical run and rode
	// it instead of starting a second suite (the one-run-per-scenario guard).
	if start.Msg.GetCoalesced() {
		fmt.Fprintf(os.Stderr, "↻ Re-attached to in-flight run %s for %s (identical request already running — no second suite).\n", runID, req.ScenarioName)
	}

	printRunBanner(os.Stderr, req.ScenarioName, runID, eta, etaKnown)

	if opts.JSONL {
		return followJSONL(ctx, client, req.ScenarioName, runID, os.Stdout)
	}

	// Auto-background a known-long run — or a run whose ETA is unknown (which
	// could be long) — unless the caller forced --wait. A known-short run follows
	// inline. This keeps an agent's tool from blocking past its timeout on a long
	// or unestimatable run.
	threshold, enabled := autoBackgroundThreshold()
	knownLong := etaKnown && eta >= threshold
	unknownLong := !etaKnown && autoBackgroundOnUnknownETA()
	if !opts.Wait && enabled && (knownLong || unknownLong) {
		if etaKnown {
			fmt.Fprintf(os.Stderr, "\n⏳ Estimated at %s — running in the background so your shell returns now.\n", humanDuration(eta))
		} else {
			fmt.Fprintf(os.Stderr, "\n⏳ ETA unknown (treating as long) — running in the background so your shell returns now.\n")
		}
		fmt.Fprintf(os.Stderr, "   Block on the result with:\n     %s\n", reattachCommand(req.ScenarioName, runID))
		return nil
	}

	return followInline(ctx, client, req.ScenarioName, runID, opts)
}

// followInline streams the run to completion, rendering progress. A SIGINT/
// SIGTERM detaches the viewer (printing the re-attach command) WITHOUT aborting
// the run.
func followInline(parent context.Context, client runs_v1connect.RunsServiceClient, scenario, runID string, opts DurableOptions) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	detached := false
	go func() {
		select {
		case <-sig:
			detached = true
			cancel()
		case <-ctx.Done():
		}
	}()

	stream, err := client.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{Scenario: scenario, RunId: runID}))
	if err != nil {
		return fmt.Errorf("follow run: %w", err)
	}

	var phasesAcc []Phase
	var terminal *runspb.RunEvent
	for stream.Receive() {
		ev := stream.Msg()
		renderLiveEvent(os.Stdout, ev, &phasesAcc)
		if ev.GetEvent() == evRunCompleted {
			terminal = ev
		}
	}
	if streamErr := stream.Err(); streamErr != nil {
		if detached || errors.Is(ctx.Err(), context.Canceled) {
			printDetached(os.Stderr, scenario, runID)
			return nil
		}
		return fmt.Errorf("follow run: %w", streamErr)
	}
	if detached {
		printDetached(os.Stderr, scenario, runID)
		return nil
	}

	resp := buildResponse(terminal, phasesAcc)
	if opts.Printer != nil {
		opts.Printer.SetStreamedObservations(true)
		opts.Printer.PrintResults(resp)
	}
	return executionResultError(resp)
}

// followJSONL streams the canonical event vocabulary as newline-delimited JSON.
// Because FollowRun replays history, the first line is always run_started
// (carrying the run id), and the last is run_completed.
func followJSONL(ctx context.Context, client runs_v1connect.RunsServiceClient, scenario, runID string, out io.Writer) error {
	stream, err := client.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{Scenario: scenario, RunId: runID}))
	if err != nil {
		_ = emitEventLine(out, map[string]any{"event": evRunCompleted, "success": false, "error": err.Error()})
		return err
	}
	var terminal *runspb.RunEvent
	for stream.Receive() {
		ev := stream.Msg()
		if err := emitEventLine(out, eventToMap(ev)); err != nil {
			return err
		}
		if ev.GetEvent() == evRunCompleted {
			terminal = ev
		}
	}
	if streamErr := stream.Err(); streamErr != nil {
		_ = emitEventLine(out, map[string]any{"event": evRunCompleted, "success": false, "error": streamErr.Error()})
		return streamErr
	}
	if terminal != nil && !terminal.GetSuccess() {
		return errors.New("suite execution completed with failures")
	}
	return nil
}

func emitEventLine(out io.Writer, fields map[string]any) error {
	line, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", line)
	return err
}

// eventToMap renders a RunEvent as the canonical JSONL object, dropping empty
// fields so each line carries only what is relevant to its event.
func eventToMap(ev *runspb.RunEvent) map[string]any {
	m := map[string]any{
		"event":           ev.GetEvent(),
		"elapsed_seconds": ev.GetElapsedSeconds(),
	}
	put := func(k, v string) {
		if v != "" {
			m[k] = v
		}
	}
	put("run_id", ev.GetRunId())
	put("scenario", ev.GetScenario())
	put("artifact_dir", ev.GetArtifactDir())
	put("preset", ev.GetPreset())
	put("phase", ev.GetPhase())
	put("status", ev.GetStatus())
	put("message", ev.GetMessage())
	put("error", ev.GetError())
	if v := ev.GetVerdict(); v != "" {
		m["verdict"] = v
	}
	if ev.GetPhaseTotal() > 0 {
		m["phase_index"] = ev.GetPhaseIndex()
		m["phase_total"] = ev.GetPhaseTotal()
	}
	if ev.GetDurationSeconds() > 0 {
		m["duration_seconds"] = ev.GetDurationSeconds()
	}
	if ev.GetQuietSeconds() > 0 {
		m["quiet_seconds"] = ev.GetQuietSeconds()
	}
	if ev.GetEvent() == evRunCompleted {
		m["success"] = ev.GetSuccess()
	}
	return m
}

// renderLiveEvent prints a concise human line for a run event.
func renderLiveEvent(w io.Writer, ev *runspb.RunEvent, phasesAcc *[]Phase) {
	switch ev.GetEvent() {
	case evPhaseStarted:
		fmt.Fprintf(w, "▶  %s\n", ev.GetPhase())
	case evPhaseProgress:
		if msg := strings.TrimSpace(ev.GetMessage()); msg != "" {
			fmt.Fprintf(w, "   %s\n", msg)
		}
	case evPhaseHeartbeat:
		fmt.Fprintf(w, "   … %s still running (%.0fs quiet)\n", ev.GetPhase(), ev.GetQuietSeconds())
	case evPhaseCompleted:
		fmt.Fprintf(w, "✓  %s (%ds)\n", ev.GetPhase(), ev.GetDurationSeconds())
		*phasesAcc = append(*phasesAcc, phaseFromEvent(ev))
	case evPhaseFailed:
		fmt.Fprintf(w, "✗  %s (%ds)\n", ev.GetPhase(), ev.GetDurationSeconds())
		*phasesAcc = append(*phasesAcc, phaseFromEvent(ev))
	}
}

func phaseFromEvent(ev *runspb.RunEvent) Phase {
	return Phase{
		Name:            ev.GetPhase(),
		Status:          ev.GetStatus(),
		DurationSeconds: float64(ev.GetDurationSeconds()),
		Error:           ev.GetError(),
	}
}

// buildResponse assembles a Response for the final summary from the accumulated
// phase events and the terminal run_completed event.
func buildResponse(terminal *runspb.RunEvent, phasesAcc []Phase) Response {
	resp := Response{Phases: phasesAcc}
	if terminal != nil {
		resp.Success = terminal.GetSuccess()
		resp.Verdict = terminal.GetVerdict()
		resp.Error = terminal.GetError()
	}
	for _, p := range phasesAcc {
		resp.PhaseSummary.Total++
		switch p.Status {
		case "passed":
			resp.PhaseSummary.Passed++
		case "failed":
			resp.PhaseSummary.Failed++
		case "skipped":
			resp.PhaseSummary.Skipped++
		}
	}
	return resp
}

func printRunBanner(w io.Writer, scenario, runID string, eta int, etaKnown bool) {
	etaStr := "unknown"
	if etaKnown {
		etaStr = humanDuration(eta)
	}
	fmt.Fprintf(w, "▶ run %s started (estimated %s)\n", runID, etaStr)
	fmt.Fprintf(w, "  The test-genie server owns this run — if your shell or tool times out, the run keeps going.\n")
	fmt.Fprintf(w, "  That is expected and recoverable: re-attach to live progress with\n    %s\n\n", reattachCommand(scenario, runID))
}

func printDetached(w io.Writer, scenario, runID string) {
	fmt.Fprintf(w, "\n⏸ Detached from run %s (still running).\n   Re-attach with:\n     %s\n", runID, reattachCommand(scenario, runID))
}

// runBusyGuidance extracts the one-run-per-scenario rejection (a divergent run
// is already in flight) and renders wait/abort guidance. ok=false for any other
// error. No client polling — the agent waits server-side or aborts.
func runBusyGuidance(err error) (string, bool) {
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeFailedPrecondition {
		return "", false
	}
	for _, d := range ce.Details() {
		msg, derr := d.Value()
		if derr != nil {
			continue
		}
		bi, ok := msg.(*runspb.RunBusyInfo)
		if !ok {
			continue
		}
		preset := bi.GetPreset()
		if preset == "" {
			preset = "default"
		}
		var b strings.Builder
		fmt.Fprintf(&b, "✗ %s already has an in-progress run %s (preset %s) — only one run per scenario at a time.\n", bi.GetScenario(), bi.GetRunId(), preset)
		fmt.Fprintf(&b, "  wait:  test-genie runs follow %s %s\n", bi.GetScenario(), bi.GetRunId())
		fmt.Fprintf(&b, "  abort: test-genie runs abort %s %s\n", bi.GetScenario(), bi.GetRunId())
		return b.String(), true
	}
	return "", false
}

func toStartRunRequest(req Request) *runspb.StartRunRequest {
	return &runspb.StartRunRequest{
		Scenario:               req.ScenarioName,
		Preset:                 req.Preset,
		Phases:                 req.Phases,
		Skip:                   req.Skip,
		FailFast:               req.FailFast,
		DiagnosticsPreset:      req.DiagnosticsPreset,
		UiUrl:                  req.UIURL,
		ApiUrl:                 req.APIURL,
		ScenarioPath:           req.ScenarioPath,
		LogicalRepoRoot:        req.LogicalRepoRoot,
		LogicalScenarioRelPath: req.LogicalScenarioRelPath,
		SuiteRequestId:         req.SuiteRequestID,
	}
}
