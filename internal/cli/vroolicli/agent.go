package vroolicli

import (
	"context"
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
	"github.com/vrooli/vrooli/internal/recovery"
)

const (
	agentGrok = "grok"
)

const (
	agentClaude   = "claude"
	agentCodex    = "codex"
	agentOpencode = "opencode"
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

func (app *App) runAgentCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return clipolicy.UsageErrorf("agent", "an agent subcommand is required (supported: launch, list, recover, thaw)")
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
	if args[0] != "launch" {
		return clipolicy.UsageErrorf("agent", "unsupported agent subcommand %q (supported: launch, list, recover, thaw)", args[0])
	}

	fs := commandtree.NewFlagSet("vrooli agent launch")
	fs.SetOutput(ctx.Stderr)
	runner := fs.String("runner", agentClaude, "coding-agent runner: claude, codex, opencode, or grok")
	prompt := fs.String("prompt", "", "optional non-interactive prompt")
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

	binary, err := agentRunnerBinary(*runner)
	if err != nil {
		return err
	}
	childArgs := agentInvocationArgs(binary, extra, *prompt)
	launch, err := cliutil.LaunchCodingAgentResult(context.Background(), cliutil.AgentLaunchRequest{
		Agent:      *runner,
		Args:       childArgs,
		WorkingDir: strings.TrimSpace(*cwd),
		Claims:     absoluteClaims(claims, strings.TrimSpace(*cwd)),
		Stdin:      ctx.Stdin,
		Stdout:     ctx.Stdout,
		Stderr:     ctx.Stderr,
	})
	if err != nil {
		return err
	}
	if launch.AttachFailure != "" && ctx.Stderr != nil {
		_, _ = fmt.Fprintf(ctx.Stderr, "agent launch attribution degraded at %s: %s\n", launch.Tier, launch.AttachFailure)
	}
	return nil
}

func (app *App) runAgentRecovery(ctx *CommandContext, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		path, err := recovery.DefaultRecordPath()
		if err != nil {
			return fmt.Errorf("resolve recovery record path: %w", err)
		}
		broker, err := recovery.New(path, nil)
		if err != nil {
			return err
		}
		for _, record := range broker.Records() {
			if ctx.Stdout != nil {
				_, _ = fmt.Fprintf(ctx.Stdout, "%s scenario=%s tier=%s budget_remaining=%d outcome=%s\n", record.ID, record.Scenario, record.TierReached, record.BudgetRemaining, record.Outcome)
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
		home, _ := os.UserHomeDir()
		binary, resolveErr := cliinvoke.Resolve(cliinvoke.ResolveOptions{RuntimeHome: home})
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
// Vrooli flag, while the underlying tools disagree: Claude uses -p, Codex and
// Grok accept a positional prompt, and OpenCode's non-interactive prompt lives
// under its run subcommand. Extra argv tokens remain operator-controlled but
// are always passed as typed arguments, never through a shell.
func agentInvocationArgs(binary string, extra []string, prompt string) []string {
	args := append([]string(nil), extra...)
	if strings.TrimSpace(prompt) == "" {
		return args
	}
	switch binary {
	case agentClaude:
		return append(args, "-p", prompt)
	case agentOpencode:
		args = append([]string{"run"}, args...)
		return append(args, prompt)
	case agentCodex, agentGrok:
		return append(args, prompt)
	default:
		// agentRunnerBinary is the admission boundary. Keep this fallback
		// conservative if a future caller bypasses that boundary.
		return append(args, prompt)
	}
}

func agentRunnerBinary(name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case agentClaude, "claude-code":
		return agentClaude, nil
	case agentCodex:
		return agentCodex, nil
	case agentOpencode:
		return agentOpencode, nil
	case agentGrok:
		return agentGrok, nil
	default:
		return "", fmt.Errorf("unsupported coding-agent runner %q (supported: claude, claude-code, codex, opencode, grok)", name)
	}
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
