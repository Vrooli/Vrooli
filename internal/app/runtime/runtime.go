package runtimeapp

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const HelpText = `vrooli runtime - Manage Vrooli runtime control-plane services

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

// Run dispatches the runtime command family.
func (app *App) Run(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, HelpText)
		return nil
	}
	if args[0] == "recovery" {
		return app.runRuntimeRecovery(ctx, args[1:])
	}
	if args[0] != "supervisor" {
		return rootcli.UsageErrorf("runtime", "unknown runtime command: %s", args[0])
	}
	if len(args) == 1 {
		_, _ = io.WriteString(ctx.Stdout, HelpText)
		return nil
	}
	switch args[1] {
	case "run":
		return app.runSupervisor(ctx, args[2:])
	case "status":
		return app.statusSupervisor(ctx, args[2:])
	case "install":
		return app.installSupervisor(ctx, args[2:])
	case "uninstall":
		return app.uninstallSupervisor(ctx, args[2:])
	default:
		return rootcli.UsageErrorf("runtime supervisor", "unknown runtime supervisor command: %s", args[1])
	}
}

func (app *App) runSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, HelpText)
		return nil
	}
	if len(args) > 0 {
		return rootcli.UsageErrorf("runtime supervisor run", "runtime supervisor run does not accept positional arguments")
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	cfg := runtimesupervisor.EnvConfig()
	cfg.HomeDir = home
	cfg.Version = app.Version
	return runtimesupervisor.Run(context.Background(), cfg)
}

func (app *App) statusSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, HelpText)
		return nil
	}
	jsonOutput := ctx.Globals.JSON
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			return rootcli.UsageErrorf("runtime supervisor status", "unknown option for runtime supervisor status: %s", arg)
		}
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	cfg := runtimesupervisor.EnvConfig()
	cfg.HomeDir = home
	svc := runtimesupervisor.New(cfg)
	defer svc.Close()
	report, err := svc.Status(context.Background())
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeSupervisorStatusJSON(ctx.Stdout, report)
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "Runtime supervisor: %s\n", report.Status)
	if report.StatusReason != "" {
		_, _ = fmt.Fprintf(ctx.Stdout, "Reason: %s\n", report.StatusReason)
	}
	if report.SupervisorID != "" {
		_, _ = fmt.Fprintf(ctx.Stdout, "Supervisor ID: %s\n", report.SupervisorID)
		_, _ = fmt.Fprintf(ctx.Stdout, "Host boot/session: %s / %s\n", report.HostBootID, report.HostSessionID)
		_, _ = fmt.Fprintf(ctx.Stdout, "Heartbeat: %s -> %s\n", report.LastHeartbeatAt.Format(time.RFC3339), report.HeartbeatDeadlineAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "Supervised running instances: %d\n", report.SupervisedInstanceCount)
	_, _ = fmt.Fprintf(ctx.Stdout, "Unverified running instances: %d\n", report.UnverifiedInstanceCount)
	_, _ = fmt.Fprintf(ctx.Stdout, "Renew interval: %s\n", report.EffectiveRenewInterval)
	_, _ = fmt.Fprintf(ctx.Stdout, "Lease TTL: %s\n", report.EffectiveLeaseTTL)
	_, _ = fmt.Fprintf(ctx.Stdout, "Health interval: %s\n", report.EffectiveHealthInterval)
	_, _ = fmt.Fprintf(ctx.Stdout, "Max health concurrency: %d\n", report.EffectiveMaxHealthConcurrency)
	_, _ = fmt.Fprintf(ctx.Stdout, "Batch size: %d\n", report.EffectiveBatchSize)
	_, _ = fmt.Fprintf(ctx.Stdout, "Recovery quiet period: %s\n", report.EffectiveRecoveryQuietPeriod)
	_, _ = fmt.Fprintf(ctx.Stdout, "Recovery cooldown: %s\n", report.EffectiveRecoveryCooldown)
	_, _ = fmt.Fprintf(ctx.Stdout, "Recovery concurrency: %d\n", report.EffectiveRecoveryConcurrency)
	if report.Status != scenarioruntime.SupervisorStatusRunning {
		_, _ = io.WriteString(ctx.Stdout, "Next steps:\n  vrooli runtime supervisor install --user\n")
		if hint := runtimesupervisor.ServiceStartHint(); hint != "" {
			_, _ = io.WriteString(ctx.Stdout, "  "+hint+"\n")
		}
		_, _ = io.WriteString(ctx.Stdout, "  vrooli runtime supervisor status\n")
	}
	return nil
}

func (app *App) installSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, HelpText)
		return nil
	}
	userService := true
	for _, arg := range args {
		if arg != "--user" {
			return rootcli.UsageErrorf("runtime supervisor install", "unknown option for runtime supervisor install: %s", arg)
		}
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	root, err := app.resolveRoot()
	if err != nil {
		return err
	}
	result, err := runtimesupervisor.InstallService(context.Background(), runtimesupervisor.ServiceInstallOptions{HomeDir: home, Executable: exe, SourceRoot: root, User: userService})
	if err != nil {
		return err
	}
	if err := cliinstall.RecordServiceInstall(home, cliinstall.ScopeRuntime, result.UnitPath, nativeServiceManager(), result.UnitName, result.Scope); err != nil {
		return fmt.Errorf("record runtime supervisor install: %w", err)
	}
	if ctx.Globals.JSON {
		return writeSupervisorServiceResultJSON(ctx.Stdout, result)
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "Installed runtime supervisor service: %s\n", result.UnitPath)
	_, _ = fmt.Fprintf(ctx.Stdout, "  Runs: %s\n", result.Executable)
	if result.LogPath != "" {
		_, _ = fmt.Fprintf(ctx.Stdout, "  Logs: %s\n", result.LogPath)
	}
	if !result.ExecutableIsCanonical {
		_, _ = fmt.Fprintln(ctx.Stdout, "  Warning: that is not the installed CLI. Run `make install` and re-run this command so the service is not pinned to a build output.")
	}
	return nil
}

func nativeServiceManager() string {
	switch runtime.GOOS {
	case "darwin":
		return "launchd"
	case "linux":
		return "systemd"
	default:
		return runtime.GOOS
	}
}

func (app *App) uninstallSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, HelpText)
		return nil
	}
	userService := true
	for _, arg := range args {
		if arg != "--user" {
			return rootcli.UsageErrorf("runtime supervisor uninstall", "unknown option for runtime supervisor uninstall: %s", arg)
		}
	}
	result, err := runtimesupervisor.UninstallService(context.Background(), runtimesupervisor.ServiceInstallOptions{User: userService})
	if err != nil {
		return err
	}
	if ctx.Globals.JSON {
		return writeSupervisorServiceResultJSON(ctx.Stdout, result)
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "Uninstalled runtime supervisor service: %s\n", result.UnitPath)
	return nil
}

func commandWantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
