package main

import (
	"time"

	"vrooli-autoheal/cli/domains"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "vrooli-autoheal"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
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

	disableStatus := false
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "Vrooli Autoheal CLI",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL", "VROOLI_AUTOHEAL_API_BASE"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		// A full fleet tick is intentionally bounded by the API's six-minute
		// safety valve and normally completes in a few minutes. Keep the CLI
		// timeout above that bound so a slow but live supervisor is not reported
		// as dead by its own operator command.
		DefaultHTTPTimeout: 7 * time.Minute,
		IncludeStatusCommand: &disableStatus,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			return domains.CommandGroups(core, support.Dependencies{
				RunLoop:      app.runLoop,
				DiagnosePort: app.diagnosePort,
			})
		},
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			return domains.SubcommandGroups(core, support.Dependencies{})
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
