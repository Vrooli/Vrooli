package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/resources"
)

type App struct {
	resolveSourceRoot     func() (string, error)
	homeDir               func() (string, error)
	isStale               func() bool
	checkStaleness        func() (buildinfo.StaleCheck, error)
	rebuildAndReexec      func([]string) error
	lookPath              func(string) (string, error)
	execCommand           func(commandSpec) error
	newLogger             func(globalOptions, io.Writer) (*slog.Logger, func())
	runProjectSetup       func(string, string, []string, io.Writer, io.Writer) error
	runProjectDevelop     func(string, string, []string, io.Writer, io.Writer) error
	runScenarioSubprocess func(scenarioSubprocessSpec) error
	scenarioExecutable    func() (string, error)
}

type commandContext struct {
	Root     string
	Globals  globalOptions
	Stdout   io.Writer
	Stderr   io.Writer
	Logger   *slog.Logger
	app      *App
	home     string
	homeErr  error
	homeSeen bool
}

func configuredApp() *App {
	return &App{
		resolveSourceRoot:     resolveSourceRootFn,
		homeDir:               config.HomeDir,
		isStale:               isStaleFn,
		checkStaleness:        checkStalenessFn,
		rebuildAndReexec:      rebuildAndReexecFn,
		lookPath:              lookPathFn,
		execCommand:           execCommandFn,
		newLogger:             newLoggerFn,
		runProjectSetup:       runProjectSetupFn,
		runProjectDevelop:     runProjectDevelopFn,
		runScenarioSubprocess: runScenarioSubprocessFn,
		scenarioExecutable:    scenarioExecutableFn,
	}
}

func (app *App) Run(args []string, stdout, stderr io.Writer) int {
	parsed, err := parseArgs(args)
	if err != nil {
		printErrorWithContext(stderr, newErrorWithCategory(err, errorCategoryUsage, "Use --help for available commands", nil))
		return 1
	}
	parsed.globals, parsed.args = consumeInlineGlobalFlags(parsed.globals, parsed.args)
	logger, restoreLogger := app.newLogger(parsed.globals, stderr)
	defer restoreLogger()
	debugLog(logger, "Parsed command", "command", parsed.command, "args", parsed.args, "json", parsed.globals.json, "verbose", parsed.globals.verbose)

	root, err := app.resolveRoot()
	if err != nil {
		printErrorWithContext(stderr, newErrorWithCategory(err, errorCategoryEnvironment, "Run from a Vrooli repository root or set VROOLI_SOURCE_ROOT", nil))
		return 1
	}
	debugLog(logger, "Resolved root", "path", root)
	primeRootEnv(root)

	ctx := &commandContext{
		Root:    root,
		Globals: parsed.globals,
		Stdout:  stdout,
		Stderr:  stderr,
		Logger:  logger,
		app:     app,
	}

	if parsed.globals.noColor {
		_ = os.Setenv("NO_COLOR", "1")
		debugLog(logger, "NO_COLOR requested by user flags")
	}

	if !parsed.globals.noStaleCheck {
		stale, err := app.shouldRebuild()
		if err != nil {
			printErrorWithContext(stderr, newErrorWithCategory(
				fmt.Errorf("stale binary check failed: %w", err),
				errorCategoryRuntime,
				"Use --no-stale-check for local experiments",
				nil,
			))
			return 1
		}
		if stale {
			debugLog(logger, "Stale check triggered")
			if err := app.rebuildAndReexec(args); err != nil {
				printErrorWithContext(stderr, newErrorWithCategory(
					fmt.Errorf("stale binary check failed: %w", err),
					errorCategoryRuntime,
					"Use --no-stale-check for local experiments",
					nil,
				))
				return 1
			}
			debugLog(logger, "Rebuilt command binary and re-executed")
			return 0
		}
	}

	if err := dispatch(app, ctx, parsed); err != nil {
		printErrorWithContext(stderr, err)
		return exitCode(err)
	}
	return 0
}

func (app *App) shouldRebuild() (bool, error) {
	if app.checkStaleness != nil {
		status, err := app.checkStaleness()
		if err != nil {
			return false, err
		}
		return status.Stale, nil
	}
	if app.isStale == nil {
		return false, nil
	}
	return app.isStale(), nil
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
	if globals.noColor {
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

func (app *App) passthroughFlags(globals globalOptions, existing []string) []string {
	return passthroughFlags(globals, existing)
}

func (app *App) newScenarioLifecycleRunner(ctx *commandContext) (*lifecycle.Runner, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return nil, err
	}
	return lifecycle.NewRunner(ctx.Root, home, ctx.Stdout, ctx.Stderr)
}

func (app *App) newScenarioService(ctx *commandContext) (*orchestrator.Service, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return nil, err
	}
	return orchestrator.New(ctx.Root, home, ctx.Stdout, ctx.Stderr), nil
}

func (app *App) newResourceController(ctx *commandContext) (*resources.Controller, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return nil, err
	}
	return resources.NewController(ctx.Root, home), nil
}

func (app *App) runTopLevelSetup(ctx *commandContext, args []string) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.runProjectSetup(ctx.Root, home, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelDevelop(ctx *commandContext, args []string) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	return app.runProjectDevelop(ctx.Root, home, args, ctx.Stdout, ctx.Stderr)
}
