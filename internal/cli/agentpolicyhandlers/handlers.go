// Package agentpolicyhandlers wires the `vrooli agent-policy` command
// tree to a subprocess fan-out that calls each installed coding-agent
// resource CLI's `permissions <verb>` surface.
//
// Why subprocess fan-out rather than in-process: each resource CLI
// owns the on-disk shape of its agent's permission file (JSON for
// Claude, JSON-map for OpenCode, TOML for Codex) and the agent gate
// for that file. Embedding all three adapters here would re-create
// the duplication we just deleted in Phase 4. The shared substrate
// (packages/cli-core/agentpolicy) lives in each resource's binary;
// the top-level fan-out's job is purely to dispatch.
package agentpolicyhandlers

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/agentpolicycli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// HandlerDeps are the per-context dependencies the agent-policy
// handlers need. Tests inject ResolveBinary + RunCommand to avoid
// touching $PATH or spawning real subprocesses.
type HandlerDeps[C any] struct {
	Stdout func(C) io.Writer
	Stderr func(C) io.Writer
	// ResolveBinary returns ("", false) when the named CLI is not
	// installed. The default uses exec.LookPath.
	ResolveBinary func(name string) (string, bool)
	// RunCommand executes the resolved binary with the given args and
	// returns its combined stdout/stderr and exit error. The default
	// uses exec.Command with CombinedOutput.
	RunCommand func(binary string, args []string) (output string, err error)
	// Agents overrides the SupportedAgents list (used by tests).
	Agents []string
}

// RootHandler returns the top-level handler for `vrooli agent-policy`.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	deps = applyDefaults(deps)
	handlerMap := map[agentpolicycli.CommandID]rootcli.Handler[C]{
		agentpolicycli.CommandList:   forwardingHandler(deps, "list", false),
		agentpolicycli.CommandDeny:   forwardingHandler(deps, "deny", true),
		agentpolicycli.CommandAllow:  forwardingHandler(deps, "allow", true),
		agentpolicycli.CommandAsk:    forwardingHandler(deps, "ask", true),
		agentpolicycli.CommandRemove: forwardingHandler(deps, "remove", true),
		agentpolicycli.CommandReset:  forwardingHandler(deps, "reset", true),
		agentpolicycli.CommandDoctor: forwardingHandler(deps, "doctor", false),
	}
	specs := commandtree.BindSpecs(agentpolicycli.CommandSpecs(), handlerMap)
	commandHandlers := commandtree.BuildHandlerMap(specs)
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, agentpolicycli.RenderCommandHelp, "agent-policy", commandHandlers, deps.Stdout)
	}
}

func applyDefaults[C any](deps HandlerDeps[C]) HandlerDeps[C] {
	if deps.ResolveBinary == nil {
		deps.ResolveBinary = defaultResolveBinary
	}
	if deps.RunCommand == nil {
		deps.RunCommand = defaultRunCommand
	}
	if len(deps.Agents) == 0 {
		deps.Agents = agentpolicycli.SupportedAgents
	}
	return deps
}

func defaultResolveBinary(name string) (string, bool) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return path, true
}

func defaultRunCommand(binary string, args []string) (string, error) {
	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// forwardingHandler returns a handler that runs `<resource-cli>
// permissions <verb> [args...]` against every installed agent. mutating
// is set for verbs that should carry the OverrideFlag through to each
// resource (the fan-out passes it through verbatim — each resource
// re-evaluates the gate locally with its own DetectCallerKind).
func forwardingHandler[C any](deps HandlerDeps[C], verb string, mutating bool) rootcli.Handler[C] {
	_ = mutating
	return func(ctx C, args []string) error {
		stdout := deps.Stdout(ctx)
		failures := 0
		ran := 0
		for _, agent := range deps.Agents {
			binary, ok := deps.ResolveBinary(agent)
			if !ok {
				fmt.Fprintf(stdout, "==> %s: not-installed (skipped)\n", agent)
				continue
			}
			ran++
			invokeArgs := append([]string{"permissions", verb}, args...)
			fmt.Fprintf(stdout, "==> %s permissions %s%s\n", agent, verb, joinForDisplay(args))
			out, err := deps.RunCommand(binary, invokeArgs)
			if strings.TrimSpace(out) != "" {
				if !strings.HasSuffix(out, "\n") {
					out += "\n"
				}
				_, _ = io.WriteString(stdout, out)
			}
			if err != nil {
				failures++
				fmt.Fprintf(stdout, "    -> failed: %v\n", err)
			}
		}
		if ran == 0 {
			return errors.New("no coding-agent resources installed (looked for: " + strings.Join(deps.Agents, ", ") + ")")
		}
		if failures > 0 {
			return fmt.Errorf("%d of %d coding-agent resources reported failure", failures, ran)
		}
		return nil
	}
}

func joinForDisplay(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return " " + strings.Join(args, " ")
}
