package main

import (
	"io"

	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

var (
	runProjectSetupFn   = projectsetup.RunSetup
	runProjectDevelopFn = projectsetup.RunDevelop
)

func runProjectSetupCommand(root string, args []string, stdout, stderr io.Writer) error {
	app := configuredApp()
	return app.runTopLevelSetup(&commandContext{
		Root:   root,
		Stdout: stdout,
		Stderr: stderr,
		app:    app,
	}, args)
}

func runProjectDevelopCommand(root string, args []string, stdout, stderr io.Writer) error {
	app := configuredApp()
	return app.runTopLevelDevelop(&commandContext{
		Root:   root,
		Stdout: stdout,
		Stderr: stderr,
		app:    app,
	}, args)
}
