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
	"os/signal"
	"sort"
	"strings"
	"syscall"
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
)

// baselineClientTimeout bounds snapshot/diff, which block while one
// comprehensive, server-durable test-genie run executes (the server's own runs
// client allows 15m). It is a CEILING, not the expected wait: the snapshot
// fast-skips an unreachable backend in seconds, and an interrupt detaches
// without aborting the durable run. The scenario-default client timeout would
// abort mid-run, so the baseline client gets its own long, bounded deadline.
const baselineClientTimeout = 30 * time.Minute

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
		{Name: "snapshot", NeedsAPI: true, Description: "Capture a baseline from one durable test-genie run (--scenario --name [--branch] [--include w,t,...] [--fast|--full] [--reason]); Ctrl-C detaches (re-attach via test-genie runs wait), never aborts", Run: func(a []string) error { return runSnapshot(core, a) }},
		{Name: "diff", NeedsAPI: true, Description: "Diff a baseline against the working tree (--scenario --name [--branch] [--surface]); exit 1 on regression, 2 on not-comparable", Run: func(a []string) error { return runDiff(core, a) }},
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

	// Up-front signal (signal-and-feedback-surface-design): a snapshot triggers
	// ONE comprehensive, server-durable test-genie run. The heavy work survives
	// this CLI disconnecting; we just block to relay the verdict. Tell the
	// operator that before the long wait so it never reads as a silent hang.
	if !c.json {
		fmt.Printf("Capturing baseline %q for %s — triggering one comprehensive test-genie run (durable server-side).\n", c.name, c.scenario)
		fmt.Printf("  The run continues even if this command is interrupted; bounded client deadline %s.\n", humanDuration(baselineClientTimeout))
		fmt.Printf("  If the wait drops, find the run with: test-genie runs list --scenario %s   then re-attach: test-genie runs wait %s <run-id>\n", c.scenario, c.scenario)
	}

	// Bounded deadline (not bare context.Background()) so the CLI can never hang
	// forever; SIGINT detaches WITHOUT aborting the server-side run.
	ctx, cancel := context.WithTimeout(context.Background(), baselineClientTimeout)
	defer cancel()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := clientFactory(core)
	resp, err := client.SnapshotForBaseline(ctx, connect.NewRequest(&baselinesv1.SnapshotForBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch,
		Include: splitCSV(include), Fast: fast, Reason: reason, CreatedBy: "agent",
	}))
	if err != nil {
		// Cancel ≠ abort: an interrupt or deadline detaches from the durable
		// run, it does not stop it. Guide the operator back to it and exit 0.
		if isDetach(ctx, err) {
			fmt.Fprintf(os.Stderr, "\nDetached from the snapshot — the comprehensive run continues server-side (not aborted).\n")
			fmt.Fprintf(os.Stderr, "Re-attach: test-genie runs list --scenario %s   then: test-genie runs wait %s <run-id>\n", c.scenario, c.scenario)
			return nil
		}
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	printSnapshot(resp.Msg)
	return nil
}

// humanDuration renders a duration compactly (e.g. "30m0s") for the snapshot
// banner. Trivial wrapper kept local so the banner reads in one place.
func humanDuration(d time.Duration) string { return d.String() }

// isDetach reports whether a SnapshotForBaseline error is a client-side
// detach (interrupt or bounded-deadline elapsed) rather than a real failure.
// A detach is NOT an abort: the durable run continues server-side, so the CLI
// exits 0 with re-attach guidance instead of surfacing an error.
func isDetach(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func runDiff(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var surface string
	fs := newFlagSet("baseline diff")
	c.bind(fs)
	fs.StringVar(&surface, "surface", "", "Restrict to one surface")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}

	client := clientFactory(core)
	resp, err := client.DiffBaseline(context.Background(), connect.NewRequest(&baselinesv1.DiffBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch, Surface: surface,
	}))
	if err != nil {
		return err
	}
	if c.json {
		if err := printJSON(resp.Msg); err != nil {
			return err
		}
	} else {
		printDiff(resp.Msg)
	}
	os.Exit(exitCodeForVerdict(resp.Msg.GetVerdict()))
	return nil
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
// regression → 1; not-comparable → 2; new-failure/preexisting/clean → 0
// (new failures and preexisting failures are not caused by the current change,
// so they do not block — Plan A §3.2 / Decision 5).
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
