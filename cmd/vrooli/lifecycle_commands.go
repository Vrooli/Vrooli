package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func runTopLevelLifecycleCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		showLifecycleHelp(ctx.Stdout)
		return nil
	}

	switch args[0] {
	case "protect":
		return runLifecycleProtectCommandWithApp(app, ctx, args[1:])
	default:
		return newUsageError(fmt.Sprintf("unknown lifecycle subcommand: %s", args[0]), "lifecycle")
	}
}

func runLifecycleProtectCommandWithApp(app *App, ctx *commandContext, args []string) error {
	commandArgs, err := parseLifecycleProtectArgs(args)
	if err != nil {
		return err
	}
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		return exitCodeError{code: 1, message: lifecycleProtectErrorMessage()}
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
			return exitCodeError{code: exitErr.ExitCode(), silent: true}
		}
		return err
	}
	return nil
}

func parseLifecycleProtectArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, commandHelpOnly(lifecycleProtectHelpText())
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return nil, commandHelpOnly(lifecycleProtectHelpText())
	}
	if args[0] != "--" {
		return nil, newUsageError("lifecycle protect requires '--' before the protected command", "lifecycle protect")
	}
	if len(args) == 1 {
		return nil, newUsageError("lifecycle protect requires a command after '--'", "lifecycle protect")
	}
	return append([]string(nil), args[1:]...), nil
}

func showLifecycleHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, lifecycleHelpText())
}

func lifecycleHelpText() string {
	return "Usage: vrooli lifecycle protect -- <command> [args...]\n"
}

func lifecycleProtectHelpText() string {
	return "Usage: vrooli lifecycle protect -- <command> [args...]\n"
}

func lifecycleProtectErrorMessage() string {
	return "This UI must be run through the Vrooli lifecycle system.\n\nInstead, use:\n   vrooli scenario start <scenario-name>\n\nThe lifecycle system provides environment variables, port allocation,\nand dependency management automatically. Direct execution is not supported.\n"
}
