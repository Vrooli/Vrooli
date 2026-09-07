package runtimeapp

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	runtimesupervisorsafeguard "github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type SupervisorRunOptions struct{ Takeover bool }

type SupervisorStatusOptions struct{ JSON bool }

type SupervisorServiceOptions struct {
	User bool
	JSON bool
}

// RunSupervisor starts the runtime supervisor with typed options.
func (app *Service) RunSupervisor(operationCtx context.Context, home string, opts SupervisorRunOptions) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	cfg := runtimesupervisor.EnvConfig()
	cfg.HomeDir = home
	cfg.Version = app.Version
	cfg.BuildIdentity = buildinfo.Fingerprint
	cfg.Takeover = opts.Takeover
	return runtimesupervisor.Run(operationCtx, cfg)
}

// SupervisorStatus returns and renders the current supervisor status.
func (app *Service) SupervisorStatus(operationCtx context.Context, home string, out io.Writer, opts SupervisorStatusOptions) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	jsonOutput := opts.JSON
	cfg := runtimesupervisor.EnvConfig()
	cfg.HomeDir = home
	cfg.Version = app.Version
	cfg.BuildIdentity = buildinfo.Fingerprint
	svc := runtimesupervisor.New(cfg)
	defer svc.Close()
	report, err := svc.Status(operationCtx)
	if err != nil {
		return err
	}
	if jsonOutput {
		return cliout.WriteProtoJSON(out, supervisorStatusMessage(report))
	}
	_, _ = fmt.Fprintf(out, "Runtime supervisor: %s\n", report.Status)
	if report.StatusReason != "" {
		_, _ = fmt.Fprintf(out, "Reason: %s\n", report.StatusReason)
	}
	if report.SupervisorID != "" {
		_, _ = fmt.Fprintf(out, "Supervisor ID: %s\n", report.SupervisorID)
		_, _ = fmt.Fprintf(out, "Host boot/session: %s / %s\n", report.HostBootID, report.HostSessionID)
		_, _ = fmt.Fprintf(out, "Heartbeat: %s -> %s\n", report.LastHeartbeatAt.Format(time.RFC3339), report.HeartbeatDeadlineAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintf(out, "Supervised running instances: %d\n", report.SupervisedInstanceCount)
	_, _ = fmt.Fprintf(out, "Unverified running instances: %d\n", report.UnverifiedInstanceCount)
	_, _ = fmt.Fprintf(out, "Renew interval: %s\n", report.EffectiveRenewInterval)
	_, _ = fmt.Fprintf(out, "Lease TTL: %s\n", report.EffectiveLeaseTTL)
	_, _ = fmt.Fprintf(out, "Health interval: %s\n", report.EffectiveHealthInterval)
	_, _ = fmt.Fprintf(out, "Max health concurrency: %d\n", report.EffectiveMaxHealthConcurrency)
	_, _ = fmt.Fprintf(out, "Batch size: %d\n", report.EffectiveBatchSize)
	_, _ = fmt.Fprintf(out, "Recovery quiet period: %s\n", report.EffectiveRecoveryQuietPeriod)
	_, _ = fmt.Fprintf(out, "Recovery cooldown: %s\n", report.EffectiveRecoveryCooldown)
	_, _ = fmt.Fprintf(out, "Recovery concurrency: %d\n", report.EffectiveRecoveryConcurrency)
	if report.Status != scenarioruntime.SupervisorStatusRunning {
		_, _ = io.WriteString(out, "Next steps:\n  vrooli runtime supervisor install --user\n")
		if hint := runtimesupervisor.ServiceStartHint(); hint != "" {
			_, _ = io.WriteString(out, "  "+hint+"\n")
		}
		_, _ = io.WriteString(out, "  vrooli runtime supervisor status\n")
	}
	return nil
}

// InstallSupervisor installs the runtime supervisor service.
func (app *Service) InstallSupervisor(operationCtx context.Context, home string, out io.Writer, opts SupervisorServiceOptions) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	root, err := app.resolveRoot()
	if err != nil {
		return err
	}
	result, err := installSupervisorService(operationCtx, home, exe, root, opts.User)
	if err != nil {
		return err
	}
	if err := cliinstall.RecordServiceInstall(home, cliinstall.ScopeRuntime, result.UnitPath, nativeServiceManager(), result.UnitName, result.Scope); err != nil {
		return fmt.Errorf("record runtime supervisor install: %w", err)
	}
	if opts.JSON {
		return cliout.WriteProtoJSON(out, supervisorServiceResultMessage(result))
	}
	_, _ = fmt.Fprintf(out, "Installed runtime supervisor service: %s\n", result.UnitPath)
	_, _ = fmt.Fprintf(out, "  Runs: %s\n", result.Executable)
	if result.LogPath != "" {
		_, _ = fmt.Fprintf(out, "  Logs: %s\n", result.LogPath)
	}
	if !result.ExecutableIsCanonical {
		_, _ = fmt.Fprintln(out, "  Warning: that is not the installed CLI. Run `make install` and re-run this command so the service is not pinned to a build output.")
	}
	return nil
}

func nativeServiceManager() string {
	switch runtime.GOOS {
	case string(hostreqspec.PlatformDarwin):
		return "launchd"
	case string(hostreqspec.PlatformLinux):
		return "systemd"
	default:
		return runtime.GOOS
	}
}

// UninstallSupervisor removes the runtime supervisor service.
func (app *Service) UninstallSupervisor(operationCtx context.Context, out io.Writer, opts SupervisorServiceOptions) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	result, err := runtimesupervisor.UninstallService(operationCtx, runtimesupervisor.ServiceInstallOptions{User: opts.User})
	if err != nil {
		return err
	}
	if opts.JSON {
		return cliout.WriteProtoJSON(out, supervisorServiceResultMessage(result))
	}
	_, _ = fmt.Fprintf(out, "Uninstalled runtime supervisor service: %s\n", result.UnitPath)
	return nil
}

// installSupervisorService is the CLI's install path. A user install goes
// through the runtime_supervisor safeguard's Converge, the same code `vrooli
// setup` runs and the readiness phase re-inspects, so the CLI cannot render a
// unit setup would later call stale. System-scope installs keep the platform
// path, which refuses them until broker support exists.
func installSupervisorService(ctx context.Context, home, exe, root string, userService bool) (runtimesupervisor.ServiceInstallResult, error) {
	if !userService {
		return runtimesupervisor.InstallService(ctx, runtimesupervisor.ServiceInstallOptions{HomeDir: home, Executable: exe, SourceRoot: root, User: false})
	}
	converged, err := runtimesupervisorsafeguard.Converge(ctx, runtimesupervisorsafeguard.ConvergeOptions{OS: runtime.GOOS, Home: home, Root: root, Executable: exe})
	rendered := converged.Rendered
	result := runtimesupervisor.ServiceInstallResult{
		UnitName:              rendered.UnitName,
		UnitPath:              rendered.Path,
		Scope:                 "user",
		Active:                converged.Active,
		LogPath:               rendered.LogPath,
		Executable:            rendered.Executable,
		ExecutableIsCanonical: rendered.Canonical,
	}
	return result, err
}
