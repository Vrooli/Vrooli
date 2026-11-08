package vroolicli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/envkit-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	configpkg "github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/recovery"
)

const (
	agentGrok = "grok"
)

const (
	agentClaude      = "claude"
	agentCodex       = "codex"
	agentOpencode    = "opencode"
	agentAntigravity = "agy"
)

// agentArgs is deliberately a repeatable argv surface. A coding-agent launch
// is allowed to delegate to the selected executable, but it never accepts a
// shell command string and never constructs one.
type agentArgs []string

func (a *agentArgs) String() string { return strings.Join(*a, ",") }

func (a *agentArgs) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--arg cannot be empty")
	}
	*a = append(*a, value)
	return nil
}

// agentSubcommands is the `vrooli agent` family in help order.
var agentSubcommands = [][2]string{
	{"launch", "Launch an allowlisted coding-agent CLI with inherited terminal input and output"},
	{"exec", "Run typed command argv inside a protected local native session"},
	{"list", "List coding-agent sessions with owner, state, memory and reapability"},
	{"reap", "Stop coding-agent sessions whose owner is confirmed gone"},
	{"recover", "Attempt budgeted mechanical lifecycle recovery of a scenario"},
	{"thaw", "Thaw a session scope frozen by storm containment (works with the API down)"},
	{"identity-binding", "Emit the versioned proof binding a harness session to its agent identity (--json)"},
	{"authorize", "Prepare reviewable consent for a session or run; a human approves it in Agent Manager"},
	{"authorization", "List a session's or run's operation authorization grants"},
	{"revoke-authorization", "Print the human decision link that revokes a session's approved grant"},
}

func printAgentHelp(ctx *AppContext) error {
	var b strings.Builder
	b.WriteString("Vrooli Agent Commands\n\nRun coding-agent sessions and bounded scenario recovery through the governed project surface.\n\nUsage:\n  vrooli agent <subcommand> [options]\n\nSubcommands:\n")
	for _, sub := range agentSubcommands {
		fmt.Fprintf(&b, "  %-22s %s\n", sub[0], sub[1])
	}
	b.WriteString("\nRun 'vrooli agent <subcommand> --help' for subcommand options.\n")
	_, err := fmt.Fprint(ctx.Stdout, b.String())
	return err
}

func (app *App) runAgentCommand(ctx *AppContext, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return printAgentHelp(ctx)
	}
	switch args[0] {
	case "help", "-h", "--help":
		return printAgentHelp(ctx)
	}
	if args[0] == "authorize" || args[0] == "authorization" || args[0] == "revoke-authorization" {
		return app.runAgentAuthorization(ctx, args[0], args[1:])
	}
	if args[0] == "exec" {
		return app.runAgentExec(ctx, args[1:])
	}
	if args[0] == "recover" {
		return app.runAgentRecovery(ctx, args[1:])
	}
	if args[0] == "thaw" {
		return app.runAgentThaw(ctx, args[1:])
	}
	if args[0] == "list" {
		return app.runAgentList(ctx, args[1:])
	}
	if args[0] == "reap" {
		return app.runAgentReap(ctx, args[1:])
	}
	if args[0] == "identity-binding" {
		return app.runAgentIdentityBinding(ctx, args[1:])
	}
	if args[0] != "launch" {
		names := make([]string, 0, len(agentSubcommands))
		for _, sub := range agentSubcommands {
			names = append(names, sub[0])
		}
		return clipolicy.UsageErrorf("agent", "unsupported agent subcommand %q (supported: %s)", args[0], strings.Join(names, ", "))
	}

	fs := commandtree.NewFlagSet("vrooli agent launch")
	fs.SetOutput(ctx.Stderr)
	runner := fs.String("runner", agentClaude, "coding-agent runner: claude, codex, opencode, grok, or antigravity")
	prompt := fs.String("prompt", "", "optional non-interactive prompt")
	requireIdentity := fs.Bool("require-native-identity", false, "refuse protected startup without native identity; broker refusal is terminal")
	timeout := fs.Duration("timeout", 0, "bound a waiting agent invocation; zero uses the caller's deadline")
	cwd := fs.String("cwd", "", "optional working directory")
	var extra agentArgs
	fs.Var(&extra, "arg", "additional argv token passed to the runner (repeatable)")
	var claims agentArgs
	fs.Var(&claims, "claim", "path this session will edit; a live holder of an overlapping claim is named and the launch continues (repeatable)")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return clipolicy.UsageErrorf("agent launch", "unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *timeout < 0 {
		return clipolicy.UsageErrorf("agent launch", "--timeout must be non-negative")
	}
	launchCtx := ctx.OperationContext()
	if *timeout > 0 {
		var cancel context.CancelFunc
		launchCtx, cancel = context.WithTimeout(launchCtx, *timeout)
		defer cancel()
	}

	binary, err := agentRunnerBinary(*runner)
	if err != nil {
		return err
	}
	childArgs := agentInvocationArgs(binary, extra, *prompt)
	launch, err := cliutil.LaunchCodingAgentResult(launchCtx, cliutil.AgentLaunchRequest{
		Agent:                 *runner,
		RequireNativeIdentity: *requireIdentity,
		Args:                  childArgs,
		WorkingDir:            strings.TrimSpace(*cwd),
		Claims:                absoluteClaims(claims, strings.TrimSpace(*cwd)),
		Stdin:                 ctx.Stdin,
		Stdout:                ctx.Stdout,
		Stderr:                ctx.Stderr,
	})
	if err != nil {
		return err
	}
	if launch.AttachFailure != "" && ctx.Stderr != nil {
		_, _ = fmt.Fprintf(ctx.Stderr, "agent launch attribution degraded at %s: %s\n", launch.Tier, launch.AttachFailure)
	}
	return nil
}

func (app *App) runAgentRecovery(ctx *AppContext, args []string) error {
	if len(args) > 0 && args[0] == "investigate" {
		return runCodingAgentInvestigation(ctx, args[1:], newCodingAgentInvestigationBroker)
	}
	if len(args) > 0 && args[0] == "reconcile" {
		fs := commandtree.NewFlagSet("vrooli agent recover reconcile")
		fs.SetOutput(ctx.Stderr)
		limit := fs.Int("limit", 32, "maximum unverified execution claims to reconcile (1–128)")
		asJSON := fs.Bool("json", ctx.Globals.JSON, "emit the bounded metadata reconciliation; no execution effects")
		if err := fs.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if fs.NArg() != 0 {
			return clipolicy.UsageErrorf("agent recover reconcile", "unexpected positional arguments")
		}
		path, err := recovery.DefaultCodingAgentRecordPath()
		if err != nil {
			return err
		}
		broker, err := recovery.NewCodingAgentBroker(path, nil)
		if err != nil {
			return err
		}
		report, err := broker.ReconcileUnverified(ctx.OperationContext(), *limit)
		if err != nil {
			return err
		}
		if ctx.Stdout == nil {
			return nil
		}
		if *asJSON {
			return json.NewEncoder(ctx.Stdout).Encode(report)
		}
		_, err = fmt.Fprintf(ctx.Stdout, "Reconciled %d execution claims as unconfirmed; %d remain. No process was stopped or started; duplicate suppression remains.\n", len(report.Changed), report.Remaining)
		return err
	}
	if len(args) > 0 && args[0] == "list" {
		fs := commandtree.NewFlagSet("vrooli agent recover list")
		fs.SetOutput(ctx.Stderr)
		asJSON := fs.Bool("json", ctx.Globals.JSON, "print lifecycle and coding-agent recovery receipts as JSON")
		if err := fs.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if fs.NArg() != 0 {
			return clipolicy.UsageErrorf("agent recover list", "unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
		}
		path, err := recovery.DefaultRecordPath()
		if err != nil {
			return fmt.Errorf("resolve recovery record path: %w", err)
		}
		broker, err := recovery.New(path, nil)
		if err != nil {
			return err
		}
		codingPath, err := recovery.DefaultCodingAgentRecordPath()
		if err != nil {
			return err
		}
		codingBroker, err := recovery.NewCodingAgentBroker(codingPath, nil)
		if err != nil {
			return err
		}
		codingRecords, err := codingBroker.Snapshot()
		if err != nil {
			return err
		}
		if *asJSON {
			if ctx.Stdout == nil {
				return nil
			}
			return json.NewEncoder(ctx.Stdout).Encode(struct {
				Lifecycle    []recovery.Record            `json:"lifecycle"`
				CodingAgents []recovery.CodingAgentRecord `json:"coding_agents"`
			}{broker.Records(), codingRecords})
		}
		for _, record := range broker.Records() {
			if ctx.Stdout != nil {
				_, _ = fmt.Fprintf(ctx.Stdout, "%s scenario=%s tier=%s budget_remaining=%d outcome=%s\n", record.ID, record.Scenario, record.TierReached, record.BudgetRemaining, record.Outcome)
			}
		}
		for _, record := range codingRecords {
			if ctx.Stdout != nil {
				_, _ = fmt.Fprintf(ctx.Stdout, "%s scenario=%s kind=coding-agent runner=%s outcome=%s created_at=%s deadline_at=%s error=%q reconciliation=%q\n", record.ID, record.Scenario, record.Runner, record.Outcome, record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), record.DeadlineAt.Format("2006-01-02T15:04:05Z07:00"), record.Error, record.ReconciliationReason)
			}
		}
		return nil
	}
	fs := commandtree.NewFlagSet("vrooli agent recover")
	fs.SetOutput(ctx.Stderr)
	scenario := fs.String("scenario", "", "scenario to recover")
	reason := fs.String("reason", "", "reason for recovery")
	requester := fs.String("requester", "vrooli-cli", "component requesting recovery")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return clipolicy.UsageErrorf("agent recover", "unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	path, err := recovery.DefaultRecordPath()
	if err != nil {
		return fmt.Errorf("resolve recovery record path: %w", err)
	}
	broker, err := recovery.New(path, func(runCtx context.Context, tier, target string, childEnv []string) error {
		home, _ := configpkg.HomeDir()
		binary, resolveErr := cliinvoke.Resolve(cliinvoke.ResolveOptions{UserHome: home})
		if resolveErr != nil {
			return resolveErr
		}
		if tier == recovery.RecoveryTierThree {
			setup := cliinvoke.Run(runCtx, cliinvoke.Invocation{Binary: binary, Args: cliinvoke.ScenarioSetup(target), Env: childEnv})
			if setupErr := setup.Error(); setupErr != nil {
				return fmt.Errorf("setup: %w", setupErr)
			}
		}
		start := cliinvoke.Run(runCtx, cliinvoke.Invocation{
			Binary: binary,
			Args:   cliinvoke.ScenarioLifecycle("start", target, tier == recovery.RecoveryTierTwo),
			Env:    childEnv,
			Stdout: ctx.Stdout,
			Stderr: ctx.Stderr,
		})
		return start.Error()
	})
	if err != nil {
		return err
	}
	record, recoverErr := broker.Recover(context.Background(), recovery.Request{Scenario: strings.TrimSpace(*scenario), Reason: strings.TrimSpace(*reason), Requester: strings.TrimSpace(*requester)}, envkit.Env(os.Environ()))
	if ctx.Stdout != nil {
		_, _ = fmt.Fprintf(ctx.Stdout, "agent recovery %s: tier=%s budget_remaining=%d outcome=%s\n", record.ID, record.TierReached, record.BudgetRemaining, record.Outcome)
	}
	return recoverErr
}

// agentInvocationArgs keeps the governed wrapper's prompt contract aligned
// with each runner's real CLI. The wrapper deliberately exposes one stable
// Vrooli flag, while the underlying tools disagree: Claude and Antigravity use
// -p, Codex uses exec, and OpenCode's non-interactive prompt lives
// under its run subcommand. Extra argv tokens remain operator-controlled but
// are always passed as typed arguments, never through a shell.
func agentInvocationArgs(binary string, extra []string, prompt string) []string {
	args := append([]string(nil), extra...)
	if strings.TrimSpace(prompt) == "" {
		return args
	}
	headless := recovery.CodingAgentArguments(binary, prompt)
	if binary == agentOpencode || binary == agentCodex {
		args = append(headless[:1:1], args...)
		return append(args, prompt)
	}
	return append(args, headless...)
}

// agentRunnerBinary admits exactly the runners the harness descriptors declare.
func agentRunnerBinary(name string) (string, error) {
	return cliutil.CodingAgentBinary(name)
}

// absoluteClaims resolves claim paths against the launch directory so two
// sessions in different working directories compare the same tree.
func absoluteClaims(claims []string, cwd string) []string {
	if len(claims) == 0 {
		return nil
	}
	base := cwd
	if base == "" {
		base, _ = os.Getwd()
	}
	out := make([]string, 0, len(claims))
	for _, claim := range claims {
		claim = strings.TrimSpace(claim)
		if claim == "" {
			continue
		}
		if !filepath.IsAbs(claim) {
			claim = filepath.Join(base, claim)
		}
		out = append(out, filepath.Clean(claim))
	}
	return out
}

// runAgentExec wraps commands inside an existing local task without another model.
func (app *App) runAgentExec(ctx *AppContext, args []string) error {
	fs := commandtree.NewFlagSet("vrooli agent exec")
	fs.SetOutput(ctx.Stderr)
	task := fs.String("task", "", "existing native task UUID; omit if the local task has no native task row")
	cwd := fs.String("cwd", "", "command working directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return clipolicy.UsageErrorf("agent exec", "command argv required after --")
	}
	_, err := cliutil.LaunchCommandSessionResult(context.Background(), cliutil.AgentLaunchRequest{
		TaskID: *task, Args: fs.Args(), WorkingDir: *cwd,
		Stdin: ctx.Stdin, Stdout: ctx.Stdout, Stderr: ctx.Stderr,
	})
	return err
}
