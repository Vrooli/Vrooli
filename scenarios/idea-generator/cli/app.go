package main

import (
	"idea-generator/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "idea-generator"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
	// apiPrefix matches the scenario's API, which mounts endpoints at /api (not
	// the default /api/v1). Root-level /health and /status are served outside
	// this prefix and must be reached via core.GetRoot.
	apiPrefix = "/api"
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
		Description:      "Idea Generator CLI - AI-powered idea generation platform",
		DefaultAPIBase:   defaultAPIBase,
		APIPrefix:        apiPrefix,
		ExtraAPIEnvVars:  []string{"IDEA_GENERATOR_API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
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
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
