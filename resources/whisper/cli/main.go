package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"

	"github.com/vrooli/vrooli/resources/whisper/cli/internal/activityproxy"
	"github.com/vrooli/vrooli/resources/whisper/cli/internal/recommend"
)

const (
	appName    = "whisper"
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
		Description:         "Whisper resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	groups := app.StandardLifecycleCommands()
	groups = append(groups, cliapp.CommandGroup{
		Title:    "Capability",
		Commands: []cliapp.Command{recommend.Commands(nil), recommend.DegradeCommand(nil), activityproxy.Command(nil)},
	})
	app.SetCommands(groups)
	return app, nil
}
