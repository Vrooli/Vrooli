package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
)

func main() {
	env := cliapp.StandardResourceEnv("music-mir", cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name: "music-mir", Version: "0.1.0", Description: "music-mir resource CLI",
		SourceRootEnvVars: env.SourceRootEnvVars, ControlPlaneEnvVars: env.ControlPlaneEnvVars,
	})
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	edge, err := activityedge.ForResource("music-mir")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app.SetCommands(append(app.StandardLifecycleCommands(), cliapp.CommandGroup{
		Title: "Capacity",
		Commands: []cliapp.Command{activityedge.Command(edge)},
	}))
	if err := app.CLI.Run(os.Args[1:]); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
