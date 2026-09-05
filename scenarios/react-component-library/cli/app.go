package main

import (
	"react-component-library/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "react-component-library"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func promotedStartHere() []cliapp.Command {
	return []cliapp.Command{{Name: "asset check", Description: "Run the one-asset validation loop"}, {Name: "components draft-begin", Description: "Start a governed asset draft"}, {Name: "components content-set", Description: "Set one draft file"}, {Name: "components draft-publish", Description: "Publish a validated draft"}, {Name: "components ingest", Description: "Ingest a component asset"}, {Name: "components test", Description: "Run component tests"}, {Name: "catalog build", Description: "Build derived catalog artifacts"}, {Name: "adoptions preflight", Description: "Check an adoption before linking"}, {Name: "adoptions link", Description: "Link an adoption into a workbench"}, {Name: "versions list", Description: "List recorded component versions"}, {Name: "versions reap", Description: "Preview and confirm safe version retirement"}, {Name: "versions doctor", Description: "Inspect and repair version state"}, {Name: "workflows start", Description: "Start durable assisted work"}}
}

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	// Local manifest bindings use the command name as their schema-level key.
	app := &App{}
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		groups, err := domains.SubcommandGroups(core, manifestBytes)
		if err != nil {
			panic(err)
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "React Component Library CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		SubcommandGroups: subcommandGroups,
		StartHere:        promotedStartHere(),
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
