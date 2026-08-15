package vroolicli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	packageapp "github.com/vrooli/vrooli/internal/app/package"
	projectapp "github.com/vrooli/vrooli/internal/app/project"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/bootstrap"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/authhandlers"
	"github.com/vrooli/vrooli/internal/cli/capacityhandlers"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/contracthandlers"
	"github.com/vrooli/vrooli/internal/cli/hygienehandlers"
	"github.com/vrooli/vrooli/internal/cli/metrics"
	"github.com/vrooli/vrooli/internal/cli/packagehandlers"
	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/cli/recoveryhandlers"
	"github.com/vrooli/vrooli/internal/cli/resourcehandlers"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/scenariohandlers"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/cliout"
	configpkg "github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/structureprovider"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type VersionInfo struct {
	CLIVersion      string
	PlatformVersion string
}

type Config struct {
	VersionInfo           VersionInfo
	ResolveSourceRootFn   func() (string, error)
	HomeDirFn             func() (string, error)
	CheckStalenessFn      func() (buildinfo.StaleCheck, error)
	RebuildAndReexecFn    func([]string) error
	LookPathFn            func(string) (string, error)
	NewLoggerFn           func(rootcli.GlobalOptions, io.Writer) (*slog.Logger, func())
	DebugLogFn            func(*slog.Logger, string, ...any)
	RunProjectBuildFn     func(string, string, io.Writer, io.Writer) error
	RunProjectSetupFn     func(string, string, projectsetup.Options, io.Writer, io.Writer) error
	RunProjectDevelopFn   func(string, string, projectsetup.Options, io.Writer, io.Writer) error
	EnsureScenarioCLIFn   func(string, string, string) error
	EnsureResourceCLIFn   func(string, string, string) error
	RunScenarioSubprocess func(scenarioexec.SubprocessSpec) error
	ScenarioExecutableFn  func() (string, error)
}

type App struct {
	VersionInfo           VersionInfo
	ResolveSourceRootFn   func() (string, error)
	HomeDirFn             func() (string, error)
	CheckStalenessFn      func() (buildinfo.StaleCheck, error)
	RebuildAndReexecFn    func([]string) error
	LookPathFn            func(string) (string, error)
	NewLoggerFn           func(rootcli.GlobalOptions, io.Writer) (*slog.Logger, func())
	DebugLogFn            func(*slog.Logger, string, ...any)
	RunProjectBuildFn     func(string, string, io.Writer, io.Writer) error
	RunProjectSetupFn     func(string, string, projectsetup.Options, io.Writer, io.Writer) error
	RunProjectDevelopFn   func(string, string, projectsetup.Options, io.Writer, io.Writer) error
	EnsureScenarioCLIFn   func(string, string, string) error
	EnsureResourceCLIFn   func(string, string, string) error
	RunScenarioSubprocess func(scenarioexec.SubprocessSpec) error
	ScenarioExecutableFn  func() (string, error)

	registry *rootcli.Registry[*CommandContext]

	scenarioCLINamesOnce  sync.Once
	scenarioCLINamesCache []string
}

type CommandContext struct {
	Root         string
	Globals      rootcli.GlobalOptions
	Stdout       io.Writer
	Stderr       io.Writer
	Logger       *slog.Logger
	app          *App
	home         string
	homeErr      error
	homeSeen     bool
	services     *bootstrap.Services
	servicesErr  error
	servicesSeen bool
}

type versionOutput struct {
	CLIVersion      string `json:"cli_version"`
	PlatformVersion string `json:"platform_version"`
	Root            string `json:"root"`
}

func New(config Config) *App {
	app := &App{
		VersionInfo:           config.VersionInfo,
		ResolveSourceRootFn:   config.ResolveSourceRootFn,
		HomeDirFn:             config.HomeDirFn,
		CheckStalenessFn:      config.CheckStalenessFn,
		RebuildAndReexecFn:    config.RebuildAndReexecFn,
		LookPathFn:            config.LookPathFn,
		NewLoggerFn:           config.NewLoggerFn,
		DebugLogFn:            config.DebugLogFn,
		RunProjectBuildFn:     config.RunProjectBuildFn,
		RunProjectSetupFn:     config.RunProjectSetupFn,
		RunProjectDevelopFn:   config.RunProjectDevelopFn,
		EnsureScenarioCLIFn:   config.EnsureScenarioCLIFn,
		EnsureResourceCLIFn:   config.EnsureResourceCLIFn,
		RunScenarioSubprocess: config.RunScenarioSubprocess,
		ScenarioExecutableFn:  config.ScenarioExecutableFn,
	}
	if app.HomeDirFn == nil {
		app.HomeDirFn = config.HomeDirFn
	}
	if app.HomeDirFn == nil {
		app.HomeDirFn = configpkg.HomeDir
	}
	if app.LookPathFn == nil {
		app.LookPathFn = exec.LookPath
	}
	if app.RunScenarioSubprocess == nil {
		app.RunScenarioSubprocess = scenarioexec.RunSubprocess
	}
	if app.ScenarioExecutableFn == nil {
		app.ScenarioExecutableFn = os.Executable
	}
	app.registry = rootcli.NewRegistry(app.buildTopLevelHandlerMap(), app.buildScenarioHandlerMap())
	return app
}

func (app *App) Registry() *rootcli.Registry[*CommandContext] {
	if app.registry == nil {
		app.registry = rootcli.NewRegistry(app.buildTopLevelHandlerMap(), app.buildScenarioHandlerMap())
	}
	return app.registry
}

func (app *App) Runner() *rootcli.Runner[*CommandContext] {
	return rootcli.NewRunner(rootcli.RunnerConfig[*CommandContext]{
		Registry:         app.Registry(),
		NewLogger:        app.NewLoggerFn,
		ResolveRoot:      app.resolveRoot,
		PrimeRootEnv:     primeRootEnv,
		ShouldRebuild:    app.shouldRebuild,
		RebuildAndReexec: app.RebuildAndReexecFn,
		NewContext: func(globals rootcli.GlobalOptions, stdout, stderr io.Writer, logger *slog.Logger) *CommandContext {
			return &CommandContext{
				Globals: globals,
				Stdout:  stdout,
				Stderr:  stderr,
				Logger:  logger,
				app:     app,
			}
		},
		SetRoot: func(ctx *CommandContext, root string) {
			ctx.Root = root
		},
		ShowMainHelp: func(ctx *CommandContext) {
			topcli.RenderMainHelp(ctx.Stdout, topcli.CommandSpecs())
		},
		ShowVersion: func(ctx *CommandContext) error {
			return WriteVersion(ctx.Stdout, ctx.Root, ctx.Globals, app.VersionInfo)
		},
		DebugLog:          app.DebugLogFn,
		MetricsRecorder:   app.newMetricsRecorder(),
		CLIVersion:        app.VersionInfo.CLIVersion,
		PlatformVersion:   app.VersionInfo.PlatformVersion,
		ScenarioCLILister: app.installedScenarioCLINames,
	})
}

// installedScenarioCLINames is consulted on the unknown-command path to
// detect invocations like `vrooli prompt-manager ...` so we can suggest
// running the scenario CLI directly. Result is cached for the process
// lifetime; failures are swallowed (returning nil falls through to the
// existing unknown-command rendering).
func (app *App) installedScenarioCLINames() []string {
	app.scenarioCLINamesOnce.Do(func() {
		if app.HomeDirFn == nil {
			return
		}
		home, err := app.HomeDirFn()
		if err != nil || strings.TrimSpace(home) == "" {
			return
		}
		manager, err := cliinstall.NewManager("", home)
		if err != nil {
			return
		}
		names, err := manager.InstalledScenarioCLINames()
		if err != nil {
			return
		}
		app.scenarioCLINamesCache = names
	})
	return app.scenarioCLINamesCache
}

func (app *App) newMetricsRecorder() *metrics.Recorder {
	home, err := app.HomeDirFn()
	if err != nil || strings.TrimSpace(home) == "" {
		return nil
	}
	return metrics.New(home, nil)
}

func (app *App) Run(args []string, stdout, stderr io.Writer) int {
	return app.Runner().Run(args, stdout, stderr)
}

func (app *App) NewCommandContext(root string, globals rootcli.GlobalOptions, stdout, stderr io.Writer) *CommandContext {
	return &CommandContext{
		Root:    root,
		Globals: globals,
		Stdout:  stdout,
		Stderr:  stderr,
		app:     app,
	}
}

func WriteVersion(w io.Writer, root string, globals rootcli.GlobalOptions, info VersionInfo) error {
	format, err := cliout.ParseFormat("", globals.JSON)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return writeCliVersionJSON(w, versionOutput{
			CLIVersion:      info.CLIVersion,
			PlatformVersion: info.PlatformVersion,
			Root:            root,
		})
	}
	_, _ = fmt.Fprintf(w, "Vrooli CLI v%s\n", info.CLIVersion)
	_, _ = fmt.Fprintf(w, "Vrooli Platform v%s\n", info.PlatformVersion)
	_, _ = fmt.Fprintf(w, "Root: %s\n", root)
	return nil
}

func (app *App) resolveRoot() (string, error) {
	root, err := app.ResolveSourceRootFn()
	if err != nil {
		return "", fmt.Errorf("resolve Vrooli root: %w", err)
	}
	return filepath.Clean(root), nil
}

func (app *App) shouldRebuild() (bool, error) {
	if app.CheckStalenessFn == nil {
		return false, nil
	}
	status, err := app.CheckStalenessFn()
	if err != nil {
		return false, err
	}
	return status.Stale, nil
}

func (ctx *CommandContext) HomeDir() (string, error) {
	if ctx.homeSeen {
		return ctx.home, ctx.homeErr
	}
	ctx.homeSeen = true
	ctx.home, ctx.homeErr = ctx.app.HomeDirFn()
	return ctx.home, ctx.homeErr
}

func (ctx *CommandContext) Services() (*bootstrap.Services, error) {
	if ctx.servicesSeen {
		return ctx.services, ctx.servicesErr
	}
	ctx.servicesSeen = true
	home, err := ctx.HomeDir()
	if err != nil {
		ctx.servicesErr = err
		return nil, err
	}
	stdout := ctx.Stdout
	if ctx.Globals.JSON {
		stdout = ctx.Stderr
	}
	ctx.services = bootstrap.New(ctx.Root, home, stdout, ctx.Stderr, ctx.Logger)
	return ctx.services, nil
}

func (app *App) CommandEnv(root string, globals rootcli.GlobalOptions) []string {
	env := os.Environ()
	env = setEnvValue(env, "VROOLI_ROOT", root)
	if strings.TrimSpace(os.Getenv(buildinfo.SourceRootEnvVar)) == "" {
		env = setEnvValue(env, buildinfo.SourceRootEnvVar, root)
	}
	if globals.NoColor {
		env = setEnvValue(env, "NO_COLOR", "1")
	}
	return env
}

func (app *App) newScenarioLifecycleRunner(ctx *CommandContext) (*lifecycle.Runner, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	runner, err := services.LifecycleRunner()
	if err != nil {
		return nil, err
	}
	return runner.WithVerbosity(verbosityFromGlobals(ctx.Globals)), nil
}

// verbosityFromGlobals translates CLI globals into the lifecycle package's
// own Verbosity enum. The two types are intentionally not the same — the
// lifecycle package does not import rootcli — so we map them here at the
// seam between the CLI shell and the scenario runner.
func verbosityFromGlobals(g rootcli.GlobalOptions) lifecycle.Verbosity {
	switch g.Output() {
	case rootcli.VerbosityQuiet:
		return lifecycle.VerbosityQuiet
	case rootcli.VerbosityVerbose:
		return lifecycle.VerbosityVerbose
	default:
		return lifecycle.VerbosityNormal
	}
}

func (app *App) newScenarioService(ctx *CommandContext) (*orchestrator.Service, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Orchestrator(), nil
}

func (app *App) NewScenarioService(ctx *CommandContext) (*orchestrator.Service, error) {
	return app.newScenarioService(ctx)
}

func (app *App) newResourceController(ctx *CommandContext) (*resources.Controller, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Resources(), nil
}

func (app *App) newProjectController(ctx *CommandContext) (*project.Controller, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Project(), nil
}

func (app *App) newMaintenanceController(ctx *CommandContext) (*maintenance.Controller, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Maintenance(), nil
}

func (app *App) newProjectCommandService(ctx *CommandContext) (projectapp.Service, error) {
	projectController, err := app.newProjectController(ctx)
	if err != nil {
		return projectapp.Service{}, err
	}
	maintenanceController, err := app.newMaintenanceController(ctx)
	if err != nil {
		return projectapp.Service{}, err
	}
	return projectapp.Service{
		Project:     projectController,
		Maintenance: maintenanceController,
	}, nil
}

func (app *App) runTopLevelSetup(ctx *CommandContext, opts projectsetup.Options) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	if ctx.Globals.Verbose {
		opts.Verbose = true
	}
	return app.RunProjectSetupFn(ctx.Root, home, opts, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelBuild(ctx *CommandContext) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.RunProjectBuildFn(ctx.Root, home, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelDevelop(ctx *CommandContext, opts projectsetup.Options) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	if ctx.Globals.Verbose {
		opts.Verbose = true
	}
	return app.RunProjectDevelopFn(ctx.Root, home, opts, ctx.Stdout, ctx.Stderr)
}

func (app *App) ensureScenarioCLI(ctx *CommandContext, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	// The scenario CLI and its source tree are variant-independent: they live at
	// scenarios/<scenario> regardless of which instance (live or a shadow) is
	// being started. Resolve by the bare scenario name so `--instance shadow`
	// (name "scenario@shadow") does not look for scenarios/scenario@shadow.
	if key, parseErr := scenarioruntime.ParseInstanceKey(name, ""); parseErr == nil {
		name = key.Scenario
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	manager, err := cliinstall.NewManager(ctx.Root, home)
	if err != nil {
		return err
	}
	preStatus, _ := manager.InspectScenarioCLIInstallLocation(name, app.LookPathFn)
	if app.EnsureScenarioCLIFn != nil {
		if err := app.EnsureScenarioCLIFn(ctx.Root, home, name); err != nil {
			return err
		}
	} else {
		if err := manager.EnsureScenarioCLI(name); err != nil {
			return err
		}
	}
	postStatus, inspectErr := manager.InspectScenarioCLIInstallLocation(name, app.LookPathFn)
	if inspectErr == nil {
		if warning := formatScenarioCLIInstallWarning(preStatus, postStatus); warning != "" {
			_, _ = fmt.Fprint(ctx.Stderr, warning)
		}
	}
	return nil
}

func (app *App) ensureResourceCLI(ctx *CommandContext, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	if app.EnsureResourceCLIFn != nil {
		return app.EnsureResourceCLIFn(ctx.Root, home, name)
	}
	manager, err := cliinstall.NewManager(ctx.Root, home)
	if err != nil {
		return err
	}
	return manager.EnsureResourceCLI(name)
}

func (app *App) locateTestGenieCLI(root, home string) (string, error) {
	return app.resolveScenarioCLIExecutable(root, home, "test-genie")
}

func (app *App) LocateTestGenieCLI(root, home string) (string, error) {
	return app.locateTestGenieCLI(root, home)
}

func (app *App) locateScenarioCompletenessCLI(root, home string) (string, error) {
	return app.resolveScenarioCLIExecutable(root, home, "scenario-completeness-scoring")
}

func (app *App) resolveScenarioCLIExecutable(root, home, name string) (string, error) {
	manager, err := cliinstall.NewManager(root, home)
	if err != nil {
		return "", err
	}
	item, err := manager.DiscoverScenarioCLI(name)
	if err != nil {
		return "", err
	}
	if app.EnsureScenarioCLIFn != nil {
		if err := app.EnsureScenarioCLIFn(root, home, name); err != nil {
			return "", err
		}
	} else if err := manager.EnsureScenarioCLI(name); err != nil {
		return "", err
	}
	return manager.InstalledBinaryPath(item), nil
}

func (app *App) openScenarioURL(url string) error {
	return scenarioexec.OpenURL(app.LookPathFn, app.RunScenarioSubprocess, url)
}

func (app *App) OpenScenarioURL(url string) error {
	return app.openScenarioURL(url)
}

func (app *App) launchDetachedScenario(root string, globals rootcli.GlobalOptions, args ...string) error {
	executable, err := app.ScenarioExecutableFn()
	if err != nil {
		return err
	}
	return scenarioexec.LaunchDetachedScenario(executable, root, globals, app.CommandEnv(root, globals), args...)
}

func (app *App) LaunchDetachedScenario(root string, globals rootcli.GlobalOptions, args ...string) error {
	return app.launchDetachedScenario(root, globals, args...)
}

func commandStdout(ctx *CommandContext) io.Writer {
	return ctx.Stdout
}

func formatScenarioCLIInstallWarning(before, after cliinstall.InstallLocationStatus) string {
	command := strings.TrimSpace(after.Command)
	if command == "" {
		command = strings.TrimSpace(before.Command)
	}
	if command == "" {
		return ""
	}

	var lines []string
	switch {
	case before.PathMismatch() && after.CanonicalExists && after.ResolvedCanonical:
		lines = append(lines,
			fmt.Sprintf("[WARN] %s previously resolved to a non-canonical CLI path.", command),
			fmt.Sprintf("  previous: %s", before.ResolvedPath),
			fmt.Sprintf("  canonical: %s", after.CanonicalPath),
			"  The canonical CLI install has been ensured.",
			"  If your current shell continues to invoke the previous path, refresh command lookup:",
			"    hash -r",
			"",
		)
	case after.PathMismatch():
		lines = append(lines,
			fmt.Sprintf("[WARN] %s resolves to a non-canonical CLI path.", command),
			fmt.Sprintf("  resolved: %s", after.ResolvedPath),
			fmt.Sprintf("  canonical: %s", after.CanonicalPath),
			"  The command on your PATH may not match the managed install.",
			"  Recommended fixes:",
			"    hash -r",
			"    ensure ~/.vrooli/bin appears before other install directories on PATH",
			"",
		)
	case after.CanonicalExists && !after.Resolved:
		lines = append(lines,
			fmt.Sprintf("[WARN] %s is installed canonically but is not currently resolvable on PATH.", command),
			fmt.Sprintf("  canonical: %s", after.CanonicalPath),
			"  Add ~/.vrooli/bin to PATH or invoke the canonical binary directly.",
			"",
		)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func projectOutputFormat(ctx *CommandContext) (cliout.Format, error) {
	return cliout.ParseFormat("", ctx.Globals.JSON)
}

func runProjectPhaseFromContext(ctx *CommandContext, phase string, args []string) error {
	controller, err := ctx.app.newProjectController(ctx)
	if err != nil {
		return err
	}
	return controller.RunProjectPhase(phase, args)
}

func (app *App) runLifecycleProtectCommand(ctx *CommandContext, args []string) error {
	commandArgs, err := projectcli.ParseLifecycleProtectArgs(args)
	if err != nil {
		return err
	}
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		return rootcli.ExitCodeError{Code: 1, Message: projectcli.LifecycleProtectErrorMessage()}
	}

	if err := app.RunScenarioSubprocess(scenarioexec.SubprocessSpec{
		Name:   commandArgs[0],
		Args:   commandArgs[1:],
		Dir:    ".",
		Env:    os.Environ(),
		Stdin:  os.Stdin,
		Stdout: ctx.Stdout,
		Stderr: ctx.Stderr,
	}); err != nil {
		var exitErr *exec.ExitError
		if ok := errorAs(err, &exitErr); ok {
			return rootcli.ExitCodeError{Code: exitErr.ExitCode(), Silent_: true}
		}
		return err
	}
	return nil
}

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

func (app *App) runRuntimeCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, runtimeHelpText)
		return nil
	}
	if args[0] == "recovery" {
		return app.runRuntimeRecovery(ctx, args[1:])
	}
	if args[0] != "supervisor" {
		return rootcli.UsageErrorf("runtime", "unknown runtime command: %s", args[0])
	}
	if len(args) == 1 {
		_, _ = io.WriteString(ctx.Stdout, runtimeHelpText)
		return nil
	}
	switch args[1] {
	case "run":
		return app.runRuntimeSupervisor(ctx, args[2:])
	case "status":
		return app.statusRuntimeSupervisor(ctx, args[2:])
	case "install":
		return app.installRuntimeSupervisor(ctx, args[2:])
	case "uninstall":
		return app.uninstallRuntimeSupervisor(ctx, args[2:])
	default:
		return rootcli.UsageErrorf("runtime supervisor", "unknown runtime supervisor command: %s", args[1])
	}
}

func (app *App) runRuntimeSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, runtimeHelpText)
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
	cfg.Version = ctx.app.VersionInfo.CLIVersion
	return runtimesupervisor.Run(context.Background(), cfg)
}

func (app *App) statusRuntimeSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, runtimeHelpText)
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
		return writeCliSupervisorStatusJSON(ctx.Stdout, report)
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
		_, _ = io.WriteString(ctx.Stdout, "Next steps:\n")
		_, _ = io.WriteString(ctx.Stdout, "  vrooli runtime supervisor install --user\n")
		if hint := runtimesupervisor.ServiceStartHint(); hint != "" {
			_, _ = io.WriteString(ctx.Stdout, "  "+hint+"\n")
		}
		_, _ = io.WriteString(ctx.Stdout, "  vrooli runtime supervisor status\n")
	}
	return nil
}

func (app *App) installRuntimeSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, runtimeHelpText)
		return nil
	}
	userService := true
	for _, arg := range args {
		switch arg {
		case "--user":
			userService = true
		default:
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
	result, err := runtimesupervisor.InstallService(context.Background(), runtimesupervisor.ServiceInstallOptions{
		HomeDir:    home,
		Executable: exe,
		SourceRoot: root,
		User:       userService,
	})
	if err != nil {
		return err
	}
	if ctx.Globals.JSON {
		return writeCliSupervisorServiceResultJSON(ctx.Stdout, result)
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "Installed runtime supervisor service: %s\n", result.UnitPath)
	return nil
}

func (app *App) uninstallRuntimeSupervisor(ctx *CommandContext, args []string) error {
	if commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, runtimeHelpText)
		return nil
	}
	userService := true
	for _, arg := range args {
		switch arg {
		case "--user":
			userService = true
		default:
			return rootcli.UsageErrorf("runtime supervisor uninstall", "unknown option for runtime supervisor uninstall: %s", arg)
		}
	}
	result, err := runtimesupervisor.UninstallService(context.Background(), runtimesupervisor.ServiceInstallOptions{User: userService})
	if err != nil {
		return err
	}
	if ctx.Globals.JSON {
		return writeCliSupervisorServiceResultJSON(ctx.Stdout, result)
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

func (app *App) buildTopLevelHandlerMap() map[topcli.CommandID]rootcli.Handler[*CommandContext] {
	handlers := map[topcli.CommandID]rootcli.Handler[*CommandContext]{
		topcli.CommandSetup: projectcli.SetupHandler(commandStdout, func(ctx *CommandContext, opts projectsetup.Options) error { return ctx.app.runTopLevelSetup(ctx, opts) }),
		topcli.CommandDevelop: projectcli.DevelopHandler(commandStdout, func(ctx *CommandContext, opts projectsetup.Options) error {
			return ctx.app.runTopLevelDevelop(ctx, opts)
		}),
		topcli.CommandBuild: projectcli.BuildHandler(commandStdout, func(ctx *CommandContext) error { return ctx.app.runTopLevelBuild(ctx) }),
		topcli.CommandClean: projectcli.ProjectPhaseHandler(commandStdout, "clean", func(ctx *CommandContext, args []string) error { return runProjectPhaseFromContext(ctx, "clean", args) }),
		topcli.CommandStatus: projectcli.StatusHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext, req projectcli.StatusRequest) (project.StatusReport, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return project.StatusReport{}, err
			}
			return command.Status(projectapp.StatusRequest{ResourcesOnly: req.ResourcesOnly, ScenariosOnly: req.ScenariosOnly, Fast: req.Fast})
		}),
		topcli.CommandStop: projectcli.StopHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext, req projectcli.StopRequest) (control.StopReport, error) {
			if err := enforceAgentCommandPolicy("stop", req.Targets); err != nil {
				return control.StopReport{}, err
			}
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return control.StopReport{}, err
			}
			return command.Stop(projectapp.StopRequest{Targets: req.Targets})
		}),
		topcli.CommandBackup: projectcli.ProjectPhaseHandler(commandStdout, "backup", func(ctx *CommandContext, args []string) error { return runProjectPhaseFromContext(ctx, "backup", args) }),
		topcli.CommandRestore: projectcli.ProjectPhaseHandler(commandStdout, "restore", func(ctx *CommandContext, args []string) error {
			return runProjectPhaseFromContext(ctx, "restore", args)
		}),
		topcli.CommandScenario: func(ctx *CommandContext, args []string) error {
			return scenariohandlers.RootHandler(commandStdout, ctx.app.Registry().ScenarioHandler, ctx.app.Registry().SuggestScenario)(ctx, args)
		},
		topcli.CommandPackage: packagehandlers.RootHandler(packagehandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			Stderr:       func(ctx *CommandContext) io.Writer { return ctx.Stderr },
			Root:         func(ctx *CommandContext) string { return ctx.Root },
			OutputFormat: projectOutputFormat,
			ScenarioOperations: func(ctx *CommandContext) (packageapp.ScenarioRuntime, error) {
				return ctx.app.newScenarioService(ctx)
			},
			LifecycleRunner: func(ctx *CommandContext) (packageapp.ScenarioPhaseRunner, error) {
				return ctx.app.newScenarioLifecycleRunner(ctx)
			},
		}),
		topcli.CommandResource: resourcehandlers.RootHandler(resourcehandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			Stderr:       func(ctx *CommandContext) io.Writer { return ctx.Stderr },
			Globals:      func(ctx *CommandContext) rootcli.GlobalOptions { return ctx.Globals },
			OutputFormat: projectOutputFormat,
			EnsureCLI: func(ctx *CommandContext, name string) error {
				return ctx.app.ensureResourceCLI(ctx, name)
			},
			ResourceController: func(ctx *CommandContext) (*resources.Controller, error) {
				return ctx.app.newResourceController(ctx)
			},
		}),
		topcli.CommandRuntime: func(ctx *CommandContext, args []string) error {
			return ctx.app.runRuntimeCommand(ctx, args)
		},
		topcli.CommandDoctor: projectcli.DoctorHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext) (project.DoctorReport, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return project.DoctorReport{}, err
			}
			return command.Doctor()
		}),
		topcli.CommandOrphans: projectcli.OrphansHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext, req projectcli.OrphansRequest) (projectcli.OrphansResponse, error) {
			policyArgs := []string{"orphans"}
			if req.Kill {
				policyArgs = append(policyArgs, "kill")
			}
			if req.DryRun {
				policyArgs = append(policyArgs, "--dry-run")
			}
			if err := enforceAgentCommandPolicy(policyArgs[0], policyArgs[1:]); err != nil {
				return projectcli.OrphansResponse{}, err
			}
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return projectcli.OrphansResponse{}, err
			}
			resp, err := command.Orphans(projectapp.OrphansRequest{Kill: req.Kill, DryRun: req.DryRun})
			if err != nil {
				return projectcli.OrphansResponse{}, err
			}
			return projectcli.OrphansResponse{List: resp.List, KillReport: resp.KillReport, DryRun: resp.DryRun}, nil
		}),
		topcli.CommandLocks: projectcli.LocksHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext, req projectcli.LocksRequest) (projectcli.LocksResponse, error) {
			policyArgs := []string{"locks"}
			if req.Clean {
				policyArgs = append(policyArgs, "clean")
			}
			if err := enforceAgentCommandPolicy(policyArgs[0], policyArgs[1:]); err != nil {
				return projectcli.LocksResponse{}, err
			}
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return projectcli.LocksResponse{}, err
			}
			resp, err := command.Locks(projectapp.LocksRequest{Clean: req.Clean})
			if err != nil {
				return projectcli.LocksResponse{}, err
			}
			return projectcli.LocksResponse{RuntimeClaims: resp.RuntimeClaims, CleanReport: resp.CleanReport, ShowAll: req.ShowAll}, nil
		}),
		topcli.CommandDiagnosePort: projectcli.DiagnosePortHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext, req projectcli.DiagnosePortRequest) (maintenance.PortDiagnostic, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return maintenance.PortDiagnostic{}, err
			}
			return command.DiagnosePort(projectapp.DiagnosePortRequest{Port: req.Port, ScenarioName: req.ScenarioName})
		}),
		topcli.CommandContract: contracthandlers.RootHandler(contracthandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			OutputFormat: projectOutputFormat,
			Service: func(ctx *CommandContext) contractapp.Service {
				return contractapp.NewDefaultService()
			},
			Validate: func(*CommandContext) (contractapp.ValidationOutput, error) {
				root, err := contractapp.ResolveRoot()
				if err != nil {
					return contractapp.ValidationOutput{}, err
				}
				return structureprovider.NewDefault().Validate(context.Background(), root)
			},
		}),
		topcli.CommandHygiene: hygienehandlers.Handler(hygienehandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			Root:         func(ctx *CommandContext) string { return ctx.Root },
			Home:         func(ctx *CommandContext) (string, error) { return ctx.HomeDir() },
			OutputFormat: projectOutputFormat,
		}),
		topcli.CommandAuth: authhandlers.RootHandler(authhandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			OutputFormat: projectOutputFormat,
		}),
		topcli.CommandRecovery: recoveryhandlers.RootHandler(recoveryhandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			Root:         func(ctx *CommandContext) string { return ctx.Root },
			OutputFormat: projectOutputFormat,
		}),
		topcli.CommandHost: func(ctx *CommandContext, args []string) error { return ctx.app.runHostCommand(ctx, args) },
		topcli.CommandCapacity: capacityhandlers.RootHandler(capacityhandlers.HandlerDeps[*CommandContext]{
			Stdout:       commandStdout,
			OutputFormat: projectOutputFormat,
		}),
		topcli.CommandCapability:       func(ctx *CommandContext, args []string) error { return app.runCapabilityCommand(ctx, args) },
		topcli.CommandCredentials:      func(ctx *CommandContext, args []string) error { return ctx.app.runCredentialsCommand(ctx, args) },
		topcli.CommandReleaseAuthority: func(ctx *CommandContext, args []string) error { return ctx.app.runReleaseAuthorityCommand(ctx, args) },
		topcli.CommandLifecycle:        projectcli.LifecycleHandler(commandStdout, func(ctx *CommandContext, args []string) error { return ctx.app.runLifecycleProtectCommand(ctx, args) }),
	}
	templateValidationCleanupHandler := projectcli.TemplateValidationCleanupHandler(commandStdout, projectOutputFormat, func(ctx *CommandContext, req projectcli.TemplateValidationCleanupRequest) (projectcli.TemplateValidationCleanupResponse, error) {
		command, err := ctx.app.newProjectCommandService(ctx)
		if err != nil {
			return projectcli.TemplateValidationCleanupResponse{}, err
		}
		opts, err := projectTemplateValidationCleanupOptions(req)
		if err != nil {
			return projectcli.TemplateValidationCleanupResponse{}, err
		}
		result, err := command.TemplateValidationCleanup(opts)
		if err != nil {
			return projectcli.TemplateValidationCleanupResponse{}, err
		}
		return projectcli.TemplateValidationCleanupResponse{Result: result}, nil
	})
	handlers[topcli.CommandCleanup] = projectcli.CleanupHandler(commandStdout, handlers[topcli.CommandOrphans], handlers[topcli.CommandLocks], templateValidationCleanupHandler)
	return handlers
}

func (app *App) runHostCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandWantsHelp(args) {
		commandtree.RenderHelp(ctx.Stdout, commandtree.Help{
			Title:        "Vrooli Host",
			Description:  "Inspect local host facts through internal/hostinventory.",
			Usage:        "vrooli host <command> [options]",
			DefaultGroup: "Host Commands",
		}, []commandtree.Spec[string]{
			hostInventorySpec(),
			hostInstallSpec(),
			hostSafeguardSpec(),
		})
		return nil
	}
	switch args[0] {
	case "inventory":
		return app.runHostInventoryCommand(ctx, args[1:])
	case "install":
		return app.runHostInstallCommand(ctx, args[1:])
	case "safeguard":
		return app.runHostSafeguardCommand(ctx, args[1:])
	default:
		return rootcli.NewUnknownCommandError(args[0], []string{"inventory", "install", "safeguard"})
	}
}

func hostInventorySpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "inventory",
		Summary: "Collect local host inventory facts",
		Help: commandtree.Help{
			Description: "Collects CPU, memory, GPU, and Docker GPU-runtime facts using the shared Go host inventory package.",
			Usage:       "vrooli host inventory [--json] [--field <name>]",
			Options: []commandtree.OptionArg{
				commandtree.JSONOption(),
				{Name: "--field", ValueName: "name", Description: "Print one shell-friendly field"},
			},
			Examples: []string{
				"vrooli host inventory --json",
				"vrooli host inventory --field has_nvidia_gpu",
				"vrooli host inventory --field memory_total_mb",
			},
		},
		Args: commandtree.ArgSchema{
			Options: []commandtree.OptionArg{
				commandtree.JSONOption(),
				{Name: "--field", ValueName: "name", Description: "Print one shell-friendly field"},
			},
		},
		Handler: "inventory",
	}
}

func (app *App) runHostInventoryCommand(ctx *CommandContext, args []string) error {
	spec := hostInventorySpec()
	parsed, err := commandtree.ParseArgs("host inventory", commandtree.SpecHelpText("", "vrooli host inventory", spec), spec.Args, args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("host inventory", "%s", err.Error())
	}

	snapshot, err := hostinventory.Collect(context.Background())
	if err != nil {
		return err
	}
	if field := strings.TrimSpace(parsed.FlagValue("--field")); field != "" {
		value, err := hostInventoryField(snapshot, field)
		if err != nil {
			return rootcli.UsageErrorf("host inventory", "%s", err.Error())
		}
		_, _ = fmt.Fprintln(ctx.Stdout, value)
		return nil
	}
	if ctx.Globals.JSON || parsed.HasFlag("--json") {
		return writeHostSnapshotJSON(ctx.Stdout, snapshot)
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "OS: %s/%s\n", snapshot.OS, snapshot.Arch)
	_, _ = fmt.Fprintf(ctx.Stdout, "CPU cores: %d\n", snapshot.CPU.Cores)
	_, _ = fmt.Fprintf(ctx.Stdout, "Memory total: %d MB\n", snapshot.Memory.TotalBytes/1024/1024)
	_, _ = fmt.Fprintf(ctx.Stdout, "NVIDIA GPU: %s\n", cliout.BoolLabel(snapshot.HasNvidiaGPU()))
	_, _ = fmt.Fprintf(ctx.Stdout, "Docker NVIDIA runtime: %s\n", cliout.BoolLabel(snapshot.HasDockerNvidiaRuntime()))
	if len(snapshot.GPUs) > 0 {
		_, _ = fmt.Fprintln(ctx.Stdout, "GPUs:")
		for _, gpu := range snapshot.GPUs {
			_, _ = fmt.Fprintf(ctx.Stdout, "- %s (%d MB, source=%s)\n", gpu.Name, gpu.VRAMBytes/1024/1024, gpu.Source)
		}
	}
	if len(snapshot.Warnings) > 0 {
		_, _ = fmt.Fprintln(ctx.Stdout, "Warnings:")
		for _, warning := range snapshot.Warnings {
			_, _ = fmt.Fprintf(ctx.Stdout, "- %s\n", warning)
		}
	}
	return nil
}

func hostInventoryField(snapshot hostinventory.Snapshot, field string) (string, error) {
	switch field {
	case "has_nvidia_gpu":
		return fmt.Sprintf("%t", snapshot.HasNvidiaGPU()), nil
	case "has_docker_nvidia_runtime":
		return fmt.Sprintf("%t", snapshot.HasDockerNvidiaRuntime()), nil
	case "has_docker_addressable_nvidia_gpu":
		return fmt.Sprintf("%t", snapshot.HasDockerAddressableNvidiaGPU()), nil
	case "gpu_count":
		return fmt.Sprintf("%d", len(snapshot.GPUs)), nil
	case "first_gpu_summary":
		for _, gpu := range snapshot.GPUs {
			if gpu.Name == "" {
				continue
			}
			return fmt.Sprintf("%s,%d,%d", gpu.Name, gpu.VRAMUsedBytes/1024/1024, gpu.VRAMBytes/1024/1024), nil
		}
		return "", nil
	case "cpu_cores":
		return fmt.Sprintf("%d", snapshot.CPU.Cores), nil
	case "memory_total_mb":
		return fmt.Sprintf("%d", snapshot.Memory.TotalBytes/1024/1024), nil
	case "memory_available_mb":
		return fmt.Sprintf("%d", snapshot.Memory.AvailableBytes/1024/1024), nil
	default:
		return "", fmt.Errorf("unknown host inventory field %q", field)
	}
}

func enforceAgentCommandPolicy(command string, args []string) error {
	argv := append([]string{"vrooli", command}, args...)
	return clipolicy.NewCommandPolicyError(clipolicy.ClassifyAgentCommand(argv, os.Environ()))
}

func projectTemplateValidationCleanupOptions(req projectcli.TemplateValidationCleanupRequest) (templatevalidation.CleanupOptions, error) {
	olderThan := templatevalidation.DefaultCleanupOlderThan
	if strings.TrimSpace(req.OlderThan) != "" {
		parsed, err := time.ParseDuration(req.OlderThan)
		if err != nil {
			return templatevalidation.CleanupOptions{}, fmt.Errorf("invalid --older-than duration %q: %w", req.OlderThan, err)
		}
		olderThan = parsed
	}
	return templatevalidation.CleanupOptions{
		OlderThan:       olderThan,
		IncludeRetained: req.IncludeRetained,
		RunID:           req.RunID,
		DryRun:          req.DryRun,
	}, nil
}

func (app *App) buildScenarioHandlerMap() map[scenariocli.CommandID]rootcli.Handler[*CommandContext] {
	return scenariohandlers.BuildHandlers(scenariohandlers.HandlerDeps[*CommandContext]{
		Stdout:       commandStdout,
		Stderr:       func(ctx *CommandContext) io.Writer { return ctx.Stderr },
		Root:         func(ctx *CommandContext) string { return ctx.Root },
		Globals:      func(ctx *CommandContext) rootcli.GlobalOptions { return ctx.Globals },
		OutputFormat: projectOutputFormat,
		HomeDir:      func(ctx *CommandContext) (string, error) { return ctx.HomeDir() },
		EnsureCLI: func(ctx *CommandContext, name string) error {
			return ctx.app.ensureScenarioCLI(ctx, name)
		},
		ScenarioOperations: func(ctx *CommandContext) (scenarioapp.ScenarioOperations, error) {
			return ctx.app.newScenarioService(ctx)
		},
		LifecycleRunner: func(ctx *CommandContext) (scenarioapp.PhaseRunner, error) {
			return ctx.app.newScenarioLifecycleRunner(ctx)
		},
		EnvValidator: func(ctx *CommandContext) (scenarioapp.EnvironmentValidator, error) {
			services, err := ctx.Services()
			if err != nil {
				return nil, err
			}
			return services.Resources(), nil
		},
		OpenURL: func(ctx *CommandContext, url string) error {
			return ctx.app.openScenarioURL(url)
		},
		LaunchDetached: func(ctx *CommandContext, args ...string) error {
			return ctx.app.launchDetachedScenario(ctx.Root, ctx.Globals, args...)
		},
		RunSubprocess: func(ctx *CommandContext, spec scenarioexec.SubprocessSpec) error {
			return ctx.app.RunScenarioSubprocess(spec)
		},
		LocateTestGenieCLI: func(ctx *CommandContext) (string, error) {
			home, err := ctx.HomeDir()
			if err != nil {
				return "", err
			}
			return ctx.app.locateTestGenieCLI(ctx.Root, home)
		},
		LocateBusinessHealthCLI: func(ctx *CommandContext) (string, error) {
			home, err := ctx.HomeDir()
			if err != nil {
				return "", err
			}
			return ctx.app.resolveScenarioCLIExecutable(ctx.Root, home, "business-health")
		},
		LocateCompleteCLI: func(ctx *CommandContext) (string, error) {
			home, err := ctx.HomeDir()
			if err != nil {
				return "", err
			}
			return ctx.app.locateScenarioCompletenessCLI(ctx.Root, home)
		},
		CommandEnv: func(ctx *CommandContext) []string {
			return ctx.app.CommandEnv(ctx.Root, ctx.Globals)
		},
	})
}

func primeRootEnv(root string) {
	_ = os.Setenv("VROOLI_ROOT", root)
	if strings.TrimSpace(os.Getenv(buildinfo.SourceRootEnvVar)) == "" {
		_ = os.Setenv(buildinfo.SourceRootEnvVar, root)
	}
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}

func errorAs(err error, target any) bool { return errors.As(err, target) }
