package main

import (
	"campaign-content-studio/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "campaign-content-studio"
	appVersion     = "2.0.0"
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
		Description:      "Campaign Content Studio CLI",
		DefaultAPIBase:   defaultAPIBase,
		APIPrefix:        "/",
		HealthPath:       "/health",
		ExtraAPIEnvVars:  []string{"CAMPAIGN_CONTENT_STUDIO_API_URL", "API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
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
