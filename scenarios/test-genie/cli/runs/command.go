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
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

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
	case "freshness":
		return runFreshness(apiClient, args[1:], os.Stdout)
	case "wait":
		return runWait(apiClient, args[1:], os.Stdout)
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
  wait     <scenario> <runID> [--timeout N] [--json]             Block until the run is terminal
                                                                 (exit 0 passed, 1 failed/aborted,
                                                                 124 if --timeout elapses first)
  follow   <scenario> <runID>                                    Stream a run's live events to the end
  status   <scenario> <runID> [--json]                           Live snapshot (status, phase, ETA,
                                                                 recommended next-check backoff)
  abort    <scenario> <runID> [--json]                           Cancel a running run (→ aborted)
  freshness --scenario <s> [--phases a,b]                        Check whether required phases ran
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
		Scenario: scen, Status: strings.TrimSpace(*status), Limit: int32(*limit),
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
	resp, err := cl.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Scenario: scen, RunId: rest[0]}))
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
	resp, err := cl.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{Scenario: scen, RunId: rest[0], Force: *force}))
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
		Scenario: scen, RunId: rest[0], PinnedBy: strings.TrimSpace(*by), Reason: strings.TrimSpace(*reason),
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
	resp, err := cl.UnpinRun(context.Background(), connect.NewRequest(&runspb.UnpinRunRequest{Scenario: scen, RunId: rest[0], PinnedBy: strings.TrimSpace(*by)}))
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
		Scenario: scen, RunIdA: rest[0], RunIdB: rest[1], Phase: strings.TrimSpace(*phase),
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
	phasesCSV := fs.String("phases", "", "Comma-separated phases to check (default: the required set = quick preset)")
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
		Scenario: scen, Phases: phases,
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
