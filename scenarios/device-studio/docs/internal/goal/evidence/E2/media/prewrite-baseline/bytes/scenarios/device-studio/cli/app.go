package main

import (
	"device-studio/cli/domains"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "device-studio"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
	testModeEnv    = "DEVICE_STUDIO_TEST_MODE"
	testModeHeader = "X-Vrooli-Test-Mode"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	app := &App{}
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		groups, err := domains.SubcommandGroups(core, manifestBytes)
		if err != nil {
			// Manifest parse / binding wiring is a programmer error caught
			// at NewApp time; surface it as a panic so misconfigured builds
			// fail loudly during the first CLI invocation rather than after
			// a user actually runs a command.
			panic(err)
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Device Studio CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups:    domains.CommandGroups,
		SubcommandGroups: subcommandGroups,
	})
	if err != nil {
		return nil, err
	}
	core.HTTPClient.SetHeaderSource(func() map[string]string {
		if os.Getenv(testModeEnv) == "1" {
			return map[string]string{testModeHeader: "1"}
		}
		return nil
	})
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
