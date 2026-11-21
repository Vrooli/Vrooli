package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

func main() {
	env := cliapp.StandardResourceEnv("cogvideo", cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name: "cogvideo", Version: "0.1.0", Description: "cogvideo resource CLI",
		SourceRootEnvVars: env.SourceRootEnvVars, ControlPlaneEnvVars: env.ControlPlaneEnvVars,
	})
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	app.SetCommands(app.StandardLifecycleCommands())
	if err := app.CLI.Run(os.Args[1:]); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
