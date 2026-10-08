package main

import (
	"fall-foliage-explorer/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "fall-foliage-explorer"
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
		Description:      "Fall Foliage Explorer CLI",
		DefaultAPIBase:   defaultAPIBase,
		APIPrefix:        "/api",
		HealthPath:       "/health",
		ExtraAPIEnvVars:  []string{"FALL_FOLIAGE_EXPLORER_API_URL", "API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups:    domains.CommandGroups,
		SubcommandGroups: domains.SubcommandGroups,
	})
	if err != nil {
		return nil, err
	}
	return &App{core: core}, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
