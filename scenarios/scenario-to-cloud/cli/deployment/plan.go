package deployment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
	"scenario-to-cloud/cli/operation"
)

// Plan outcomes (plans.v1.ExecutablePlan.outcome).
const (
	OutcomeApply      = "apply"
	OutcomeNoOp       = "no_op"
	OutcomeNeedsInput = "needs_input"
)

// planFlags are the flags shared by plan, apply, execute and start.
type planFlags struct {
	sel          *selector.Flags
	scope        *string
	forceBundle  *bool
	jsonOutput   *bool
	showCommands *bool
}

func registerPlanFlags(fs *flag.FlagSet) planFlags {
	return planFlags{
		sel:          selector.Register(fs),
		scope:        fs.String("scope", "", "Plan scope: full (default), install, runtime, start"),
		forceBundle:  fs.Bool("force-bundle", false, "Rebuild the release bundle before compiling so the plan pins a fresh release digest"),
		jsonOutput:   fs.Bool("json", false, "Output proto JSON"),
		showCommands: fs.Bool("show-commands", false, "Also print the derived shell preview (display only; never executed as such)"),
	}
}

func (pf planFlags) compile() CompileOptions {
	return CompileOptions{Scope: *pf.scope, ForceBundleBuild: *pf.forceBundle}
}

// applyFlags are the admission flags shared by apply, execute and start.
type applyFlags struct {
	requestKey *string
	preflight  *bool
	noWait     *bool
	timeout    *time.Duration
}

func registerApplyFlags(fs *flag.FlagSet) applyFlags {
	return applyFlags{
		requestKey: fs.String("request-key", "", "Idempotency key; the same key with the same digest replays the admitted operation"),
		preflight:  fs.Bool("preflight", false, "Run the target preflight checks as part of the operation before any effect"),
		noWait:     fs.Bool("no-wait", false, "Return after admission (exit 3) instead of waiting once"),
		timeout:    fs.Duration("timeout", operation.DefaultWaitTimeout, "Observer bound for the wait"),
	}
}

func runPlan(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment plan", flag.ContinueOnError)
	pf := registerPlanFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, pf.sel)
	if err != nil {
		return err
	}
	compiled, err := client.CompilePlan(context.Background(), ref.GetId(), pf.compile())
	if err != nil {
		return err
	}
	if *pf.jsonOutput {
		return protoout.Print(compiled)
	}
	WritePreview(os.Stdout, ref, compiled, *pf.showCommands)
	if compiled.GetPlan().GetOutcome() == OutcomeNeedsInput {
		return needsInput(compiled)
	}
	return nil
}

func runApply(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment apply", flag.ContinueOnError)
	sel := selector.Register(fs)
	scope := fs.String("scope", "", "Plan scope the digest was reviewed for: full (default), install, runtime, start")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	digest := fs.String("plan-digest", "", "The plan digest reviewed with 'deployment plan' (required)")
	af := registerApplyFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*digest) == "" {
		return apierr.Refused("--plan-digest is required: run 'scenario-to-cloud deployment plan %s' and pass the digest it printed", selector.Usage)
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	return admitAndWait(client, ref, *digest, ApplyOptions{Scope: *scope, RequestKey: *af.requestKey, RunPreflight: *af.preflight}, !*af.noWait, *af.timeout, *jsonOutput)
}

func runExecute(client *Client, args []string) error {
	return runPlanThenApply(client, "execute", "", args)
}

func runStart(client *Client, args []string) error {
	return runPlanThenApply(client, "start", "start", args)
}

// runPlanThenApply is execute/start: compile, print the review (unless
// --yes), apply the exact digest that was shown and wait once.
func runPlanThenApply(client *Client, verb, fixedScope string, args []string) error {
	fs := flag.NewFlagSet("deployment "+verb, flag.ContinueOnError)
	pf := registerPlanFlags(fs)
	yes := fs.Bool("yes", false, "Skip printing the plan review before applying")
	af := registerApplyFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scope := *pf.scope
	if fixedScope != "" {
		if scope != "" && scope != fixedScope {
			return apierr.Refused("deployment %s always plans the %q scope; use 'deployment execute --scope %s' instead", verb, fixedScope, scope)
		}
		scope = fixedScope
	}
	ref, err := resolveArgs(client, fs, pf.sel)
	if err != nil {
		return err
	}
	compiled, err := client.CompilePlan(context.Background(), ref.GetId(), CompileOptions{Scope: scope, ForceBundleBuild: *pf.forceBundle})
	if err != nil {
		return err
	}
	switch compiled.GetPlan().GetOutcome() {
	case OutcomeNeedsInput:
		if *pf.jsonOutput {
			if err := protoout.Print(compiled); err != nil {
				return err
			}
		} else {
			WritePreview(os.Stdout, ref, compiled, *pf.showCommands)
		}
		return needsInput(compiled)
	case OutcomeNoOp:
		if *pf.jsonOutput {
			return protoout.Print(compiled)
		}
		fmt.Println(selector.Identity(ref))
		fmt.Printf("no change: the observed state already satisfies the desired state (plan digest %s)\n", compiled.GetPlanDigest())
		return nil
	}
	if !*yes && !*pf.jsonOutput {
		WritePreview(os.Stdout, ref, compiled, *pf.showCommands)
		fmt.Println()
	}
	return admitAndWait(client, ref, compiled.GetPlanDigest(), ApplyOptions{Scope: scope, RequestKey: *af.requestKey, RunPreflight: *af.preflight}, !*af.noWait, *af.timeout, *pf.jsonOutput)
}

// admitAndWait applies the reviewed digest, prints the identities and waits
// once on the admitted operation.
func admitAndWait(client *Client, ref *identityv1.DeploymentRef, digest string, opts ApplyOptions, wait bool, timeout time.Duration, jsonOutput bool) error {
	if strings.TrimSpace(opts.RequestKey) == "" {
		opts.RequestKey = newRequestKey()
	}
	applied, err := client.ApplyPlan(context.Background(), ref.GetId(), digest, opts)
	if err != nil {
		return err
	}
	if applied.GetState() == OutcomeNoOp || applied.GetOperationId() == "" {
		if jsonOutput {
			return protoout.Print(applied)
		}
		fmt.Println(selector.Identity(ref))
		fmt.Printf("no change: nothing to apply (plan digest %s)\n", applied.GetPlanDigest())
		return nil
	}
	if jsonOutput {
		if err := protoout.Print(applied); err != nil {
			return err
		}
	} else {
		fmt.Println(selector.Identity(ref))
		fmt.Printf("operation: %s  state: %s\n", applied.GetOperationId(), applied.GetState())
		fmt.Printf("plan digest: %s\n", applied.GetPlanDigest())
		fmt.Printf("request key: %s\n", opts.RequestKey)
	}
	if !wait {
		return apierr.Pending("operation %s admitted; reattach: scenario-to-cloud operation wait %s", applied.GetOperationId(), applied.GetOperationId())
	}
	return operation.WaitAndReport(context.Background(), client.Operations, applied.GetOperationId(), timeout, jsonOutput)
}

// needsInput is the noninteractive handoff: the durable reference is
// printed and the command exits 3 without prompting.
func needsInput(compiled *plansv1.CompilePlanResponse) error {
	h := compiled.GetPlan().GetHandoff()
	if h == nil {
		h = compiled.GetPreview().GetHandoff()
	}
	msg := "input required before this plan can be applied"
	if h != nil {
		msg = fmt.Sprintf("input required before this plan can be applied; resume through %s (%s): %s", h.GetOwner(), h.GetKind(), h.GetReference())
		if len(h.GetMissing()) > 0 {
			msg += "\nmissing: " + strings.Join(h.GetMissing(), ", ")
		}
	}
	return apierr.Pending("%s", msg)
}

func newRequestKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "cli:" + time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	return "cli:" + hex.EncodeToString(b[:])
}

// WritePreview renders the review: identity, digest, outcome, changes, data
// effects, downtime, recovery strategy and handoff; the shell preview only
// on request.
func WritePreview(w io.Writer, ref *identityv1.DeploymentRef, compiled *plansv1.CompilePlanResponse, showCommands bool) {
	plan := compiled.GetPlan()
	preview := compiled.GetPreview()
	fmt.Fprintln(w, selector.Identity(ref))
	fmt.Fprintf(w, "plan digest: %s\n", compiled.GetPlanDigest())
	fmt.Fprintf(w, "scope: %s  outcome: %s  desired revision: %d\n", plan.GetScope(), plan.GetOutcome(), plan.GetDesiredRevision())
	if plan.GetReleaseDigest() != "" {
		fmt.Fprintf(w, "release digest: %s\n", plan.GetReleaseDigest())
	}
	if plan.GetConfigurationDigest() != "" {
		fmt.Fprintf(w, "configuration digest: %s\n", plan.GetConfigurationDigest())
	}
	if plan.GetClosureDigest() != "" {
		fmt.Fprintf(w, "closure digest: %s\n", plan.GetClosureDigest())
	}
	if compiled.GetClosureStatus() != "" && compiled.GetClosureStatus() != "derived" {
		fmt.Fprintf(w, "closure: %s\n", compiled.GetClosureStatus())
	}
	if preview.GetTarget() != "" {
		fmt.Fprintf(w, "target: %s\n", preview.GetTarget())
	}
	if p := plan.GetPresentation(); p != nil && p.GetSummary() != "" {
		fmt.Fprintf(w, "summary: %s\n", p.GetSummary())
	}
	if len(preview.GetChanges()) > 0 {
		fmt.Fprintln(w, "changes:")
		for i, c := range preview.GetChanges() {
			fmt.Fprintf(w, "  %2d. %s  [%s]", i+1, c.GetActionId(), c.GetEffect())
			if c.GetSummary() != "" {
				fmt.Fprintf(w, " %s", c.GetSummary())
			}
			fmt.Fprintln(w)
			details := []string{}
			if c.GetCapability() != "" {
				details = append(details, "capability="+c.GetCapability())
			}
			if c.GetVerification() != "" {
				details = append(details, "verify="+c.GetVerification())
			}
			if c.GetRecovery() != "" {
				details = append(details, "recovery="+c.GetRecovery())
			}
			if c.GetRetry() != "" {
				details = append(details, "retry="+c.GetRetry())
			}
			if c.GetCancelPoint() {
				details = append(details, "cancel-point")
			}
			if len(details) > 0 {
				fmt.Fprintf(w, "      %s\n", strings.Join(details, " "))
			}
		}
	} else if plan.GetOutcome() == OutcomeApply {
		fmt.Fprintln(w, "changes: none listed")
	}
	if len(preview.GetDataEffects()) > 0 {
		fmt.Fprintln(w, "data effects:")
		for _, d := range preview.GetDataEffects() {
			fmt.Fprintf(w, "  %s: %s (%s)\n", d.GetSubject(), d.GetEffect(), d.GetActionId())
		}
	}
	if dt := preview.GetDowntime(); dt != nil && (dt.GetExpectedSeconds() > 0 || dt.GetReason() != "") {
		fmt.Fprintf(w, "downtime: ~%ds", dt.GetExpectedSeconds())
		if dt.GetReason() != "" {
			fmt.Fprintf(w, " (%s)", dt.GetReason())
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintln(w, "downtime: none declared")
	}
	if preview.GetRecoveryStrategy() != "" {
		fmt.Fprintf(w, "recovery strategy: %s\n", preview.GetRecoveryStrategy())
	}
	if p := plan.GetPresentation(); p != nil {
		if p.GetDowntimeNote() != "" {
			fmt.Fprintf(w, "downtime note: %s\n", p.GetDowntimeNote())
		}
		if p.GetRecoveryNote() != "" {
			fmt.Fprintf(w, "recovery note: %s\n", p.GetRecoveryNote())
		}
	}
	if h := plan.GetHandoff(); h != nil && h.GetReference() != "" {
		fmt.Fprintf(w, "handoff: %s (%s) %s\n", h.GetOwner(), h.GetKind(), h.GetReference())
		if len(h.GetMissing()) > 0 {
			fmt.Fprintf(w, "  missing: %s\n", strings.Join(h.GetMissing(), ", "))
		}
	}
	if showCommands {
		if len(preview.GetShellPreview()) == 0 {
			fmt.Fprintln(w, "shell preview: none")
		} else {
			fmt.Fprintln(w, "shell preview (derived; execution uses typed verbs, not these strings):")
			for _, line := range preview.GetShellPreview() {
				fmt.Fprintf(w, "  [%s] %s\n", line.GetActionId(), line.GetCommand())
			}
		}
	}
}
