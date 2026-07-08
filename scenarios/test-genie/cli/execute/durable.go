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

	"github.com/vrooli/cli-core/cliapp"

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
	// CI and callers that need the real exit code inline).
	Wait bool
	// JSONL emits the canonical newline-delimited event stream to stdout instead
	// of human rendering.
	JSONL bool
	// JSON blocks to completion and emits the final execute Response as a single
	// JSON object to stdout. Like JSONL it shares the durable StartRun path — the
	// only difference from the human and JSONL modes is rendering, so --json is an
	// output contract, not a separate execution path.
	JSON bool
	// Printer renders the final human summary (nil in JSONL/JSON mode).
	Printer *report.Printer
}

// RunDurable starts a server-owned, cancel-survivable run and either follows it
// inline (printing the run id and re-attach command up front) or — for a
// known-long run without --wait — launches it in the background and returns
// immediately with the re-attach command. The run is NEVER aborted by this
// process exiting or being interrupted.
//
// It is built on the cli-core durable_run primitive (cliapp.RunDurable): the run
// STARTS mode-blind, and the output mode selects ONLY the renderer. Human,
// --json, and --jsonl therefore share one server-owned StartRun/follow path by
// construction — --json cannot select a separate synchronous execution lifecycle.
func RunDurable(baseURL string, req Request, opts DurableOptions) error {
	client := newRunsClient(baseURL)
	mode := cliapp.DurableRunModeFrom(opts.JSON, opts.JSONL)

	return cliapp.RunDurable(mode, cliapp.DurableRunSpec[*startedRun]{
		Start: func() (*startedRun, error) {
			start, err := client.StartRun(context.Background(), connect.NewRequest(toStartRunRequest(req)))
			if err != nil {
				return nil, err
			}
			runID := start.Msg.GetRunId()
			coalesced := start.Msg.GetCoalesced()
			return &startedRun{
				client:    client,
				req:       req,
				opts:      opts,
				runID:     runID,
				eta:       int(start.Msg.GetEstimatedTotalSeconds()),
				etaKnown:  start.Msg.GetEtaKnown(),
				coalesced: coalesced,
				handle:    newRunHandle(req.ScenarioName, runID, coalesced),
			}, nil
		},
		RenderStartError: func(mode cliapp.DurableRunMode, err error) error {
			if mode == cliapp.DurableRunJSON {
				// Machine consumers get a JSON error object on stdout, matching the
				// success-path shape, carrying the scenario (run id is unknown before a
				// successful start) instead of the human busy-guidance block.
				emitJSONError(os.Stdout, req.ScenarioName, "", err)
				return fmt.Errorf("start run: %w", err)
			}
			if msg, ok := runBusyGuidance(err); ok {
				fmt.Fprint(os.Stderr, msg)
				return fmt.Errorf("scenario %s is busy with another run", req.ScenarioName)
			}
			return fmt.Errorf("start run: %w", err)
		},
		Human: func(sr *startedRun) error { return sr.renderHuman() },
		JSON:  func(sr *startedRun) error { return sr.renderJSON() },
		JSONL: func(sr *startedRun) error { return sr.renderJSONL() },
	})
}

// startedRun is the durable_run handle: everything the per-mode renderers need
// once the mode-blind StartRun has returned.
type startedRun struct {
	client    runs_v1connect.RunsServiceClient
	req       Request
	opts      DurableOptions
	runID     string
	eta       int
	etaKnown  bool
	coalesced bool
	handle    RunHandle
}

// renderJSONL streams the canonical newline-delimited event vocabulary.
func (sr *startedRun) renderJSONL() error {
	return followJSONL(context.Background(), sr.client, sr.req.ScenarioName, sr.runID, os.Stdout)
}

// renderJSON emits the early run handle to stderr (so a long run is never opaque)
// and blocks to a single final JSON object on stdout.
func (sr *startedRun) renderJSON() error {
	emitStartHandle(os.Stderr, sr.handle, sr.eta, sr.etaKnown)
	return followJSONFinal(context.Background(), sr.client, sr.req.ScenarioName, sr.runID, sr.handle, os.Stdout)
}

// renderHuman prints the run banner and either auto-backgrounds a known/unknown
// long run (unless --wait) or follows it inline.
func (sr *startedRun) renderHuman() error {
	// Coalesced: this request matched an already-in-flight identical run and rode
	// it instead of starting a second suite (the one-run-per-scenario guard).
	if sr.coalesced {
		fmt.Fprintf(os.Stderr, "↻ Re-attached to in-flight run %s for %s (identical request already running — no second suite).\n", sr.runID, sr.req.ScenarioName)
	}

	// The banner is human/stderr guidance.
	printRunBanner(os.Stderr, sr.req.ScenarioName, sr.runID, sr.eta, sr.etaKnown)

	// Auto-background a known-long run — or a run whose ETA is unknown (which
	// could be long) — unless the caller forced --wait. A known-short run follows
	// inline. This keeps an agent's tool from blocking past its timeout on a long
	// or unestimatable run.
	threshold, enabled := autoBackgroundThreshold()
	knownLong := sr.etaKnown && sr.eta >= threshold
	unknownLong := !sr.etaKnown && autoBackgroundOnUnknownETA()
	if !sr.opts.Wait && enabled && (knownLong || unknownLong) {
		if sr.etaKnown {
			fmt.Fprintf(os.Stderr, "\n⏳ Estimated at %s — running in the background so your shell returns now.\n", humanDuration(sr.eta))
		} else {
			fmt.Fprintf(os.Stderr, "\n⏳ ETA unknown (treating as long) — running in the background so your shell returns now.\n")
		}
		fmt.Fprintf(os.Stderr, "   Block on the result (quiet, exits with the verdict) with:\n     %s\n", reattachCommand(sr.req.ScenarioName, sr.runID))
		fmt.Fprintf(os.Stderr, "   Review maturity findings with:\n     %s\n", findingsCommand(sr.req.ScenarioName, sr.runID))
		fmt.Fprintf(os.Stderr, "   Or watch live progress with:\n     %s\n", followCommand(sr.req.ScenarioName, sr.runID))
		return nil
	}

	return followInline(context.Background(), sr.client, sr.req.ScenarioName, sr.runID, sr.opts)
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

	// Suppress heartbeat keep-alives when stdout is not a terminal (piped or
	// backgrounded) so an agent following inline is not re-woken on every beat;
	// an interactive terminal keeps them as a liveness signal.
	stream, err := client.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID, SuppressHeartbeats: !isStdoutTTY(),
	}))
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
	// A machine consumer never wants heartbeat keep-alive noise — always suppress.
	stream, err := client.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID, SuppressHeartbeats: true,
	}))
	if err != nil {
		_ = emitEventLine(out, jsonlErrorEvent(scenario, runID, err))
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
		_ = emitEventLine(out, jsonlErrorEvent(scenario, runID, streamErr))
		return streamErr
	}
	if terminal != nil && !terminal.GetSuccess() {
		return errors.New("suite execution completed with failures")
	}
	return nil
}

// followJSONFinal follows the durable run to completion (no human rendering) and
// emits the assembled execute Response as one indented JSON object on stdout.
// This is the durable replacement for the legacy blocking `client.Run` path:
// human, --json, and --jsonl now all start from StartRun and differ only in how
// they render the same server-owned run. The Response carries success, verdict,
// the run id (as executionId), the per-phase results, and any terminal error.
func followJSONFinal(ctx context.Context, client runs_v1connect.RunsServiceClient, scenario, runID string, handle RunHandle, out io.Writer) error {
	stream, err := client.FollowRun(ctx, connect.NewRequest(&runspb.FollowRunRequest{
		Scenario: scenario, RunId: runID, SuppressHeartbeats: true,
	}))
	if err != nil {
		emitJSONError(out, scenario, runID, err)
		return fmt.Errorf("follow run: %w", err)
	}
	var phasesAcc []Phase
	var terminal *runspb.RunEvent
	for stream.Receive() {
		ev := stream.Msg()
		switch ev.GetEvent() {
		case evPhaseCompleted, evPhaseFailed:
			phasesAcc = append(phasesAcc, phaseFromEvent(ev))
		case evRunCompleted:
			terminal = ev
		}
	}
	if streamErr := stream.Err(); streamErr != nil {
		emitJSONError(out, scenario, runID, streamErr)
		return fmt.Errorf("follow run: %w", streamErr)
	}

	resp := buildResponse(terminal, phasesAcc)
	if resp.ExecutionID == "" {
		resp.ExecutionID = runID
	}
	// The terminal object carries the durable run identity + reattach/follow
	// commands so a consumer can reattach or audit from the final object alone.
	h := handle
	resp.RunHandle = &h
	if err := writeResponseJSON(out, resp); err != nil {
		return err
	}
	return executionResultError(resp)
}

// writeResponseJSON emits the execute Response as indented JSON with a trailing
// newline, matching the pretty-printed shape the legacy --json path produced.
func writeResponseJSON(out io.Writer, resp Response) error {
	body, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal execute response: %w", err)
	}
	_, err = fmt.Fprintf(out, "%s\n", body)
	return err
}

// emitJSONError writes a {"success":false,"error":...,"scenario":...,"runId":...}
// object so a --json consumer always receives parseable, actionable JSON on
// stdout even on a start/follow failure. The scenario is always known; runID is
// included only once a run has started (empty before StartRun succeeds), so a
// busy/coalescing/follow failure carries the id needed to reattach or abort.
func emitJSONError(out io.Writer, scenario, runID string, cause error) {
	fields := map[string]any{"success": false, "error": cause.Error()}
	if scenario != "" {
		fields["scenario"] = scenario
	}
	if runID != "" {
		fields["runId"] = runID
	}
	body, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		fmt.Fprintf(out, "{\"success\":false,\"error\":%q}\n", cause.Error())
		return
	}
	fmt.Fprintf(out, "%s\n", body)
}

// newRunHandle builds the structured, server-owned run identity shared by the
// early start-handle and the terminal --json object.
func newRunHandle(scenario, runID string, coalesced bool) RunHandle {
	return RunHandle{
		RunID:     runID,
		Scenario:  scenario,
		Reattach:  reattachCommand(scenario, runID),
		Follow:    followCommand(scenario, runID),
		Coalesced: coalesced,
	}
}

// emitStartHandle writes the early run-handle as one structured JSON line so a
// long --json run exposes its durable run identity immediately (on stderr —
// stdout stays the single terminal SuiteExecutionResult object). It mirrors the
// JSONL run_started event: event marker + identity + ETA + reattach breadcrumb.
func emitStartHandle(out io.Writer, handle RunHandle, eta int, etaKnown bool) {
	fields := map[string]any{
		"event":     evRunStarted,
		"run_id":    handle.RunID,
		"scenario":  handle.Scenario,
		"reattach":  handle.Reattach,
		"follow":    handle.Follow,
		"eta_known": etaKnown,
	}
	if etaKnown {
		fields["estimated_total_seconds"] = eta
	}
	if handle.Coalesced {
		fields["coalesced"] = true
	}
	line, err := json.Marshal(fields)
	if err != nil {
		return
	}
	fmt.Fprintf(out, "%s\n", line)
}

// jsonlErrorEvent builds the terminal run_completed error line for the JSONL
// stream, carrying the run identity so a consumer can reattach/abort instead of
// receiving only an opaque error string.
func jsonlErrorEvent(scenario, runID string, cause error) map[string]any {
	m := map[string]any{"event": evRunCompleted, "success": false, "error": cause.Error()}
	if scenario != "" {
		m["scenario"] = scenario
	}
	if runID != "" {
		m["run_id"] = runID
	}
	return m
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
		Name:             ev.GetPhase(),
		Status:           ev.GetStatus(),
		DurationSeconds:  float64(ev.GetDurationSeconds()),
		Error:            ev.GetError(),
		MaturityStanding: StandingFromProto(ev.GetMaturityStanding()),
		FindingsSummary:  FindingsSummaryFromProto(ev.GetFindingsSummary()),
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
	waitSeconds := recommendedWaitSeconds(eta, etaKnown)
	fmt.Fprintf(w, "▶ run %s started (estimated %s)\n", runID, etaStr)
	fmt.Fprintf(w, "  The test-genie server owns this run — if your shell or tool times out, the run keeps going.\n")
	printAgentWaitBlock(w, scenario, runID, eta, etaKnown, waitSeconds)
	fmt.Fprintf(w, "  Watch live progress with:\n    %s\n\n", followCommand(scenario, runID))
}

func printDetached(w io.Writer, scenario, runID string) {
	fmt.Fprintf(w, "\n⏸ Detached from run %s (still running).\n   Re-attach with:\n     %s\n   Review maturity findings with:\n     %s\n   (or watch live: %s)\n", runID, reattachCommand(scenario, runID), findingsCommand(scenario, runID), followCommand(scenario, runID))
}

func printAgentWaitBlock(w io.Writer, scenario, runID string, eta int, etaKnown bool, waitSeconds int) {
	etaStr := "unknown"
	if etaKnown {
		etaStr = humanDuration(eta)
	}
	fmt.Fprintf(w, "  Agent wait protocol:\n")
	fmt.Fprintf(w, "    Run exactly once:\n      %s\n", reattachCommandWithTimeout(scenario, runID, waitSeconds))
	fmt.Fprintf(w, "    Then inspect the maturity scorecard:\n      %s\n", findingsCommand(scenario, runID))
	fmt.Fprintf(w, "    Expected duration: ~%s; recommended wait timeout: %s.\n", etaStr, humanDuration(waitSeconds))
	fmt.Fprintf(w, "    In coding-agent tool execution, give the command at least this timeout and do not poll with short output checks.\n")
	fmt.Fprintf(w, "    If a wait process was already started and then interrupted:\n")
	fmt.Fprintf(w, "      pgrep -af 'test-genie runs wait --json .* %s %s'\n", scenario, runID)
	fmt.Fprintf(w, "      tail --pid=<pid> -f /dev/null\n")
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
		printAgentWaitBlock(&b, bi.GetScenario(), bi.GetRunId(), 0, false, recommendedWaitSeconds(0, false))
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
