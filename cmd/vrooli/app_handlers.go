package main

import (
	"errors"
	"io"
	"os"
	"os/exec"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	packageapp "github.com/vrooli/vrooli/internal/app/package"
	"github.com/vrooli/vrooli/internal/cli/contracthandlers"
	"github.com/vrooli/vrooli/internal/cli/packagehandlers"
	projectapp "github.com/vrooli/vrooli/internal/app/project"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/cli/resourcehandlers"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/scenariohandlers"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

func commandStdout(ctx *commandContext) io.Writer {
	return ctx.Stdout
}

func runInfoTopLevelCommand(app *App, ctx *commandContext, args []string) error {
	req, err := topcli.ParseInfoRequest(args)
	if err != nil {
		if helpErr, ok := err.(interface{ HelpText() string }); ok {
			_, _ = io.WriteString(ctx.Stdout, helpErr.HelpText())
			if text := helpErr.HelpText(); text == "" || text[len(text)-1] != '\n' {
				_, _ = io.WriteString(ctx.Stdout, "\n")
			}
			return nil
		}
		return err
	}
	format, err := formatFromJSON(ctx.Globals.JSON)
	if err != nil {
		return err
	}
	return topcli.RunInfo(ctx.Root, format, req, ctx.Stdout, ctx.Stderr)
}

func runScenarioRootCommand(app *App, ctx *commandContext, args []string) error {
	return scenariohandlers.RootHandler(commandStdout, ctx.app.registry.ScenarioHandler, ctx.app.registry.SuggestScenario)(ctx, args)
}

func runLifecycleProtectCommand(app *App, ctx *commandContext, args []string) error {
	commandArgs, err := projectcli.ParseLifecycleProtectArgs(args)
	if err != nil {
		return err
	}
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		return rootcli.ExitCodeError{Code: 1, Message: projectcli.LifecycleProtectErrorMessage()}
	}

	if err := app.runScenarioSubprocess(scenarioSubprocessSpec{
		name:   commandArgs[0],
		args:   commandArgs[1:],
		dir:    ".",
		env:    os.Environ(),
		stdin:  os.Stdin,
		stdout: ctx.Stdout,
		stderr: ctx.Stderr,
	}); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return rootcli.ExitCodeError{Code: exitErr.ExitCode(), Silent_: true}
		}
		return err
	}
	return nil
}

func runCleanupCommand(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	_, ctx := newConfiguredCommandContext(root, parsed.Globals, stdout, stderr)
	return projectcli.CleanupHandler(commandStdout,
		func(ctx *commandContext, args []string) error {
			return buildTopLevelHandlerMap()[topcli.CommandOrphans](ctx, args)
		},
		func(ctx *commandContext, args []string) error {
			return buildTopLevelHandlerMap()[topcli.CommandLocks](ctx, args)
		},
	)(ctx, parsed.Args)
}

func projectOutputFormat(ctx *commandContext) (cliout.Format, error) {
	return parseOutputFormat(ctx.Globals)
}

func runProjectPhaseFromContext(ctx *commandContext, phase string, args []string) error {
	controller, err := ctx.app.newProjectController(ctx)
	if err != nil {
		return err
	}
	return controller.RunProjectPhase(phase, args)
}

func buildTopLevelHandlerMap() map[topcli.CommandID]rootcli.Handler[*commandContext] {
	handlers := map[topcli.CommandID]rootcli.Handler[*commandContext]{
		topcli.CommandSetup: projectcli.SetupHandler(commandStdout, func(ctx *commandContext, opts projectsetup.Options) error { return ctx.app.runTopLevelSetup(ctx, opts) }),
		topcli.CommandDevelop: projectcli.DevelopHandler(commandStdout, func(ctx *commandContext, opts projectsetup.Options) error {
			return ctx.app.runTopLevelDevelop(ctx, opts)
		}),
		topcli.CommandBuild: projectcli.BuildHandler(commandStdout, func(ctx *commandContext) error { return ctx.app.runTopLevelBuild(ctx) }),
		topcli.CommandClean: projectcli.ProjectPhaseHandler(commandStdout, "clean", func(ctx *commandContext, args []string) error { return runProjectPhaseFromContext(ctx, "clean", args) }),
		topcli.CommandStatus: projectcli.StatusHandler(commandStdout, projectOutputFormat, func(ctx *commandContext, req projectcli.StatusRequest) (project.StatusReport, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return project.StatusReport{}, err
			}
			return command.Status(projectapp.StatusRequest{ResourcesOnly: req.ResourcesOnly, ScenariosOnly: req.ScenariosOnly, Fast: req.Fast})
		}),
		topcli.CommandStop: projectcli.StopHandler(commandStdout, projectOutputFormat, func(ctx *commandContext, req projectcli.StopRequest) (control.StopReport, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return control.StopReport{}, err
			}
			return command.Stop(projectapp.StopRequest{Targets: req.Targets})
		}),
		topcli.CommandBackup: projectcli.ProjectPhaseHandler(commandStdout, "backup", func(ctx *commandContext, args []string) error { return runProjectPhaseFromContext(ctx, "backup", args) }),
		topcli.CommandRestore: projectcli.ProjectPhaseHandler(commandStdout, "restore", func(ctx *commandContext, args []string) error {
			return runProjectPhaseFromContext(ctx, "restore", args)
		}),
		topcli.CommandInfo: func(ctx *commandContext, args []string) error {
			return runInfoTopLevelCommand(ctx.app, ctx, args)
		},
		topcli.CommandScenario: func(ctx *commandContext, args []string) error {
			return runScenarioRootCommand(ctx.app, ctx, args)
		},
		topcli.CommandPackage: packagehandlers.RootHandler(packagehandlers.HandlerDeps[*commandContext]{
			Stdout:       commandStdout,
			Stderr:       func(ctx *commandContext) io.Writer { return ctx.Stderr },
			Root:         func(ctx *commandContext) string { return ctx.Root },
			OutputFormat: projectOutputFormat,
			ScenarioOperations: func(ctx *commandContext) (packageapp.ScenarioRuntime, error) {
				return ctx.app.newScenarioService(ctx)
			},
			LifecycleRunner: func(ctx *commandContext) (packageapp.ScenarioPhaseRunner, error) {
				return ctx.app.newScenarioLifecycleRunner(ctx)
			},
		}),
		topcli.CommandResource: resourcehandlers.RootHandler(resourcehandlers.HandlerDeps[*commandContext]{
			Stdout:       commandStdout,
			Stderr:       func(ctx *commandContext) io.Writer { return ctx.Stderr },
			Globals:      func(ctx *commandContext) rootcli.GlobalOptions { return ctx.Globals },
			OutputFormat: projectOutputFormat,
			ResourceController: func(ctx *commandContext) (*resources.Controller, error) {
				return ctx.app.newResourceController(ctx)
			},
		}),
		topcli.CommandDoctor: projectcli.DoctorHandler(commandStdout, projectOutputFormat, func(ctx *commandContext) (project.DoctorReport, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return project.DoctorReport{}, err
			}
			return command.Doctor()
		}),
		topcli.CommandOrphans: projectcli.OrphansHandler(commandStdout, projectOutputFormat, func(ctx *commandContext, req projectcli.OrphansRequest) (projectcli.OrphansResponse, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return projectcli.OrphansResponse{}, err
			}
			resp, err := command.Orphans(projectapp.OrphansRequest{Kill: req.Kill})
			if err != nil {
				return projectcli.OrphansResponse{}, err
			}
			return projectcli.OrphansResponse{List: resp.List, KillReport: resp.KillReport}, nil
		}),
		topcli.CommandLocks: projectcli.LocksHandler(commandStdout, projectOutputFormat, func(ctx *commandContext, req projectcli.LocksRequest) (projectcli.LocksResponse, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return projectcli.LocksResponse{}, err
			}
			resp, err := command.Locks(projectapp.LocksRequest{Clean: req.Clean})
			if err != nil {
				return projectcli.LocksResponse{}, err
			}
			return projectcli.LocksResponse{List: resp.List, CleanReport: resp.CleanReport}, nil
		}),
		topcli.CommandDiagnosePort: projectcli.DiagnosePortHandler(commandStdout, projectOutputFormat, func(ctx *commandContext, req projectcli.DiagnosePortRequest) (maintenance.PortDiagnostic, error) {
			command, err := ctx.app.newProjectCommandService(ctx)
			if err != nil {
				return maintenance.PortDiagnostic{}, err
			}
			return command.DiagnosePort(projectapp.DiagnosePortRequest{Port: req.Port, ScenarioName: req.ScenarioName})
		}),
		topcli.CommandContract: contracthandlers.RootHandler(contracthandlers.HandlerDeps[*commandContext]{
			Stdout:       commandStdout,
			OutputFormat: projectOutputFormat,
			Service: func(ctx *commandContext) contractapp.Service {
				return contractapp.NewDefaultService()
			},
		}),
		topcli.CommandLifecycle: projectcli.LifecycleHandler(commandStdout, func(ctx *commandContext, args []string) error { return runLifecycleProtectCommand(ctx.app, ctx, args) }),
	}
	handlers[topcli.CommandCleanup] = projectcli.CleanupHandler(commandStdout, handlers[topcli.CommandOrphans], handlers[topcli.CommandLocks])
	return handlers
}

func buildScenarioHandlerMap() map[scenariocli.CommandID]rootcli.Handler[*commandContext] {
	return scenariohandlers.BuildHandlers(scenariohandlers.HandlerDeps[*commandContext]{
		Stdout:       commandStdout,
		Stderr:       func(ctx *commandContext) io.Writer { return ctx.Stderr },
		Root:         func(ctx *commandContext) string { return ctx.Root },
		Globals:      func(ctx *commandContext) rootcli.GlobalOptions { return ctx.Globals },
		OutputFormat: projectOutputFormat,
		HomeDir:      func(ctx *commandContext) (string, error) { return ctx.HomeDir() },
		ScenarioOperations: func(ctx *commandContext) (scenarioapp.ScenarioOperations, error) {
			return ctx.app.newScenarioService(ctx)
		},
		LifecycleRunner: func(ctx *commandContext) (scenarioapp.PhaseRunner, error) {
			return ctx.app.newScenarioLifecycleRunner(ctx)
		},
		EnvValidator: func(ctx *commandContext) (scenarioapp.EnvironmentValidator, error) {
			services, err := ctx.Services()
			if err != nil {
				return nil, err
			}
			return services.Resources(), nil
		},
		OpenURL: func(ctx *commandContext, url string) error {
			return ctx.app.openScenarioURL(url)
		},
		LaunchDetached: func(ctx *commandContext, args ...string) error {
			return ctx.app.launchDetachedScenario(ctx.Root, ctx.Globals, args...)
		},
		RunSubprocess: func(ctx *commandContext, spec scenarioexec.SubprocessSpec) error {
			return ctx.app.runScenarioSubprocess(scenarioSubprocessSpec{
				name:   spec.Name,
				args:   append([]string(nil), spec.Args...),
				dir:    spec.Dir,
				env:    append([]string(nil), spec.Env...),
				stdin:  spec.Stdin,
				stdout: spec.Stdout,
				stderr: spec.Stderr,
			})
		},
		LocateTestGenieCLI: func(ctx *commandContext) (string, error) {
			home, err := ctx.HomeDir()
			if err != nil {
				return "", err
			}
			return ctx.app.locateTestGenieCLI(ctx.Root, home)
		},
		LocateCompleteCLI: func(ctx *commandContext) (string, error) {
			return ctx.app.locateScenarioCompletenessCLI(ctx.Root)
		},
		CommandEnv: func(ctx *commandContext) []string {
			return ctx.app.commandEnv(ctx.Root, ctx.Globals)
		},
	})
}
