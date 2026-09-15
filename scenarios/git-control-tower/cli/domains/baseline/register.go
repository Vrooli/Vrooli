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
	"io"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"git-control-tower/cli/internal/callerheader"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

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
// (snapshot/diff/list/show/delete) plus the stateful *engagement*
// verbs (start/check/promote/abandon/status/gc — Baseline Modes P2, in
// engagement.go + promote.go).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	subcommands := []cliapp.Command{
		{Name: "snapshot", NeedsAPI: true, Description: "Capture a baseline from one durable test-genie run, or inspect it with `snapshot status` (--scenario --name [--run] [--branch]); Ctrl-C detaches, never aborts", Run: func(a []string) error { return runSnapshot(core, a) }},
		{Name: "diff", NeedsAPI: true, Description: "Start a durable descriptor-driven diff of a baseline vs the working tree, returning a run id + re-attach command (--scenario --name [--branch] [--wait]); `diff status --run R` resolves the verdict, `diff status --latest` recovers an interrupted wait (exit 1 regression, 2 not-comparable, 3 not-ready); `diff wait-all --run s:name:R …` resolves several started diffs in one call; Ctrl-C detaches, never aborts", Run: func(a []string) error { return runDiff(core, a) }},
		{Name: "list", NeedsAPI: true, Description: "List baselines (--scenario [--branch] [--all-branches])", Run: func(a []string) error { return runList(core, a) }},
		{Name: "show", NeedsAPI: true, Description: "Show one baseline (--scenario --name [--branch])", Run: func(a []string) error { return runShow(core, a) }},
		{Name: "delete", NeedsAPI: true, Description: "Delete a baseline and unpin its single Test Genie run (--scenario --name [--branch])", Run: func(a []string) error { return runDelete(core, a) }},
		{Name: "repair", NeedsAPI: true, Description: "Plan deterministic lifecycle repair for a baseline (--scenario --name [--branch]); pass --apply to write the lifecycle audit and converge a tombstoned manifest", Run: func(a []string) error { return runRepair(core, a) }},
		{Name: "collection", NeedsAPI: true, Description: "Start, inspect, wait, extend, and diff durable multi-scenario baseline collections: `collection capture --name N --member scenario ...` prints its native wait; `collection show --name N --wait` reattaches; `collection extend` is append-only before edits; `collection diff --name N --operation-id stable-id [--member S ...]` starts an idempotent comparison, then `diff status` inspects and `diff wait` owns the final verdict", Run: func(a []string) error { return runCollection(core, a) }},
		{Name: "path", NeedsAPI: true, Description: "Capture and compare bounded informational source evidence: `path capture --name N --path glob ...`, `path diff --before N --after N`, `path show`, or `path delete`", Run: func(a []string) error { return runPathSnapshot(core, a) }},
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
	if len(args) > 0 && args[0] == "status" {
		return runSnapshotStatus(core, args[1:])
	}

	var c commonFlags
	var reason string
	fs := newFlagSet("baseline snapshot")
	c.bind(fs)
	fs.StringVar(&reason, "reason", "", "Optional reason recorded on pinned runs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
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
		Reason: reason, CreatedBy: "agent",
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

func runSnapshotStatus(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var run string
	var wait bool
	fs := newFlagSet("baseline snapshot status")
	c.bind(fs)
	fs.StringVar(&run, "run", "", "Snapshot run id (optional; narrows to one returned run)")
	fs.BoolVar(&wait, "wait", false, "Block server-side until the snapshot run is terminal before reconciling")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}

	timeout := snapshotStartCeiling
	if wait {
		timeout = baselineClientTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := clientFactory(core)
	resp, recovered, err := durableReadWithEOFRecovery(ctx, wait, func(callCtx context.Context, blocking bool) (*connect.Response[baselinesv1.GetSnapshotStatusResponse], error) {
		return client.GetSnapshotStatus(callCtx, connect.NewRequest(&baselinesv1.GetSnapshotStatusRequest{
			Scenario: c.scenario, Name: c.name, Branch: c.branch, RunId: run, Wait: blocking,
		}))
	})
	if err != nil {
		return err
	}
	if recovered {
		fmt.Fprintf(os.Stderr, "snapshot wait attachment ended unexpectedly; recovered current state once by durable run id %s\n", run)
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	printSnapshotStatus(resp.Msg)
	switch resp.Msg.GetStatus() {
	case "ready":
		return nil
	case "pending":
		os.Exit(exitNotReady)
	default:
		os.Exit(exitNotComparable)
	}
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
	var wait bool
	fs := newFlagSet("baseline diff")
	c.bind(fs)
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
		Scenario: c.scenario, Name: c.name, Branch: c.branch,
	}))
	if err != nil {
		if handled, code := renderRunBusy(err, c.scenario); handled {
			os.Exit(code)
		}
		if handled := renderBaselineNotFound(core, c, "diff", err); handled {
			os.Exit(exitNotComparable)
		}
		return err
	}

	// A reused run is already terminal — resolve its verdict inline regardless of
	// caller (nothing to wait on, so nothing to park).
	if start.Msg.GetReusedRun() {
		return resolveDiff(core, c, start.Msg.GetRunId(), wait)
	}

	// Inside an agent-manager run, park instead of blocking on the verdict:
	// agent-manager performs the wait on the agent's behalf (re-running the
	// blocking diff from its own non-agent context) and wakes the run with the
	// result. The await-handle key is "<scenario>/<name>" — the gct Waiter
	// resolves it via `baseline diff --scenario --name --wait`. Outside an AM run
	// this is a no-op and we fall through to today's behaviour.
	if park, parked, perr := parkForDiff(c.scenario, c.name); parked {
		if perr == nil {
			fmt.Println(park.Message)
			return nil
		}
		fmt.Fprintf(os.Stderr, "agent-manager park unavailable (%v) — resolving inline instead\n", perr)
		return resolveDiff(core, c, start.Msg.GetRunId(), true)
	}

	// --wait: resolve the verdict inline. Otherwise return fast with the
	// re-attach banner (no client polling).
	if wait {
		if c.json {
			fmt.Fprintf(os.Stderr, "baseline diff started: scenario=%s name=%s run=%s; recover with `git-control-tower baseline diff status --scenario %s --name %s --run %s --wait --json`\n",
				c.scenario, c.name, start.Msg.GetRunId(), c.scenario, c.name, start.Msg.GetRunId())
		} else {
			fmt.Fprint(os.Stderr, diffStartBanner(start.Msg))
			fmt.Fprintf(os.Stderr, "  waiting once for verdict; interrupt detaches but the server-side diff continues.\n")
		}
		return resolveDiff(core, c, start.Msg.GetRunId(), wait)
	}
	if c.json {
		return printJSON(start.Msg)
	}
	printDiffStart(start.Msg)
	return nil
}

// parkForDiff asks agent-manager to park the current run on this baseline diff
// when the diff is invoked inside an agent-manager run. It is a thin wrapper over
// cliutil.ParkForAwait fixing the producer + key encoding; see that primitive for
// the (result, parked, err) contract.
func parkForDiff(scenario, name string) (*cliutil.ParkResult, bool, error) {
	return cliutil.ParkForAwait(cliutil.ParkRequest{
		Producer: cliutil.ParkProducerGCT,
		Key:      scenario + "/" + name,
	})
}

// runDiffStatus resolves a started diff's cached verdict. It exits 0/1/2 by
// verdict, or 3 when the run is still in flight (with follow/resolve guidance).
func runDiffStatus(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var run string
	var wait, latest bool
	fs := newFlagSet("baseline diff status")
	c.bind(fs)
	fs.StringVar(&run, "run", "", "The diff's run id (from `baseline diff`) (required)")
	fs.BoolVar(&latest, "latest", false, "Recover the latest diff run recorded for this baseline (use when a wait was interrupted before you captured the run id)")
	fs.BoolVar(&wait, "wait", false, "Block server-side until the verdict is ready")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}
	if strings.TrimSpace(run) == "" && !latest {
		return fmt.Errorf("--run is required (the run id printed by `baseline diff`), or pass --latest to recover the latest run for this baseline")
	}
	if strings.TrimSpace(run) != "" && latest {
		return fmt.Errorf("--run and --latest are mutually exclusive")
	}
	return resolveDiff(core, c, run, wait, latest)
}

// resolveDiff fetches a diff verdict via GetDiffResult and renders it, exiting by
// verdict (0/1/2) or 3 when still in flight. wait blocks server-side.
func resolveDiff(core *cliapp.ScenarioApp, c commonFlags, run string, wait bool, latest ...bool) error {
	timeout := snapshotStartCeiling
	if wait {
		timeout = baselineClientTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := clientFactory(core)
	useLatest := len(latest) > 0 && latest[0]
	resp, recovered, err := durableReadWithEOFRecovery(ctx, wait, func(callCtx context.Context, blocking bool) (*connect.Response[baselinesv1.GetDiffResultResponse], error) {
		return client.GetDiffResult(callCtx, connect.NewRequest(&baselinesv1.GetDiffResultRequest{
			Scenario: c.scenario, Name: c.name, Branch: c.branch, RunId: run, Wait: blocking, Latest: useLatest,
		}))
	})
	if err != nil {
		return err
	}
	if recovered {
		fmt.Fprintf(os.Stderr, "diff wait attachment ended unexpectedly; recovered current state once by durable run id %s\n", run)
	}
	msg := resp.Msg
	effectiveRun := run
	if msg.GetRunId() != "" {
		effectiveRun = msg.GetRunId()
	}
	if msg.GetStatus() == "in_progress" {
		if c.json {
			if err := printJSON(msg); err != nil {
				return err
			}
		} else {
			printDiffPending(c.scenario, c.name, effectiveRun, int(msg.GetRecommendedNextCheckSeconds()))
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
	if msg.GetDiff() == nil {
		os.Exit(exitNotComparable)
	}
	os.Exit(exitCodeForVerdict(msg.GetDiff().GetVerdict()))
	return nil
}

// durableReadWithEOFRecovery performs one blocking attachment and, only when
// that attachment ends in EOF/cancel/deadline, one non-blocking read by the same durable id.
// The callback must never start work; this helper intentionally cannot retry a
// mutation and contains no polling loop.
func durableReadWithEOFRecovery[T any](ctx context.Context, wait bool, read func(context.Context, bool) (T, error)) (T, bool, error) {
	value, err := read(ctx, wait)
	if err == nil || !wait || !isAttachmentEnd(err) {
		return value, false, err
	}
	recoveryCtx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	value, err = read(recoveryCtx, false)
	return value, true, err
}

func isUnexpectedEOF(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) || strings.Contains(strings.ToLower(err.Error()), "unexpected eof")
}

func isAttachmentEnd(err error) bool {
	code := connect.CodeOf(err)
	return isUnexpectedEOF(err) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || code == connect.CodeDeadlineExceeded || code == connect.CodeCanceled || code == connect.CodeUnavailable
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
		fmt.Fprint(os.Stderr, agentWaitBlock(bi.GetScenario(), bi.GetRunId(), 0, false))
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
		if handled := renderBaselineNotFound(core, c, "show", err); handled {
			os.Exit(exitNotComparable)
		}
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	printShow(resp.Msg.GetBaseline())
	return nil
}

func renderBaselineNotFound(core *cliapp.ScenarioApp, c commonFlags, op string, cause error) bool {
	if connect.CodeOf(cause) != connect.CodeNotFound {
		return false
	}
	if strings.TrimSpace(c.scenario) == "" || strings.TrimSpace(c.name) == "" {
		return false
	}
	client := clientFactory(core)
	ctx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	st, err := client.GetSnapshotStatus(ctx, connect.NewRequest(&baselinesv1.GetSnapshotStatusRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch,
	}))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return false
	}
	fmt.Fprintf(os.Stderr, "✗ baseline %q for %s was not found", c.name, c.scenario)
	if c.branch != "" {
		fmt.Fprintf(os.Stderr, " on branch %s", c.branch)
	}
	fmt.Fprintln(os.Stderr, ".")
	if err == nil {
		printSnapshotStatusDiagnostics(os.Stderr, st.Msg)
	}
	fmt.Fprintf(os.Stderr, "  inspect snapshots: git-control-tower baseline snapshot status --scenario %s --name %s", c.scenario, c.name)
	if c.branch != "" {
		fmt.Fprintf(os.Stderr, " --branch %s", c.branch)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  list baselines:    git-control-tower baseline list --scenario %s --all-branches\n", c.scenario)
	fmt.Fprintf(os.Stderr, "  recent runs:       test-genie runs list --scenario %s --json --limit 10\n", c.scenario)
	if op == "diff" {
		fmt.Fprintln(os.Stderr, "  diff skipped because there is no resolved baseline manifest to compare.")
	}
	return true
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

func runRepair(core *cliapp.ScenarioApp, args []string) error {
	var c commonFlags
	var apply bool
	fs := newFlagSet("baseline repair")
	c.bind(fs)
	fs.BoolVar(&apply, "apply", false, "Apply the deterministic repair and write lifecycle audit evidence")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.requireScenarioName(); err != nil {
		return err
	}
	resp, err := clientFactory(core).RepairBaseline(context.Background(), connect.NewRequest(&baselinesv1.RepairBaselineRequest{
		Scenario: c.scenario, Name: c.name, Branch: c.branch, Apply: apply,
	}))
	if err != nil {
		return err
	}
	if c.json {
		return printJSON(resp.Msg)
	}
	if apply {
		fmt.Printf("Applied repair for baseline %q (generation %d)\n", c.name, resp.Msg.GetGeneration())
	} else {
		fmt.Printf("Repair plan for baseline %q (generation %d; dry run)\n", c.name, resp.Msg.GetGeneration())
	}
	for _, action := range resp.Msg.GetActions() {
		fmt.Printf("- %s\n", action)
	}
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
