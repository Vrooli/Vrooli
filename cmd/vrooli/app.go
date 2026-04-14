package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	projectapp "github.com/vrooli/vrooli/internal/app/project"
	"github.com/vrooli/vrooli/internal/bootstrap"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

type scenarioSubprocessSpec struct {
	name   string
	args   []string
	dir    string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type App struct {
	resolveSourceRoot     func() (string, error)
	homeDir               func() (string, error)
	checkStaleness        func() (buildinfo.StaleCheck, error)
	rebuildAndReexec      func([]string) error
	lookPath              func(string) (string, error)
	newLogger             func(globalOptions, io.Writer) (*slog.Logger, func())
	runProjectBuild       func(string, string, io.Writer, io.Writer) error
	runProjectSetup       func(string, string, projectsetup.Options, io.Writer, io.Writer) error
	runProjectDevelop     func(string, string, projectsetup.Options, io.Writer, io.Writer) error
	runScenarioSubprocess func(scenarioSubprocessSpec) error
	scenarioExecutable    func() (string, error)
	registry              *rootcli.Registry[*commandContext]
}

type commandContext struct {
	Root         string
	Globals      globalOptions
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

func configuredApp() *App {
	return &App{
		resolveSourceRoot: resolveSourceRootFn,
		homeDir:           config.HomeDir,
		checkStaleness:    checkStalenessFn,
		rebuildAndReexec:  rebuildAndReexecFn,
		lookPath:          lookPathFn,
		newLogger:         newLoggerFn,
		runProjectBuild:   projectsetup.RunBuild,
		runProjectSetup:   projectsetup.RunSetupWithOptions,
		runProjectDevelop: projectsetup.RunDevelopWithOptions,
		runScenarioSubprocess: func(spec scenarioSubprocessSpec) error {
			return scenarioexec.RunSubprocess(scenarioexec.SubprocessSpec{
				Name:   spec.name,
				Args:   spec.args,
				Dir:    spec.dir,
				Env:    spec.env,
				Stdin:  spec.stdin,
				Stdout: spec.stdout,
				Stderr: spec.stderr,
			})
		},
		scenarioExecutable: scenarioExecutableFn,
	}
}

func newConfiguredCommandContext(root string, globals globalOptions, stdout, stderr io.Writer) (*App, *commandContext) {
	app := configuredApp()
	app.registry = rootcli.NewRegistry(buildTopLevelHandlerMap(), buildScenarioHandlerMap())
	return app, &commandContext{
		Root:    root,
		Globals: globals,
		Stdout:  stdout,
		Stderr:  stderr,
		app:     app,
	}
}

func configuredRunner() *rootcli.Runner[*commandContext] {
	app := configuredApp()
	return app.runner()
}

func (app *App) runner() *rootcli.Runner[*commandContext] {
	if app.registry == nil {
		app.registry = rootcli.NewRegistry(buildTopLevelHandlerMap(), buildScenarioHandlerMap())
	}
	return rootcli.NewRunner(rootcli.RunnerConfig[*commandContext]{
		Registry:         app.registry,
		NewLogger:        app.newLogger,
		ResolveRoot:      app.resolveRoot,
		PrimeRootEnv:     primeRootEnv,
		ShouldRebuild:    app.shouldRebuild,
		RebuildAndReexec: app.rebuildAndReexec,
		NewContext: func(globals globalOptions, stdout, stderr io.Writer, logger *slog.Logger) *commandContext {
			return &commandContext{
				Globals: globals,
				Stdout:  stdout,
				Stderr:  stderr,
				Logger:  logger,
				app:     app,
			}
		},
		SetRoot: func(ctx *commandContext, root string) {
			ctx.Root = root
		},
		ShowMainHelp: func(ctx *commandContext) {
			topcli.RenderMainHelp(ctx.Stdout, topcli.CommandSpecs())
		},
		ShowVersion: func(ctx *commandContext) error {
			return showVersion(ctx.Stdout, ctx.Root, ctx.Globals)
		},
		DebugLog: debugLog,
	})
}

func (app *App) Run(args []string, stdout, stderr io.Writer) int {
	return app.runner().Run(args, stdout, stderr)
}

func (app *App) shouldRebuild() (bool, error) {
	if app.checkStaleness != nil {
		status, err := app.checkStaleness()
		if err != nil {
			return false, err
		}
		return status.Stale, nil
	}
	return false, nil
}

func (app *App) resolveRoot() (string, error) {
	root, err := app.resolveSourceRoot()
	if err != nil {
		return "", fmt.Errorf("resolve Vrooli root: %w", err)
	}
	return filepath.Clean(root), nil
}

func (app *App) commandEnv(root string, globals globalOptions) []string {
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

func (ctx *commandContext) HomeDir() (string, error) {
	if ctx.homeSeen {
		return ctx.home, ctx.homeErr
	}
	ctx.homeSeen = true
	ctx.home, ctx.homeErr = ctx.app.homeDir()
	return ctx.home, ctx.homeErr
}

func (ctx *commandContext) Services() (*bootstrap.Services, error) {
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

func (app *App) passthroughFlags(globals globalOptions, existing []string) []string {
	return passthroughFlags(globals, existing)
}

func (app *App) newScenarioLifecycleRunner(ctx *commandContext) (*lifecycle.Runner, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.LifecycleRunner()
}

func (app *App) newScenarioService(ctx *commandContext) (*orchestrator.Service, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Orchestrator(), nil
}

func (app *App) newResourceController(ctx *commandContext) (*resources.Controller, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Resources(), nil
}

func (app *App) newProjectController(ctx *commandContext) (*project.Controller, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Project(), nil
}

func (app *App) newMaintenanceController(ctx *commandContext) (*maintenance.Controller, error) {
	services, err := ctx.Services()
	if err != nil {
		return nil, err
	}
	return services.Maintenance(), nil
}

func (app *App) newProjectCommandService(ctx *commandContext) (projectapp.Service, error) {
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

func (app *App) runTopLevelSetup(ctx *commandContext, opts projectsetup.Options) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.runProjectSetup(ctx.Root, home, opts, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelBuild(ctx *commandContext) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.runProjectBuild(ctx.Root, home, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelDevelop(ctx *commandContext, opts projectsetup.Options) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.runProjectDevelop(ctx.Root, home, opts, ctx.Stdout, ctx.Stderr)
}

func scenarioExecutableFn() (string, error) {
	return os.Executable()
}

func (app *App) locateTestGenieCLI(root, home string) (string, error) {
	return scenarioexec.LocateTestGenieCLI(app.lookPath, root, home)
}

func (app *App) locateScenarioCompletenessCLI(root string) (string, error) {
	return scenarioexec.LocateScenarioCompletenessCLI(app.lookPath, root)
}

func (app *App) openScenarioURL(url string) error {
	return scenarioexec.OpenURL(app.lookPath, func(spec scenarioexec.SubprocessSpec) error {
		return app.runScenarioSubprocess(scenarioSubprocessSpec{
			name:   spec.Name,
			args:   append([]string(nil), spec.Args...),
			dir:    spec.Dir,
			env:    append([]string(nil), spec.Env...),
			stdin:  spec.Stdin,
			stdout: spec.Stdout,
			stderr: spec.Stderr,
		})
	}, url)
}

func (app *App) launchDetachedScenario(root string, globals globalOptions, args ...string) error {
	executable, err := app.scenarioExecutable()
	if err != nil {
		return err
	}
	return scenarioexec.LaunchDetachedScenario(executable, root, globals, app.commandEnv(root, globals), args...)
}
