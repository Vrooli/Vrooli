package scenariohandlers

import (
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // thin glue layer; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

const (
	testRuntimeFollow = "follow"
)

const (
	testRuntimeHelp = "--help"
	testRuntimeLogs = "logs"
)

// TestHandler routes `vrooli scenario test …`.
//
// The bare form is a thin alias for `test-genie --auto-start execute <scenario>
// …`. Test Genie owns foreground/background policy, stdout/stderr, JSON/JSONL,
// exit status, and the durable run id. There is no lifecycle test phase,
// root-side run store, wrapper JSON shape, or fallback command contract here.
//
// The wait/status/follow/abort subcommands proxy to `test-genie runs …`, which
// owns the durable per-scenario run history. testRuntimeLogs is a friendly alias for the
// server's event replay (`runs follow`).
func TestHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		if len(args) > 0 {
			switch args[0] {
			case "wait", "status", testRuntimeFollow, "abort", testRuntimeLogs:
				return proxyToTestGenieRuns(deps, ctx, args[0], args[1:])
			}
		}
		return testRunHandler(deps, ctx, args)
	}
}

// testRunHandler directly delegates to Test Genie so its run banner, JSON modes,
// and exit code are preserved exactly.
func testRunHandler[C any](deps HandlerDeps[C], ctx C, args []string) error {
	req, err := ParseTestRequest(deps.Globals(ctx).JSON, deps.Globals(ctx).Verbose, args)
	if err != nil {
		return err
	}
	if deps.LocateTestGenieCLI == nil || deps.RunSubprocess == nil {
		return rootcli.RuntimeErrorf("Run `test-genie execute …` directly", "the test-genie CLI alias is not available in this context")
	}
	cliPath, err := deps.LocateTestGenieCLI(ctx)
	if err != nil {
		return err
	}

	commandArgs := []string{"--auto-start", "execute", req.Name}
	commandArgs = append(commandArgs, req.Args...)
	if deps.Globals(ctx).JSON && !rootcli.ContainsFlag(commandArgs, "--json") && !rootcli.ContainsFlag(commandArgs, "--jsonl") {
		commandArgs = append(commandArgs, "--json")
	}

	return deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
		Name:   cliPath,
		Args:   commandArgs,
		Dir:    deps.Root(ctx),
		Env:    deps.CommandEnv(ctx),
		Stdin:  nil,
		Stdout: deps.Stdout(ctx),
		Stderr: deps.Stderr(ctx),
	})
}

// proxyToTestGenieRuns forwards a run-handle subcommand to `test-genie runs
// <verb> <scenario> <run-id> …`. The test-genie server owns durable run state,
// so the root CLI is a thin pass-through (no proto dependency, consistent with
// how the lifecycle already shells test-genie). testRuntimeLogs maps to `runs follow`.
func proxyToTestGenieRuns[C any](deps HandlerDeps[C], ctx C, verb string, args []string) error {
	if deps.LocateTestGenieCLI == nil || deps.RunSubprocess == nil {
		return rootcli.RuntimeErrorf("Run `test-genie runs …` directly", "the test-genie CLI proxy is not available in this context")
	}
	for _, arg := range args {
		if arg == testRuntimeHelp || arg == "-h" {
			return rootcli.CommandHelpOnly(testRunHandleHelp(verb))
		}
	}

	runsVerb := verb
	if verb == testRuntimeLogs {
		runsVerb = testRuntimeFollow
	}

	cliPath, err := deps.LocateTestGenieCLI(ctx)
	if err != nil {
		return err
	}

	commandArgs := append([]string{"runs", runsVerb}, args...)
	// wait/status/abort honor --json; follow is a live stream and ignores it.
	globals := deps.Globals(ctx)
	if globals.JSON && runsVerb != testRuntimeFollow && !rootcli.ContainsFlag(commandArgs, "--json") {
		commandArgs = append(commandArgs, "--json")
	}

	return deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
		Name:   cliPath,
		Args:   commandArgs,
		Dir:    deps.Root(ctx),
		Env:    deps.CommandEnv(ctx),
		Stdout: deps.Stdout(ctx),
		Stderr: deps.Stderr(ctx),
	})
}

func testRunHandleHelp(verb string) string {
	switch verb {
	case "wait":
		return "vrooli scenario test wait <scenario> <run-id> [--timeout <seconds>] [--json]\n\n" +
			"Block until the run is terminal. Human mode streams live progress (same as follow); " +
			"--json emits one quiet snapshot for scripts. Exit 0 passed, 1 failed/aborted, 124 if --timeout elapses first."
	case "status":
		return "vrooli scenario test status <scenario> <run-id> [--json]\n\n" +
			"Live snapshot: status, active phase, elapsed, remaining-ETA, recommended next-check backoff."
	case testRuntimeFollow, testRuntimeLogs:
		return "vrooli scenario test " + verb + " <scenario> <run-id>\n\n" +
			"Stream the run's events to completion (replays history for a finished run)."
	case "abort":
		return "vrooli scenario test abort <scenario> <run-id> [--json]\n\n" +
			"Cancel a running run (transitions it to aborted)."
	default:
		return "vrooli scenario test " + verb + " <scenario> <run-id>"
	}
}
