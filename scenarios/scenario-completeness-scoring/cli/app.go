package main

import (
	"scenario-completeness-scoring/cli/domains"
	"scenario-completeness-scoring/cli/internal/format"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "scenario-completeness-scoring"
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
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Scenario Completeness Scoring CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"SCENARIO_COMPLETENESS_SCORING_API_URL", "SCORING_API_BASE", "API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		OnColor:          format.SetColorEnabled,
		CommandGroups:    domains.CommandGroups,
		SubcommandGroups: domains.SubcommandGroups,
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
