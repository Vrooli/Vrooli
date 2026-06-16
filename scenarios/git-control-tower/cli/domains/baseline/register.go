// Package baseline implements the `git-control-tower baseline` command group:
// the agent-facing primitive that replaces `git stash` for regression
// diagnosis. Capture a baseline before implementing, diff it afterwards to ask
// "did my change cause this, or was it preexisting?" without touching the tree.
//
// Hand-written registrar (like repo/branch/audit) rather than manifest-driven,
// so it does not require BaselinesService proto-descriptor cross-validation in
// the worktree manifest test.
package baseline

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"git-control-tower/cli/internal/callerheader"

	"github.com/vrooli/cli-core/cliapp"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

// Exit codes (Decision 5 — agents/CI branch on exit code without parsing).
const (
	exitOK            = 0
	exitRegression    = 1
	exitNotComparable = 2
	// exitNotReady is `diff status` when the run is still in flight — distinct
	// from a verdict so scripts don't misread "not done yet" as "clean".
	exitNotReady = 3
)

// baselineClientTimeout bounds snapshot/diff, which block while one
// comprehensive, server-durable test-genie run executes (the server's own runs
// client allows 15m). It is a CEILING, not the expected wait: the snapshot
// fast-skips an unreachable backend in seconds, and an interrupt detaches
// without aborting the durable run. The scenario-default client timeout would
// abort mid-run, so the baseline client gets its own long, bounded deadline.
const baselineClientTimeout = 30 * time.Minute

// snapshotStartCeiling bounds the snapshot START call. The RPC returns as soon as
// the durable run has started (git read + reachability probe + StartRun), so this
// is small — the heavy run continues server-side and is followed by run id.
const snapshotStartCeiling = 2 * time.Minute

// clientFactory builds the BaselinesService Connect client. Overridable in tests.
var clientFactory = func(core *cliapp.ScenarioApp) baselines_v1connect.BaselinesServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, baselineClientTimeout)
	return baselines_v1connect.NewBaselinesServiceClient(httpClient, baseURL,
		connect.WithInterceptors(callerheader.New()))
}

// Register wires the baseline command group: the passive *record* verbs
// (snapshot/diff/list/show/delete/create/edit) plus the stateful *engagement*
// verbs (start/check/promote/abandon/status/gc — Baseline Modes P2, in
// engagement.go + promote.go).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	subcommands := []cliapp.Command{
		{Name: "snapshot", NeedsAPI: true, Description: "Capture a baseline from one durable test-genie run (--scenario --name [--branch] [--include w,t,...] [--fast|--full] [--reason]); Ctrl-C detaches (re-attach via test-genie runs wait --json), never aborts", Run: func(a []string) error { return runSnapshot(core, a) }},
		{Name: "diff", NeedsAPI: true, Description: "Start a durable diff of a baseline vs the working tree, returning a run id + re-attach command (--scenario --name [--branch] [--surface] [--wait]); `diff status --run R` resolves the verdict (exit 1 regression, 2 not-comparable, 3 not-ready); `diff wait-all --run s:name:R …` resolves several started diffs in one call; reuses a clean-tree run when possible; Ctrl-C detaches, never aborts", Run: func(a []string) error { return runDiff(core, a) }},
		{Name: "list", NeedsAPI: true, Description: "List baselines (--scenario [--branch] [--all-branches])", Run: func(a []string) error { return runList(core, a) }},
		{Name: "show", NeedsAPI: true, Description: "Show one baseline (--scenario --name [--branch])", Run: func(a []string) error { return runShow(core, a) }},
		{Name: "delete", NeedsAPI: true, Description: "Delete a baseline and unpin its test-genie runs (--scenario --name [--branch])", Run: func(a []string) error { return runDelete(core, a) }},
		{Name: "create", NeedsAPI: true, Description: "Create an empty baseline (no capture) (--scenario --name [--branch])", Run: func(a []string) error { return runCreate(core, a) }},
		{Name: "edit", NeedsAPI: true, Description: "Re-point a surface at a pinned test-genie run (--scenario --name --surface --pin-run <runID> [--branch])", Run: func(a []string) error { return runEdit(core, a) }},
	}
	subcommands = append(subcommands, registerEngagementVerbs(core)...)
	return cliapp.SubcommandGroup{
		Name:        "baseline",
		Description: "Capture/diff review baselines and run shadow/live engagements (Baseline Modes)",
		NeedsAPI:    true,
		Subcommands: subcommands,
	}
}

// commonFlags are shared across most subcommands.
type commonFlags struct {
	scenario string
	name     string
	branch   string
	json     bool
}

func (c *commonFlags) bind(fs *flag.FlagSet) {
	fs.StringVar(&c.scenario, "scenario", "", "Scenario slug (required)")
	fs.StringVar(&c.name, "name", "", "Baseline name")
	fs.StringVar(&c.branch, "branch", "", "Git branch (default: current)")
	fs.BoolVar(&c.json, "json", false, "Emit JSON")
}

func (c commonFlags) requireScenarioName() error {
	if strings.TrimSpace(c.scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	if strings.TrimSpace(c.name) == "" {
		return fmt.Errorf("--name is required")
	}
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func runSnapshot(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var include, reason string
	var fast, full bool
	fs := newFlagSet("baseline snapshot")
	c.bind(fs)
	fs.StringVar(&include, "include", "", "Comma-separated surfaces to capture (default: all available)")
	fs.StringVar(&reason, "reason", "", "Optional reason recorded on pinned runs")
	fs.BoolVar(&fast, "fast", true, "Fast capture: skip heavy diagnostics (default)")
	fs.BoolVar(&full, "full", false, "Full capture: video/HAR/trace/console (slower)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}
	// --full overrides the default --fast.
	if full {
		fast = false
	}

	// A snapshot now STARTS one comprehensive, server-durable run and returns its
	// handle immediately (the pin happens server-side on completion). So the CLI
	// returns fast with the run id + ETA + a streaming follow command, instead of
	// blocking silently for the whole run.
	if !c.json {
		fmt.Printf("Capturing baseline %q for %s — starting one comprehensive test-genie run...\n", c.name, c.scenario)
	}

	ctx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()

	client := clientFactory(core)
	resp, err := client.SnapshotForBaseline(ctx, connect.NewRequest(&baselinesv1.SnapshotForBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch,
		Include: splitCSV(include), Fast: fast, Reason: reason, CreatedBy: "agent",
	}))
	if err != nil {
		if handled, code := renderRunBusy(err, c.scenario); handled {
			os.Exit(code)
		}
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	printSnapshot(resp.Msg)
	return nil
}

// runDiff STARTS a durable diff and returns immediately with the run handle +
// re-attach banner (mirror snapshot) — it never silently blocks (the
// anti-polling contract). `baseline diff status --run R` (dispatched here)
// resolves the cached verdict. With --wait it blocks server-side and prints the
// verdict inline; a clean-tree reuse resolves instantly with no second suite.
func runDiff(core *cliapp.ScenarioApp, args []string) error {
	// Sub-dispatch: `baseline diff status …` resolves a started diff's verdict;
	// `baseline diff wait-all …` resolves several started diffs in one call.
	if len(args) > 0 && args[0] == "status" {
		return runDiffStatus(core, args[1:])
	}
	if len(args) > 0 && args[0] == "wait-all" {
		return runDiffWaitAll(core, args[1:])
	}

	var c commonFlags
	var surface string
	var wait bool
	fs := newFlagSet("baseline diff")
	c.bind(fs)
	fs.StringVar(&surface, "surface", "", "Restrict to one surface")
	fs.BoolVar(&wait, "wait", false, "Block until the verdict is ready and print it inline (CI); default returns a run id to resolve with `diff status`")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}

	startCtx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	client := clientFactory(core)
	start, err := client.StartDiff(startCtx, connect.NewRequest(&baselinesv1.StartDiffRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch, Surface: surface,
	}))
	if err != nil {
		if handled, code := renderRunBusy(err, c.scenario); handled {
			os.Exit(code)
		}
		return err
	}

	// --wait, or a reused run (already terminal): resolve the verdict inline.
	// Otherwise return fast with the re-attach banner (no client polling).
	if wait || start.Msg.GetReusedRun() {
		return resolveDiff(core, c, surface, start.Msg.GetRunId(), wait)
	}
	if c.json {
		return printJSON(start.Msg)
	}
	printDiffStart(start.Msg)
	return nil
}

// runDiffStatus resolves a started diff's cached verdict. It exits 0/1/2 by
// verdict, or 3 when the run is still in flight (with follow/resolve guidance).
func runDiffStatus(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var surface, run string
	var wait bool
	fs := newFlagSet("baseline diff status")
	c.bind(fs)
	fs.StringVar(&surface, "surface", "", "Restrict to one surface")
	fs.StringVar(&run, "run", "", "The diff's run id (from `baseline diff`) (required)")
	fs.BoolVar(&wait, "wait", false, "Block server-side until the verdict is ready")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}
	if strings.TrimSpace(run) == "" {
		return fmt.Errorf("--run is required (the run id printed by `baseline diff`)")
	}
	return resolveDiff(core, c, surface, run, wait)
}

// resolveDiff fetches a diff verdict via GetDiffResult and renders it, exiting by
// verdict (0/1/2) or 3 when still in flight. wait blocks server-side.
func resolveDiff(core *cliapp.ScenarioApp, c commonFlags, surface, run string, wait bool) error {
	timeout := snapshotStartCeiling
	if wait {
		timeout = baselineClientTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := clientFactory(core)
	resp, err := client.GetDiffResult(ctx, connect.NewRequest(&baselinesv1.GetDiffResultRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch, RunId: run, Surface: surface, Wait: wait,
	}))
	if err != nil {
		return err
	}
	msg := resp.Msg
	if msg.GetStatus() == "in_progress" {
		if c.json {
			if err := printJSON(msg); err != nil {
				return err
			}
		} else {
			printDiffPending(c.scenario, c.name, run, int(msg.GetRecommendedNextCheckSeconds()))
		}
		os.Exit(exitNotReady)
	}
	if c.json {
		if err := printJSON(msg); err != nil {
			return err
		}
	} else if d := msg.GetDiff(); d != nil {
		printDiff(d)
	}
	os.Exit(exitCodeForVerdict(msg.GetDiff().GetVerdict()))
	return nil
}

// renderRunBusy renders the one-run-per-scenario rejection (a divergent run is
// in flight) with wait/abort guidance and returns the exit code, or false when
// err is not a busy rejection.
func renderRunBusy(err error, scenario string) (bool, int) {
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeFailedPrecondition {
		return false, 0
	}
	for _, d := range ce.Details() {
		msg, derr := d.Value()
		if derr != nil {
			continue
		}
		bi, ok := msg.(*baselinesv1.RunBusyInfo)
		if !ok {
			continue
		}
		fmt.Fprintf(os.Stderr, "✗ %s already has an in-progress run %s (preset %s) — only one run per scenario at a time.\n", bi.GetScenario(), bi.GetRunId(), busyPreset(bi.GetPreset()))
		fmt.Fprintf(os.Stderr, "  wait:  test-genie runs wait --json %s %s\n", bi.GetScenario(), bi.GetRunId())
		fmt.Fprintf(os.Stderr, "  abort: test-genie runs abort %s %s\n", bi.GetScenario(), bi.GetRunId())
		return true, exitNotComparable
	}
	return false, 0
}

func busyPreset(p string) string {
	if strings.TrimSpace(p) == "" {
		return "default"
	}
	return p
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var allBranches bool
	fs := newFlagSet("baseline list")
	c.bind(fs)
	fs.BoolVar(&allBranches, "all-branches", false, "List baselines across all branches")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(c.scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}

	client := clientFactory(core)
	resp, err := client.ListBaselines(context.Background(), connect.NewRequest(&baselinesv1.ListBaselinesRequest{
		Scenario: c.scenario, Branch: c.branch, AllBranches: allBranches,
	}))
	if err != nil {
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	printList(resp.Msg.GetBaselines())
	return nil
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	fs := newFlagSet("baseline show")
	c.bind(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}

	client := clientFactory(core)
	resp, err := client.GetBaseline(context.Background(), connect.NewRequest(&baselinesv1.GetBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch,
	}))
	if err != nil {
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	printShow(resp.Msg.GetBaseline())
	return nil
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var force bool
	fs := newFlagSet("baseline delete")
	c.bind(fs)
	fs.BoolVar(&force, "force", false, "Force delete")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}

	client := clientFactory(core)
	if _, err := client.DeleteBaseline(context.Background(), connect.NewRequest(&baselinesv1.DeleteBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch,
	})); err != nil {
		return err
	}
	if c.json {
		return printJSON(map[string]any{"deleted": true, "name": c.name})
	}
	fmt.Printf("✓ Deleted baseline %q (test-genie runs unpinned)\n", c.name)
	return nil
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	fs := newFlagSet("baseline create")
	c.bind(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}

	client := clientFactory(core)
	resp, err := client.CreateBaseline(context.Background(), connect.NewRequest(&baselinesv1.CreateBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch, CreatedBy: "agent",
	}))
	if err != nil {
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	fmt.Printf("✓ Created empty baseline %q for %s (branch: %s)\n", resp.Msg.GetBaseline().GetName(), c.scenario, resp.Msg.GetBaseline().GetBranch())
	return nil
}

func runEdit(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var surface, pinRun, reason string
	fs := newFlagSet("baseline edit")
	c.bind(fs)
	fs.StringVar(&surface, "surface", "", "Surface ID to re-point (required)")
	fs.StringVar(&pinRun, "pin-run", "", "test-genie runID to pin (required)")
	fs.StringVar(&reason, "reason", "", "Optional pin reason")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}
	if strings.TrimSpace(surface) == "" || strings.TrimSpace(pinRun) == "" {
		return fmt.Errorf("--surface and --pin-run are required")
	}

	client := clientFactory(core)
	resp, err := client.EditBaseline(context.Background(), connect.NewRequest(&baselinesv1.EditBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch, Surface: surface, PinRunId: pinRun, Reason: reason,
	}))
	if err != nil {
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	fmt.Printf("✓ Re-pointed surface %q of baseline %q to run %s\n", surface, c.name, pinRun)
	return nil
}

// exitCodeForVerdict maps the overall diff verdict to a process exit code.
// regression → 1; not-comparable → 2; changed/new-failure/preexisting/clean → 0
// (new/preexisting failures are not caused by the current change, and `changed`
// is the neutral advisory visual tier — none of them block — Plan A §3.2 /
// Decision 5).
func exitCodeForVerdict(verdict string) int {
	switch verdict {
	case "regression":
		return exitRegression
	case "not-comparable":
		return exitNotComparable
	default:
		return exitOK
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printJSON(v any) error {
	if msg, ok := v.(proto.Message); ok {
		b, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// surfaceIDsSorted returns surface IDs in a stable display order.
func surfaceIDsSorted(surfaces map[string]*baselinesv1.SurfacePointer) []string {
	ids := make([]string, 0, len(surfaces))
	for id := range surfaces {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
