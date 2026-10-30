package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/capacitysync"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/commands"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

const (
	appName    = "ollaya"
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
		Description:         "Ollaya decision-model resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	handlers := commands.Default()
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{
			Title: "Capacity",
			Commands: []cliapp.Command{{
				Name:        "capacity-sync",
				Description: "Keep the Ollaya VRAM capacity claim alive",
				Usage:       "resource-ollaya capacity-sync [--interval 15s] [--once]",
				Run:         capacitysync.Command(nil),
			}},
		}),
		[]cliapp.SubcommandGroup{
			commands.GatewayCommands(handlers),
			commands.PolicyCommands(handlers),
			commands.ModelCommands(handlers),
			companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{Resource: appName, Steps: companion.DeviceSteps("gpu")}),
		},
	)
	return app, nil
}
