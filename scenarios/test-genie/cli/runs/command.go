// Package runs implements the `test-genie runs` CLI subcommand tree. It is a
// thin client over the test-genie API's RunsService Connect-RPC, exposing the
// append-only run history for humans (default) and machines (--json).
package runs

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// Exit-code contract for `runs compare` (mirrors baseline diff semantics).
const (
	exitOK            = 0 // clean / only new-failures
	exitRegression    = 1 // a regression exists
	exitNotComparable = 2 // baseline missing a surface / not comparable
)

// newClient is a package var so tests can substitute a fake RunsServiceClient.
var newClient = func(apiClient *cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) {
	baseURL := strings.TrimRight(apiClient.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("test-genie API base URL is not configured")
	}
	return runs_v1connect.NewRunsServiceClient(http.DefaultClient, baseURL), nil
}

// Run dispatches `runs` subcommands.
func Run(apiClient *cliutil.APIClient, args []string) error {
	if len(args) == 0 {
		return printUsage(os.Stdout)
	}
	switch args[0] {
	case "list":
		return runList(apiClient, args[1:], os.Stdout)
	case "show":
		return runShow(apiClient, args[1:], os.Stdout)
	case "delete":
		return runDelete(apiClient, args[1:], os.Stdout)
	case "pin":
		return runPin(apiClient, args[1:], os.Stdout)
	case "unpin":
		return runUnpin(apiClient, args[1:], os.Stdout)
	case "compare":
		return runCompare(apiClient, args[1:], os.Stdout)
	case "cost":
		return runCost(apiClient, args[1:], os.Stdout)
	case "freshness":
		return runFreshness(apiClient, args[1:], os.Stdout)
	case "wait":
		return runWait(apiClient, args[1:], os.Stdout)
	case "wait-all":
		return runWaitAll(apiClient, args[1:], os.Stdout)
	case "follow":
		return runFollow(apiClient, args[1:], os.Stdout)
	case "abort":
		return runAbort(apiClient, args[1:], os.Stdout)
	case "status":
		return runStatus(apiClient, args[1:], os.Stdout)
	case "help", "-h", "--help":
		return printUsage(os.Stdout)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'test-genie runs help' for usage", args[0])
	}
}

func printUsage(w io.Writer) error {
	fmt.Fprintln(w, `Usage: test-genie runs <command>

Commands:
  list     --scenario <s> [--status passed|failed] [--limit N]   List runs (newest-first)
  show     --scenario <s> <runID>                                Show a run's phases and pins
  delete   --scenario <s> <runID> [--force]                      Delete a run (force for pinned)
  pin      --scenario <s> <runID> --by <id> [--reason <text>]    Pin a run (protect from GC)
  unpin    --scenario <s> <runID> --by <id>                      Remove a pin
  compare  --scenario <s> <runID-a> <runID-b> [--phase <name>]   Compare two runs
  cost     [--scenario <s>] [--window 168h] [--fleet] [--json]   Measured phase cost. --fleet
                                                                 aggregates by phase across every
                                                                 scenario, with provider attribution,
                                                                 queue latency, and repeat-failure cost
  wait     <scenario> <runID> [--timeout N] [--json]             Block until the run is terminal
                                                                 (exit 0 passed, 1 failed/aborted,
                                                                 124 if --timeout elapses first).
                                                                 --json is the quiet agent path (one
                                                                 snapshot, no stream)
  wait-all --run <s>:<runID> [--run ...] [--timeout N] [--json]  Block until ALL named runs are terminal
                                                                 (one call for parallel runs). Aggregate
                                                                 exit: 0 all passed, 1 any failed, 124 any
                                                                 still in-flight at --timeout, 2 any error
  follow   <scenario> <runID> [--heartbeats]                     Stream a run's live events to the end
                                                                 (human live-watch; heartbeats are dropped
                                                                 on a non-TTY unless --heartbeats)
  status   <scenario> <runID> [--json]                           Live snapshot (status, phase, ETA,
                                                                 recommended next-check backoff)
  abort    <scenario> <runID> [--json]                           Cancel a running run (→ aborted)
  freshness --scenario <s> [--phases a,b]                        Check whether required freshness
                                                                 phases ran
                                                                 against the scenario's current tree
                                                                 (exit 1 if any phase is stale/unknown;
                                                                 digest scope is the scenario dir only —
                                                                 shared packages/* edits don't count)
  freshness --changed                                            Same check for every scenario the
                                                                 current git change-set touches
                                                                 (advisory fan-out used by vrooli
                                                                 hygiene; degrades to checked=false
                                                                 instead of erroring; exit 1 only
                                                                 when stale scenarios are found)

Add --json to any command for machine-readable output.`)
	return nil
}

func client(apiClient *cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) {
	return newClient(apiClient)
}

func costReportCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.GetCostReportResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.GetCostReportResponse, error) {
		window, err := parseCostWindow(ctx.Flag("window"), 7*24*time.Hour)
		if err != nil {
			return nil, err
		}
		compareWindow, err := parseCostWindow(ctx.Flag("compare-window"), 0)
		if err != nil {
			return nil, err
		}
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.GetCostReport(context.Background(), connect.NewRequest(&runspb.GetCostReportRequest{
			Target:               ctx.Flag("scenario"),
			WindowSeconds:        int64(window.Seconds()),
			CompareWindowSeconds: int64(compareWindow.Seconds()),
			Fleet:                ctx.Flag("fleet") == "true",
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
}

func costReportReport(_ cliapp.OperationContext, msg *runspb.GetCostReportResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetPhases()))
	var totalWall, totalRepeat int64
	for _, phase := range msg.GetPhases() {
		totalWall += phase.GetTotalWallClockMs()
		totalRepeat += phase.GetRepeatFailureWallClockMs()
		results = append(results, formatCostRow(phase))
	}
	summary := []string{fmt.Sprintf("Measured phase cost: %d phase(s), window=%ds, compare=%ds", len(results), msg.GetWindowSeconds(), msg.GetCompareWindowSeconds())}
	if totalWall > 0 && totalRepeat > 0 {
		summary = append(summary, fmt.Sprintf(
			"Repeat-failure cost: %s of %s (%.1f%%) spent re-deriving failures already produced",
			formatCostDuration(totalRepeat), formatCostDuration(totalWall),
			float64(totalRepeat)/float64(totalWall)*100))
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Phase cost",
		Results:        results,
	}
}

// formatCostRow renders one cost row. Provider attribution and queue latency
// appear only when they are known, so a line never asserts a fact the data does
// not contain.
func formatCostRow(phase *runspb.CostPhaseSummary) string {
	name := phase.GetTarget() + "/" + phase.GetPhase()
	if provider := phase.GetProviderScenario(); provider != "" {
		name += " [" + provider + "]"
	}
	line := fmt.Sprintf("%s samples=%d reliable=%d excluded=%d total=%dms median=%dms p90=%dms cpu=%dms peak_rss=%d change=%dms (%.1f%%) prediction=%d mae=%dms (%.1f%%) cache_hit=%.1f%%",
		name, phase.GetSampleCount(), phase.GetReliableSampleCount(), phase.GetExcludedSampleCount(),
		phase.GetTotalWallClockMs(), phase.GetMedianWallClockMs(), phase.GetP90WallClockMs(),
		phase.GetTotalCpuUserMs(), phase.GetMaxPeakRssBytes(), phase.GetChangeWallClockMs(), phase.GetChangePercent(),
		phase.GetPredictionSampleCount(), phase.GetPredictionMeanAbsoluteErrorMs(), phase.GetPredictionMeanAbsoluteErrorPercent(),
		phase.GetCacheHitRatePercent())
	if phase.GetQueueLatencyMedianMs() >= 0 {
		line += fmt.Sprintf(" queue_median=%dms queue_p90=%dms", phase.GetQueueLatencyMedianMs(), phase.GetQueueLatencyP90Ms())
	} else {
		line += " queue=unknown"
	}
	if phase.GetRepeatFailureSampleCount() > 0 {
		line += fmt.Sprintf(" repeat_failure=%dms over %d sample(s)", phase.GetRepeatFailureWallClockMs(), phase.GetRepeatFailureSampleCount())
	}
	return line
}

// formatCostDuration renders milliseconds in the unit a reader can act on.
func formatCostDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%.1fh", d.Hours())
	case d >= time.Minute:
		return fmt.Sprintf("%.1fm", d.Minutes())
	default:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

func runCost(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs cost", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scenario := fs.String("scenario", "", "scenario")
	windowText := fs.String("window", "168h", "look-back window (for example 168h or 7d)")
	compareText := fs.String("compare-window", "0", "preceding comparison window (for example 168h)")
	fleet := fs.Bool("fleet", false, "aggregate by phase across every scenario instead of one row per scenario/phase")
	jsonOutput := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	window, err := parseCostWindow(*windowText, 7*24*time.Hour)
	if err != nil {
		return err
	}
	compareWindow, err := parseCostWindow(*compareText, 0)
	if err != nil {
		return err
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.GetCostReport(context.Background(), connect.NewRequest(&runspb.GetCostReportRequest{
		Target: *scenario, WindowSeconds: int64(window.Seconds()), CompareWindowSeconds: int64(compareWindow.Seconds()),
		Fleet: *fleet,
	}))
	if err != nil {
		return err
	}
	if *jsonOutput {
		data, err := (protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: false}).Marshal(resp.Msg)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(data))
		return err
	}
	// The report renderer does not inspect operation inputs; a nil context is
	// sufficient for this legacy direct-dispatch path.
	report := costReportReport(nil, resp.Msg)
	for _, line := range report.Summary {
		fmt.Fprintln(w, line)
	}
	for _, line := range report.Results {
		fmt.Fprintln(w, line)
	}
	return nil
}

func parseCostWindow(raw string, defaultWindow time.Duration) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultWindow, nil
	}
	if strings.HasSuffix(strings.ToLower(raw), "d") {
		var days float64
		if _, err := fmt.Sscanf(strings.TrimSpace(raw[:len(raw)-1]), "%f", &days); err != nil || days < 0 {
			return 0, fmt.Errorf("invalid cost window %q", raw)
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("invalid cost window %q", raw)
	}
	return d, nil
}

// Register returns the manifest-backed runs command group with cli-core
// primitive evidence for single-call RPC commands. Long-lived follow/wait
// commands remain in omitted manifest coverage until streaming primitives exist.
func Register(manifest []byte, apiClient *cliutil.APIClient) (cliapp.SubcommandGroup, error) {
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "runs", map[string]cliapp.PrimitiveHandler{
		"RunsService.ListRuns":         cliapp.ProtoList(listRunsCall(apiClient), listRunsReport),
		"RunsService.GetRun":           cliapp.ProtoList(getRunCall(apiClient), getRunReport),
		"RunsService.ListRunArtifacts": cliapp.ProtoList(listRunArtifactsCall(apiClient), listRunArtifactsReport),
		"RunsService.GetRunArtifact":   cliapp.ProtoList(getRunArtifactCall(apiClient), getRunArtifactReport),
		"RunsService.DeleteRun":        cliapp.ProtoMutation(deleteRunCall(apiClient), deleteRunReport),
		"RunsService.PinRun":           cliapp.ProtoMutation(pinRunCall(apiClient), pinRunReport),
		"RunsService.UnpinRun":         cliapp.ProtoMutation(unpinRunCall(apiClient), unpinRunReport),
		"RunsService.CompareRuns":      cliapp.ProtoListOutcome(compareRunsCall(apiClient), compareRunsReport, compareExitFromResponse),
		"RunsService.GetCostReport":    cliapp.ProtoList(costReportCall(apiClient), costReportReport),
		"RunsService.CheckFreshness":   cliapp.ProtoListOutcome(checkFreshnessCall(apiClient), checkFreshnessReport, freshnessExit),
		"RunsService.GetRunFindings":   cliapp.ProtoList(getRunFindingsCall(apiClient), getRunFindingsReport),
		"RunsService.GetSelfHealth":    cliapp.ProtoOperational(selfHealthCall(apiClient), selfHealthReport),
		"RunsService.GetFleetHealth":   cliapp.ProtoOperational(fleetHealthCall(apiClient), fleetHealthReport),
		"RunsService.AbortRun":         cliapp.ProtoMutation(abortRunCall(apiClient), abortRunReport),
		"RunsService.GetRunStatus":     cliapp.ProtoOperational(runStatusCall(apiClient), runStatusReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	group.Subcommands = append(group.Subcommands,
		cliapp.Command{
			Name:        "wait",
			NeedsAPI:    true,
			Description: "Block until a server-owned run is terminal",
			Architecture: cliapp.CommandArchitecture{
				Exception:       cliapp.ExceptionDurableRun,
				ExceptionReason: "waits on an existing server-owned durable run and maps the terminal run state to the command exit code",
			},
		}.WithLegacyPrimitive(cliapp.DurableRunLegacy(func(args []string) error {
			return runWait(apiClient, args, os.Stdout)
		})),
		cliapp.Command{
			Name:        "wait-all",
			NeedsAPI:    true,
			Description: "Block until all named server-owned runs are terminal",
			Architecture: cliapp.CommandArchitecture{
				Exception:       cliapp.ExceptionDurableRun,
				ExceptionReason: "waits on multiple existing server-owned durable runs and maps their aggregate terminal state to the command exit code",
			},
		}.WithLegacyPrimitive(cliapp.DurableRunLegacy(func(args []string) error {
			return runWaitAll(apiClient, args, os.Stdout)
		})),
		cliapp.Command{
			Name:        "follow",
			NeedsAPI:    true,
			Description: "Stream live events for a server-owned run",
			Architecture: cliapp.CommandArchitecture{
				Exception:       cliapp.ExceptionStreaming,
				ExceptionReason: "owns a long-lived server-stream follow lifecycle rather than a single unary RPC call",
			},
		}.WithLegacyPrimitive(cliapp.StreamingLegacy(func(args []string) error {
			return runFollow(apiClient, args, os.Stdout)
		})),
	)
	return group, nil
}

func listRunArtifactsCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.ListRunArtifactsResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.ListRunArtifactsResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		var kinds []string
		for _, kind := range strings.Split(ctx.Flag("kinds"), ",") {
			if kind = strings.TrimSpace(kind); kind != "" {
				kinds = append(kinds, kind)
			}
		}
		resp, err := cl.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{
			Target: ctx.Flag("scenario"), RunId: ctx.Positional("run_id"), Kinds: kinds, ProducingPhase: ctx.Flag("phase"),
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
}

func listRunArtifactsReport(_ cliapp.OperationContext, msg *runspb.ListRunArtifactsResponse) cliapp.ListReport {
	summary := []string{fmt.Sprintf("Artifact catalog v%d: %d artifact(s)", msg.GetSchemaVersion(), len(msg.GetArtifacts()))}
	if msg.GetLegacyDiscovered() || len(msg.GetDegradedReasons()) > 0 {
		summary = append(summary, "Evidence degraded: "+strings.Join(msg.GetDegradedReasons(), "; "))
	}
	results := make([]string, 0, len(msg.GetArtifacts()))
	for _, artifact := range msg.GetArtifacts() {
		results = append(results, fmt.Sprintf("%s  kind=%s  phase=%s  access=%s", artifact.GetId(), artifact.GetKind(), artifact.GetProducingPhase(), artifact.GetAccessPath()))
	}
	if len(results) == 0 {
		results = append(results, "No cataloged artifacts.")
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Artifacts", Results: results}
}

func getRunArtifactCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.GetRunArtifactResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.GetRunArtifactResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{
			Target: ctx.Flag("scenario"), RunId: ctx.Positional("run_id"), ArtifactId: ctx.Positional("artifact_id"),
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
}

func getRunArtifactReport(_ cliapp.OperationContext, msg *runspb.GetRunArtifactResponse) cliapp.ListReport {
	a := msg.GetArtifact()
	summary := []string{fmt.Sprintf("Artifact: %s", a.GetId()), fmt.Sprintf("Kind: %s", a.GetKind()), fmt.Sprintf("Access: %s", a.GetAccessPath())}
	if msg.GetLegacyDiscovered() {
		summary = append(summary, "Evidence degraded: discovered from a historical run without a persisted catalog.")
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Metadata", Results: []string{fmt.Sprintf("phase=%s provenance=%s", a.GetProducingPhase(), a.GetProvenance())}}
}

func listRunsCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.ListRunsResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.ListRunsResponse, error) {
		limit, err := intFlag(ctx, "limit")
		if err != nil {
			return nil, err
		}
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.ListRuns(context.Background(), connect.NewRequest(&runspb.ListRunsRequest{
			Target: strings.TrimSpace(ctx.Flag("scenario")),
			Status: strings.TrimSpace(ctx.Flag("status")),
			Limit:  int32(limit),
		}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func listRunsReport(_ cliapp.OperationContext, msg *runspb.ListRunsResponse) cliapp.ListReport {
	if len(msg.GetRuns()) == 0 {
		return cliapp.ListReport{Summary: []string{"No runs recorded."}}
	}
	results := make([]string, 0, len(msg.GetRuns()))
	for _, r := range msg.GetRuns() {
		pinMark := ""
		if len(r.GetPins()) > 0 {
			pinMark = "  [pinned]"
		}
		results = append(results, fmt.Sprintf("%s  %-11s  %s%s", r.GetRunId(), r.GetStatus(), r.GetStartedAt(), pinMark))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d run(s) recorded.", len(results))}, ResultsHeading: "Runs", Results: results}
}

func getRunCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.GetRunResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.GetRunResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Target: ctx.Flag("scenario"), RunId: ctx.Positional("run_id")}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func getRunReport(_ cliapp.OperationContext, msg *runspb.GetRunResponse) cliapp.ListReport {
	r := msg.GetRun()
	summary := []string{
		fmt.Sprintf("Run: %s", r.GetRunId()),
		fmt.Sprintf("Status: %s", r.GetStatus()),
		fmt.Sprintf("Started: %s", r.GetStartedAt()),
	}
	if r.GetCompletedAt() != "" {
		summary = append(summary, fmt.Sprintf("Ended: %s", r.GetCompletedAt()))
	}
	if reasons := msg.GetDegradedReasons(); len(reasons) > 0 {
		summary = append(summary, "Evidence degraded: "+strings.Join(reasons, "; "))
	}
	results := make([]string, 0, len(r.GetPhases())+len(r.GetPins()))
	for _, p := range r.GetPhases() {
		results = append(results, fmt.Sprintf("phase %-14s %s", p.GetName(), p.GetStatus()))
	}
	for _, p := range r.GetPins() {
		results = append(results, fmt.Sprintf("pin %s (%s)", p.GetPinnedBy(), p.GetReason()))
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Details", Results: results}
}

func deleteRunCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.DeleteRunResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.DeleteRunResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{
			Target: ctx.Flag("scenario"),
			RunId:  ctx.Positional("run_id"),
			Force:  ctx.BoolFlag("force"),
		}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func deleteRunReport(ctx cliapp.OperationContext, _ *runspb.DeleteRunResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Deleted run %s", ctx.Positional("run_id"))}}
}

func pinRunCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.PinRunResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.PinRunResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.PinRun(context.Background(), connect.NewRequest(&runspb.PinRunRequest{
			Target:   ctx.Flag("scenario"),
			RunId:    ctx.Positional("run_id"),
			PinnedBy: strings.TrimSpace(ctx.Flag("by")),
			Reason:   strings.TrimSpace(ctx.Flag("reason")),
		}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func pinRunReport(ctx cliapp.OperationContext, _ *runspb.PinRunResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Pinned run %s by %s", ctx.Positional("run_id"), ctx.Flag("by"))}}
}

func unpinRunCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.UnpinRunResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.UnpinRunResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.UnpinRun(context.Background(), connect.NewRequest(&runspb.UnpinRunRequest{
			Target:   ctx.Flag("scenario"),
			RunId:    ctx.Positional("run_id"),
			PinnedBy: strings.TrimSpace(ctx.Flag("by")),
		}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func unpinRunReport(ctx cliapp.OperationContext, _ *runspb.UnpinRunResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Unpinned run %s (%s)", ctx.Positional("run_id"), ctx.Flag("by"))}}
}

func compareRunsCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.CompareRunsResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.CompareRunsResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{
			Target: ctx.Flag("scenario"),
			RunIdA: ctx.Positional("left_run_id"),
			RunIdB: ctx.Positional("right_run_id"),
			Phase:  strings.TrimSpace(ctx.Flag("phase")),
		}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func compareRunsReport(_ cliapp.OperationContext, msg *runspb.CompareRunsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetPhases()))
	for _, p := range msg.GetPhases() {
		results = append(results, fmt.Sprintf("%-14s %s (%s -> %s)", p.GetPhase(), p.GetVerdict(), p.GetStatusA(), p.GetStatusB()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Verdict: %s", msg.GetVerdict())}, ResultsHeading: "Phases", Results: results}
}

func compareExitFromResponse(msg *runspb.CompareRunsResponse) error {
	return compareExit(msg.GetVerdict())
}

func checkFreshnessCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.CheckFreshnessResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.CheckFreshnessResponse, error) {
		scenario := strings.TrimSpace(ctx.Flag("scenario"))
		if scenario == "" {
			scenario = strings.TrimSpace(ctx.Positional("scenario_arg"))
		}
		if scenario == "" {
			return nil, errors.New("scenario is required")
		}
		phases := splitCSV(ctx.Flag("phases"))
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.CheckFreshness(context.Background(), connect.NewRequest(&runspb.CheckFreshnessRequest{Target: scenario, Phases: phases}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func checkFreshnessReport(_ cliapp.OperationContext, msg *runspb.CheckFreshnessResponse) cliapp.ListReport {
	stale := countNotFresh(msg)
	results := make([]string, 0, len(msg.GetPhases()))
	for _, p := range msg.GetPhases() {
		detail := ""
		if p.GetStatus() != "fresh" && p.GetLastRunCompletedAt() != "" {
			detail = fmt.Sprintf(" (last passed %s, before the latest changes)", p.GetLastRunCompletedAt())
		}
		results = append(results, fmt.Sprintf("%-14s %s%s", p.GetPhase(), p.GetStatus(), detail))
	}
	summary := []string{fmt.Sprintf("%d phase(s) checked; %d stale or unknown.", len(msg.GetPhases()), stale)}
	hints := []string{}
	if stale > 0 && msg.GetSuggestedCommand() != "" {
		hints = append(hints, msg.GetSuggestedCommand())
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Phases", Results: results, RetrievalHints: hints}
}

func selfHealthCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.GetSelfHealthResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.GetSelfHealthResponse, error) {
		window, err := intFlag(ctx, "window-days")
		if err != nil {
			return nil, err
		}
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.GetSelfHealth(context.Background(), connect.NewRequest(&runspb.GetSelfHealthRequest{WindowDays: int32(window)}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func selfHealthReport(_ cliapp.OperationContext, msg *runspb.GetSelfHealthResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{Status: []string{fmt.Sprintf("self health present: %t", msg.GetSelfHealth() != nil)}}
}

func fleetHealthCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.GetFleetHealthResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.GetFleetHealthResponse, error) {
		window, err := intFlag(ctx, "window-days")
		if err != nil {
			return nil, err
		}
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.GetFleetHealth(context.Background(), connect.NewRequest(&runspb.GetFleetHealthRequest{
			WindowDays:    int32(window),
			IncludeRoster: ctx.BoolFlag("roster"),
		}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func fleetHealthReport(_ cliapp.OperationContext, msg *runspb.GetFleetHealthResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{Status: []string{fmt.Sprintf("fleet health present: %t", msg.GetFleetHealth() != nil)}}
}

func abortRunCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.AbortRunResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.AbortRunResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.AbortRun(context.Background(), connect.NewRequest(&runspb.AbortRunRequest{Target: ctx.Positional("scenario"), RunId: ctx.Positional("run_id")}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func abortRunReport(ctx cliapp.OperationContext, msg *runspb.AbortRunResponse) cliapp.MutationReport {
	status := "aborted"
	if msg.GetStatus() != nil && msg.GetStatus().GetStatus() != "" {
		status = msg.GetStatus().GetStatus()
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Run %s is %s", ctx.Positional("run_id"), status)}}
}

func runStatusCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.RunLiveStatus, error) {
	return func(ctx cliapp.OperationContext) (*runspb.RunLiveStatus, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		resp, err := cl.GetRunStatus(context.Background(), connect.NewRequest(&runspb.GetRunStatusRequest{Target: ctx.Positional("scenario"), RunId: ctx.Positional("run_id")}))
		if err != nil {
			return nil, &exitErr{code: exitNotComparable, err: err}
		}
		return resp.Msg, nil
	}
}

func runStatusReport(_ cliapp.OperationContext, msg *runspb.RunLiveStatus) cliapp.OperationalReport {
	status := "(unknown)"
	if msg.GetStatus() != "" {
		status = msg.GetStatus()
	}
	return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Run status: %s", status)}}
}

func intFlag(ctx cliapp.OperationContext, name string) (int, error) {
	value := strings.TrimSpace(ctx.Flag(name))
	if value == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer: %w", name, err)
	}
	return n, nil
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func requireScenario(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("--scenario is required")
	}
	return s, nil
}

func runList(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs list", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	status := fs.String("status", "", "Filter by status (passed|failed|...)")
	limit := fs.Int("limit", 0, "Max runs to return (0 = all)")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.ListRuns(context.Background(), connect.NewRequest(&runspb.ListRunsRequest{
		Target: scen, Status: strings.TrimSpace(*status), Limit: int32(*limit),
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg)
	}
	runs := resp.Msg.GetRuns()
	if len(runs) == 0 {
		fmt.Fprintln(w, "No runs recorded.")
		return nil
	}
	for _, r := range runs {
		pinMark := ""
		if len(r.GetPins()) > 0 {
			pinMark = "  [pinned]"
		}
		fmt.Fprintf(w, "%s  %-11s  %s%s\n", r.GetRunId(), r.GetStatus(), r.GetStartedAt(), pinMark)
	}
	return nil
}

func runShow(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs show", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("exactly one runID is required")
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Target: scen, RunId: rest[0]}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg)
	}
	r := resp.Msg.GetRun()
	fmt.Fprintf(w, "Run:     %s\n", r.GetRunId())
	fmt.Fprintf(w, "Status:  %s\n", r.GetStatus())
	fmt.Fprintf(w, "Started: %s\n", r.GetStartedAt())
	if r.GetCompletedAt() != "" {
		fmt.Fprintf(w, "Ended:   %s\n", r.GetCompletedAt())
	}
	if len(r.GetPhases()) > 0 {
		fmt.Fprintln(w, "Phases:")
		for _, p := range r.GetPhases() {
			fmt.Fprintf(w, "  %-14s %s\n", p.GetName(), p.GetStatus())
		}
	}
	if len(r.GetPins()) > 0 {
		fmt.Fprintln(w, "Pins:")
		for _, p := range r.GetPins() {
			fmt.Fprintf(w, "  %s (%s)\n", p.GetPinnedBy(), p.GetReason())
		}
	}
	return nil
}

func runDelete(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs delete", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	force := fs.Bool("force", false, "Delete even if pinned")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("exactly one runID is required")
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{Target: scen, RunId: rest[0], Force: *force}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg)
	}
	fmt.Fprintf(w, "Deleted run %s\n", rest[0])
	return nil
}

func runPin(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs pin", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	by := fs.String("by", "", "Pin owner identifier (e.g. gct:baseline:plan-7c3)")
	reason := fs.String("reason", "", "Reason for the pin")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("exactly one runID is required")
	}
	if strings.TrimSpace(*by) == "" {
		return errors.New("--by is required")
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.PinRun(context.Background(), connect.NewRequest(&runspb.PinRunRequest{
		Target: scen, RunId: rest[0], PinnedBy: strings.TrimSpace(*by), Reason: strings.TrimSpace(*reason),
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg)
	}
	fmt.Fprintf(w, "Pinned run %s by %s\n", rest[0], *by)
	return nil
}

func runUnpin(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs unpin", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	by := fs.String("by", "", "Pin owner identifier")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("exactly one runID is required")
	}
	if strings.TrimSpace(*by) == "" {
		return errors.New("--by is required")
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.UnpinRun(context.Background(), connect.NewRequest(&runspb.UnpinRunRequest{Target: scen, RunId: rest[0], PinnedBy: strings.TrimSpace(*by)}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		return writeJSON(w, resp.Msg)
	}
	fmt.Fprintf(w, "Unpinned run %s (%s)\n", rest[0], *by)
	return nil
}

func runCompare(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs compare", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	phase := fs.String("phase", "", "Restrict to one phase")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 2 {
		return errors.New("exactly two runIDs are required: <runID-a> <runID-b>")
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{
		Target: scen, RunIdA: rest[0], RunIdB: rest[1], Phase: strings.TrimSpace(*phase),
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		if err := writeJSON(w, resp.Msg); err != nil {
			return err
		}
	} else {
		writeCompareHuman(w, resp.Msg)
	}
	return compareExit(resp.Msg.GetVerdict())
}

func writeCompareHuman(w io.Writer, msg *runspb.CompareRunsResponse) {
	for _, p := range msg.GetPhases() {
		mark := "✓"
		if p.GetVerdict() != "clean" {
			mark = "✗"
		}
		fmt.Fprintf(w, "%s %-14s %s (%s → %s)\n", mark, p.GetPhase(), p.GetVerdict(), p.GetStatusA(), p.GetStatusB())
	}
	fmt.Fprintf(w, "\nVerdict: %s\n", msg.GetVerdict())
}

func compareExit(verdict string) error {
	switch verdict {
	case "regression":
		return &exitErr{code: exitRegression, err: errors.New("regression detected")}
	case "not-comparable":
		return &exitErr{code: exitNotComparable, err: errors.New("runs not comparable")}
	default:
		return nil
	}
}

// runFreshness reports whether the required phases (default: the quick
// preset — a global code-level SSOT, not per-scenario configurable) have run
// against the scenario's CURRENT working tree. Exit 0 when everything is
// fresh; exit 1 when any phase is stale or unknown so scripts can gate on it.
// Documented v1 limitation: the digest scopes to the scenario directory only;
// edits to shared packages/* do not invalidate freshness.
func runFreshness(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs freshness", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	phasesCSV := fs.String("phases", "", "Comma-separated phases to check (default: required freshness profile)")
	changed := fs.Bool("changed", false, "Check every scenario touched by the current git change-set (advisory: degrades to checked=false, never errors)")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *changed {
		if strings.TrimSpace(*scenario) != "" || len(fs.Args()) > 0 {
			return errors.New("--changed checks the whole change-set; it cannot be combined with a scenario")
		}
		if strings.TrimSpace(*phasesCSV) != "" {
			return errors.New("--changed always checks the required phase set; it cannot be combined with --phases")
		}
		return runFreshnessChanged(apiClient, *jsonOut, w)
	}
	scen := strings.TrimSpace(*scenario)
	if scen == "" && len(fs.Args()) == 1 {
		scen = strings.TrimSpace(fs.Args()[0])
	}
	scen, err := requireScenario(scen)
	if err != nil {
		return err
	}
	var phases []string
	for _, p := range strings.Split(*phasesCSV, ",") {
		if p = strings.TrimSpace(p); p != "" {
			phases = append(phases, p)
		}
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	resp, err := cl.CheckFreshness(context.Background(), connect.NewRequest(&runspb.CheckFreshnessRequest{
		Target: scen, Phases: phases,
	}))
	if err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		if err := writeJSON(w, resp.Msg); err != nil {
			return err
		}
		return freshnessExit(resp.Msg)
	}

	stale := 0
	for _, p := range resp.Msg.GetPhases() {
		mark := "✓"
		detail := ""
		if p.GetStatus() != "fresh" {
			mark = "✗"
			stale++
			if p.GetLastRunCompletedAt() != "" {
				detail = fmt.Sprintf("  (last passed %s, before the latest changes)", p.GetLastRunCompletedAt())
			}
		}
		fmt.Fprintf(w, "%s %-14s %s%s\n", mark, p.GetPhase(), p.GetStatus(), detail)
	}
	if stale == 0 {
		fmt.Fprintf(w, "\nAll %d phase(s) fresh against the current tree.\n", len(resp.Msg.GetPhases()))
		return nil
	}
	fmt.Fprintf(w, "\n%d phase(s) have not run against the current tree.\nRun: %s\n", stale, resp.Msg.GetSuggestedCommand())
	return freshnessExit(resp.Msg)
}

func freshnessExit(msg *runspb.CheckFreshnessResponse) error {
	for _, p := range msg.GetPhases() {
		if p.GetStatus() != "fresh" {
			return &exitErr{code: exitRegression, err: fmt.Errorf("%d phase(s) stale or unknown", countNotFresh(msg))}
		}
	}
	return nil
}

func countNotFresh(msg *runspb.CheckFreshnessResponse) int {
	n := 0
	for _, p := range msg.GetPhases() {
		if p.GetStatus() != "fresh" {
			n++
		}
	}
	return n
}

func writeJSON(w io.Writer, msg proto.Message) error {
	encoded, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(encoded))
	return nil
}

// exitErr wraps an error with a documented exit code (cli-core inspects ExitCode()).
type exitErr struct {
	code int
	err  error
}

func (e *exitErr) Error() string { return e.err.Error() }
func (e *exitErr) Unwrap() error { return e.err }
func (e *exitErr) ExitCode() int { return e.code }
