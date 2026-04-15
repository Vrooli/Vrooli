package main

import (
	"io"
	"os"

	"tunnel-manager/cli/domains"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "tunnel-manager"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func boolPtr(v bool) *bool { return &v }

type App struct {
	core  *cliapp.ScenarioApp
	Stdin io.Reader
}

func NewApp() (*App, error) {
	app := &App{Stdin: os.Stdin}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "Tunnel Manager CLI",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		IncludeStatusCommand: boolPtr(false),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return domains.CommandGroups(app.dependencies())
		},
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			app.core = core
			return domains.SubcommandGroups(app.dependencies())
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		Core: func() *cliapp.ScenarioApp {
			return a.core
		},
		Stdin: func() io.Reader {
			if a.Stdin == nil {
				return os.Stdin
			}
			return a.Stdin
		},
	}
}

func (a *App) commandGroups() []cliapp.CommandGroup {
	return domains.CommandGroups(a.dependencies())
}

func (a *App) subcommandGroups() []cliapp.SubcommandGroup {
	return domains.SubcommandGroups(a.dependencies())
}
