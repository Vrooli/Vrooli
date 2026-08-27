package vroolicli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
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
		return clipolicy.UsageErrorf("agent", "an agent subcommand is required (supported: launch)")
	}
	if args[0] != "launch" {
		return clipolicy.UsageErrorf("agent", "unsupported agent subcommand %q (supported: launch)", args[0])
	}

	fs := flag.NewFlagSet("vrooli agent launch", flag.ContinueOnError)
	fs.SetOutput(ctx.Stderr)
	runner := fs.String("runner", agentClaude, "coding-agent runner: claude, codex, opencode, or grok")
	prompt := fs.String("prompt", "", "optional non-interactive prompt")
	cwd := fs.String("cwd", "", "optional working directory")
	var extra agentArgs
	fs.Var(&extra, "arg", "additional argv token passed to the runner (repeatable)")
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
	command := exec.CommandContext(context.Background(), binary, childArgs...)
	command.Stdin = ctx.Stdin
	command.Stdout = ctx.Stdout
	command.Stderr = ctx.Stderr
	if strings.TrimSpace(*cwd) != "" {
		command.Dir = strings.TrimSpace(*cwd)
	}
	return command.Run()
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
