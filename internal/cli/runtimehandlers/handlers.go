package runtimehandlers

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type runtimeService struct {
	run func([]string) error
}

const defaultRecoveryInspectLimit = 50

var (
	supervisorCommandNames = []string{"run", "status", "install", "uninstall"}
	recoveryPolicyNames    = []string{"set", "list"}
)

const runtimeHelpText = `vrooli runtime - Manage Vrooli runtime control-plane services

Usage:
  vrooli runtime supervisor run [options]
  vrooli runtime supervisor status [--json]
  vrooli runtime supervisor install [--user]
  vrooli runtime supervisor uninstall [--user]
  vrooli runtime recovery policy set <scenario> [options]
  vrooli runtime recovery policy list

Options:
  --json                    Emit JSON output when supported
  --help, -h                Show this help message

Environment:
  VROOLI_RUNTIME_SUPERVISOR                  Supervisor mode: off, auto, or on (default auto)
  VROOLI_RUNTIME_SUPERVISOR_RENEW_INTERVAL   Supervisor heartbeat interval (default 10s)
  VROOLI_RUNTIME_SUPERVISOR_LEASE_TTL        Runtime lease deadline extension (default 45s)
  VROOLI_RUNTIME_SUPERVISOR_HEALTH_INTERVAL  Health refresh planning interval (default 45s)
  VROOLI_RUNTIME_SUPERVISOR_MAX_HEALTH_CONCURRENCY
                                             Maximum concurrent health probes (default 16)
  VROOLI_RUNTIME_SUPERVISOR_BATCH_SIZE       Lease renewal batch size (default 250)
  VROOLI_RUNTIME_RECOVERY_QUIET_PERIOD       Pressure-clear duration before recovery (default 2m)
  VROOLI_RUNTIME_RECOVERY_COOLDOWN           Delay after a failed recovery (default 5m)
  VROOLI_RUNTIME_RECOVERY_CONCURRENCY        Maximum lifecycle recoveries per tier/tick (default 1)
  VROOLI_RUNTIME_PRESSURE_SOME_AVG10         Memory PSI some.avg10 recovery threshold (default 10)
`

// RegisteredCommandPaths returns the child paths bound by the runtime handler.
func RegisteredCommandPaths() []string {
	paths := make([]string, 0, len(supervisorCommandNames)+len(recoveryPolicyNames))
	for _, name := range supervisorCommandNames {
		paths = append(paths, "runtime supervisor "+name)
	}
	for _, name := range recoveryPolicyNames {
		paths = append(paths, "runtime recovery policy "+name)
	}
	return paths
}

// RootHandler dispatches `vrooli runtime` through the runtime application.
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (runtimeService, error) {
			operationCtx := rootcli.ResolveOperationContext(deps, ctx)
			commandCtx := &rootcli.CommandContext{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: operationCtx, HomeDirFn: func() (string, error) { return deps.HomeDir(ctx) }, ResolveRootFn: func() (string, error) { return deps.ResolveRoot(ctx) }, Version: deps.Version(ctx)}
			app := &runtimeapp.Service{Version: deps.Version(ctx), ResolveRootFn: func() (string, error) { return deps.ResolveRoot(ctx) }}
			return runtimeService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return writeRuntimeHelp(deps.Stdout(ctx))
				}
				if args[0] == "recovery" {
					return runRecoveryManifest(operationCtx, app, commandCtx, args[1:], deps.Stdout(ctx), deps.Stderr(ctx))
				}
				group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "runtime/supervisor", runtimeBindings(operationCtx, app, commandCtx, []string{"supervisor"}, supervisorCommandNames))
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli runtime", SubcommandGroups: []cliapp.SubcommandGroup{group}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service runtimeService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

func runRecoveryManifest(operationCtx context.Context, app *runtimeapp.Service, ctx *rootcli.CommandContext, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || manifestdispatch.WantsHelp(args) {
		return writeRecoveryHelp(stdout)
	}
	if args[0] == "inspect" {
		return runRecoveryInspect(operationCtx, app, ctx, args[1:])
	}
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "runtime/recovery/policy", runtimeBindings(operationCtx, app, ctx, []string{"recovery", "policy"}, recoveryPolicyNames))
	if err != nil {
		return err
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli runtime recovery", SubcommandGroups: []cliapp.SubcommandGroup{group}})
	return core.RunWithWriters(manifestdispatch.WithJSON(args, ctx.Globals.JSON), stdout, stderr)
}

func runtimeBindings(operationCtx context.Context, app *runtimeapp.Service, ctx *rootcli.CommandContext, prefix []string, names []string) map[string]func(cliapp.RunContext) error {
	bindings := make(map[string]func(cliapp.RunContext) error, len(names))
	for _, name := range names {
		command := append(append([]string(nil), prefix...), name)
		bindings[name] = func(command []string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				jsonOutput := ctx.Globals.JSON || runCtx.JSON()
				home, err := ctx.HomeDir()
				if err != nil {
					return err
				}
				switch command[len(command)-1] {
				case "run":
					return app.RunSupervisor(operationCtx, home, runtimeapp.SupervisorRunOptions{Takeover: runCtx.BoolFlag("takeover")})
				case "status":
					return app.SupervisorStatus(operationCtx, home, ctx.Stdout, runtimeapp.SupervisorStatusOptions{JSON: jsonOutput})
				case "install":
					return app.InstallSupervisor(operationCtx, home, ctx.Stdout, runtimeapp.SupervisorServiceOptions{User: true, JSON: jsonOutput})
				case "uninstall":
					return app.UninstallSupervisor(operationCtx, ctx.Stdout, runtimeapp.SupervisorServiceOptions{User: true, JSON: jsonOutput})
				case "list":
					return app.RecoveryPolicyList(operationCtx, home, ctx.Stdout)
				case "set":
					policy, err := recoveryPolicyFromRunContext(runCtx)
					if err != nil {
						return err
					}
					return app.RecoveryPolicySet(operationCtx, home, ctx.Stdout, policy)
				default:
					return fmt.Errorf("runtime command %q has no handler", command[len(command)-1])
				}
			}
		}(command)
	}
	return bindings
}

func writeRuntimeHelp(out io.Writer) error {
	_, err := io.WriteString(out, runtimeHelpText)
	return err
}

func writeRecoveryHelp(out io.Writer) error {
	_, err := io.WriteString(out, "Usage:\n  vrooli runtime recovery policy set <scenario> --critical --enabled --tier <n> --retry-budget <n> [--variant <name>] [--opt-out]\n  vrooli runtime recovery policy list\n  vrooli runtime recovery inspect [--limit <n>] [--json]\n")
	return err
}

func runRecoveryInspect(operationCtx context.Context, app *runtimeapp.Service, ctx *rootcli.CommandContext, args []string) error {
	opts := runtimeapp.RecoveryInspectOptions{Limit: defaultRecoveryInspectLimit, JSON: ctx.Globals.JSON}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			opts.JSON = true
		case "--limit":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: runtime recovery inspect: --limit requires a value")
			}
			value, err := strconv.Atoi(args[i+1])
			i++
			if err != nil || value < 1 || value > 1000 {
				return fmt.Errorf("usage: runtime recovery inspect: --limit must be between 1 and 1000")
			}
			opts.Limit = value
		default:
			return fmt.Errorf("usage: runtime recovery inspect: unknown option: %s", args[i])
		}
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.RecoveryInspect(operationCtx, home, ctx.Stdout, opts)
}

func recoveryPolicyFromRunContext(runCtx cliapp.RunContext) (scenarioruntime.RecoveryPolicy, error) {
	policy := scenarioruntime.RecoveryPolicy{Scenario: strings.TrimSpace(runCtx.Positional("scenario"))}
	if policy.Scenario == "" {
		return policy, rootcli.UsageErrorf("runtime recovery policy set", "scenario is required")
	}
	policy.Critical = runCtx.BoolFlag("critical")
	if runCtx.BoolFlag("not-critical") {
		policy.Critical = false
	}
	policy.Enabled = runCtx.BoolFlag("enabled")
	if runCtx.BoolFlag("disabled") {
		policy.Enabled = false
	}
	policy.OptOut = runCtx.BoolFlag("opt-out")
	if runCtx.BoolFlag("clear-opt-out") {
		policy.OptOut = false
	}
	policy.Variant = strings.TrimSpace(runCtx.Flag("variant"))
	var err error
	if raw := strings.TrimSpace(runCtx.Flag("tier")); raw != "" {
		policy.DependencyTier, err = strconv.Atoi(raw)
		if err != nil || policy.DependencyTier < 0 {
			return policy, rootcli.UsageErrorf("runtime recovery policy set", "--tier must be a non-negative integer")
		}
	}
	if raw := strings.TrimSpace(runCtx.Flag("retry-budget")); raw != "" {
		policy.RetryBudget, err = strconv.Atoi(raw)
		if err != nil || policy.RetryBudget < 0 {
			return policy, rootcli.UsageErrorf("runtime recovery policy set", "--retry-budget must be a non-negative integer")
		}
	}
	return policy, nil
}
