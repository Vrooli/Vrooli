package main

import (
	"agent-metareasoning-manager/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "agent-metareasoning-manager"
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
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:        appName,
		Version:     appVersion,
		Description: "Agent Metareasoning Manager CLI",
		// The API mounts all routes at the root (no /api/v1 prefix).
		APIPrefix:      "/",
		DefaultAPIBase: defaultAPIBase,
		ExtraAPIEnvVars: []string{
			"AGENT_METAREASONING_MANAGER_API_BASE",
			"AGENT_METAREASONING_MANAGER_API_URL",
			"API_BASE_URL",
		},
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
