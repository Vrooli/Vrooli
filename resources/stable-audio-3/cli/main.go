package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

const appName = "stable-audio-3"

var buildFingerprint, buildTimestamp, buildSourceRoot = "unknown", "unknown", ""

func main() {
	app, err := newApp()
	if err == nil {
		err = app.CLI.Run(os.Args[1:])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name: appName, Version: "0.1.0", Description: "Stable Audio 3 Medium local music resource",
		SourceRootEnvVars: env.SourceRootEnvVars, ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint: buildFingerprint, BuildTimestamp: buildTimestamp, BuildSourceRoot: buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	edge, err := activityedge.ForResource(appName)
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(append(app.StandardLifecycleCommands(), cliapp.CommandGroup{
		Title: "Capacity", Commands: []cliapp.Command{activityedge.Command(edge)},
	}), []cliapp.SubcommandGroup{pilotCapacityCommands(nil), {Name: "content", Description: "Generate and export locally owned audio", Subcommands: []cliapp.Command{
		{Name: "compose", Description: "Generate one take and export a verified WAV/receipt to a new directory", Usage: "resource-stable-audio-3 content compose --request FILE --out NEW_DIRECTORY [--timeout 330s] [--url LOOPBACK_URL]", Run: composeCommand},
	}}})
	return app, nil
}

func pilotCapacityCommands(exec func(context.Context, []string, string, ...string) error) cliapp.SubcommandGroup {
	return companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{
		Resource: appName, Exec: exec,
		Steps: []companion.LifecycleStep{{Label: "medium", Action: "restart", Description: "Apply the fixed Medium CUDA pilot profile through normal resource lifecycle"}},
	})
}
