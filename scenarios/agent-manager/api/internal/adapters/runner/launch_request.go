package runner

import (
	"fmt"
	"strings"
)

// buildEnvWrappedLaunchRequest builds a [LaunchRequest] for an agent that
// runs through the `env` shim so a per-run tag lands in `/proc/<pid>/cmdline`
// (the agent-manager reconciler reads it to detect orphaned processes).
//
// All three coding-agent runners (claude_code, codex, opencode) construct
// this request the same way:
//
//   - Command is always "env" — never the agent binary directly. The
//     leading `<TAG_KEY>=<tag>` env arg surfaces in /proc and lets the
//     reconciler match running processes back to their RunID.
//   - The first positional after the env arg is the resolved binary path.
//   - The remaining positionals are the agent's CLI args.
//   - The full process environment comes from the runner's buildEnv().
//   - Stdin is the prompt as a string reader (the prompt-via-stdin
//     pattern shared across runners). When prompt is empty, Stdin is nil
//     and the launcher closes the pipe immediately.
//   - IdleTimeout is always [DefaultStreamIdleTimeout] for the streaming
//     path; durable-transcript paths choose their own and bypass this
//     helper entirely.
//
// Centralising the construction here keeps the per-runner Execute
// methods short and means one place to fix when the env shim pattern
// changes or a new runner needs the same wiring.
func buildEnvWrappedLaunchRequest(
	tagEnvKey, binaryPath string,
	args []string,
	tag, prompt string,
	env []string,
	workingDir string,
) LaunchRequest {
	envArgs := make([]string, 0, 2+len(args))
	envArgs = append(envArgs, fmt.Sprintf("%s=%s", tagEnvKey, tag))
	envArgs = append(envArgs, binaryPath)
	envArgs = append(envArgs, args...)

	req := LaunchRequest{
		Command:     "env",
		Args:        envArgs,
		Env:         env,
		WorkingDir:  workingDir,
		IdleTimeout: DefaultStreamIdleTimeout,
	}
	if prompt != "" {
		req.Stdin = strings.NewReader(prompt)
	}
	return req
}
