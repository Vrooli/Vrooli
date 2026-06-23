package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacity"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacitysync"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/gateway"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policycmd"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName    = "ollama"
	appVersion = "0.1.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Ollama resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), ensure.CommandGroup(),
			cliapp.CommandGroup{Title: "Capacity", Commands: []cliapp.Command{capacitysync.Command(nil)}}),
		[]cliapp.SubcommandGroup{gateway.Commands(nil), capacity.Commands(nil), policycmd.Commands(nil)},
	)
	return app, nil
}
