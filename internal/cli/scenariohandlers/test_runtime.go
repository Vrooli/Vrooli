package scenariohandlers

import (
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // thin glue layer; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

// TestHandler routes `vrooli scenario test …`.
//
// The bare form runs the scenario's lifecycle `test` phase to completion. That
// phase shells `test-genie execute <scenario> --auto-start --wait`, and the run
// itself is owned by the test-genie SERVER — so an interrupted command leaves
// the run alive and re-attachable (the run id + re-attach command are printed
// up front by `execute`). There is no root-side run store, run-id scheme, or
// detach dance.
//
// The wait/status/follow/abort subcommands proxy to `test-genie runs …`, which
// owns the durable per-scenario run history. `logs` is a friendly alias for the
// server's event replay (`runs follow`).
func TestHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		if len(args) > 0 {
			switch args[0] {
			case "wait", "status", "follow", "abort", "logs":
				return proxyToTestGenieRuns(deps, ctx, args[0], args[1:])
			}
		}
		return testRunHandler(deps, ctx, args)
	}
}

// testRunHandler runs the lifecycle test phase to completion and returns the
// phase's exit status. Under --json it emits the typed TestPhaseResult summary.
func testRunHandler[C any](deps HandlerDeps[C], ctx C, args []string) error {
	req, err := ParseTestRequest(deps.Globals(ctx).JSON, deps.Globals(ctx).Verbose, args)
	if err != nil {
		return err
	}
	if req.Opts.CustomPath == "" {
		if err := ensureScenarioCLIs(deps, ctx, req.Name); err != nil {
			return err
		}
	}
	emitScenarioStaleWarning(deps.Stderr(ctx), deps.Root(ctx), req.Name, deps.Globals(ctx))

	runner, err := deps.LifecycleRunner(ctx)
	if err != nil {
		return err
	}
	service := NewRunnerService(runner)
	result, runErr := service.TestDetailed(scenarioapp.TestRequest{Name: req.Name, Opts: req.Opts})
	if result.Scenario == "" {
		result.Scenario = req.Name
	}

	if req.JSON {
		if writeErr := WriteTestPhaseResultJSON(deps.Stdout(ctx), result, runErr); writeErr != nil && runErr == nil {
			return writeErr
		}
	}
	return runErr
}

// proxyToTestGenieRuns forwards a run-handle subcommand to `test-genie runs
// <verb> <scenario> <run-id> …`. The test-genie server owns durable run state,
// so the root CLI is a thin pass-through (no proto dependency, consistent with
// how the lifecycle already shells test-genie). `logs` maps to `runs follow`.
func proxyToTestGenieRuns[C any](deps HandlerDeps[C], ctx C, verb string, args []string) error {
	if deps.LocateTestGenieCLI == nil || deps.RunSubprocess == nil {
		return rootcli.RuntimeErrorf("Run `test-genie runs …` directly", "the test-genie CLI proxy is not available in this context")
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return rootcli.CommandHelpOnly(testRunHandleHelp(verb))
		}
	}

	runsVerb := verb
	if verb == "logs" {
		runsVerb = "follow"
	}

	cliPath, err := deps.LocateTestGenieCLI(ctx)
	if err != nil {
		return err
	}

	commandArgs := append([]string{"runs", runsVerb}, args...)
	// wait/status/abort honor --json; follow is a live stream and ignores it.
	globals := deps.Globals(ctx)
	if globals.JSON && runsVerb != "follow" && !rootcli.ContainsFlag(commandArgs, "--json") {
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
			"Block until the run is terminal. Human mode streams live progress (same as `follow`); " +
			"--json emits one quiet snapshot for scripts. Exit 0 passed, 1 failed/aborted, 124 if --timeout elapses first."
	case "status":
		return "vrooli scenario test status <scenario> <run-id> [--json]\n\n" +
			"Live snapshot: status, active phase, elapsed, remaining-ETA, recommended next-check backoff."
	case "follow", "logs":
		return "vrooli scenario test " + verb + " <scenario> <run-id>\n\n" +
			"Stream the run's events to completion (replays history for a finished run)."
	case "abort":
		return "vrooli scenario test abort <scenario> <run-id> [--json]\n\n" +
			"Cancel a running run (transitions it to aborted)."
	default:
		return "vrooli scenario test " + verb + " <scenario> <run-id>"
	}
}
