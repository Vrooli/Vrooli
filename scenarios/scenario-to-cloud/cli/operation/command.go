package operation

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/operationstanding"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
)

// Exit codes of the operation commands (shared contract with the API):
// 0 ok, 1 failed, 2 refused, 3 pending (non-terminal standing returned),
// 124 observer timeout (wait returned still_pending).
const (
	ExitOK              = apierr.ExitOK
	ExitFailed          = apierr.ExitFailed
	ExitRefused         = apierr.ExitRefused
	ExitPending         = apierr.ExitPending
	ExitObserverTimeout = apierr.ExitObserverTimeout
)

// Commands is the operation command group over the operations client and
// the deployments resolver (for `list` selectors).
type Commands struct {
	Client      *Client
	Deployments deploymentsv1connect.DeploymentsServiceClient
}

// Run executes operation subcommands.
func (c Commands) Run(args []string) error {
	if len(args) == 0 {
		return printUsage()
	}
	switch args[0] {
	case "get":
		return c.runGet(args[1:])
	case "wait":
		return c.runWait("wait", args[1:])
	case "resume":
		return c.runWait("resume", args[1:])
	case "cancel":
		return c.runCancel(args[1:])
	case "list":
		return c.runList(args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud operation help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud operation <command> [arguments]

Durable cloud operations. The operation id is the wait reference a client
keeps across disconnects and owner restarts; the owner holds progress. An
operation started from the API or the UI is attached here by the same id.

Commands:
  get <operation-id>        Typed standing (state, active step, receipts, next action)
  wait <operation-id>       Block once server-side until terminal (--timeout, default 300s)
  resume <operation-id>     Attach to an operation after a disconnect (same as wait)
  cancel <operation-id>     Record a cancellation intent (honoured at the next safe point)
  list <selector>           Operations admitted against a deployment ` + selector.Usage + `

Exit codes: 0 succeeded, 1 failed/failed_recovery/cancelled, 2 refused,
3 pending (non-terminal), 124 observer timeout (wait returned still_pending;
the operation is unchanged and the reattach command is printed).

Run 'scenario-to-cloud operation <command> -h' for command-specific options.`)
	return nil
}

func (c Commands) runGet(args []string) error {
	fs := flag.NewFlagSet("operation get", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud operation get <operation-id>")
	}
	st, err := c.Client.Get(context.Background(), fs.Arg(0))
	if err != nil {
		return err
	}
	if err := emit(st, *jsonOutput); err != nil {
		return err
	}
	return ExitFor(st, false)
}

func (c Commands) runWait(verb string, args []string) error {
	fs := flag.NewFlagSet("operation "+verb, flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	timeout := fs.Duration("timeout", DefaultWaitTimeout, "Observer bound (server-side block); capped by the owner")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud operation %s <operation-id> [--timeout 300s]", verb)
	}
	st, err := c.Client.Wait(context.Background(), fs.Arg(0), *timeout)
	if err != nil {
		return err
	}
	if err := emit(st, *jsonOutput); err != nil {
		return err
	}
	return ExitFor(st, true)
}

func (c Commands) runCancel(args []string) error {
	fs := flag.NewFlagSet("operation cancel", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud operation cancel <operation-id>")
	}
	st, err := c.Client.Cancel(context.Background(), fs.Arg(0))
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(st)
	}
	fmt.Printf("Cancellation recorded for %s\n", st.GetOperationId())
	return WriteStanding(os.Stdout, st)
}

func (c Commands) runList(args []string) error {
	fs := flag.NewFlagSet("operation list", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	chosen, err := sel.Selector(fs.Args())
	if err != nil {
		return err
	}
	ref, err := selector.Resolve(context.Background(), c.Deployments, chosen)
	if err != nil {
		return err
	}
	resp, err := c.Client.List(context.Background(), ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(resp)
	}
	fmt.Println(selector.Identity(ref))
	if len(resp.GetOperations()) == 0 {
		fmt.Printf("No operations for deployment %s\n", resp.GetDeploymentId())
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "OPERATION\tSTATE\tFENCE\tPLAN DIGEST\tACTIVE STEP\tCOMPLETED\tUPDATED")
	for _, op := range resp.GetOperations() {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%d\t%s\n", op.GetOperationId(), op.GetState(), op.GetFence(), op.GetPlanDigest(), op.GetActiveStep(), len(op.GetCompletedSteps()), op.GetUpdatedAt())
	}
	return w.Flush()
}

func emit(st *operationsv1.OperationStanding, jsonOutput bool) error {
	if jsonOutput {
		return protoout.Print(st)
	}
	return WriteStanding(os.Stdout, st)
}

// WriteStanding renders the shared human view plus the operation-specific
// facts (identities, receipts, unknown effects, result, next action).
func WriteStanding(w io.Writer, st *operationsv1.OperationStanding) error {
	fmt.Fprintf(w, "Operation %s (deployment %s)\n", st.GetOperationId(), st.GetDeploymentId())
	if err := operationstanding.WriteText(w, standingView{st}); err != nil {
		return err
	}
	fmt.Fprintf(w, "  fence: %d\n", st.GetFence())
	if st.GetPlanDigest() != "" {
		fmt.Fprintf(w, "  plan digest: %s\n", st.GetPlanDigest())
	}
	if st.GetRequestKey() != "" {
		fmt.Fprintf(w, "  request key: %s\n", st.GetRequestKey())
	}
	if len(st.GetCompletedSteps()) > 0 {
		fmt.Fprintf(w, "  completed steps: %s\n", strings.Join(st.GetCompletedSteps(), ", "))
	}
	for _, r := range st.GetStepReceipts() {
		if r.GetOutcome() == "failed" || r.GetOutcome() == "unknown" {
			fmt.Fprintf(w, "  step %s: %s", r.GetStep(), r.GetOutcome())
			if r.GetError() != "" {
				fmt.Fprintf(w, " (%s)", r.GetError())
			}
			fmt.Fprintln(w)
		}
	}
	for _, e := range st.GetUnknownEffects() {
		fmt.Fprintf(w, "  unknown effect at %s: %s -> %s\n", e.GetStep(), e.GetReason(), e.GetNextAction())
	}
	if res := st.GetResult(); res != nil {
		fmt.Fprintf(w, "  outcome: %s", res.GetOutcome())
		if res.GetRecoveryOutcome() != "" {
			fmt.Fprintf(w, " (recovery: %s)", res.GetRecoveryOutcome())
		}
		if res.GetMessage() != "" {
			fmt.Fprintf(w, ": %s", res.GetMessage())
		}
		fmt.Fprintln(w)
	}
	if e := st.GetError(); e != nil {
		fmt.Fprintf(w, "  error: %s (%s)\n", e.GetMessage(), e.GetCode())
	}
	if na := st.GetNextAction(); na != nil && na.GetLabel() != "" {
		fmt.Fprintf(w, "  next: %s", na.GetLabel())
		if na.GetReference() != "" {
			fmt.Fprintf(w, " (%s)", na.GetReference())
		}
		fmt.Fprintln(w)
	}
	return nil
}

// standingView adapts the generated standing to the shared renderer.
type standingView struct {
	st *operationsv1.OperationStanding
}

func (v standingView) GetLifecycle() string   { return v.st.GetState() }
func (v standingView) GetActivePhase() string { return v.st.GetActiveStep() }
func (v standingView) GetEtaKnown() bool      { return false }
func (v standingView) GetEstimatedRemainingSeconds() int32 {
	return 0
}

func (v standingView) GetDirective() string {
	switch v.st.GetState() {
	case "succeeded":
		return "done"
	case "failed", "failed_recovery", "cancelled":
		return "inspect"
	case "waiting_input":
		return "provide input"
	case "reconciling", "recovering":
		return "reconcile"
	default:
		return "wait"
	}
}

func (v standingView) GetReattachCommand() string {
	if v.st.GetTerminal() {
		return ""
	}
	return v.st.GetReattachCommand()
}

// ExitFor maps a standing to the exit contract. observer=true marks a wait
// whose bound elapsed (still_pending) as 124 rather than 3.
func ExitFor(st *operationsv1.OperationStanding, observer bool) error {
	switch st.GetState() {
	case "succeeded":
		return nil
	case "failed", "failed_recovery", "cancelled":
		return apierr.Failed("operation %s %s", st.GetOperationId(), st.GetState())
	}
	if observer && st.GetStillPending() {
		msg := fmt.Sprintf("operation %s still %s after the observer bound; the operation is unchanged. Re-run: %s", st.GetOperationId(), st.GetState(), ReattachCommand(st))
		if st.GetRecommendedNextCheckSeconds() > 0 {
			msg += fmt.Sprintf(" (recommended next check in %ds)", st.GetRecommendedNextCheckSeconds())
		}
		return &apierr.Exit{Code: ExitObserverTimeout, Message: msg}
	}
	return apierr.Pending("operation %s is %s; reattach: %s", st.GetOperationId(), st.GetState(), ReattachCommand(st))
}

// ReattachCommand is the server's reattach command, or the CLI's own wait
// command when the server gave none.
func ReattachCommand(st *operationsv1.OperationStanding) string {
	if cmd := strings.TrimSpace(st.GetReattachCommand()); cmd != "" {
		return cmd
	}
	return "scenario-to-cloud operation wait " + st.GetOperationId()
}

// WaitAndReport blocks once on the operation and prints the standing; it is
// the shared tail of every command that admits an operation.
func WaitAndReport(ctx context.Context, client *Client, id string, timeout time.Duration, jsonOutput bool) error {
	st, err := client.Wait(ctx, id, timeout)
	if err != nil {
		return err
	}
	if err := emit(st, jsonOutput); err != nil {
		return err
	}
	return ExitFor(st, true)
}
