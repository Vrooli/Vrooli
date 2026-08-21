// Command vrooli-agent-launcher is the shared, best-effort attribution
// boundary for native coding-agent binaries. It never interprets agent
// arguments as shell text.
//
// It has two modes, chosen by argv[0]:
//
//   - Shim mode, when invoked under an agent's own name (a link named `codex`,
//     `claude`, …). This is how attribution reaches every shell, including
//     non-interactive ones, without editing a shell profile.
//   - Explicit mode, when invoked as itself with --agent.
//
// Both modes end the same way: the process attributes the run and then replaces
// its own image with the agent, so nothing of the launcher survives into the
// agent's lifetime. See cliutil.LaunchCodingAgent.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func main() {
	if runner, ok := cliutil.ShimAliasFromArgv0(os.Args[0]); ok {
		os.Exit(runShim(runner, os.Args[1:]))
	}
	if err := run(os.Args[1:]); err != nil {
		os.Exit(reportLaunchError(err))
	}
}

// reportLaunchError maps a launch outcome onto this process's exit status,
// preserving the agent's own exit code whenever the agent actually ran.
func reportLaunchError(err error) int {
	var launchErr *cliutil.AgentLaunchError
	if errors.As(err, &launchErr) {
		fmt.Fprintln(os.Stderr, "vrooli-agent-launcher:", err)
		return 1
	}
	if code := cliutil.ChildExitCode(err); code >= 0 {
		return code
	}
	fmt.Fprintln(os.Stderr, "vrooli-agent-launcher:", err)
	return 1
}

// runShim launches the agent this binary was invoked as.
//
// The contract is fail-open: attribution is observability, so no failure inside
// this process may stop the operator's agent from starting. The only outcome
// that legitimately fails is "the real agent is not installed", because then
// there is nothing to fall back to.
func runShim(runner string, args []string) int {
	self := shimSelfPath()

	err := launchWithRecovery(runner, args, self)
	if err == nil {
		return 0
	}
	return reportLaunchError(err)
}

// launchWithRecovery runs the attributed launch and converts a panic in the
// attribution path into a direct, unattributed exec of the real agent. A bug in
// Vrooli's own code must cost the operator their attribution, never their
// session.
func launchWithRecovery(runner string, args []string, self string) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		fmt.Fprintf(os.Stderr, "vrooli-agent-launcher: attribution failed (%v); starting %s unattributed\n", recovered, runner)
		err = execUnattributed(runner, args, self)
	}()

	return cliutil.LaunchCodingAgent(context.Background(), cliutil.AgentLaunchRequest{
		Agent:  runner,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		LookPath: func(binary string) (string, error) {
			return cliutil.ResolveAgentBinaryExcluding(binary, self)
		},
	})
}

// execUnattributed is the last-resort path: resolve the real agent and replace
// this process with it, carrying no identity at all.
func execUnattributed(runner string, args []string, self string) error {
	binary, err := cliutil.CodingAgentBinary(runner)
	if err != nil {
		return err
	}
	path, err := cliutil.ResolveAgentBinaryExcluding(binary, self)
	if err != nil {
		return &cliutil.AgentLaunchError{Agent: runner, Err: err}
	}
	return cliutil.ExecAgent(path, binary, args, os.Environ())
}

// shimSelfPath resolves this executable so the PATH search can refuse to
// resolve back to it. An unresolvable path yields "", which the resolver treats
// as "cannot be self" — the safe direction, since a failed comparison must
// never hide the real agent.
func shimSelfPath() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}

func run(args []string) error {
	fs := flag.NewFlagSet("vrooli-agent-launcher", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agent := fs.String("agent", "", "coding-agent runner")
	task := fs.String("task", "", "optional task UUID to associate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if strings.TrimSpace(*agent) == "" {
		if len(remaining) == 0 {
			return errors.New("usage: vrooli-agent-launcher --agent <runner> [--task <uuid>] -- [agent args]")
		}
		*agent = remaining[0]
		remaining = remaining[1:]
	}

	// Resolve past our own shims here too. Without this, an explicit
	// `vrooli-agent-launcher --agent codex` resolves `codex` to the shim we
	// installed on PATH and execs through it, adding a pointless hop.
	self := shimSelfPath()
	return cliutil.LaunchCodingAgent(context.Background(), cliutil.AgentLaunchRequest{
		Agent:  *agent,
		TaskID: *task,
		Args:   remaining,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		LookPath: func(binary string) (string, error) {
			return cliutil.ResolveAgentBinaryExcluding(binary, self)
		},
	})
}
