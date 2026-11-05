// Package main provides the Test Scenario CLI.
//
// This CLI is intentionally minimal. test-scenario exists as a fixture for
// validating Vrooli's installer pipeline and security scanners; the CLI only
// needs to exist and install cleanly. Scenario-specific commands are
// deliberately omitted — cli-core's built-in `status`, `version`, and `help`
// are sufficient.
package main

import "github.com/vrooli/cli-core/cliapp"

const (
	appName        = "test-scenario"
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
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Test Scenario CLI (installer validation fixture)",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"TEST_SCENARIO_API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
	})
	if err != nil {
		return nil, err
	}
	return &App{core: core}, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
